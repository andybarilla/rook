package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [workspace] [service]",
		Short: "Restart a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}
			wsName := ws.Name
			orch := cctx.newOrchestrator(wsName)
			orch.Reconnect(*ws)

			if len(args) > 1 {
				svcName := args[1]
				if _, ok := ws.Services[svcName]; !ok {
					return fmt.Errorf("unknown service: %s", svcName)
				}
				containerName := fmt.Sprintf("rook_%s_%s", wsName, svcName)
				runner.StopContainer(containerName)
				fmt.Printf("Restarting %s...\n", svcName)
				if err := orch.StartService(context.Background(), *ws, svcName); err != nil {
					return err
				}
				fmt.Printf("Restarted %s.\n", svcName)
				return nil
			}

			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, _ := runner.FindContainers(prefix)
			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				return nil
			}
			for _, c := range containers {
				runner.StopContainer(c)
			}
			profile := "all"
			if _, ok := ws.Profiles["default"]; ok {
				profile = "default"
			}
			fmt.Printf("Restarting %s (profile: %s)...\n", wsName, profile)
			if err := orch.Up(context.Background(), *ws, profile); err != nil {
				return err
			}
			fmt.Println("Restarted.")
			return nil
		},
	}
}
