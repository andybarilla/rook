package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env <workspace>",
		Short: "Print generated environment variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := registry.NewFileRegistry(filepath.Join(configDir(), "workspaces.json"))
			if err != nil {
				return err
			}
			entry, err := reg.Get(args[0])
			if err != nil {
				return err
			}
			m, err := workspace.ParseManifest(filepath.Join(entry.Path, "rook.yaml"))
			if err != nil {
				return err
			}
			alloc, err := ports.NewFileAllocator(filepath.Join(configDir(), "ports.json"), 10000, 60000)
			if err != nil {
				return err
			}
			portMap := make(map[string]int)
			for name := range m.Services {
				if result := alloc.Get(m.Name, name); result.OK {
					portMap[name] = result.Port
				}
			}
			for name, svc := range m.Services {
				resolved, err := envgen.ResolveTemplates(svc.Environment, portMap, false)
				if err != nil {
					return err
				}
				for k, v := range resolved {
					fmt.Printf("%s.%s: %s=%s\n", m.Name, name, k, v)
				}
			}
			return nil
		},
	}
}
