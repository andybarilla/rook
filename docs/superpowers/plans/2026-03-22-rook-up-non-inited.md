# Rook Up from Non-Inited Directory — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a user runs a workspace command from a directory with `rook.yaml` that isn't registered, prompt to auto-initialize instead of failing.

**Architecture:** Add `resolveAndLoadWorkspace` to `cliContext` that combines name resolution + loading + auto-init prompt. Extract `initFromManifest` from the init command's registration logic. Update all commands that call `resolveWorkspaceName` + `loadWorkspace` to use the new method.

**Tech Stack:** Go, stdlib `testing`, `bufio` for TTY prompting

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/cli/context.go` | Modify | Add `resolveAndLoadWorkspace`, `initFromManifest` |
| `internal/cli/context_test.go` | Modify | Tests for new methods |
| `internal/cli/init.go` | Modify | Reuse `initFromManifest` instead of inline registration logic |
| `internal/cli/up.go` | Modify | Switch to `resolveAndLoadWorkspace` |
| `internal/cli/restart.go` | Modify | Switch to `resolveAndLoadWorkspace` |
| `internal/cli/logs.go` | Modify | Switch to `resolveAndLoadWorkspace` |
| `internal/cli/check_builds.go` | Modify | Switch to `resolveAndLoadWorkspace` |
| `internal/cli/status.go` | Modify | Both branches switch to `resolveAndLoadWorkspace` |

---

### Task 1: Extract `initFromManifest` helper and refactor `init.go`

**Files:**
- Modify: `internal/cli/context.go` (add the helper)
- Modify: `internal/cli/init.go:147-194` (replace inline registration with helper call)
- Test: `internal/cli/context_test.go`

- [ ] **Step 1: Write the failing test for `initFromManifest`**

In `internal/cli/context_test.go`, add:

```go
func TestInitFromManifest(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	// Create a workspace directory with rook.yaml
	wsDir := t.TempDir()
	manifest := []byte("name: testws\nservices:\n  web:\n    image: nginx\n    ports:\n      - 8080\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	cctx, err := newCLIContext()
	if err != nil {
		t.Fatal(err)
	}

	err = cctx.initFromManifest(wsDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify workspace is registered
	entry, err := cctx.registry.Get("testws")
	if err != nil {
		t.Fatalf("workspace not registered: %v", err)
	}
	if entry.Path != wsDir {
		t.Errorf("expected path %s, got %s", wsDir, entry.Path)
	}

	// Verify port was allocated (Get returns LookupResult, not (int, error))
	result := cctx.portAlloc.Get("testws", "web")
	if !result.OK {
		t.Fatal("port not allocated")
	}
	if result.Port < 10000 || result.Port > 60000 {
		t.Errorf("port %d out of expected range", result.Port)
	}

	// Verify .rook/.gitignore was created
	gitignorePath := filepath.Join(wsDir, ".rook", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		t.Error(".rook/.gitignore was not created")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/cli/ -run TestInitFromManifest -v`
Expected: FAIL — `initFromManifest` does not exist yet.

- [ ] **Step 3: Implement `initFromManifest` on `cliContext`**

In `internal/cli/context.go`, add:

```go
func (c *cliContext) initFromManifest(dir string) error {
	manifestPath := filepath.Join(dir, "rook.yaml")
	m, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return err
	}

	rookDir := filepath.Join(dir, ".rook")
	if err := ensureRookGitignore(rookDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot create .rook/.gitignore: %v\n", err)
	}

	ensureAgentMDRookSection(dir, m)

	if err := c.registry.Register(m.Name, dir); err != nil {
		return err
	}

	for name, svc := range m.Services {
		if svc.PinPort > 0 {
			allocated, err := c.portAlloc.AllocatePinned(m.Name, name, svc.PinPort)
			if err != nil {
				return fmt.Errorf("pinning port for %s: %w", name, err)
			}
			fmt.Printf("  %s.%s -> :%d (pinned)\n", m.Name, name, allocated)
		} else if len(svc.Ports) > 0 {
			allocated, err := c.portAlloc.Allocate(m.Name, name)
			if err != nil {
				return fmt.Errorf("allocating port for %s: %w", name, err)
			}
			fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
		}
	}
	fmt.Printf("Workspace %q registered from %s\n", m.Name, dir)
	return nil
}
```

Add `"os"` to `context.go` imports (if not already present).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/cli/ -run TestInitFromManifest -v`
Expected: PASS

- [ ] **Step 5: Refactor `init.go` to call `initFromManifest`**

In `internal/cli/init.go`, replace lines 147–194 (from `m, err := workspace.ParseManifest(manifestPath)` through `return nil`) with:

```go
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			if err := cctx.initFromManifest(dir); err != nil {
				return err
			}
			warns.print(os.Stderr)
			return nil
```

Remove the now-unused imports from `init.go`: `"github.com/andybarilla/rook/internal/ports"` and `"github.com/andybarilla/rook/internal/registry"`.

**Keep** `"github.com/andybarilla/rook/internal/workspace"` — it's still used by the discovery block (`workspace.WriteManifest`, `workspace.Manifest`, `workspace.TypeSingle`).

- [ ] **Step 6: Run all tests to verify refactor didn't break anything**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/cli/ -v`
Expected: All tests pass. Also: `go build ./cmd/rook/`

- [ ] **Step 7: Commit**

```bash
git add internal/cli/context.go internal/cli/context_test.go internal/cli/init.go
git commit -m "feat(cli): extract initFromManifest helper from init command"
```

---

### Task 2: Add `resolveAndLoadWorkspace` with auto-init prompt

**Files:**
- Modify: `internal/cli/context.go` (add `resolveAndLoadWorkspace`)
- Test: `internal/cli/context_test.go`

- [ ] **Step 1: Write the failing test — happy path (prompt answered yes)**

In `internal/cli/context_test.go`, add `"strings"` to the import block, then add:

```go
func TestResolveAndLoadWorkspace_AutoInit(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	// Create workspace dir with rook.yaml but don't register it
	wsDir := t.TempDir()
	manifest := []byte("name: autows\nservices:\n  api:\n    image: node\n    ports:\n      - 3000\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	cctx, _ := newCLIContext()

	// Simulate user typing "y\n" on stdin
	r, w, _ := os.Pipe()
	w.WriteString("y\n")
	w.Close()

	ws, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "autows" {
		t.Errorf("expected workspace name autows, got %s", ws.Name)
	}

	// Verify it was registered
	_, err = cctx.registry.Get("autows")
	if err != nil {
		t.Error("workspace was not registered after auto-init")
	}
}
```

- [ ] **Step 2: Write the failing test — prompt answered no**

```go
func TestResolveAndLoadWorkspace_Declined(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: declinews\nservices:\n  api:\n    image: node\n    ports:\n      - 3000\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	cctx, _ := newCLIContext()

	// Simulate user typing "n\n"
	r, w, _ := os.Pipe()
	w.WriteString("n\n")
	w.Close()

	_, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err == nil {
		t.Fatal("expected error when user declines")
	}
	if !strings.Contains(err.Error(), "rook init") {
		t.Errorf("expected hint about rook init, got: %s", err.Error())
	}
}
```

- [ ] **Step 3: Write the failing test — explicit name arg (no prompt)**

```go
func TestResolveAndLoadWorkspace_ExplicitNameNotFound(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	cctx, _ := newCLIContext()

	// Pass explicit name that doesn't exist — should error without prompting
	r, w, _ := os.Pipe()
	w.Close() // Empty stdin — if it tries to read, it'll get EOF

	_, err := cctx.resolveAndLoadWorkspace([]string{"nonexistent"}, r)
	if err == nil {
		t.Fatal("expected error for unregistered explicit name")
	}
	if strings.Contains(err.Error(), "Initialize") {
		t.Error("should not prompt for explicit name arg")
	}
}
```

- [ ] **Step 4: Write the failing test — already registered (normal path)**

```go
func TestResolveAndLoadWorkspace_AlreadyRegistered(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: existingws\nservices:\n  db:\n    image: postgres\n    ports:\n      - 5432\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	cctx, _ := newCLIContext()
	// Pre-register the workspace
	cctx.registry.Register("existingws", wsDir)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	// Stdin is closed — should not need to prompt
	r, w, _ := os.Pipe()
	w.Close()

	ws, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "existingws" {
		t.Errorf("expected existingws, got %s", ws.Name)
	}
}
```

- [ ] **Step 5: Write the failing test — non-TTY (EOF on stdin, no prompt)**

```go
func TestResolveAndLoadWorkspace_NonTTY(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir := t.TempDir()
	manifest := []byte("name: nonttyws\nservices:\n  api:\n    image: node\n    ports:\n      - 3000\n")
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifest, 0644)

	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	cctx, _ := newCLIContext()

	// Closed pipe simulates non-interactive stdin (EOF immediately)
	r, w, _ := os.Pipe()
	w.Close()

	_, err := cctx.resolveAndLoadWorkspace(nil, r)
	if err == nil {
		t.Fatal("expected error for non-TTY stdin")
	}
	if !strings.Contains(err.Error(), "rook init") {
		t.Errorf("expected hint about rook init, got: %s", err.Error())
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/cli/ -run TestResolveAndLoadWorkspace -v`
Expected: FAIL — `resolveAndLoadWorkspace` does not exist yet.

- [ ] **Step 7: Implement `resolveAndLoadWorkspace`**

In `internal/cli/context.go`, add:

```go
func (c *cliContext) resolveAndLoadWorkspace(args []string, stdin *os.File) (*workspace.Workspace, error) {
	fromCwd := len(args) == 0
	name, err := c.resolveWorkspaceName(args)
	if err != nil {
		return nil, err
	}

	ws, err := c.loadWorkspace(name)
	if err == nil {
		return ws, nil
	}

	// Only offer auto-init when name was inferred from cwd rook.yaml
	if !fromCwd {
		return nil, err
	}

	// Check if the error is "not found" (vs some other failure)
	if !strings.Contains(err.Error(), "not found") {
		return nil, err
	}

	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		return nil, err
	}

	// Prompt for confirmation — read from stdin; EOF means non-interactive
	fmt.Printf("Workspace %q is not registered. Initialize it now? [Y/n]: ", name)
	reader := bufio.NewReader(stdin)
	input, readErr := reader.ReadString('\n')
	if readErr != nil {
		// Non-TTY or EOF — fall back to error
		return nil, fmt.Errorf("workspace %q not found. Run \"rook init .\" to register this workspace", name)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "n" || input == "no" {
		return nil, fmt.Errorf("run \"rook init .\" to register this workspace")
	}

	if err := c.initFromManifest(cwd); err != nil {
		return nil, fmt.Errorf("auto-init failed: %w", err)
	}

	return c.loadWorkspace(name)
}
```

Add imports to `context.go`: `"bufio"`, `"strings"`.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/cli/ -run TestResolveAndLoadWorkspace -v`
Expected: All 5 tests pass.

- [ ] **Step 9: Commit**

```bash
git add internal/cli/context.go internal/cli/context_test.go
git commit -m "feat(cli): add resolveAndLoadWorkspace with auto-init prompt"
```

---

### Task 3: Wire commands to use `resolveAndLoadWorkspace`

**Files:**
- Modify: `internal/cli/up.go:41-48`
- Modify: `internal/cli/restart.go:20-27`
- Modify: `internal/cli/logs.go:30-38`
- Modify: `internal/cli/check_builds.go:32-40`
- Modify: `internal/cli/status.go:12-27, 71-75`

Each command currently does:
```go
wsName, err := cctx.resolveWorkspaceName(args)
if err != nil { return err }
ws, err := cctx.loadWorkspace(wsName)
if err != nil { return err }
```

Replace with:
```go
ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
if err != nil { return err }
```

Then use `ws.Name` wherever `wsName` was used after this point.

- [ ] **Step 1: Update `up.go`**

In `internal/cli/up.go`, replace lines 41-48:
```go
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}
```
with:
```go
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}
			wsName := ws.Name
```

- [ ] **Step 2: Update `restart.go`**

In `internal/cli/restart.go`, replace lines 20-27:
```go
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}
			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}
