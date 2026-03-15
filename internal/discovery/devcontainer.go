package discovery

import (
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
)

// DevcontainerDiscoverer detects devcontainer configuration files.
type DevcontainerDiscoverer struct{}

// NewDevcontainerDiscoverer creates a new DevcontainerDiscoverer.
func NewDevcontainerDiscoverer() *DevcontainerDiscoverer { return &DevcontainerDiscoverer{} }

// Name returns the discoverer name.
func (d *DevcontainerDiscoverer) Name() string { return "devcontainer" }

// Detect returns true if a devcontainer.json file exists in the directory.
func (d *DevcontainerDiscoverer) Detect(dir string) bool {
	p := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	_, err := os.Stat(p)
	return err == nil
}

// Discover returns an informational result (no services) indicating devcontainer was detected.
func (d *DevcontainerDiscoverer) Discover(dir string) (*DiscoveryResult, error) {
	return &DiscoveryResult{
		Source:   "devcontainer",
		Services: make(map[string]workspace.Service),
		Groups:   make(map[string][]string),
	}, nil
}
