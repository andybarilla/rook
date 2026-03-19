# Force Rook-Range Ports Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the preferred port parameter from `Allocate()` so all non-pinned ports are allocated from the 10000-60000 range.

**Architecture:** The `PortAllocator` interface loses its `preferred int` parameter. `FileAllocator.Allocate()` drops the "try preferred first" branch. All call sites (CLI, orchestrator, API) are updated to the new 2-arg signature.

**Tech Stack:** Go, stdlib testing

---

## File Structure

| Action | File | Purpose |
|--------|------|---------|
| Modify | `internal/ports/allocator.go` | Interface + implementation change |
| Modify | `internal/ports/allocator_test.go` | Update tests for new signature |
| Modify | `internal/cli/up.go:154` | Drop preferred arg |
| Modify | `internal/cli/init.go:186-193` | Replace per-port loop with single call |
| Modify | `internal/orchestrator/orchestrator.go:63,231` | Drop preferred arg (2 sites) |
| Modify | `internal/api/workspace.go:211,585` | Drop preferred arg (2 sites) |
| Modify | `internal/api/workspace_test.go:28-30` | Update stub signature |

---

### Task 1: Update PortAllocator Interface and Implementation

**Files:**
- Modify: `internal/ports/allocator.go:27` (interface)
- Modify: `internal/ports/allocator.go:100-123` (implementation)
- Test: `internal/ports/allocator_test.go`

- [ ] **Step 1: Update test for new signature**

In `internal/ports/allocator_test.go`, remove the `preferred` argument from all `Allocate()` calls:

```go
// TestAllocate_AssignsFromRange — line 16: change arg from 0 to no third arg
port, err := a.Allocate("ws1", "postgres")

// TestAllocate_PreferredPort — this test no longer applies.
// Replace with a test that verifies allocation always comes from range:
func TestAllocate_AlwaysFromRange(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 49100, 49110)
	port, _ := a.Allocate("ws1", "app")
	if port < 49100 || port > 49110 {
		t.Errorf("expected port in range 49100-49110, got %d", port)
	}
}

// TestAllocate_StablePorts — line 38: drop third arg
port1, _ := a1.Allocate("ws1", "postgres")

// TestAllocate_NoConflicts — lines 52-55: drop third arg from all calls
a.Allocate("ws1", "a")
a.Allocate("ws1", "b")
a.Allocate("ws1", "c")
_, err := a.Allocate("ws1", "d")

// TestRelease — lines 74, 75, 77: drop third arg
a.Allocate("ws1", "a")
a.Allocate("ws1", "b")
// ...
_, err := a.Allocate("ws1", "c")

// TestAll — lines 86-87: drop third arg
a.Allocate("ws1", "postgres")
a.Allocate("ws2", "redis")
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ports/ -v`
Expected: compilation errors — `Allocate` called with wrong number of arguments.

- [ ] **Step 3: Update interface and implementation**

In `internal/ports/allocator.go`:

Change the interface (line 27):
```go
Allocate(workspace, service string) (int, error)
```

Change the method signature and remove the preferred branch (lines 100-123):
```go
// Allocate assigns a port for the given workspace and service from the
// allocator range. If the workspace/service already has an allocation,
// the existing port is returned.
func (a *FileAllocator) Allocate(workspace, service string) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if idx := a.findIndex(workspace, service); idx >= 0 {
		return a.entries[idx].Port, nil
	}

	for p := a.minPort; p <= a.maxPort; p++ {
		if !a.used[p] && portAvailable(p) {
			return a.assign(workspace, service, p, false)
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", a.minPort, a.maxPort)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ports/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/allocator.go internal/ports/allocator_test.go
git commit -m "feat: remove preferred port from Allocate(), always use rook range"
```

---

### Task 2: Update CLI Call Sites

**Files:**
- Modify: `internal/cli/up.go:153-154`
- Modify: `internal/cli/init.go:185-193`

- [ ] **Step 1: Update up.go**

In `internal/cli/up.go`, change line 154 from:
```go
port, err := cctx.portAlloc.Allocate(ws.Name, name, svc.Ports[0])
```
to:
```go
port, err := cctx.portAlloc.Allocate(ws.Name, name)
```

- [ ] **Step 2: Update init.go**

In `internal/cli/init.go`, replace the per-port loop (lines 185-193):
```go
} else {
	for _, port := range svc.Ports {
		allocated, err := alloc.Allocate(m.Name, name, port)
		if err != nil {
			return fmt.Errorf("allocating port for %s: %w", name, err)
		}
		fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
	}
}
```

with a single allocation call:
```go
} else if len(svc.Ports) > 0 {
	allocated, err := alloc.Allocate(m.Name, name)
	if err != nil {
		return fmt.Errorf("allocating port for %s: %w", name, err)
	}
	fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/cli/`
Expected: compiles cleanly

- [ ] **Step 4: Commit**

```bash
git add internal/cli/up.go internal/cli/init.go
git commit -m "feat: update CLI call sites for new Allocate() signature"
```

---

### Task 3: Update Orchestrator Call Sites

**Files:**
- Modify: `internal/orchestrator/orchestrator.go:63,231`

- [ ] **Step 1: Update Up() method**

In `internal/orchestrator/orchestrator.go`, change line 63 from:
```go
port, err := o.portAllocator.Allocate(ws.Name, name, svc.Ports[0])
```
to:
```go
port, err := o.portAllocator.Allocate(ws.Name, name)
```

- [ ] **Step 2: Update StartService() method**

Change line 231 from:
```go
port, err := o.portAllocator.Allocate(ws.Name, serviceName, svc.Ports[0])
```
to:
```go
port, err := o.portAllocator.Allocate(ws.Name, serviceName)
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/orchestrator/`
Expected: compiles cleanly

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/orchestrator.go
git commit -m "feat: update orchestrator call sites for new Allocate() signature"
```

---

### Task 4: Update API Layer and Test Stub

**Files:**
- Modify: `internal/api/workspace.go:211,585`
- Modify: `internal/api/workspace_test.go:28-30`

- [ ] **Step 1: Update workspace.go call sites**

In `internal/api/workspace.go`, change line 211 from:
```go
if _, err := w.portAlloc.Allocate(name, svcName, svc.Ports[0]); err != nil {
```
to:
```go
if _, err := w.portAlloc.Allocate(name, svcName); err != nil {
```

Change line 585 from:
```go
if _, err := w.portAlloc.Allocate(name, svcName, svc.Ports[0]); err != nil {
```
to:
```go
if _, err := w.portAlloc.Allocate(name, svcName); err != nil {
```

- [ ] **Step 2: Update test stub**

In `internal/api/workspace_test.go`, change the stub (lines 28-30) from:
```go
func (s *stubPortAlloc) Allocate(workspace, service string, preferred int) (int, error) {
	return preferred, nil
}
```
to:
```go
func (s *stubPortAlloc) Allocate(workspace, service string) (int, error) {
	return 10000, nil
}
```

- [ ] **Step 3: Run all tests**

Run: `go test ./internal/api/ -v`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/api/workspace.go internal/api/workspace_test.go
git commit -m "feat: update API layer for new Allocate() signature"
```

---

### Task 5: Full Test Suite and Cleanup

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: all packages PASS. If any fail due to missed `Allocate` call sites, fix them.

- [ ] **Step 2: Update CLAUDE.md**

Remove "Force rook ports flag" from the "What's Not Yet Implemented" section.

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: remove force rook ports from not-yet-implemented list"
```
