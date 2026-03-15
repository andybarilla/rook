package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDevcontainerDiscoverer_Detect(t *testing.T) {
	d := NewDevcontainerDiscoverer()

	t.Run("detects devcontainer.json", func(t *testing.T) {
		dir := t.TempDir()
		dcDir := filepath.Join(dir, ".devcontainer")
		os.MkdirAll(dcDir, 0755)
		os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte("{}"), 0644)
		if !d.Detect(dir) {
			t.Error("expected Detect to return true for .devcontainer/devcontainer.json")
		}
	})

	t.Run("returns false without file", func(t *testing.T) {
		dir := t.TempDir()
		if d.Detect(dir) {
			t.Error("expected Detect to return false when no devcontainer file exists")
		}
	})
}

func TestDevcontainerDiscoverer_ForwardedPorts(t *testing.T) {
	d := NewDevcontainerDiscoverer()
	dir := t.TempDir()
	dcDir := filepath.Join(dir, ".devcontainer")
	os.MkdirAll(dcDir, 0755)

	content := `{
		"name": "My Dev Container",
		"image": "mcr.microsoft.com/devcontainers/go:1.22",
		"forwardPorts": [8080, 3000]
	}`
	os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(content), 0644)

	result, err := d.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Source != "devcontainer" {
		t.Errorf("expected source 'devcontainer', got %q", result.Source)
	}

	// Devcontainer returns informational result with no services
	if len(result.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(result.Services))
	}
}
