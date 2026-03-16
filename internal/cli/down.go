package cli

import (
	"fmt"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
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
				return nil
			}
			for _, name := range containers {
				fmt.Printf("Stopping %s...\n", name)
				runner.StopContainer(name)
			}
			fmt.Printf("Stopped %d container(s) for %s.\n", len(containers), wsName)
			return nil
		},
	}
}
