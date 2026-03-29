package e2e_test

import (
	"strings"
	"testing"
)

func TestDown_NoContainers(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: downws
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

	output, err := runRook(t, binPath, configDir, "down", "downws")
	if err != nil {
		t.Fatalf("down failed: %s", output)
	}
	if !strings.Contains(output, "No running containers") {
		t.Errorf("expected no containers message, got: %s", output)
	}
}

func TestDown_UnknownWorkspace(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	// down doesn't fail for unknown workspace - it just reports no containers found
	output, err := runRook(t, binPath, configDir, "down", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %s", output)
	}
	if !strings.Contains(output, "No running containers") {
		t.Errorf("expected no containers message, got: %s", output)
	}
}
