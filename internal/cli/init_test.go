package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd_GeneratesGitignore(t *testing.T) {
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

	// Check that .rook/.gitignore exists
	gitignorePath := filepath.Join(wsDir, ".rook", ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .rook/.gitignore to exist, got error: %v", err)
	}

	// Check that it contains .cache/
	if !strings.Contains(string(content), ".cache/") {
		t.Errorf("expected .rook/.gitignore to contain '.cache/', got: %s", string(content))
	}
}

func TestInitCmd_CreatesScriptsDir(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	wsName := filepath.Base(wsDir)

	// Create .devcontainer directory with docker-compose.yml
	devcontainerDir := filepath.Join(wsDir, ".devcontainer")
	os.MkdirAll(devcontainerDir, 0755)

	// Use the actual workspace name in the paths
	composeContent := `
services:
  app:
    build:
      context: ..
      dockerfile: Dockerfile
    command: sh /workspaces/` + wsName + `/.devcontainer/start.sh
    volumes:
      - ..:/workspaces/` + wsName + `
`
	os.WriteFile(filepath.Join(devcontainerDir, "docker-compose.yml"), []byte(composeContent), 0644)

	// Create the script that should be copied
	scriptContent := `#!/bin/sh
echo "Starting devcontainer..."
sleep infinity
`
	os.WriteFile(filepath.Join(devcontainerDir, "start.sh"), []byte(scriptContent), 0755)

	// Run init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Check that script was copied to .rook/scripts/ instead of .rook/
	scriptPath := filepath.Join(wsDir, ".rook", "scripts", "start.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Errorf("expected script to be copied to .rook/scripts/start.sh, got error: %v", err)
	}

	// Verify script content was copied correctly
	copiedContent, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read copied script: %v", err)
	}
	if string(copiedContent) != scriptContent {
		t.Errorf("script content mismatch: got %q, want %q", string(copiedContent), scriptContent)
	}

	// Verify .rook/.gitignore also exists (it should be created regardless)
	gitignorePath := filepath.Join(wsDir, ".rook", ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		t.Errorf("expected .rook/.gitignore to exist, got error: %v", err)
	}
}
