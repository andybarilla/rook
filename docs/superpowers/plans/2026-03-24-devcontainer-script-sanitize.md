# Devcontainer Script Sanitization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automatically strip devcontainer-specific patterns (wait loops, keep-alive commands, background operators) from scripts copied during `rook init`.

**Architecture:** A pure function `SanitizeScript` in the discovery package processes script content line-by-line, removing known devcontainer patterns and reporting changes. The init command calls it between reading and writing the script.

**Tech Stack:** Go, stdlib `testing`, `strings`, `regexp`

**Spec:** `docs/superpowers/specs/2026-03-24-devcontainer-script-sanitize-design.md`

---

### File Map

- **Create:** `internal/discovery/sanitize.go` — `ScriptChange` type + `SanitizeScript` function
- **Create:** `internal/discovery/sanitize_test.go` — all test cases
- **Modify:** `internal/cli/init.go` — call `SanitizeScript` on copied script content, print changes, adjust warning

---

### Task 1: SanitizeScript — wait loop removal

**Files:**
- Create: `internal/discovery/sanitize_test.go`
- Create: `internal/discovery/sanitize.go`

- [ ] **Step 1: Write failing test for wait loop removal**

```go
package discovery

import (
	"strings"
	"testing"
)

func TestSanitizeScript(t *testing.T) {
	t.Run("removes_wait_loop_with_comments", func(t *testing.T) {
		input := `#!/bin/bash
cd /workspaces/app

# Wait for post-create.sh to finish (it creates this marker file)
echo "Waiting for post-create to finish..."
while [ ! -f /tmp/.devcontainer-ready ]; do
  sleep 1
done

