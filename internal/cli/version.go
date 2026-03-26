package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the rook version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				printJSON(map[string]string{"version": Version})
				return
			}
			fmt.Println("rook " + Version)
		},
	}
}
