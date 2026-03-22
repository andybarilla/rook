package cli

import (
	"fmt"
	"os"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env [workspace]",
		Short: "Print generated environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}

			portMap := make(map[string]int)
			for name := range ws.Services {
				if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
					portMap[name] = result.Port
				}
			}
			for name, svc := range ws.Services {
				resolved, err := envgen.ResolveTemplates(svc.Environment, portMap)
				if err != nil {
					return err
				}
				for k, v := range resolved {
					fmt.Printf("%s.%s: %s=%s\n", ws.Name, name, k, v)
				}
			}
			return nil
		},
	}
}
