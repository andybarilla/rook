package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
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

func TestInitCmd_GeneratesGitignore_ExistingManifest(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	// Create rook.yaml directly (skip auto-discovery)
	manifestContent := `name: testws
type: single
services:
  web:
    image: nginx:latest
    ports:
      - 8080
`
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte(manifestContent), 0644)

	// Run init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Check that .rook/.gitignore exists even without auto-discovery
	gitignorePath := filepath.Join(wsDir, ".rook", ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .rook/.gitignore to exist for pre-existing manifest: %v", err)
	}
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

func testManifest() *workspace.Manifest {
	return &workspace.Manifest{
		Name: "myapp",
		Type: workspace.TypeSingle,
		Services: map[string]workspace.Service{
			"web":      {Image: "nginx:latest", Ports: []int{8080}},
			"postgres": {Image: "postgres:16", Ports: []int{5432}},
		},
	}
}

func TestEnsureAgentMD_AppendsToCLAUDEMD(t *testing.T) {
	dir := t.TempDir()
	existing := "# My Project\n\nSome existing content.\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := testManifest()
	ensureAgentMDRookSection(dir, m)

	content, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}
	s := string(content)

	// Should preserve existing content
	if !strings.HasPrefix(s, existing) {
		t.Errorf("existing content should be preserved at start")
	}
	// Should have rook tags
	if !strings.Contains(s, "<!-- rook -->") {
		t.Error("expected <!-- rook --> opening tag")
	}
	if !strings.Contains(s, "<!-- /rook -->") {
		t.Error("expected <!-- /rook --> closing tag")
	}
	// Should mention workspace name
	if !strings.Contains(s, "myapp") {
		t.Error("expected workspace name in section")
	}
	// Should list services
	if !strings.Contains(s, "web") || !strings.Contains(s, "postgres") {
		t.Error("expected service names in section")
	}
	// Should include common commands
	if !strings.Contains(s, "rook up") {
		t.Error("expected rook up command in section")
	}
}

func TestEnsureAgentMD_AppendsToAGENTSMD(t *testing.T) {
	dir := t.TempDir()
	existing := "# Agent Instructions\n"
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(existing), 0644)

	m := testManifest()
	ensureAgentMDRookSection(dir, m)

	content, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	s := string(content)

	if !strings.HasPrefix(s, existing) {
		t.Errorf("existing content should be preserved")
	}
	if !strings.Contains(s, "<!-- rook -->") {
		t.Error("expected rook section in AGENTS.md")
	}
}

func TestEnsureAgentMD_PrefersCLAUDEMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Claude\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0644)

	m := testManifest()
	ensureAgentMDRookSection(dir, m)

	// CLAUDE.md should get the section
	claude, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(string(claude), "<!-- rook -->") {
		t.Error("expected rook section in CLAUDE.md")
	}
	// AGENTS.md should be untouched
	agents, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if strings.Contains(string(agents), "<!-- rook -->") {
		t.Error("AGENTS.md should not be modified when CLAUDE.md exists")
	}
}

func TestEnsureAgentMD_SkipsIfTagExists(t *testing.T) {
	dir := t.TempDir()
	existing := "# Project\n\n<!-- rook -->\nold rook content\n<!-- /rook -->\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := testManifest()
	ensureAgentMDRookSection(dir, m)

	content, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if string(content) != existing {
		t.Errorf("file should be unchanged when rook tags exist, got:\n%s", string(content))
	}
}

func TestEnsureAgentMD_NoFileDoesNothing(t *testing.T) {
	dir := t.TempDir()

	m := testManifest()
	ensureAgentMDRookSection(dir, m)

	// Neither file should be created
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		t.Error("should not create CLAUDE.md when it doesn't exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		t.Error("should not create AGENTS.md when it doesn't exist")
	}
}
