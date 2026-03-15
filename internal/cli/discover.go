package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover <workspace>",
		Short: "Re-scan workspace and show changes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Discover command not yet fully implemented")
			return nil
		},
	}
}
