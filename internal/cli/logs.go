package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <workspace> [service]",
		Short: "Tail logs (all or specific service)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Logs command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
