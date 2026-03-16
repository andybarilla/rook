package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

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

			profile := "all"
			if len(args) > 1 {
				profile = args[1]
			} else if _, ok := ws.Profiles["default"]; ok {
				profile = "default"
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
					port, err := cctx.portAlloc.Allocate(ws.Name, name, svc.Ports[0])
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
						resolved, err = envgen.ResolveTemplates(svc.Environment, portMap, false)
					}
					if err != nil {
						return fmt.Errorf("resolving env for %s: %w", name, err)
					}
					svc.Environment = resolved
					ws.Services[name] = svc
				}
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
				fmt.Fprintf(os.Stderr, "warning: reconnect failed: %v\n", err)
			}
			fmt.Printf("Starting %s (profile: %s)...\n", wsName, profile)
			if err := orch.Up(ctx, *ws, profile); err != nil {
				return err
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
				fmt.Println("Services started in detach mode.")
				return nil
			}

			fmt.Print("\nStreaming logs (Ctrl+C to stop)...\n\n")
			mux := newLogMux(os.Stdout)
			docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))

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
						fmt.Fprintf(os.Stderr, "warning: cannot stream logs for %s: %v\n", svcName, err)
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
						fmt.Fprintf(os.Stderr, "warning: cannot stream logs for %s: %v\n", svcName, err)
						continue
					}
					wg.Add(1)
					go func() {
						defer wg.Done()
						mux.addStream(svcName, reader, idx)
					}()
				}
			}

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
