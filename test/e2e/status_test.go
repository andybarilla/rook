package e2e_test

import (
	"strings"
	"testing"
)

func TestStatus_AllWorkspaces(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	output, err := runRook(t, binPath, configDir, "status")
	if err != nil {
		t.Fatalf("status failed: %s", output)
	}
	if !strings.Contains(output, "No workspaces registered") {
		t.Errorf("expected no workspaces message, got: %s", output)
	}
}

func TestStatus_WithWorkspaces(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: statusws
type: single
services:
  web:
    image: nginx
    ports: [8080]
  db:
    image: postgres
    ports: [5432]
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "status", "statusws")
	if err != nil {
		t.Fatalf("status failed: %s", output)
	}
	if !strings.Contains(output, "web") || !strings.Contains(output, "db") {
		t.Errorf("expected both services in output, got: %s", output)
	}
	// Container services should show as "stopped" when not running
	if !strings.Contains(output, "stopped") && !strings.Contains(output, "container") {
		t.Errorf("expected status info, got: %s", output)
	}
}

func TestStatus_SingleService(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: singlestatus
type: single
services:
  api:
    image: node
    ports: [3000]
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "status", "singlestatus")
	if err != nil {
		t.Fatalf("status failed: %s", output)
	}
	if !strings.Contains(output, "api") {
		t.Errorf("expected api service in output, got: %s", output)
	}
}
