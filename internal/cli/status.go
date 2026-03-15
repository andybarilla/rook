package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show all workspaces and running services",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Status display not yet implemented (requires running daemon)")
			return nil
		},
	}
}
