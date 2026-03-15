package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <path>",
		Short: "Initialize a workspace from a project directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			manifestPath := filepath.Join(dir, "rook.yaml")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				// Run auto-discovery
				discoverers := []discovery.Discoverer{
					discovery.NewComposeDiscoverer(),
					discovery.NewDevcontainerDiscoverer(),
					discovery.NewMiseDiscoverer(),
				}
				result, err := discovery.RunAll(dir, discoverers)
				if err != nil {
					return fmt.Errorf("discovery failed: %w", err)
				}
				if len(result.Services) == 0 {
					return fmt.Errorf("no services discovered in %s — create a rook.yaml manually", dir)
				}
				fmt.Printf("Discovered from %s:\n", result.Source)
				for name, svc := range result.Services {
					if svc.IsContainer() {
						fmt.Printf("  %s (container: %s)\n", name, svc.Image)
					} else {
						fmt.Printf("  %s (process)\n", name)
					}
				}
				wsName := filepath.Base(dir)
				m := &workspace.Manifest{
					Name:     wsName,
					Type:     workspace.TypeSingle,
					Services: result.Services,
					Groups:   result.Groups,
				}
				if err := workspace.WriteManifest(manifestPath, m); err != nil {
					return fmt.Errorf("writing manifest: %w", err)
				}
				fmt.Printf("Generated %s\n", manifestPath)
			}

			m, err := workspace.ParseManifest(manifestPath)
			if err != nil {
				return err
			}

			cfgDir := configDir()
			os.MkdirAll(cfgDir, 0755)

			reg, err := registry.NewFileRegistry(filepath.Join(cfgDir, "workspaces.json"))
			if err != nil {
				return err
			}
			if err := reg.Register(m.Name, dir); err != nil {
				return err
			}

			alloc, err := ports.NewFileAllocator(filepath.Join(cfgDir, "ports.json"), 10000, 60000)
			if err != nil {
				return err
			}

			for name, svc := range m.Services {
				if svc.PinPort > 0 {
					allocated, err := alloc.AllocatePinned(m.Name, name, svc.PinPort)
					if err != nil {
						return fmt.Errorf("pinning port for %s: %w", name, err)
					}
					fmt.Printf("  %s.%s -> :%d (pinned)\n", m.Name, name, allocated)
				} else {
					for _, port := range svc.Ports {
						allocated, err := alloc.Allocate(m.Name, name, port)
						if err != nil {
							return fmt.Errorf("allocating port for %s: %w", name, err)
						}
						fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
					}
				}
			}
			fmt.Printf("Workspace %q registered from %s\n", m.Name, dir)
			return nil
		},
	}
}
