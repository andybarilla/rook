package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/ports"
	"github.com/spf13/cobra"
)

func newPortsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ports",
		Short: "Show global port allocation table",
		RunE: func(cmd *cobra.Command, args []string) error {
			alloc, err := ports.NewFileAllocator(filepath.Join(configDir(), "ports.json"), 10000, 60000)
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
}
