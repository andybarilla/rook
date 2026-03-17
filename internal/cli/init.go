package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/workspace"
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
	return &cobra.Command{
		Use:   "init <path>",
		Short: "Initialize a workspace from a project directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			manifestPath := filepath.Join(dir, "rook.yaml")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				// Run auto-discovery
				discoverers := []discovery.Discoverer{
					discovery.NewComposeDiscoverer(),
					discovery.NewDevcontainerDiscoverer(),
					discovery.NewMiseDiscoverer(),
				}
				result, err := discovery.RunAll(dir, discoverers)
				if err != nil {
					return fmt.Errorf("discovery failed: %w", err)
				}
				if len(result.Services) == 0 {
					return fmt.Errorf("no services discovered in %s — create a rook.yaml manually", dir)
				}
				fmt.Printf("Discovered from %s:\n", result.Source)
				for name, svc := range result.Services {
					if svc.IsContainer() {
						fmt.Printf("  %s (container: %s)\n", name, svc.Image)
					} else {
						fmt.Printf("  %s (process)\n", name)
					}
				}
				// Copy devcontainer scripts to .rook/ so they can be modified
				// without affecting the devcontainer setup
				rookDir := filepath.Join(dir, ".rook")
				for name, svc := range result.Services {
					if svc.Command == "" {
						continue
					}
					// Check if the command references a .devcontainer script
					if !strings.Contains(svc.Command, ".devcontainer/") {
						continue
					}

					// Extract the script path from the command
					// Handle both "/path/.devcontainer/start.sh" and "sh /path/.devcontainer/start.sh"
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

					// Resolve to a host path — the script path is a container path
					// that maps via the volume mount. Try common patterns.
					var hostScript string
					// Try relative to project dir (e.g., .devcontainer/start.sh)
					rel := scriptPath
					// Strip leading workspace mount prefix if present
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
					if err := os.WriteFile(destPath, content, 0755); err != nil {
						continue
					}

					// Update the service command to use the .rook/scripts/ copy
					newCommand := strings.Replace(svc.Command, scriptPath, "/workspaces/"+filepath.Base(dir)+"/.rook/scripts/"+scriptName, 1)
					svc.Command = newCommand
					result.Services[name] = svc

					fmt.Printf("  Copied %s to .rook/scripts/%s\n", rel, scriptName)
					warns.add("Review .rook/scripts/%s and adjust for rook (e.g., remove devcontainer-specific wait loops)", scriptName)
				}
				// Ensure .rook/.gitignore exists with .cache/ entry
				if err := ensureRookGitignore(rookDir); err != nil {
					return fmt.Errorf("creating .rook/.gitignore: %w", err)
				}

				wsName := filepath.Base(dir)
				m := &workspace.Manifest{
					Name:     wsName,
					Type:     workspace.TypeSingle,
					Services: result.Services,
					Groups:   result.Groups,
				}
				if err := workspace.WriteManifest(manifestPath, m); err != nil {
					return fmt.Errorf("writing manifest: %w", err)
				}
				fmt.Printf("Generated %s\n", manifestPath)
			}

			m, err := workspace.ParseManifest(manifestPath)
			if err != nil {
				return err
			}

			cfgDir := configDir()
			os.MkdirAll(cfgDir, 0755)

			reg, err := registry.NewFileRegistry(filepath.Join(cfgDir, "workspaces.json"))
			if err != nil {
				return err
			}
			if err := reg.Register(m.Name, dir); err != nil {
				return err
			}

			alloc, err := ports.NewFileAllocator(filepath.Join(cfgDir, "ports.json"), 10000, 60000)
			if err != nil {
				return err
			}

			for name, svc := range m.Services {
				if svc.PinPort > 0 {
					allocated, err := alloc.AllocatePinned(m.Name, name, svc.PinPort)
					if err != nil {
						return fmt.Errorf("pinning port for %s: %w", name, err)
					}
					fmt.Printf("  %s.%s -> :%d (pinned)\n", m.Name, name, allocated)
				} else {
					for _, port := range svc.Ports {
						allocated, err := alloc.Allocate(m.Name, name, port)
						if err != nil {
							return fmt.Errorf("allocating port for %s: %w", name, err)
						}
						fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
					}
				}
			}
			fmt.Printf("Workspace %q registered from %s\n", m.Name, dir)
			warns.print(os.Stderr)
			return nil
		},
	}
}
