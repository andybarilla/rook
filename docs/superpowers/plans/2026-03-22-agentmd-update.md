# Agentmd Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `ensureAgentMDRookSection` upsert (replace existing rook tags instead of skipping) and add a `rook agentmd` CLI command.

**Architecture:** Change the existing `ensureAgentMDRookSection` to return `(string, error)` and replace content between `<!-- rook -->` / `<!-- /rook -->` tags when they exist. Add a new Cobra command that calls the same function.

**Tech Stack:** Go, Cobra, stdlib `testing`

**Spec:** `docs/superpowers/specs/2026-03-22-agentmd-update-design.md`

---

### Task 1: Change `ensureAgentMDRookSection` to upsert

**Files:**
- Modify: `internal/cli/agentmd.go:13-49`
- Modify: `internal/cli/context.go:132`
- Test: `internal/cli/init_test.go`

- [ ] **Step 1: Update test for replacement behavior**

Replace `TestEnsureAgentMD_SkipsIfTagExists` in `internal/cli/init_test.go:232-244` with:

```go
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
```

- [ ] **Step 2: Add test for replacement with different services**

Add after the previous test:

```go
func TestEnsureAgentMD_ReplacesWithDifferentServices(t *testing.T) {
	dir := t.TempDir()
	// Start with a section listing "oldservice"
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
```

- [ ] **Step 3: Add test for preserving content outside tags**

```go
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
```

- [ ] **Step 4: Add test for tags at start of file**

```go
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

	// Should not start with a blank line
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
```

- [ ] **Step 5: Add test for missing closing tag**

```go
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
```

- [ ] **Step 6: Update existing tests for new return signature**

Update all calls to `ensureAgentMDRookSection` in `init_test.go` to capture the return values:

In `TestEnsureAgentMD_AppendsToCLAUDEMD`, change `ensureAgentMDRookSection(dir, m)` to:
```go
action, err := ensureAgentMDRookSection(dir, m)
if err != nil {
	t.Fatalf("unexpected error: %v", err)
}
if action != "added" {
	t.Errorf("expected action %q, got %q", "added", action)
}
```

In `TestEnsureAgentMD_AppendsToAGENTSMD`, change `ensureAgentMDRookSection(dir, m)` to:
```go
action, err := ensureAgentMDRookSection(dir, m)
if err != nil {
	t.Fatalf("unexpected error: %v", err)
}
if action != "added" {
	t.Errorf("expected action %q, got %q", "added", action)
}
```

In `TestEnsureAgentMD_PrefersCLAUDEMD`, change `ensureAgentMDRookSection(dir, m)` to:
```go
if _, err := ensureAgentMDRookSection(dir, m); err != nil {
	t.Fatalf("unexpected error: %v", err)
}
```

In `TestEnsureAgentMD_NoFileDoesNothing`, change `ensureAgentMDRookSection(dir, m)` to:
```go
action, err := ensureAgentMDRookSection(dir, m)
if err != nil {
	t.Fatalf("unexpected error: %v", err)
}
if action != "" {
	t.Errorf("expected empty action, got %q", action)
}
```

- [ ] **Step 7: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run "TestEnsureAgentMD" -v`
Expected: FAIL — `ensureAgentMDRookSection` still returns nothing (compilation error on return values).

- [ ] **Step 8: Implement upsert in `ensureAgentMDRookSection`**

Replace `internal/cli/agentmd.go:13-49` with:

```go
// ensureAgentMDRookSection upserts a rook section in an existing CLAUDE.md or
// AGENTS.md file. It prefers CLAUDE.md if both exist. If neither exists, it
// does nothing. Returns the action taken ("added", "updated", or "") and any error.
func ensureAgentMDRookSection(dir string, m *workspace.Manifest) (string, error) {
	// Try CLAUDE.md first, then AGENTS.md
	var target string
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			target = p
			break
		}
	}
	if target == "" {
		return "", nil
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", filepath.Base(target), err)
	}

	s := string(content)
	section := buildRookSection(m)

	openTag := "<!-- rook -->"
	closeTag := "<!-- /rook -->\n"

	startIdx := strings.Index(s, openTag)
	if startIdx >= 0 {
		// Replace existing section
		endIdx := strings.Index(s, closeTag)
		if endIdx < 0 {
			return "", fmt.Errorf("found %s without matching <!-- /rook --> in %s", openTag, filepath.Base(target))
		}
		result := s[:startIdx] + section + s[endIdx+len(closeTag):]
		if err := os.WriteFile(target, []byte(result), 0644); err != nil {
			return "", fmt.Errorf("writing %s: %w", filepath.Base(target), err)
		}
		return "updated", nil
	}

	// Append new section
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "\n" + section

	if err := os.WriteFile(target, []byte(s), 0644); err != nil {
		return "", fmt.Errorf("writing %s: %w", filepath.Base(target), err)
	}
	return "added", nil
}
```

- [ ] **Step 9: Update call site in `context.go`**

Change `internal/cli/context.go:132` from:

```go
ensureAgentMDRookSection(dir, m)
```

to:

```go
if _, err := ensureAgentMDRookSection(dir, m); err != nil {
	fmt.Fprintf(os.Stderr, "Warning: cannot update agent md: %v\n", err)
}
```

- [ ] **Step 10: Run all tests**

Run: `go test ./internal/cli/ -run "TestEnsureAgentMD" -v`
Expected: All PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/cli/agentmd.go internal/cli/context.go internal/cli/init_test.go
git commit -m "feat(cli): upsert agentmd section on re-init instead of skipping"
```

