package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [workspace] [service]",
		Short: "Restart a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Restart command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
