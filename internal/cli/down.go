package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down [workspace]",
		Short: "Stop all services in workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Down command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