---

### Task 2: Add `rook agentmd` CLI command

**Files:**
- Create: `internal/cli/agentmd_cmd.go`
- Modify: `internal/cli/root.go:31`
- Test: `internal/cli/init_test.go`

- [ ] **Step 1: Write test for agentmd command**

Add to `internal/cli/init_test.go`:

```go
func TestAgentMDCmd_UpdatesExistingSection(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifestContent := `name: testws
type: single
services:
  api:
    image: node:20
    ports:
      - 3000
`
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte(manifestContent), 0644)
	os.WriteFile(filepath.Join(wsDir, "CLAUDE.md"), []byte("# Project\n\n<!-- rook -->\nold\n<!-- /rook -->\n"), 0644)

	cmd := newAgentMDCmd()
	cmd.SetArgs([]string{wsDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agentmd command failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(wsDir, "CLAUDE.md"))
	s := string(content)

	if strings.Contains(s, "old") {
		t.Error("old section content should be replaced")
	}
	if !strings.Contains(s, "api") {
		t.Error("expected 'api' service in updated section")
	}
	if !strings.Contains(s, "testws") {
		t.Error("expected workspace name in updated section")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run "TestAgentMDCmd" -v`
Expected: FAIL — `newAgentMDCmd` does not exist.

- [ ] **Step 3: Extract `agentMDTarget` helper and create `internal/cli/agentmd_cmd.go`**

First, add to `internal/cli/agentmd.go` (after `buildRookSection`):

```go
// agentMDTarget returns the filename (CLAUDE.md or AGENTS.md) found in dir, or "".
func agentMDTarget(dir string) string {
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return name
		}
	}
	return ""
}
```

Then create `internal/cli/agentmd_cmd.go`:

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newAgentMDCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agentmd [path]",
		Short: "Update rook section in CLAUDE.md or AGENTS.md",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var dir string
			if len(args) > 0 {
				dir = args[0]
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting working directory: %w", err)
				}
				dir = cwd
			}

			manifestPath := filepath.Join(dir, "rook.yaml")
			m, err := workspace.ParseManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("parsing rook.yaml: %w", err)
			}

			action, err := ensureAgentMDRookSection(dir, m)
			if err != nil {
				return err
			}

			target := agentMDTarget(dir)
			switch action {
			case "added":
				fmt.Printf("Added rook section to %s\n", target)
			case "updated":
				fmt.Printf("Updated rook section in %s\n", target)
			default:
				fmt.Println("No CLAUDE.md or AGENTS.md found")
			}
			return nil
		},
	}
}
```

- [ ] **Step 4: Register command in `root.go`**

Add `newAgentMDCmd()` to the `cmd.AddCommand(...)` list in `internal/cli/root.go:20-32`.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/cli/ -run "TestAgentMDCmd" -v`
Expected: PASS.

- [ ] **Step 6: Run full test suite**

Run: `go test ./internal/cli/ -v`
Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/agentmd_cmd.go internal/cli/agentmd.go internal/cli/root.go internal/cli/init_test.go
git commit -m "feat(cli): add rook agentmd command for standalone section updates"
```

---

### Task 3: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Remove from "What's Not Yet Implemented"**

Remove the line `- \`rook init\` agentmd section update (currently append-only; re-init doesn't refresh services)` from `CLAUDE.md`.

- [ ] **Step 2: Add `rook agentmd` to CLI Usage**

Add to the CLI Usage section:

```
rook agentmd [path]           # Update rook section in CLAUDE.md/AGENTS.md
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for agentmd upsert and new command"
```
