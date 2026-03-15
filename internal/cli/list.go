package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/registry"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := registry.NewFileRegistry(filepath.Join(configDir(), "workspaces.json"))
			if err != nil {
				return err
			}
			entries := reg.List()
			if jsonOutput {
				printJSON(entries)
				return nil
			}
			if len(entries) == 0 {
				fmt.Println("No workspaces registered. Run 'rook init <path>' to add one.")
				return nil
			}
			for _, e := range entries {
				fmt.Printf("%-20s %s\n", e.Name, e.Path)
			}
			return nil
		},
	}
}
