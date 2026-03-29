package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/llm"
	"github.com/andybarilla/rook/internal/prompt"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// ensureRookGitignore creates .rook/.gitignore with .cache/ entry if it doesn't exist
func ensureRookGitignore(rookDir string) error {
	gitignorePath := filepath.Join(rookDir, ".gitignore")
	// Check if it already exists
	if _, err := os.Stat(gitignorePath); err == nil {
		return nil
	}
	// Create .rook directory if needed
	if err := os.MkdirAll(rookDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(gitignorePath, []byte(".cache/\n"), 0644)
}

func newInitCmd() *cobra.Command {
	var warns warnings
	var nonInteractive bool
	var force bool
	var add bool

	cmd := &cobra.Command{
		Use:   "init <path>",
		Short: "Initialize a workspace from a project directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			manifestPath := filepath.Join(dir, "rook.yaml")
			manifestExists := false
			if _, err := os.Stat(manifestPath); err == nil {
				manifestExists = true
			}

			// Auto-detect non-interactive mode when stdin is not a terminal
			if !nonInteractive && !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
				nonInteractive = true
			}

			// Handle existing manifest
			if manifestExists {
				switch {
				case force:
					os.Remove(manifestPath)
					// Fall through to discovery below
				case add:
					return addServices(dir, manifestPath, nonInteractive, &warns)
				default:
					// Existing rook.yaml without flags: just register it (skip discovery)
					cctx, err := newCLIContext()
					if err != nil {
						return err
					}
					if err := cctx.initFromManifest(dir); err != nil {
						return err
					}
					warns.print(os.Stderr)
					return nil
				}
			}

			var services map[string]workspace.Service
			var groups map[string][]string

			if nonInteractive {
				services, groups, err = discoverNonInteractive(dir, &warns)
			} else {
				p := prompt.NewStdinPrompter(os.Stdin, os.Stdout)
				services, groups, err = discoverInteractive(dir, p, &warns)
			}
			if err != nil {
				return err
			}

			if len(services) == 0 {
				return fmt.Errorf("no services discovered in %s — create a rook.yaml manually", dir)
			}

			// Copy and sanitize devcontainer scripts
			services = copyDevcontainerScripts(dir, services, &warns)

			wsName := filepath.Base(dir)
			m := &workspace.Manifest{
				Name:     wsName,
				Type:     workspace.TypeSingle,
				Services: services,
				Groups:   groups,
			}
			if err := workspace.WriteManifest(manifestPath, m); err != nil {
				return fmt.Errorf("writing manifest: %w", err)
			}
			fmt.Printf("Generated %s\n", manifestPath)

			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			if err := cctx.initFromManifest(dir); err != nil {
				return err
			}
			warns.print(os.Stderr)
			return nil
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Skip interactive prompts (legacy behavior)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing rook.yaml and re-initialize")
	cmd.Flags().BoolVar(&add, "add", false, "Add services to an existing rook.yaml")

	return cmd
}

// discoverNonInteractive preserves the original init behavior: first compose file wins, no prompts.
func discoverNonInteractive(dir string, warns *warnings) (map[string]workspace.Service, map[string][]string, error) {
	discoverers := []discovery.Discoverer{
		discovery.NewComposeDiscoverer(),
		discovery.NewDevcontainerDiscoverer(),
		discovery.NewMiseDiscoverer(),
	}
	result, err := discovery.RunAll(dir, discoverers)
	if err != nil {
		return nil, nil, fmt.Errorf("discovery failed: %w", err)
	}

	if len(result.Services) > 0 {
		fmt.Printf("Discovered from %s:\n", result.Source)
		for name, svc := range result.Services {
			if svc.IsContainer() {
				fmt.Printf("  %s (container: %s)\n", name, svc.Image)
			} else {
				fmt.Printf("  %s (process)\n", name)
			}
		}
	}

	return result.Services, result.Groups, nil
}

// discoverInteractive runs the interactive init flow.
func discoverInteractive(dir string, p prompt.Prompter, warns *warnings) (map[string]workspace.Service, map[string][]string, error) {
	services := make(map[string]workspace.Service)
	groups := make(map[string][]string)

	// 1. Scan for compose files
	composeFiles := discovery.ScanComposeFiles(dir)

	// 2. Scan for local signals
	localSignals := discovery.ScanLocalSignals(dir)

	// 3. Check for devcontainer
	devDiscoverer := discovery.NewDevcontainerDiscoverer()
	hasDevcontainer := devDiscoverer.Detect(dir)

	// 4. Check for mise
	miseDiscoverer := discovery.NewMiseDiscoverer()
	hasMise := miseDiscoverer.Detect(dir)

	// Display what was found
	if len(composeFiles) > 0 {
		fmt.Println("\nFound compose files:")
		options := make([]string, len(composeFiles))
		for i, cf := range composeFiles {
			options[i] = fmt.Sprintf("%s (%s)", cf.RelPath, strings.Join(cf.ServiceNames, ", "))
		}

		selected, err := p.Select("\nWhich compose file(s) should rook use?", options)
		if err != nil {
			return nil, nil, err
		}

		// Parse selected compose files
		d := discovery.NewComposeDiscoverer()
		for _, idx := range selected {
			cf := composeFiles[idx]
			result, err := d.DiscoverFile(dir, cf.Path)
			if err != nil {
				fmt.Printf("  Warning: failed to parse %s: %v\n", cf.RelPath, err)
				continue
			}
			// Merge services, asking about conflicts
			for name, svc := range result.Services {
				if existing, conflict := services[name]; conflict {
					fmt.Printf("\n  Service %q defined in multiple compose files.\n", name)
					fmt.Printf("    Existing: image=%s\n", existing.Image)
					fmt.Printf("    New:      image=%s\n", svc.Image)
					keep, err := p.Confirm(fmt.Sprintf("  Replace %q with definition from %s?", name, cf.RelPath), false)
					if err != nil {
						return nil, nil, err
					}
					if !keep {
						continue
					}
				}
				services[name] = svc
			}
			for name, group := range result.Groups {
				groups[name] = group
			}
		}
	}

	// Show other detections
	if hasDevcontainer || hasMise || len(localSignals) > 0 {
		fmt.Println("\nAlso detected:")
		if hasDevcontainer {
			fmt.Println("  - .devcontainer/devcontainer.json")
		}
		if hasMise {
			fmt.Println("  - mise/tool-versions")
		}
		for _, sig := range localSignals {
			detail := sig.Type
			if sig.Command != "" {
				detail += ": " + sig.Command
			}
			fmt.Printf("  - %s (%s)\n", sig.File, detail)
		}
	}

	// Devcontainer Dockerfile analysis (LLM)
	if hasDevcontainer {
		dfPath := filepath.Join(dir, ".devcontainer", "Dockerfile")
		if _, err := os.Stat(dfPath); err == nil {
			dfContent, err := os.ReadFile(dfPath)
			if err == nil {
				// Parse Dockerfile for structured signals first
				dfSignals := discovery.ParseDockerfile(dfContent)
				if len(dfSignals.InferredDeps) > 0 {
					fmt.Printf("\n  Dockerfile analysis (structural): inferred deps: %s\n", strings.Join(dfSignals.InferredDeps, ", "))
				}

				// Offer LLM analysis
				provider, providerErr := llm.NewAnthropicProvider()
				if providerErr == nil {
					analyzeLLM, err := p.Confirm("\nAnalyze Dockerfile with LLM to suggest service breakdown?", false)
					if err != nil {
						return nil, nil, err
					}
					if analyzeLLM {
						if err := runLLMDockerfileAnalysis(dir, provider, p, string(dfContent), services); err != nil {
							fmt.Printf("  LLM analysis failed: %v\n", err)
						}
					}
				}
			}
		}
	}

	// Local service addition
	if len(localSignals) > 0 {
		addLocal, err := p.Confirm("\nAdd local services?", true)
		if err != nil {
			return nil, nil, err
		}
		if addLocal {
			if err := addLocalServicesInteractive(p, localSignals, services); err != nil {
				return nil, nil, err
			}
		}
	} else {
		addManual, err := p.Confirm("\nAdd a local service manually?", false)
		if err != nil {
			return nil, nil, err
		}
		if addManual {
			if err := addManualServices(p, services); err != nil {
				return nil, nil, err
			}
		}
	}

	return services, groups, nil
}

// addLocalServicesInteractive presents detected local signals and lets the user add them.
func addLocalServicesInteractive(p prompt.Prompter, signals []discovery.LocalSignal, services map[string]workspace.Service) error {
	for _, sig := range signals {
		fmt.Printf("\n  Detected: %s (%s)\n", sig.File, sig.Type)
		name, err := p.Input("  Service name", sig.Name)
		if err != nil {
			return err
		}
		if name == "" {
			continue
		}

		command := sig.Command
		if command == "" {
			command, err = p.Input("  Command", "")
			if err != nil {
				return err
			}
		} else {
			command, err = p.Input("  Command", command)
			if err != nil {
				return err
			}
		}
		if command == "" {
			fmt.Println("  Skipping — no command specified")
			continue
		}

		deps, err := p.InputList("  Depends on")
		if err != nil {
			return err
		}

		services[name] = workspace.Service{
			Command:   command,
			DependsOn: deps,
		}
		fmt.Printf("  Added service %q: %s\n", name, command)
	}

	// Offer to add more manually
	return addManualServices(p, services)
}

// addManualServices lets the user add services by typing name/command/deps.
func addManualServices(p prompt.Prompter, services map[string]workspace.Service) error {
	for {
		addMore, err := p.Confirm("\n  Add another service?", false)
		if err != nil {
			return err
		}
		if !addMore {
			break
		}

		name, err := p.Input("  Service name", "")
		if err != nil {
			return err
		}
		if name == "" {
			break
		}

		command, err := p.Input("  Command", "")
		if err != nil {
			return err
		}
		if command == "" {
			continue
		}

		deps, err := p.InputList("  Depends on")
		if err != nil {
			return err
		}

		services[name] = workspace.Service{
			Command:   command,
			DependsOn: deps,
		}
		fmt.Printf("  Added service %q: %s\n", name, command)
	}
	return nil
}

// runLLMDockerfileAnalysis sends the Dockerfile to the LLM and lets the user accept suggestions.
func runLLMDockerfileAnalysis(dir string, provider llm.Provider, p prompt.Prompter, dfContent string, services map[string]workspace.Service) error {
	// Build file tree (top 2 levels)
	fileTree := buildFileTree(dir, 2)

	// Read start script if present
	var startScript string
	for _, name := range []string{"start.sh", "post-create.sh", "setup.sh"} {
		scriptPath := filepath.Join(dir, ".devcontainer", name)
		if data, err := os.ReadFile(scriptPath); err == nil {
			startScript = string(data)
			break
		}
	}

	fmt.Println("  Analyzing Dockerfile...")
	suggestions, err := llm.AnalyzeDockerfile(nil, provider, dfContent, startScript, fileTree)
	if err != nil {
		return err
	}

	if len(suggestions) == 0 {
		fmt.Println("  No services suggested.")
		return nil
	}

	fmt.Println("\n  LLM suggested services:")
	for _, s := range suggestions {
		if s.Type == "container" {
			fmt.Printf("    %s (container: %s) — %s\n", s.Name, s.Image, s.Reasoning)
		} else {
			fmt.Printf("    %s (process: %s) — %s\n", s.Name, s.Command, s.Reasoning)
		}
	}

	accept, err := p.Confirm("\n  Accept these suggestions?", true)
	if err != nil {
		return err
	}
	if !accept {
		return nil
	}

	for _, s := range suggestions {
		svc := workspace.Service{
			DependsOn: s.DependsOn,
		}
		if s.Type == "container" {
			svc.Image = s.Image
			svc.Ports = s.Ports
		} else {
			svc.Command = s.Command
		}
		services[s.Name] = svc
	}

	return nil
}

// addServices handles the --add flag: loads existing manifest and adds new services.
func addServices(dir, manifestPath string, nonInteractive bool, warns *warnings) error {
	m, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing existing manifest: %w", err)
	}

	if nonInteractive {
		return fmt.Errorf("--add requires interactive mode")
	}

	services := m.Services
	if services == nil {
		services = make(map[string]workspace.Service)
	}

	p := prompt.NewStdinPrompter(os.Stdin, os.Stdout)

	// Show existing services
	fmt.Println("Existing services:")
	for name, svc := range services {
		if svc.IsContainer() {
			fmt.Printf("  %s (container: %s)\n", name, svc.Image)
		} else {
			fmt.Printf("  %s (process: %s)\n", name, svc.Command)
		}
	}

	// Scan for local signals
	localSignals := discovery.ScanLocalSignals(dir)

	// Filter out signals that match existing service names
	var newSignals []discovery.LocalSignal
	for _, sig := range localSignals {
		if _, exists := services[sig.Name]; !exists {
			newSignals = append(newSignals, sig)
		}
	}

	if len(newSignals) > 0 {
		fmt.Println("\nDetected new local service signals:")
		if err := addLocalServicesInteractive(p, newSignals, services); err != nil {
			return err
		}
	} else {
		if err := addManualServices(p, services); err != nil {
			return err
		}
	}

	m.Services = services
	if err := workspace.WriteManifest(manifestPath, m); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	fmt.Printf("Updated %s\n", manifestPath)

	// Re-register to allocate ports for new services
	cctx, err := newCLIContext()
	if err != nil {
		return err
	}
	if err := cctx.initFromManifest(dir); err != nil {
		return err
	}
	warns.print(os.Stderr)
	return nil
}

