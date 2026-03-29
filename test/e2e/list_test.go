package e2e_test

import (
	"strings"
	"testing"
)

func TestList_Empty(t *testing.T) {
	binPath := buildRook(t)
	_, configDir := createTestWorkspace(t, "")

	output, err := runRook(t, binPath, configDir, "list")
	if err != nil {
		t.Fatalf("list failed: %s", output)
	}
	if !strings.Contains(output, "No workspaces registered") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestList_WithWorkspaces(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: listws
type: single
services:
  web:
    image: nginx
    ports: [8080]
`
	wsDir, configDir := createTestWorkspace(t, manifest)

	_, err := runRook(t, binPath, configDir, "init", wsDir)
	if err != nil {
		t.Fatalf("init failed: %s", err)
	}

	output, err := runRook(t, binPath, configDir, "list")
	if err != nil {
		t.Fatalf("list failed: %s", output)
	}
	if !strings.Contains(output, "listws") {
		t.Errorf("expected listws in output, got: %s", output)
	}
}

func TestList_JSON(t *testing.T) {
	binPath := buildRook(t)

	manifest := `
name: jsonws
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

	output, err := runRook(t, binPath, configDir, "list", "--json")
	if err != nil {
		t.Fatalf("list --json failed: %s", output)
	}
	if !strings.Contains(output, `"name"`) || !strings.Contains(output, "jsonws") {
		t.Errorf("expected JSON output with workspace name, got: %s", output)
	}
}
