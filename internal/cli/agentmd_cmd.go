package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newAgentMDCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agentmd [path]",
		Short: "Update rook section in CLAUDE.md or AGENTS.md",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var dir string
			if len(args) > 0 {
				dir = args[0]
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting working directory: %w", err)
				}
				dir = cwd
			}

			manifestPath := filepath.Join(dir, "rook.yaml")
			m, err := workspace.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parsing rook.yaml: %w", err)
			}

			action, err := ensureAgentMDRookSection(dir, m)
			if err != nil {
				return err
			}

			target := agentMDTarget(dir)
			switch action {
			case "added":
				fmt.Printf("Added rook section to %s\n", target)
			case "updated":
				fmt.Printf("Updated rook section in %s\n", target)
			default:
				fmt.Println("No CLAUDE.md or AGENTS.md found")
			}
			return nil
		},
	}
}
