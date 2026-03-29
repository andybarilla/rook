package e2e_test

import (
	"strings"
	"testing"
)

func TestLogs_NoContainers(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: logsws
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

	output, err := runRook(t, binPath, configDir, "logs", "logsws")
	if err != nil {
		t.Fatalf("logs failed: %s", output)
	}
	if !strings.Contains(output, "No running services") {
		t.Errorf("expected no services message, got: %s", output)
	}
}

func TestLogs_UnknownWorkspace(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	_, err := runRook(t, binPath, configDir, "logs", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown workspace")
	}
}
