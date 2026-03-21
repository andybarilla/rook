package cli

import (
	"fmt"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	var removeVolumes bool

	cmd := &cobra.Command{
		Use:   "down [workspace]",
		Short: "Stop all services in workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}
			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return fmt.Errorf("finding containers: %w", err)
			}
			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				// Still clean up the network
				networkName := fmt.Sprintf("rook_%s", wsName)
				runner.RemoveNetwork(networkName)
				return nil
			}

			// Collect named volumes before removing containers (must inspect while containers exist)
			seen := map[string]bool{}
			if removeVolumes {
				for _, name := range containers {
					vols, err := runner.ContainerVolumes(name)
					if err == nil {
						for _, v := range vols {
							seen[v] = true
						}
					}
				}
			}

			for _, name := range containers {
				fmt.Printf("Stopping %s...\n", name)
				runner.StopContainerWithVolumes(name, removeVolumes)
			}

			// Remove named volumes after containers are gone (deduplicated)
			if removeVolumes && len(seen) > 0 {
				volumes := make([]string, 0, len(seen))
				for v := range seen {
					volumes = append(volumes, v)
				}
				fmt.Printf("Removing %d volume(s)...\n", len(volumes))
				runner.RemoveVolumes(volumes)
			}

			// Clean up the workspace network
			networkName := fmt.Sprintf("rook_%s", wsName)
			runner.RemoveNetwork(networkName)

			fmt.Printf("Stopped %d container(s) for %s.\n", len(containers), wsName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&removeVolumes, "volumes", "v", false, "Remove volumes associated with containers")

	return cmd
}
