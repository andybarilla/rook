package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

type cliContext struct {
	registry  *registry.FileRegistry
	portAlloc *ports.FileAllocator
	process   *runner.ProcessRunner
}

func newCLIContext() (*cliContext, error) {
	cfgDir := configDir()
	os.MkdirAll(cfgDir, 0755)
	reg, err := registry.NewFileRegistry(filepath.Join(cfgDir, "workspaces.json"))
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}
	alloc, err := ports.NewFileAllocator(filepath.Join(cfgDir, "ports.json"), 10000, 60000)
	if err != nil {
		return nil, fmt.Errorf("loading port allocator: %w", err)
	}
	return &cliContext{registry: reg, portAlloc: alloc, process: runner.NewProcessRunner()}, nil
}

func (c *cliContext) newOrchestrator(wsName string) *orchestrator.Orchestrator {
	docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))
	return orchestrator.New(docker, c.process, c.portAlloc)
}

func (c *cliContext) resolveWorkspaceName(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	manifestPath := filepath.Join(cwd, "rook.yaml")
	if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
		return "", fmt.Errorf("no workspace specified and no rook.yaml in current directory")
	}
	m, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return "", fmt.Errorf("rook.yaml in current directory has errors: %w", err)
	}
	return m.Name, nil
}

func (c *cliContext) loadWorkspace(name string) (*workspace.Workspace, error) {
	entry, err := c.registry.Get(name)
	if err != nil {
		return nil, err
	}
	m, err := workspace.ParseManifest(filepath.Join(entry.Path, "rook.yaml"))
	if err != nil {
		return nil, err
	}
	return m.ToWorkspace(entry.Path)
}

func (c *cliContext) initFromManifest(dir string) error {
	manifestPath := filepath.Join(dir, "rook.yaml")
	m, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return err
	}

	rookDir := filepath.Join(dir, ".rook")
	if err := ensureRookGitignore(rookDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create .rook/.gitignore: %v\n", err)
	}

	ensureAgentMDRookSection(dir, m)

	if err := c.registry.Register(m.Name, dir); err != nil {
		return err
	}

	for name, svc := range m.Services {
		if svc.PinPort > 0 {
			allocated, err := c.portAlloc.AllocatePinned(m.Name, name, svc.PinPort)
			if err != nil {
				return fmt.Errorf("pinning port for %s: %w", name, err)
			}
			fmt.Printf("  %s.%s -> :%d (pinned)\n", m.Name, name, allocated)
		} else if len(svc.Ports) > 0 {
			allocated, err := c.portAlloc.Allocate(m.Name, name)
			if err != nil {
				return fmt.Errorf("allocating port for %s: %w", name, err)
			}
			fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
		}
	}
	fmt.Printf("Workspace %q registered from %s\n", m.Name, dir)
	return nil
}
