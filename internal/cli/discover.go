package cli

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover [workspace]",
		Short: "Re-scan workspace and show changes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			// Resolve workspace name from args or current directory
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			// Load the workspace entry to get the path
			entry, err := cctx.registry.Get(wsName)
			if err != nil {
				return err
			}

			// Load the current manifest
			manifest, err := workspace.ParseManifest(manifestPath(entry.Path))
			if err != nil {
				return fmt.Errorf("reading manifest: %w", err)
			}

			// Run discovery on the workspace directory
			discoverers := []discovery.Discoverer{
				discovery.NewComposeDiscoverer(),
				discovery.NewDevcontainerDiscoverer(),
				discovery.NewMiseDiscoverer(),
			}
			result, err := discovery.RunAll(entry.Path, discoverers)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			// Compare discovered services with manifest services
			changes := compareServices(manifest.Services, result.Services)

			// Output results
			printChanges(cmd, changes, result.Source)

			return nil
		},
	}
}

// ServiceChange represents a detected change between manifest and discovery.
type ServiceChange struct {
	NewServices     []string
	RemovedServices []string
}

// compareServices compares manifest services with discovered services.
func compareServices(manifest map[string]workspace.Service, discovered map[string]workspace.Service) ServiceChange {
	var changes ServiceChange

	// Find new services (in discovered but not in manifest)
	for name := range discovered {
		if _, exists := manifest[name]; !exists {
			changes.NewServices = append(changes.NewServices, name)
		}
	}

	// Find removed services (in manifest but not in discovered)
	for name := range manifest {
		if _, exists := discovered[name]; !exists {
			changes.RemovedServices = append(changes.RemovedServices, name)
		}
	}

	// Sort for consistent output
	sort.Strings(changes.NewServices)
	sort.Strings(changes.RemovedServices)

	return changes
}

// printChanges outputs the detected changes.
func printChanges(cmd *cobra.Command, changes ServiceChange, source string) {
	// Print discovery source
	cmd.Printf("Scanned from: %s\n\n", source)

	// Check if there are any changes
	if len(changes.NewServices) == 0 && len(changes.RemovedServices) == 0 {
		cmd.Println("No changes detected.")
		return
	}

	// Print new services
	if len(changes.NewServices) > 0 {
		cmd.Println("New services:")
		for _, name := range changes.NewServices {
			cmd.Printf("  + %s\n", name)
		}
	}

	// Print removed services
	if len(changes.RemovedServices) > 0 {
		if len(changes.NewServices) > 0 {
			cmd.Println()
		}
		cmd.Println("Removed services:")
		for _, name := range changes.RemovedServices {
			cmd.Printf("  - %s\n", name)
		}
	}

	// Print summary
	total := len(changes.NewServices) + len(changes.RemovedServices)
	cmd.Printf("\n%d change(s) detected.\n", total)
}

// manifestPath returns the path to the rook.yaml manifest in a workspace directory.
func manifestPath(workspacePath string) string {
	return filepath.Join(workspacePath, "rook.yaml")
}
