package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

// mockPrompter returns predetermined responses for testing.
type mockPrompter struct {
	selectResponses  [][]int
	confirmResponses []bool
	inputResponses   []string
	listResponses    [][]string

	selectIdx  int
	confirmIdx int
	inputIdx   int
	listIdx    int
}

func (m *mockPrompter) Select(_ string, _ []string) ([]int, error) {
	if m.selectIdx >= len(m.selectResponses) {
		return nil, nil
	}
	r := m.selectResponses[m.selectIdx]
	m.selectIdx++
	return r, nil
}

func (m *mockPrompter) Confirm(_ string, _ bool) (bool, error) {
	if m.confirmIdx >= len(m.confirmResponses) {
		return false, nil
	}
	r := m.confirmResponses[m.confirmIdx]
	m.confirmIdx++
	return r, nil
}

func (m *mockPrompter) Input(_ string, defaultValue string) (string, error) {
	if m.inputIdx >= len(m.inputResponses) {
		return defaultValue, nil
	}
	r := m.inputResponses[m.inputIdx]
	m.inputIdx++
	if r == "" {
		return defaultValue, nil
	}
	return r, nil
}

func (m *mockPrompter) InputList(_ string) ([]string, error) {
	if m.listIdx >= len(m.listResponses) {
		return nil, nil
	}
	r := m.listResponses[m.listIdx]
	m.listIdx++
	return r, nil
}

func TestDiscoverInteractive_SelectsComposeFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"),
		[]byte("services:\n  postgres:\n    image: postgres:16\n    ports:\n      - 5432:5432\n  redis:\n    image: redis:7\n    ports:\n      - 6379:6379\n"), 0644)
	os.WriteFile(filepath.Join(dir, "compose.yaml"),
		[]byte("services:\n  app:\n    image: node:22\n    ports:\n      - 3000:3000\n"), 0644)

	// ScanComposeFiles sorts alphabetically: compose.yaml < docker-compose.yml
	// So index 1 = docker-compose.yml
	p := &mockPrompter{
		selectResponses:  [][]int{{1}}, // select docker-compose.yml
		confirmResponses: []bool{false}, // don't add local services
	}

	var warns warnings
	services, _, err := discoverInteractive(dir, p, &warns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := services["postgres"]; !ok {
		t.Error("expected postgres from docker-compose.yml")
	}
	if _, ok := services["redis"]; !ok {
		t.Error("expected redis from docker-compose.yml")
	}
	if _, ok := services["app"]; ok {
		t.Error("did not expect app from compose.yaml (not selected)")
	}
}

func TestDiscoverInteractive_AddsLocalService(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"),
		[]byte("services:\n  postgres:\n    image: postgres:16\n    ports:\n      - 5432:5432\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module github.com/example/myapp\ngo 1.22\n"), 0644)

	p := &mockPrompter{
		selectResponses:  [][]int{{0}},                 // select compose file
		confirmResponses: []bool{true, false},           // add local services, then stop adding more
		inputResponses:   []string{"api", "go run ."},   // service name, command
		listResponses:    [][]string{{"postgres"}},      // depends on
	}

	var warns warnings
	services, _, err := discoverInteractive(dir, p, &warns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := services["postgres"]; !ok {
		t.Error("expected postgres from compose")
	}
	api, ok := services["api"]
	if !ok {
		t.Fatal("expected api service to be added")
	}
	if api.Command != "go run ." {
		t.Errorf("expected command 'go run .', got %q", api.Command)
	}
	if len(api.DependsOn) != 1 || api.DependsOn[0] != "postgres" {
		t.Errorf("expected depends_on [postgres], got %v", api.DependsOn)
	}
}

func TestDiscoverInteractive_SkipsEverything(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"),
		[]byte("services:\n  postgres:\n    image: postgres:16\n"), 0644)

	p := &mockPrompter{
		selectResponses:  [][]int{nil}, // skip compose selection
		confirmResponses: []bool{false}, // don't add local services
	}

	var warns warnings
	services, _, err := discoverInteractive(dir, p, &warns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(services) != 0 {
		t.Errorf("expected 0 services when skipping everything, got %d", len(services))
	}
}

func TestInitCmd_ForceReInit(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	// Create initial rook.yaml
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte("name: old\ntype: single\nservices:\n  old:\n    image: old:1\n"), 0644)

	// Create compose file for re-discovery
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"),
		[]byte("services:\n  newdb:\n    image: postgres:16\n    ports:\n      - 5432:5432\n"), 0644)

	// Run init --force
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{"--force", wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init --force failed: %v", err)
	}

	// Verify new manifest was written
	m, err := workspace.ParseManifest(filepath.Join(wsDir, "rook.yaml"))
	if err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	if _, ok := m.Services["newdb"]; !ok {
		t.Error("expected newdb service after --force re-init")
	}
	if _, ok := m.Services["old"]; ok {
		t.Error("did not expect old service after --force re-init")
	}
}

func TestInitCmd_NonInteractiveFlag(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"),
		[]byte("services:\n  web:\n    image: nginx:latest\n    ports:\n      - 80:80\n"), 0644)

	initCmd := newInitCmd()
	initCmd.SetArgs([]string{"--non-interactive", wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init --non-interactive failed: %v", err)
	}

	m, err := workspace.ParseManifest(filepath.Join(wsDir, "rook.yaml"))
	if err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	if _, ok := m.Services["web"]; !ok {
		t.Error("expected web service from non-interactive init")
	}
}

func TestInitCmd_ExistingManifestRegisters(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte("name: testws\ntype: single\nservices:\n  web:\n    image: nginx\n    ports:\n      - 80\n"), 0644)

	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init with existing manifest failed: %v", err)
	}

	// Verify .rook/.gitignore was created (proves registration ran)
	gitignorePath := filepath.Join(wsDir, ".rook", ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		t.Error("expected .rook/.gitignore to be created for existing manifest")
	}
}

func TestCopyDevcontainerScripts(t *testing.T) {
	dir := t.TempDir()
	wsName := filepath.Base(dir)

	// Create .devcontainer/start.sh
	devDir := filepath.Join(dir, ".devcontainer")
	os.MkdirAll(devDir, 0755)
	os.WriteFile(filepath.Join(devDir, "start.sh"), []byte("#!/bin/sh\nmake dev\nsleep infinity\n"), 0755)

	services := map[string]workspace.Service{
		"app": {
			Command: "sh /workspaces/" + wsName + "/.devcontainer/start.sh",
		},
	}

	var warns warnings
	result := copyDevcontainerScripts(dir, services, &warns)

	// Script should be copied
	scriptPath := filepath.Join(dir, ".rook", "scripts", "start.sh")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("expected script to be copied: %v", err)
	}
	if strings.Contains(string(content), "sleep infinity") {
		t.Error("expected sleep infinity to be sanitized out")
	}

	// Command should be updated
	svc := result["app"]
	if !strings.Contains(svc.Command, ".rook/scripts/start.sh") {
		t.Errorf("expected command to reference .rook/scripts/, got %q", svc.Command)
	}
}