echo "ready"
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "while") {
			t.Error("expected while loop to be removed")
		}
		if strings.Contains(result, "devcontainer-ready") {
			t.Error("expected devcontainer-ready references to be removed")
		}
		if !strings.Contains(result, "echo \"ready\"") {
			t.Error("expected unrelated echo to be preserved")
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}
		if !strings.Contains(changes[0].Description, "wait loop") {
			t.Errorf("expected change description to mention wait loop, got %q", changes[0].Description)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/discovery/ -run TestSanitizeScript/removes_wait_loop -v`
Expected: FAIL — `SanitizeScript` not defined

- [ ] **Step 3: Write minimal SanitizeScript with wait loop removal**

In `internal/discovery/sanitize.go`:

```go
package discovery

import (
	"regexp"
	"strings"
)

// ScriptChange describes a sanitization change made to a script.
type ScriptChange struct {
	Description string
}

var whileConditionRe = regexp.MustCompile(`^\s*while\s+(.+);\s*do\s*$`)

// SanitizeScript removes devcontainer-specific patterns from shell scripts.
// Returns the sanitized content and a list of changes made.
func SanitizeScript(content []byte) ([]byte, []ScriptChange) {
	lines := strings.Split(string(content), "\n")
	var changes []ScriptChange

	// Rule 1: Remove wait loops (while/sleep/done blocks with preceding comments/echos)
	lines, changes = removeWaitLoops(lines, changes)

	return []byte(strings.Join(lines, "\n")), changes
}

// removeWaitLoops removes while loops whose body is only sleep commands,
// along with contiguous preceding comment and echo lines.
func removeWaitLoops(lines []string, changes []ScriptChange) ([]string, []ScriptChange) {
	var result []string
	i := 0
	for i < len(lines) {
		m := whileConditionRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r"))
		if m != nil {
			// Check if body is only sleep lines, ending with done
			bodyStart := i + 1
			bodyEnd := -1
			onlySleep := true
			for j := bodyStart; j < len(lines); j++ {
				trimmed := strings.TrimSpace(lines[j])
				if trimmed == "done" {
					bodyEnd = j
					break
				}
				if !strings.HasPrefix(trimmed, "sleep ") && trimmed != "" {
					onlySleep = false
					break
				}
			}
			if bodyEnd != -1 && onlySleep {
				// Remove preceding contiguous comment and echo lines
				for len(result) > 0 {
					trimmed := strings.TrimSpace(result[len(result)-1])
					if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "echo ") {
						result = result[:len(result)-1]
					} else {
						break
					}
				}
				condition := strings.TrimSpace(m[1])
				changes = append(changes, ScriptChange{
					Description: "Removed wait loop (while " + condition + ")",
				})
				i = bodyEnd + 1
				continue
			}
		}
		result = append(result, lines[i])
		i++
	}
	return result, changes
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/discovery/ -run TestSanitizeScript/removes_wait_loop -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/sanitize.go internal/discovery/sanitize_test.go
git commit -m "feat: SanitizeScript with wait loop removal"
```

---

### Task 2: Keep-alive command removal

**Files:**
- Modify: `internal/discovery/sanitize_test.go`
- Modify: `internal/discovery/sanitize.go`

- [ ] **Step 1: Write failing test for keep-alive removal**

Add to `TestSanitizeScript` in `sanitize_test.go`:

```go
	t.Run("removes_keep_alive_with_comments", func(t *testing.T) {
		input := `#!/bin/bash
echo "starting"

# Keep the container alive
exec sleep infinity
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "sleep infinity") {
			t.Error("expected sleep infinity to be removed")
		}
		if strings.Contains(result, "Keep the container") {
			t.Error("expected preceding comment to be removed")
		}
		if !strings.Contains(result, "echo \"starting\"") {
			t.Error("expected unrelated lines to be preserved")
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}
		if !strings.Contains(changes[0].Description, "keep-alive") {
			t.Errorf("expected change to mention keep-alive, got %q", changes[0].Description)
		}
	})

	t.Run("removes_tail_keepalive", func(t *testing.T) {
		input := `#!/bin/bash
do-stuff
tail -f /dev/null
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "tail") {
			t.Error("expected tail -f /dev/null to be removed")
		}
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}
	})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/ -run TestSanitizeScript/removes_keep_alive -v && go test ./internal/discovery/ -run TestSanitizeScript/removes_tail -v`
Expected: FAIL — keep-alive lines not removed

- [ ] **Step 3: Add keep-alive removal to SanitizeScript**

Add a `removeKeepAlive` function and call it from `SanitizeScript` after `removeWaitLoops`. The function scans lines for keep-alive patterns (`exec sleep infinity`, `sleep infinity`, `exec tail -f /dev/null`, `tail -f /dev/null`) and removes them along with contiguous preceding comment lines. Returns whether any keep-alive was removed (needed by Rule 3).

```go
var keepAlivePatterns = []string{
	"exec sleep infinity",
	"sleep infinity",
	"exec tail -f /dev/null",
	"tail -f /dev/null",
}

func removeKeepAlive(lines []string, changes []ScriptChange) ([]string, []ScriptChange, bool) {
	var result []string
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isKeepAlive := false
		for _, pat := range keepAlivePatterns {
			if trimmed == pat {
				isKeepAlive = true
				break
			}
		}
		if isKeepAlive {
			// Remove preceding contiguous comment and blank lines
			for len(result) > 0 {
				t := strings.TrimSpace(result[len(result)-1])
				if strings.HasPrefix(t, "#") || t == "" {
					result = result[:len(result)-1]
				} else {
					break
				}
			}
			changes = append(changes, ScriptChange{
				Description: "Removed keep-alive command (" + trimmed + ")",
			})
			removed = true
			continue
		}
		result = append(result, line)
	}
	return result, changes, removed
}
```

Update `SanitizeScript` to call `removeKeepAlive` after `removeWaitLoops`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/ -run TestSanitizeScript -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/sanitize.go internal/discovery/sanitize_test.go
git commit -m "feat: keep-alive command removal in SanitizeScript"
```

---

### Task 3: Background operator stripping

**Files:**
- Modify: `internal/discovery/sanitize_test.go`
- Modify: `internal/discovery/sanitize.go`

