package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverCmd_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newDiscoverCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no workspace specified") {
		t.Errorf("expected error about no workspace specified, got: %v", err)
	}
}

func TestDiscoverCmd_WorkspaceNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newDiscoverCmd()
	cmd.SetArgs([]string{"nonexistent"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error about workspace not found, got: %v", err)
	}
}

func TestDiscoverCmd_NoChanges(t *testing.T) {
	// Setup: create a workspace with a docker-compose.yml and run init
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  db:
    image: postgres:15
    ports:
      - "5432:5432"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Run init to register the workspace
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Now run discover - should show no changes
	var out bytes.Buffer
	discoverCmd := newDiscoverCmd()
	discoverCmd.SetOut(&out)
	discoverCmd.SetArgs([]string{filepath.Base(wsDir)})
	err := discoverCmd.Execute()
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No changes detected") {
		t.Errorf("expected 'No changes detected', got:\n%s", output)
	}
}

func TestDiscoverCmd_NewService(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Run init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add a new service to docker-compose
	updatedCompose := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  redis:
    image: redis:7
    ports:
      - "6379:6379"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(updatedCompose), 0644)

	// Run discover - should show new service
	var out bytes.Buffer
	discoverCmd := newDiscoverCmd()
	discoverCmd.SetOut(&out)
	discoverCmd.SetArgs([]string{filepath.Base(wsDir)})
	err := discoverCmd.Execute()
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "New services") || !strings.Contains(output, "redis") {
		t.Errorf("expected new service 'redis', got:\n%s", output)
	}
}

func TestDiscoverCmd_RemovedService(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  redis:
    image: redis:7
    ports:
      - "6379:6379"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Run init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Remove a service from docker-compose
	updatedCompose := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(updatedCompose), 0644)

	// Run discover - should show removed service
	var out bytes.Buffer
	discoverCmd := newDiscoverCmd()
	discoverCmd.SetOut(&out)
	discoverCmd.SetArgs([]string{filepath.Base(wsDir)})
	err := discoverCmd.Execute()
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Removed services") || !strings.Contains(output, "redis") {
		t.Errorf("expected removed service 'redis', got:\n%s", output)
	}
}

func TestDiscoverCmd_BothNewAndRemoved(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  redis:
    image: redis:7
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Run init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Replace redis with postgres
	updatedCompose := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
  postgres:
    image: postgres:15
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(updatedCompose), 0644)

	// Run discover - should show both changes
	var out bytes.Buffer
	discoverCmd := newDiscoverCmd()
	discoverCmd.SetOut(&out)
	discoverCmd.SetArgs([]string{filepath.Base(wsDir)})
	err := discoverCmd.Execute()
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "New services") || !strings.Contains(output, "postgres") {
		t.Errorf("expected new service 'postgres', got:\n%s", output)
	}
	if !strings.Contains(output, "Removed services") || !strings.Contains(output, "redis") {
		t.Errorf("expected removed service 'redis', got:\n%s", output)
	}
}

func TestDiscoverCmd_NoDiscoverySource(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	// Create a rook.yaml manually without any discoverable files
	manifestContent := `
name: manual-workspace
type: single
services:
  app:
    command: echo hello
`
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte(manifestContent), 0644)

	// Manually register the workspace
	ctx, err := newCLIContext()
	if err != nil {
		t.Fatalf("failed to create cli context: %v", err)
	}
	ctx.registry.Register("manual-workspace", wsDir)

	// Run discover - should show that all services are removed (nothing discovered)
	var out bytes.Buffer
	discoverCmd := newDiscoverCmd()
	discoverCmd.SetOut(&out)
	discoverCmd.SetArgs([]string{"manual-workspace"})
	err = discoverCmd.Execute()
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Removed services") || !strings.Contains(output, "app") {
		t.Errorf("expected removed service 'app', got:\n%s", output)
	}
}

func TestDiscoverCmd_CurrentDirectory(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Run init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Change to workspace directory
	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	// Run discover without args - should use current directory
	var out bytes.Buffer
	discoverCmd := newDiscoverCmd()
	discoverCmd.SetOut(&out)
	discoverCmd.SetArgs([]string{})
	err := discoverCmd.Execute()
	if err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No changes detected") {
		t.Errorf("expected 'No changes detected', got:\n%s", output)
	}
}
