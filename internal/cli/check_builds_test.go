package cli

import (
	"path/filepath"
	"testing"
)

func TestCheckBuildsCmd_UsesCachePath(t *testing.T) {
	wsRoot := "/tmp/testws"
	expectedPath := filepath.Join(wsRoot, ".rook", ".cache", "build-cache.json")
	actualPath := buildCachePath(wsRoot)
	if actualPath != expectedPath {
		t.Errorf("expected %s, got %s", expectedPath, actualPath)
	}
}

func TestCheckBuildsCmd_Help(t *testing.T) {
	// Verify command is registered and has help
	cmd := NewCheckBuildsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != "check-builds [workspace]" {
		t.Errorf("Use: got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected non-empty Short")
	}
}
