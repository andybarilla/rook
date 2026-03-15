package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMiseDiscoverer_Detect(t *testing.T) {
	d := NewMiseDiscoverer()

	t.Run("detects mise.toml", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("[tools]"), 0644)
		if !d.Detect(dir) {
			t.Error("expected Detect to return true for mise.toml")
		}
	})

	t.Run("detects .mise.toml", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, ".mise.toml"), []byte("[tools]"), 0644)
		if !d.Detect(dir) {
			t.Error("expected Detect to return true for .mise.toml")
		}
	})

	t.Run("returns false without file", func(t *testing.T) {
		dir := t.TempDir()
		if d.Detect(dir) {
			t.Error("expected Detect to return false when no mise file exists")
		}
	})
}

func TestMiseDiscoverer_DetectsToolVersions(t *testing.T) {
	d := NewMiseDiscoverer()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".tool-versions"), []byte("node 20.0.0"), 0644)
	if !d.Detect(dir) {
		t.Error("expected Detect to return true for .tool-versions")
	}
}

func TestMiseDiscoverer_Discover(t *testing.T) {
	d := NewMiseDiscoverer()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("[tools]\nnode = \"20\""), 0644)

	result, err := d.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Source != "mise" {
		t.Errorf("expected source 'mise', got %q", result.Source)
	}

	// Mise returns informational result with no services
	if len(result.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(result.Services))
	}
}
