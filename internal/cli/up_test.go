package cli

import (
	"path/filepath"
	"testing"
)

func TestUpCmd_BuildCachePath(t *testing.T) {
	wsRoot := "/workspace/test"
	got := buildCachePath(wsRoot)
	want := filepath.Join(wsRoot, ".rook", ".cache", "build-cache.json")
	if got != want {
		t.Errorf("buildCachePath(%q) = %q, want %q", wsRoot, got, want)
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
