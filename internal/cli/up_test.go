package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpCmd_BuildCachePath_DefaultsToNewPath(t *testing.T) {
	wsRoot := t.TempDir()
	got := buildCachePath(wsRoot)
	want := filepath.Join(wsRoot, ".rook", ".cache", "build-cache.json")
	if got != want {
		t.Errorf("buildCachePath(%q) = %q, want %q", wsRoot, got, want)
	}
}

func TestUpCmd_BuildCachePath_MigratesOldPath(t *testing.T) {
	wsRoot := t.TempDir()
	oldPath := filepath.Join(wsRoot, ".rook", "build-cache.json")
	os.MkdirAll(filepath.Dir(oldPath), 0755)
	os.WriteFile(oldPath, []byte(`{"version":1}`), 0644)

	got := buildCachePath(wsRoot)
	if got != oldPath {
		t.Errorf("buildCachePath should return old path when it exists, got %q, want %q", got, oldPath)
	}
}

func TestUpCmd_BuildCachePath_PrefersNewPath(t *testing.T) {
	wsRoot := t.TempDir()
	// Create both old and new
	oldPath := filepath.Join(wsRoot, ".rook", "build-cache.json")
	newPath := filepath.Join(wsRoot, ".rook", ".cache", "build-cache.json")
	os.MkdirAll(filepath.Dir(oldPath), 0755)
	os.WriteFile(oldPath, []byte(`{"version":1}`), 0644)
	os.MkdirAll(filepath.Dir(newPath), 0755)
	os.WriteFile(newPath, []byte(`{"version":1}`), 0644)

	got := buildCachePath(wsRoot)
	if got != newPath {
		t.Errorf("buildCachePath should prefer new path, got %q, want %q", got, newPath)
	}
}

func TestUpCmd_ResolvedDirPath(t *testing.T) {
	wsRoot := "/workspace/test"
	got := resolvedDirPath(wsRoot)
	want := filepath.Join(wsRoot, ".rook", ".cache", "resolved")
	if got != want {
		t.Errorf("resolvedDirPath(%q) = %q, want %q", wsRoot, got, want)
	}
}
