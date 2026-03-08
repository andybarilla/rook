package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "rook",
		Short:        "Rook — local development environment manager",
		Long:         "Rook manages local development sites, SSL, PHP, Node, and database services.",
		SilenceUsage: true,
		Run:          func(cmd *cobra.Command, args []string) {},
	}

	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())

	return cmd
}

func Execute() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
