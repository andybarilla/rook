package discovery

import (
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
)

var miseFileNames = []string{
	"mise.toml",
	".mise.toml",
	".tool-versions",
}

// MiseDiscoverer detects mise/tool-versions configuration files.
type MiseDiscoverer struct{}

// NewMiseDiscoverer creates a new MiseDiscoverer.
func NewMiseDiscoverer() *MiseDiscoverer { return &MiseDiscoverer{} }

// Name returns the discoverer name.
func (d *MiseDiscoverer) Name() string { return "mise" }

// Detect returns true if a mise or tool-versions file exists in the directory.
func (d *MiseDiscoverer) Detect(dir string) bool {
	for _, name := range miseFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// Discover returns an informational result (no services) indicating mise was detected.
func (d *MiseDiscoverer) Discover(dir string) (*DiscoveryResult, error) {
	return &DiscoveryResult{
		Source:   "mise",
		Services: make(map[string]workspace.Service),
		Groups:   make(map[string][]string),
	}, nil
}
