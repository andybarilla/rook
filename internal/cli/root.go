package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "rook",
		Short:        "Local development workspace manager",
		SilenceUsage: true,
	}
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.AddCommand(
		newInitCmd(),
		newDiscoverCmd(),
		newUpCmd(),
		newDownCmd(),
		newRestartCmd(),
		newStatusCmd(),
		newListCmd(),
		newPortsCmd(),
		newLogsCmd(),
		newEnvCmd(),
		NewCheckBuildsCmd(),
		newAgentMDCmd(),
	)
	return cmd
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.ExpandEnv("$HOME/.config")
	}
	return dir + "/rook"
}
