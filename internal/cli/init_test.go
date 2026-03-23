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
	action, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "added" {
		t.Errorf("expected action %q, got %q", "added", action)
	}

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
	action, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "added" {
		t.Errorf("expected action %q, got %q", "added", action)
	}

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
	if _, err := ensureAgentMDRookSection(dir, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestEnsureAgentMD_ReplacesExistingSection(t *testing.T) {
	dir := t.TempDir()
	existing := "# Project\n\n<!-- rook -->\nold rook content\n<!-- /rook -->\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := testManifest()
	action, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "updated" {
		t.Errorf("expected action %q, got %q", "updated", action)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	s := string(content)

	// Old content should be gone
	if strings.Contains(s, "old rook content") {
		t.Error("old rook content should have been replaced")
	}
	// New content should be present
	if !strings.Contains(s, "postgres") || !strings.Contains(s, "web") {
		t.Error("expected current services in replaced section")
	}
	// Content before tags should be preserved
	if !strings.HasPrefix(s, "# Project\n\n") {
		t.Error("content before rook tags should be preserved")
	}
}

func TestEnsureAgentMD_ReplacesWithDifferentServices(t *testing.T) {
	dir := t.TempDir()
	existing := "# Project\n\n<!-- rook -->\n## Rook Workspace\n- `oldservice` — old:image\n<!-- /rook -->\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := &workspace.Manifest{
		Name: "myapp",
		Type: workspace.TypeSingle,
		Services: map[string]workspace.Service{
			"newservice": {Image: "new:image", Ports: []int{3000}},
		},
	}
	_, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	s := string(content)

	if strings.Contains(s, "oldservice") {
		t.Error("old service should not be in replaced section")
	}
	if !strings.Contains(s, "newservice") {
		t.Error("new service should be in replaced section")
	}
}

func TestEnsureAgentMD_PreservesContentOutsideTags(t *testing.T) {
	dir := t.TempDir()
	existing := "# Project\n\nSome intro.\n\n<!-- rook -->\nold\n<!-- /rook -->\n\n## Other Section\n\nMore content.\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := testManifest()
	_, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	s := string(content)

	if !strings.HasPrefix(s, "# Project\n\nSome intro.\n\n") {
		t.Error("content before rook tags should be preserved exactly")
	}
	if !strings.HasSuffix(s, "\n## Other Section\n\nMore content.\n") {
		t.Errorf("content after rook tags should be preserved exactly, got:\n%s", s)
	}
}

func TestEnsureAgentMD_ReplacesTagsAtStartOfFile(t *testing.T) {
	dir := t.TempDir()
	existing := "<!-- rook -->\nold\n<!-- /rook -->\n\n## Other\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := testManifest()
	_, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	s := string(content)

	if strings.HasPrefix(s, "\n") {
		t.Error("should not add leading blank line when tags are at start of file")
	}
	if !strings.HasPrefix(s, "<!-- rook -->") {
		t.Error("should start with rook tag")
	}
	if !strings.HasSuffix(s, "\n## Other\n") {
		t.Errorf("content after tags should be preserved, got:\n%s", s)
	}
}

func TestEnsureAgentMD_ErrorsOnMissingClosingTag(t *testing.T) {
	dir := t.TempDir()
	existing := "# Project\n\n<!-- rook -->\nsome content without closing tag\n"
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0644)

	m := testManifest()
	_, err := ensureAgentMDRookSection(dir, m)
	if err == nil {
		t.Fatal("expected error for missing closing tag")
	}
	if !strings.Contains(err.Error(), "<!-- /rook -->") {
		t.Errorf("error should mention missing closing tag, got: %v", err)
	}
}

func TestEnsureAgentMD_NoFileDoesNothing(t *testing.T) {
	dir := t.TempDir()

	m := testManifest()
	action, err := ensureAgentMDRookSection(dir, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if action != "" {
		t.Errorf("expected empty action, got %q", action)
	}

	// Neither file should be created
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		t.Error("should not create CLAUDE.md when it doesn't exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		t.Error("should not create AGENTS.md when it doesn't exist")
	}
}