```
with:
```go
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}
			wsName := ws.Name
```

Add `"os"` to `restart.go` imports if not present.

- [ ] **Step 3: Update `logs.go`**

In `internal/cli/logs.go`, replace lines 30-38:
```go
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}
```
with:
```go
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}
			wsName := ws.Name
```

- [ ] **Step 4: Update `check_builds.go`**

In `internal/cli/check_builds.go`, replace lines 32-40:
```go
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}
```
with:
```go
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}
			wsName := ws.Name
```

- [ ] **Step 5: Update `status.go`**

The `status` command has two branches: `len(args) == 0` calls `showAllWorkspaces` (lists all registered), and `len(args) > 0` calls `showWorkspaceDetail`. Both branches need updating to support auto-init from cwd.

Replace the entire command handler (lines 16-25):
```go
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return showAllWorkspaces(cctx)
			}
			return showWorkspaceDetail(cctx, args[0])
		},
```
with:
```go
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			// Try to resolve a workspace (from arg or cwd).
			// If no arg and no rook.yaml in cwd, fall back to listing all.
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				if len(args) == 0 {
					return showAllWorkspaces(cctx)
				}
				return err
			}
			return showWorkspaceDetail(cctx, ws)
		},
```

Update `showWorkspaceDetail` to accept `*workspace.Workspace` instead of `string` (lines 71-75):
```go
func showWorkspaceDetail(cctx *cliContext, ws *workspace.Workspace) error {
	prefix := fmt.Sprintf("rook_%s_", ws.Name)
	fmt.Printf("%-20s %-12s %-12s %-8s\n", "SERVICE", "TYPE", "STATUS", "PORT")
	for name, svc := range ws.Services {
```

And update the port lookup (line 87) to use `ws.Name`:
```go
		if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
```

Add `"os"` to status.go imports.

- [ ] **Step 6: Verify all tests pass and build compiles**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/cli/ -v && go build ./cmd/rook/`
Expected: All tests pass, build succeeds.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/up.go internal/cli/restart.go internal/cli/logs.go internal/cli/check_builds.go internal/cli/status.go
git commit -m "feat(cli): wire resolveAndLoadWorkspace into all workspace commands"
```
