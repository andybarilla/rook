package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newPortsCmd() *cobra.Command {
	var reset bool

	cmd := &cobra.Command{
		Use:   "ports",
		Short: "Show global port allocation table",
		RunE: func(cmd *cobra.Command, args []string) error {
			portsPath := filepath.Join(configDir(), "ports.json")

			if reset {
				// Stop all rook containers first — stale containers on old ports
				// would be adopted by reconnect, causing port mismatches
				cctx, err := newCLIContext()
				if err == nil {
					for _, e := range cctx.registry.List() {
						prefix := fmt.Sprintf("rook_%s_", e.Name)
						containers, _ := runner.FindContainers(prefix)
						for _, c := range containers {
							fmt.Printf("Stopping %s...\n", c)
							runner.StopContainer(c)
						}
					}
				}

				if err := os.Remove(portsPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("removing ports file: %w", err)
				}
				fmt.Println("Port allocations cleared. Ports will be re-allocated on next `rook up`.")
				return nil
			}

			alloc, err := ports.NewFileAllocator(portsPath, 10000, 60000)
			if err != nil {
				return err
			}
			all := alloc.All()
			if jsonOutput {
				printJSON(all)
				return nil
			}
			if len(all) == 0 {
				fmt.Println("No ports allocated.")
				return nil
			}
			fmt.Printf("%-20s %-20s %s\n", "WORKSPACE", "SERVICE", "PORT")
			for _, e := range all {
				pinned := ""
				if e.Pinned {
					pinned = " (pinned)"
				}
				fmt.Printf("%-20s %-20s %d%s\n", e.Workspace, e.Service, e.Port, pinned)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&reset, "reset", false, "Clear all port allocations (re-allocated on next rook up)")
	return cmd
}
