package e2e_test

import (
	"strings"
	"testing"
)

func TestRestart_NoContainers(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: restartws
type: single
services:
  app:
    image: nginx
    ports: [80]
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "restart", "restartws")
	if err != nil {
		t.Fatalf("restart failed: %s", output)
	}
	if !strings.Contains(output, "No running containers") {
		t.Errorf("expected no containers message, got: %s", output)
	}
}

func TestRestart_UnknownWorkspace(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	_, err := runRook(t, binPath, configDir, "restart", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown workspace")
	}
}
