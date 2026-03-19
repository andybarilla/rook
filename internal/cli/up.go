package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/andybarilla/rook/internal/buildcache"
	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	var detach bool
	var build bool

	cmd := &cobra.Command{
		Use:   "up [workspace] [profile]",
		Short: "Start services",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var warns warnings

			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}

			// Set log directory for process services
			cctx.process.SetLogDir(logDirPath(ws.Root))

			profile := "all"
			if len(args) > 1 {
				profile = args[1]
			} else if _, ok := ws.Profiles["default"]; ok {
				profile = "default"
			}

			// Create docker runner for build cache checking (reused later for log streaming)
			docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))

			// Check for stale builds
			loadCachePath := buildCachePath(ws.Root)
			saveCachePath := filepath.Join(ws.Root, ".rook", ".cache", "build-cache.json")
			cache, err := buildcache.Load(loadCachePath)
			if err != nil {
				return fmt.Errorf("loading build cache: %w", err)
			}

			staleServices := make(map[string][]string)
			for name, svc := range ws.Services {
				if svc.Build == "" {
					continue
				}
				// Get current image ID (optional - may not exist yet)
				currentImageID, _ := docker.GetImageID(name)
				result, err := buildcache.DetectStale(cache, name, svc, ws.Root, currentImageID)
				if err != nil {
					return fmt.Errorf("checking %s: %w", name, err)
				}
				if result.NeedsRebuild {
					staleServices[name] = result.Reasons
				}
			}

			// Prompt to rebuild if any stale services
			if len(staleServices) > 0 && !build {
				fmt.Println("Checking for stale builds...")
				fmt.Printf("\n%d service(s) need rebuild:\n", len(staleServices))
				for name, reasons := range staleServices {
					if len(reasons) > 0 {
						fmt.Printf("  - %s (%s)\n", name, reasons[0])
					} else {
						fmt.Printf("  - %s\n", name)
					}
				}

				// Check which services have missing images (must rebuild)
				var missingImages, staleFiles []string
				for name, reasons := range staleServices {
					for _, r := range reasons {
						if r == "image missing" {
							missingImages = append(missingImages, name)
							break
						}
					}
					if _, isMissing := contains(reasons, "image missing"); !isMissing {
						staleFiles = append(staleFiles, name)
					}
				}

				// Auto-rebuild missing images
				if len(missingImages) > 0 {
					for _, name := range missingImages {
						svc := ws.Services[name]
						svc.ForceBuild = true
						ws.Services[name] = svc
					}
					fmt.Printf("\nAuto-rebuilding %d service(s) with missing images...\n", len(missingImages))
				}

				// Prompt for file changes only in interactive mode
				if len(staleFiles) > 0 {
					if !isTerminal(os.Stdin) {
						fmt.Println("\nNon-interactive mode: skipping rebuild for stale files. Use --build to force.")
					} else {
						fmt.Print("\nRebuild all? [Y/n]: ")

						reader := bufio.NewReader(os.Stdin)
						input, _ := reader.ReadString('\n')
						input = strings.TrimSpace(strings.ToLower(input))

						if input == "n" || input == "no" {
							fmt.Println("Proceeding with existing images...")
						} else {
							// Mark stale services for rebuild
							for name := range staleServices {
								svc := ws.Services[name]
								svc.ForceBuild = true
								ws.Services[name] = svc
							}
						}
					}
				}
			}

			// Allocate ports first so templates can resolve
			portMap := make(map[string]int)
			for name, svc := range ws.Services {
				if svc.PinPort > 0 {
					port, err := cctx.portAlloc.AllocatePinned(ws.Name, name, svc.PinPort)
					if err != nil {
						return fmt.Errorf("pinning port for %s: %w", name, err)
					}
					portMap[name] = port
				} else if len(svc.Ports) > 0 {
					port, err := cctx.portAlloc.Allocate(ws.Name, name)
					if err != nil {
						return fmt.Errorf("allocating port for %s: %w", name, err)
					}
					portMap[name] = port
				}
			}
			// Build container-aware host/port maps for template resolution
			containerPrefix := fmt.Sprintf("rook_%s_", wsName)
			containerPortMap := make(map[string]int)
			containerHostMap := make(map[string]string)
			for svcName, s := range ws.Services {
				if s.IsContainer() {
					// Container-to-container: use container name + internal port
					containerHostMap[svcName] = containerPrefix + svcName
					if len(s.Ports) > 0 {
						containerPortMap[svcName] = s.Ports[0] // internal port
					} else if p, ok := portMap[svcName]; ok {
						// Fallback: use allocated port (e.g., service defined ports
						// in a previous init but devcontainer compose omits them)
						containerPortMap[svcName] = p
					}
				} else {
					containerHostMap[svcName] = "localhost"
					if p, ok := portMap[svcName]; ok {
						containerPortMap[svcName] = p
					}
				}
			}

			for name, svc := range ws.Services {
				if len(svc.Environment) > 0 {
					var resolved map[string]string
					var err error
					if svc.IsContainer() {
						// Container services use container networking
						resolved, err = envgen.ResolveWithHostMap(svc.Environment, containerPortMap, containerHostMap)
					} else {
						// Process services use localhost + allocated ports
						resolved, err = envgen.ResolveTemplates(svc.Environment, portMap)
					}
					if err != nil {
						return fmt.Errorf("resolving env for %s: %w", name, err)
					}
					svc.Environment = resolved
					ws.Services[name] = svc
				}

				// Load env_file for process services
				if svc.IsProcess() && svc.EnvFile != "" {
					envFilePath := filepath.Join(ws.Root, svc.EnvFile)
					merged, err := envgen.LoadProcessEnvFile(envFilePath, svc.Environment, portMap)
					if err != nil {
						return fmt.Errorf("loading env_file for %s: %w", name, err)
					}
					svc.Environment = merged
					ws.Services[name] = svc
				}
			}

			// Write resolved env files so they override values from mounted .env files
			// (Makefiles that -include .env bypass container -e flags)
			resolvedDir := resolvedDirPath(ws.Root)
			os.MkdirAll(resolvedDir, 0755)
			for name, svc := range ws.Services {
				if !svc.IsContainer() || len(svc.Environment) == 0 {
					continue
				}
				envPath := filepath.Join(resolvedDir, name+".env")
				if err := envgen.WriteEnvFile(envPath, svc.Environment); err != nil {
					warns.add("cannot write resolved env for %s: %v", name, err)
					continue
				}
				svc.ResolvedEnvFile = envPath
				ws.Services[name] = svc
			}

			// Resolve templates in mounted config files (e.g., Caddyfile with {{.Host.api}})
			for name, svc := range ws.Services {
				if !svc.IsContainer() || len(svc.Volumes) == 0 {
					continue
				}
				for i, vol := range svc.Volumes {
					parts := strings.SplitN(vol, ":", 2)
					if len(parts) != 2 {
						continue
					}
					hostPath := parts[0]
					containerPath := parts[1]

					// Only process relative file paths (not named volumes)
					if !strings.HasPrefix(hostPath, "./") && !strings.HasPrefix(hostPath, "/") {
						continue
					}

					// Resolve relative to workspace root
					absPath := hostPath
					if strings.HasPrefix(hostPath, "./") {
						absPath = filepath.Join(ws.Root, hostPath)
					}

					// Only process files (not directories)
					info, err := os.Stat(absPath)
					if err != nil || info.IsDir() {
						continue
					}

					// Read the file and check for templates
					content, err := os.ReadFile(absPath)
					if err != nil || !strings.Contains(string(content), "{{") {
						continue
					}

					// Resolve templates
					resolved, err := envgen.ResolveFileTemplate(string(content), containerPortMap, containerHostMap)
					if err != nil {
						warns.add("cannot resolve templates in %s: %v", hostPath, err)
						continue
					}

					// Write resolved copy
					resolvedPath := filepath.Join(resolvedDir, fmt.Sprintf("%s_%s", name, filepath.Base(hostPath)))
					if err := os.WriteFile(resolvedPath, []byte(resolved), info.Mode()); err != nil {
						warns.add("cannot write resolved %s: %v", hostPath, err)
						continue
					}

					// Replace volume mount with resolved path
					svc.Volumes[i] = resolvedPath + ":" + containerPath
				}
				ws.Services[name] = svc
			}

			if build {
				for name, svc := range ws.Services {
					if svc.Build != "" {
						svc.ForceBuild = true
						ws.Services[name] = svc
					}
				}
			}

			orch := cctx.newOrchestrator(wsName)
			if err := orch.Reconnect(*ws); err != nil {
				warns.add("reconnect failed: %v", err)
			}
			fmt.Printf("Starting %s (profile: %s)...\n", wsName, profile)
			if err := orch.Up(ctx, *ws, profile); err != nil {
				return err
			}

			// Update build cache for all services with build contexts
			for name, svc := range ws.Services {
				if svc.Build == "" {
					continue
				}
				imageID, err := docker.GetImageID(name)
				if err != nil {
					// Image might not exist yet if build failed or was skipped
					continue
				}
				buildCtx := filepath.Join(ws.Root, svc.Build)
				dockerfile := "Dockerfile"
				if svc.Dockerfile != "" {
					dockerfile = svc.Dockerfile
				}
				if err := cache.UpdateAfterBuild(name, ws.Root, buildCtx, dockerfile, imageID); err != nil {
					warns.add("cannot update build cache for %s: %v", name, err)
					continue
				}
			}
			if err := cache.Save(saveCachePath); err != nil {
				warns.add("cannot save build cache: %v", err)
			}

			services, _ := orchestrator.TopoSort(ws.Services, ws.ServiceNames())
			for _, name := range services {
				if port, ok := portMap[name]; ok {
					fmt.Printf("  %-20s :%d\n", name, port)
				} else {
					fmt.Printf("  %-20s (no port)\n", name)
				}
			}

			if detach {
				warns.print(os.Stderr)
				fmt.Println("Services started in detach mode.")
				return nil
			}

			fmt.Print("\nStreaming logs (Ctrl+C to stop)...\n\n")
			mux := newLogMux(os.Stdout)

			var wg sync.WaitGroup
			statuses, _ := orch.Status(*ws)
			colorIdx := 0
			for _, name := range services {
				if statuses[name] != runner.StatusRunning {
					continue
				}
				svc := ws.Services[name]
				svcName := name
				idx := colorIdx
				colorIdx++

				if svc.IsContainer() {
					handle := runner.RunHandle{ID: svcName, Type: "docker"}
					reader, logCmd, err := docker.StreamLogs(handle)
					if err != nil {
						warns.add("cannot stream logs for %s: %v", svcName, err)
						continue
					}
					wg.Add(1)
					go func() {
						defer wg.Done()
						mux.addStream(svcName, reader, idx)
						if logCmd != nil {
							logCmd.Wait()
						}
					}()
				} else {
					reader, err := cctx.process.StreamLogs(runner.RunHandle{ID: svcName, Type: "process"})
					if err != nil {
						warns.add("cannot stream logs for %s: %v", svcName, err)
						continue
					}
					wg.Add(1)
					go func() {
						defer wg.Done()
						mux.addStream(svcName, reader, idx)
					}()
				}
			}

			warns.print(os.Stderr)

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nShutting down...")
			cancel()
			orch.Down(context.Background(), *ws)
			wg.Wait()
			fmt.Println("All services stopped.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Start services and exit immediately")
	cmd.Flags().BoolVar(&build, "build", false, "Force rebuild of services with build context")
	return cmd
}

// isTerminal checks if a file descriptor is a terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) (int, bool) {
	for i, s := range slice {
		if s == item {
			return i, true
		}
	}
	return -1, false
}

// buildCachePath returns the path to the build cache file.
// Falls back to old location for migration from pre-.cache/ layout.
func buildCachePath(wsRoot string) string {
	newPath := filepath.Join(wsRoot, ".rook", ".cache", "build-cache.json")
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}
	oldPath := filepath.Join(wsRoot, ".rook", "build-cache.json")
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}
	return newPath
}

// resolvedDirPath returns the path to the resolved files directory.
func resolvedDirPath(wsRoot string) string {
	return filepath.Join(wsRoot, ".rook", ".cache", "resolved")
}

// logDirPath returns the path to the process log files directory.
func logDirPath(wsRoot string) string {
	return filepath.Join(wsRoot, ".rook", ".cache", "logs")
}
