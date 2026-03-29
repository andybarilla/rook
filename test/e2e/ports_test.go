package e2e_test

import (
	"strings"
	"testing"
)

func TestPorts_Empty(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	output, err := runRook(t, binPath, configDir, "ports")
	if err != nil {
		t.Fatalf("ports failed: %s", output)
	}
	if !strings.Contains(output, "No ports allocated") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestPorts_Show(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: portsws
type: single
services:
  redis:
    image: redis
    ports: [6379]
  db:
    image: postgres
    ports: [5432]
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "ports")
	if err != nil {
		t.Fatalf("ports failed: %s", output)
	}
	if !strings.Contains(output, "redis") || !strings.Contains(output, "db") {
		t.Errorf("expected both services in output, got: %s", output)
	}
}

func TestPorts_Reset(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: resetws
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

	// Verify ports exist
	output, err := runRook(t, binPath, configDir, "ports")
	if err != nil {
		t.Fatalf("ports failed: %s", output)
	}
	if !strings.Contains(output, "app") {
		t.Errorf("expected app in ports output, got: %s", output)
	}

	// Reset
	output, err = runRook(t, binPath, configDir, "ports", "--reset")
	if err != nil {
		t.Fatalf("ports --reset failed: %s", output)
	}
	if !strings.Contains(output, "cleared") {
		t.Errorf("expected cleared message, got: %s", output)
	}

	// Verify empty
	output, err = runRook(t, binPath, configDir, "ports")
	if err != nil {
		t.Fatalf("ports failed: %s", output)
	}
	if !strings.Contains(output, "No ports allocated") {
		t.Errorf("expected no ports after reset, got: %s", output)
	}
}

func TestPorts_JSON(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: jsonports
type: single
services:
  svc:
    image: nginx
    ports: [80]
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "ports", "--json")
	if err != nil {
		t.Fatalf("ports --json failed: %s", output)
	}
	if !strings.Contains(output, `"port"`) || !strings.Contains(output, "jsonports") {
		t.Errorf("expected JSON output with port data, got: %s", output)
	}
}
