# Process Crash Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the orchestrator's 1-second crash check to process services so immediate failures are detected and reported with log context.

**Architecture:** Remove the `if svc.IsContainer()` gate on the crash check in `orchestrator.go`. Both runners already implement `Status()` and `Logs()` on the `Runner` interface, so the existing code path works unchanged. Add tests using a `crashingMockRunner` that returns `StatusCrashed` and log output.

**Tech Stack:** Go, stdlib `testing`

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/orchestrator/orchestrator.go` | Modify | Remove `IsContainer()` gate on crash check (2 lines) |
| `internal/orchestrator/orchestrator_test.go` | Modify | Add crash detection tests for both service types |

---

### Task 1: Add crash detection tests and remove the gate

**Files:**
- Modify: `internal/orchestrator/orchestrator.go:131-151`
- Test: `internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write the failing test — process crash detected**

In `internal/orchestrator/orchestrator_test.go`, add a new mock runner that simulates a crash, then a test that verifies process crashes are caught:

```go
// crashingMockRunner simulates a service that crashes immediately.
type crashingMockRunner struct {
	mockRunner
	crashNames map[string]bool
	logOutput  string
}

func (m *crashingMockRunner) Status(handle runner.RunHandle) (runner.ServiceStatus, error) {
	if m.crashNames[handle.ID] {
		return runner.StatusCrashed, nil
	}
	return runner.StatusRunning, nil
}

func (m *crashingMockRunner) Logs(handle runner.RunHandle) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.logOutput)), nil
}

func TestOrchestrator_Up_DetectsProcessCrash(t *testing.T) {
	crash := &crashingMockRunner{
		crashNames: map[string]bool{"worker": true},
		logOutput:  "Error: missing DATABASE_URL",
	}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"worker": {Command: "node worker.js"},
		},
		Profiles: map[string][]string{"default": {"worker"}},
	}
	orch := orchestrator.New(crash, crash, nil)
	err := orch.Up(context.Background(), ws, "default")
	if err == nil {
		t.Fatal("expected error for crashed process")
	}
	if !strings.Contains(err.Error(), "crashed immediately") {
		t.Errorf("expected crash message, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "missing DATABASE_URL") {
		t.Errorf("expected log output in error, got: %s", err.Error())
	}
}
```

- [ ] **Step 2: Write the failing test — container crash still detected (regression check)**

```go
func TestOrchestrator_Up_DetectsContainerCrash(t *testing.T) {
	crash := &crashingMockRunner{
		crashNames: map[string]bool{"db": true},
		logOutput:  "FATAL: password authentication failed",
	}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"db": {Image: "postgres:16"},
		},
		Profiles: map[string][]string{"default": {"db"}},
	}
	orch := orchestrator.New(crash, crash, nil)
	err := orch.Up(context.Background(), ws, "default")
	if err == nil {
		t.Fatal("expected error for crashed container")
	}
	if !strings.Contains(err.Error(), "crashed immediately") {
		t.Errorf("expected crash message, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "password authentication failed") {
		t.Errorf("expected log output in error, got: %s", err.Error())
	}
}
```

- [ ] **Step 3: Write the failing test — healthy process passes check**

```go
func TestOrchestrator_Up_HealthyProcessPasses(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"worker": {Command: "node worker.js"},
		},
		Profiles: map[string][]string{"default": {"worker"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatalf("healthy process should not error: %v", err)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `cd /home/andy/dev/andybarilla/rook/.worktrees/process-status && go test ./internal/orchestrator/ -run "TestOrchestrator_Up_Detects|TestOrchestrator_Up_Healthy" -v -count=1`

Expected: `TestOrchestrator_Up_DetectsProcessCrash` FAILS (process crash not detected — error is nil). The container crash test may already pass since the gate currently allows containers. The healthy process test should pass.

- [ ] **Step 5: Remove the `if svc.IsContainer()` gate**

In `internal/orchestrator/orchestrator.go`, replace lines 131-151:

```go
		// Brief pause to catch immediate crashes (e.g., missing env vars)
		if svc.IsContainer() {
			time.Sleep(1 * time.Second)
			status, _ := r.Status(handle)
			if status == runner.StatusCrashed || status == runner.StatusStopped {
				// Fetch last logs for the error message
				var lastLogs string
				if logReader, err := r.Logs(handle); err == nil {
					if data, err := io.ReadAll(logReader); err == nil {
						lines := strings.Split(strings.TrimSpace(string(data)), "\n")
						// Show last 20 lines
						if len(lines) > 20 {
							lines = lines[len(lines)-20:]
						}
						lastLogs = "\n  " + strings.Join(lines, "\n  ")
					}
					logReader.Close()
				}
				return fmt.Errorf("service %s crashed immediately after starting%s", name, lastLogs)
			}
		}
```

with (just removing the `if` wrapper):

```go
		// Brief pause to catch immediate crashes (e.g., missing env vars)
		time.Sleep(1 * time.Second)
		status, _ := r.Status(handle)
		if status == runner.StatusCrashed || status == runner.StatusStopped {
			// Fetch last logs for the error message
			var lastLogs string
			if logReader, err := r.Logs(handle); err == nil {
				if data, err := io.ReadAll(logReader); err == nil {
					lines := strings.Split(strings.TrimSpace(string(data)), "\n")
					// Show last 20 lines
					if len(lines) > 20 {
						lines = lines[len(lines)-20:]
					}
					lastLogs = "\n  " + strings.Join(lines, "\n  ")
				}
				logReader.Close()
			}
			return fmt.Errorf("service %s crashed immediately after starting%s", name, lastLogs)
		}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /home/andy/dev/andybarilla/rook/.worktrees/process-status && go test ./internal/orchestrator/ -v -count=1`

Expected: All tests pass, including the 3 new ones. Note: the existing `TestOrchestrator_Up` test uses `mockRunner` which returns `StatusRunning`, so it still passes — the 1-second sleep is now applied to all services but healthy ones aren't affected.

Also run full test suite: `go test ./internal/... -count=1`

- [ ] **Step 7: Commit**

```bash
git add internal/orchestrator/orchestrator.go internal/orchestrator/orchestrator_test.go
git commit -m "feat(orchestrator): extend crash detection to process services

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