- [ ] **Step 1: Write failing tests for `&` stripping**

Add to `TestSanitizeScript`:

```go
	t.Run("strips_background_when_keepalive_removed", func(t *testing.T) {
		input := `#!/bin/bash
# Start dev servers in the background
make dev-servers &

exec sleep infinity
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, " &") {
			t.Error("expected trailing & to be stripped")
		}
		if !strings.Contains(result, "make dev-servers") {
			t.Error("expected command to be preserved without &")
		}
		if strings.Contains(result, "in the background") {
			t.Error("expected 'in the background' removed from comment")
		}
		// Expect 3 changes: keep-alive, background strip, comment update
		hasBackground := false
		for _, c := range changes {
			if strings.Contains(c.Description, "background") {
				hasBackground = true
			}
		}
		if !hasBackground {
			t.Error("expected a change about background operator removal")
		}
	})

	t.Run("preserves_background_when_no_keepalive", func(t *testing.T) {
		input := `#!/bin/bash
make dev-servers &
do-other-stuff
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if !strings.Contains(result, "make dev-servers &") {
			t.Error("expected & to be preserved when no keep-alive present")
		}
		if len(changes) != 0 {
			t.Errorf("expected no changes, got %d", len(changes))
		}
	})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/ -run TestSanitizeScript/strips_background -v && go test ./internal/discovery/ -run TestSanitizeScript/preserves_background -v`
Expected: `strips_background` FAIL, `preserves_background` PASS (no changes yet)

- [ ] **Step 3: Add background stripping logic**

Add `stripBackground` function, only called when `removeKeepAlive` returns `removed == true`. Strips trailing ` &` from command lines and removes "in the background" from comment lines.

```go
func stripBackground(lines []string, changes []ScriptChange) ([]string, []ScriptChange) {
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Strip trailing & from commands
		if strings.HasSuffix(trimmed, " &") && !strings.HasPrefix(trimmed, "#") {
			cmd := strings.TrimSuffix(trimmed, " &")
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			result = append(result, indent+cmd)
			changes = append(changes, ScriptChange{
				Description: "Removed background operator from '" + cmd + "'",
			})
			continue
		}
		// Clean "in the background" from comments
		if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "in the background") {
			cleaned := strings.Replace(line, " in the background", "", 1)
			result = append(result, cleaned)
			continue
		}
		result = append(result, line)
	}
	return result, changes
}
```

Update `SanitizeScript` to conditionally call `stripBackground` when `removedKeepAlive` is true.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/ -run TestSanitizeScript -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/sanitize.go internal/discovery/sanitize_test.go
git commit -m "feat: strip background operators when keep-alive removed"
```

---

### Task 4: Blank line collapsing and edge cases

**Files:**
- Modify: `internal/discovery/sanitize_test.go`
- Modify: `internal/discovery/sanitize.go`

- [ ] **Step 1: Write failing tests for blank line collapsing and edge cases**

Add to `TestSanitizeScript`:

```go
	t.Run("collapses_blank_lines", func(t *testing.T) {
		input := `#!/bin/bash
cd /workspaces/app

# Wait for ready
while [ -f /tmp/wait ]; do
  sleep 1
done



echo "done"
`
		out, _ := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "\n\n\n") {
			t.Error("expected consecutive blank lines to be collapsed")
		}
		if !strings.Contains(result, "echo \"done\"") {
			t.Error("expected content to be preserved")
		}
	})

	t.Run("no_changes_for_clean_script", func(t *testing.T) {
		input := `#!/bin/bash
cd /workspaces/app
make build
make run
`
		out, changes := SanitizeScript([]byte(input))

		if string(out) != input {
			t.Error("expected clean script to be returned unchanged")
		}
		if len(changes) != 0 {
			t.Errorf("expected no changes, got %d", len(changes))
		}
	})

	t.Run("full_emrai_script", func(t *testing.T) {
		input := `#!/bin/bash
cd /workspaces/emrai

# Wait for post-create.sh to finish (it creates this marker file)
echo "Waiting for post-create to finish..."
while [ ! -f /tmp/.devcontainer-ready ]; do
  sleep 1
done

# Start dev servers in the background
make dev-servers &

# Keep the container alive
exec sleep infinity
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		expected := "#!/bin/bash\ncd /workspaces/emrai\n\n# Start dev servers\nmake dev-servers\n"
		if result != expected {
			t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
		}
		if len(changes) != 3 {
			t.Fatalf("expected 3 changes, got %d: %v", len(changes), changes)
		}
	})

	t.Run("multiple_wait_loops", func(t *testing.T) {
		input := `#!/bin/bash
while [ ! -f /tmp/a ]; do
  sleep 1
done
echo "middle"
while [ ! -f /tmp/b ]; do
  sleep 2
done
echo "end"
`
		out, changes := SanitizeScript([]byte(input))
		result := string(out)

		if strings.Contains(result, "while") {
			t.Error("expected both while loops to be removed")
		}
		if !strings.Contains(result, "echo \"middle\"") || !strings.Contains(result, "echo \"end\"") {
			t.Error("expected non-loop content to be preserved")
		}
		waitChanges := 0
		for _, c := range changes {
			if strings.Contains(c.Description, "wait loop") {
				waitChanges++
			}
		}
		if waitChanges != 2 {
			t.Errorf("expected 2 wait loop changes, got %d", waitChanges)
		}
	})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/ -run TestSanitizeScript -v`
Expected: `collapses_blank_lines` and `full_emrai_script` FAIL (blank lines not collapsed)

- [ ] **Step 3: Add blank line collapsing**

Add `collapseBlankLines` function and call it last in `SanitizeScript`:

```go
func collapseBlankLines(lines []string) []string {
	var result []string
	prevBlank := false
	for _, line := range lines {
		blank := strings.TrimSpace(line) == ""
		if blank && prevBlank {
			continue
		}
		prevBlank = blank
		result = append(result, line)
	}
	// Remove trailing blank lines
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}
	// Ensure trailing newline
	if len(result) > 0 {
		result = append(result, "")
	}
	return result
}
```

- [ ] **Step 4: Run all tests to verify they pass**

Run: `go test ./internal/discovery/ -run TestSanitizeScript -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/sanitize.go internal/discovery/sanitize_test.go
git commit -m "feat: blank line collapsing and edge case tests"
```

---

### Task 5: Integrate into init.go

**Files:**
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Modify init.go to call SanitizeScript**

In `internal/cli/init.go`, between the `os.ReadFile(hostScript)` call and the `os.WriteFile(destPath, content, 0755)` call, add sanitization:

```go
				// Sanitize devcontainer-specific patterns
				content, scriptChanges := discovery.SanitizeScript(content)
```

Replace the existing print + warning block:

```go
				fmt.Printf("  Copied %s to .rook/scripts/%s\n", rel, scriptName)
				if len(scriptChanges) > 0 {
					for _, c := range scriptChanges {
						fmt.Printf("  Sanitized .rook/scripts/%s: %s\n", scriptName, c.Description)
					}
					warns.add("Verify .rook/scripts/%s — devcontainer patterns were automatically removed", scriptName)
				} else {
					warns.add("Review .rook/scripts/%s and adjust for rook (e.g., remove devcontainer-specific wait loops)", scriptName)
				}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/cli/`
Expected: Success

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 4: Manual test with emrai repo**

Run: `rm -rf /home/andy/dev/andybarilla/emrai/.rook /home/andy/dev/andybarilla/emrai/rook.yaml && go run ./cmd/rook init /home/andy/dev/andybarilla/emrai`
Expected: Output shows sanitization messages and the copied script has wait loop, sleep infinity, and `&` removed.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/init.go
git commit -m "feat: integrate script sanitization into rook init"
```