// copyDevcontainerScripts copies and sanitizes devcontainer scripts to .rook/scripts/.
func copyDevcontainerScripts(dir string, services map[string]workspace.Service, warns *warnings) map[string]workspace.Service {
	rookDir := filepath.Join(dir, ".rook")
	for name, svc := range services {
		if svc.Command == "" || !strings.Contains(svc.Command, ".devcontainer/") {
			continue
		}

		// Extract the script path from the command
		scriptPath := ""
		for _, part := range strings.Fields(svc.Command) {
			if strings.Contains(part, ".devcontainer/") {
				scriptPath = part
				break
			}
		}
		if scriptPath == "" {
			continue
		}

		// Resolve to a host path
		var hostScript string
		rel := scriptPath
		for _, prefix := range []string{
			"/workspaces/" + filepath.Base(dir) + "/",
			"/workspace/",
		} {
			if strings.HasPrefix(scriptPath, prefix) {
				rel = strings.TrimPrefix(scriptPath, prefix)
				break
			}
		}
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			hostScript = candidate
		}
		if hostScript == "" {
			continue
		}

		// Copy to .rook/scripts/
		scriptsDir := filepath.Join(rookDir, "scripts")
		os.MkdirAll(scriptsDir, 0755)
		scriptName := filepath.Base(hostScript)
		destPath := filepath.Join(scriptsDir, scriptName)
		content, err := os.ReadFile(hostScript)
		if err != nil {
			continue
		}
		// Sanitize devcontainer-specific patterns
		content, scriptChanges := discovery.SanitizeScript(content)
		if err := os.WriteFile(destPath, content, 0755); err != nil {
			continue
		}

		// Update the service command to use the .rook/scripts/ copy
		newCommand := strings.Replace(svc.Command, scriptPath, "/workspaces/"+filepath.Base(dir)+"/.rook/scripts/"+scriptName, 1)
		svc.Command = newCommand
		services[name] = svc

		fmt.Printf("  Copied %s to .rook/scripts/%s\n", rel, scriptName)
		if len(scriptChanges) > 0 {
			for _, c := range scriptChanges {
				fmt.Printf("  Sanitized .rook/scripts/%s: %s\n", scriptName, c.Description)
			}
			warns.add("Verify .rook/scripts/%s — devcontainer patterns were automatically removed", scriptName)
		} else {
			warns.add("Review .rook/scripts/%s and adjust for rook (e.g., remove devcontainer-specific wait loops)", scriptName)
		}
	}
	return services
}

// buildFileTree returns a string representation of the directory tree (up to maxDepth levels).
func buildFileTree(dir string, maxDepth int) string {
	var sb strings.Builder
	buildFileTreeRecursive(&sb, dir, "", 0, maxDepth)
	return sb.String()
}

func buildFileTreeRecursive(sb *strings.Builder, dir, prefix string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		// Skip hidden dirs (except .devcontainer) and common noise
		if strings.HasPrefix(name, ".") && name != ".devcontainer" {
			continue
		}
		if name == "node_modules" || name == "vendor" || name == "__pycache__" {
			continue
		}
		sb.WriteString(prefix + name)
		if e.IsDir() {
			sb.WriteString("/\n")
			buildFileTreeRecursive(sb, filepath.Join(dir, name), prefix+"  ", depth+1, maxDepth)
		} else {
			sb.WriteString("\n")
		}
	}
}
