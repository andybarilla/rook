package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up [workspace] [profile]",
		Short: "Start services",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Up command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
