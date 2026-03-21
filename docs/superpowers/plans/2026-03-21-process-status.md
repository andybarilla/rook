# Process Service Status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `rook status` show accurate status (running/stopped/crashed) for process services across CLI sessions using PID file tracking.

**Architecture:** New `pidfile.go` in `internal/runner/` handles PID file I/O and cross-platform liveness checks. `ProcessRunner` writes/removes PID files during Start/Stop and gains a `Reconnect` method for adopting processes from previous sessions. The CLI status command reads PID files directly for quick status queries. The orchestrator's `Reconnect` is extended to also discover process services.

**Tech Stack:** Go stdlib (`os`, `encoding/json`, `syscall`), existing `internal/runner` and `internal/orchestrator` packages.

**Spec:** `docs/superpowers/specs/2026-03-21-process-status-design.md`

---

### Task 1: PID File I/O — `internal/runner/pidfile.go`

**Files:**
- Create: `internal/runner/pidfile.go`
- Create: `internal/runner/pidfile_test.go`

- [ ] **Step 1: Write failing tests for PIDDirPath, WritePIDFile, ReadPIDFile, RemovePIDFile, ListPIDFiles**

In `internal/runner/pidfile_test.go`:

```go
package runner_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
)

func TestPIDDirPath(t *testing.T) {
	got := runner.PIDDirPath("/home/user/myproject")
	want := filepath.Join("/home/user/myproject", ".rook", ".cache", "pids")
	if got != want {
		t.Errorf("PIDDirPath = %q, want %q", got, want)
	}
}

func TestWriteReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	info := runner.PIDInfo{
		PID:       12345,
		Command:   "make run",
		StartedAt: time.Now().Truncate(time.Second),
	}
	if err := runner.WritePIDFile(dir, "api", info); err != nil {
		t.Fatal(err)
	}
	got, err := runner.ReadPIDFile(dir, "api")
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != info.PID {
		t.Errorf("PID = %d, want %d", got.PID, info.PID)
	}
	if got.Command != info.Command {
		t.Errorf("Command = %q, want %q", got.Command, info.Command)
	}
	if !got.StartedAt.Equal(info.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, info.StartedAt)
	}
}

func TestReadPIDFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := runner.ReadPIDFile(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent PID file")
	}
}

func TestRemovePIDFile(t *testing.T) {
	dir := t.TempDir()
	info := runner.PIDInfo{PID: 1, Command: "echo", StartedAt: time.Now()}
	runner.WritePIDFile(dir, "svc", info)
	if err := runner.RemovePIDFile(dir, "svc"); err != nil {
		t.Fatal(err)
	}
	_, err := runner.ReadPIDFile(dir, "svc")
	if err == nil {
		t.Error("expected error after removal")
	}
}

func TestRemovePIDFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	// Should not error on missing file
	if err := runner.RemovePIDFile(dir, "nonexistent"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListPIDFiles(t *testing.T) {
	dir := t.TempDir()
	runner.WritePIDFile(dir, "api", runner.PIDInfo{PID: 1, Command: "a", StartedAt: time.Now()})
	runner.WritePIDFile(dir, "worker", runner.PIDInfo{PID: 2, Command: "b", StartedAt: time.Now()})

	names, err := runner.ListPIDFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 PID files, got %d", len(names))
	}
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	if !got["api"] || !got["worker"] {
		t.Errorf("expected api and worker, got %v", names)
	}
}

func TestListPIDFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names, err := runner.ListPIDFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0, got %d", len(names))
	}
}

func TestListPIDFiles_NonexistentDir(t *testing.T) {
	names, err := runner.ListPIDFiles("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0, got %d", len(names))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -run "TestPIDDirPath|TestWriteReadPIDFile|TestReadPIDFile_NotFound|TestRemovePIDFile|TestListPIDFiles" -v`
Expected: compilation errors — `PIDDirPath`, `PIDInfo`, `WritePIDFile`, etc. not defined

- [ ] **Step 3: Implement pidfile.go**

Create `internal/runner/pidfile.go`:

```go
package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PIDInfo is the data stored in a PID file for a running process service.
type PIDInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	StartedAt time.Time `json:"started_at"`
}

// PIDDirPath returns the directory where PID files are stored for a workspace.
func PIDDirPath(wsRoot string) string {
	return filepath.Join(wsRoot, ".rook", ".cache", "pids")
}

// WritePIDFile writes a PID file for the named service into dir.
func WritePIDFile(dir, serviceName string, info PIDInfo) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating pid dir: %w", err)
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(pidFilePath(dir, serviceName), data, 0644)
}

// ReadPIDFile reads the PID file for the named service from dir.
func ReadPIDFile(dir, serviceName string) (*PIDInfo, error) {
	data, err := os.ReadFile(pidFilePath(dir, serviceName))
	if err != nil {
		return nil, err
	}
	var info PIDInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// RemovePIDFile removes the PID file for the named service. No error if missing.
func RemovePIDFile(dir, serviceName string) error {
	err := os.Remove(pidFilePath(dir, serviceName))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ListPIDFiles returns service names that have PID files in dir.
// Returns empty slice (not error) if dir does not exist.
func ListPIDFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pid") {
			names = append(names, strings.TrimSuffix(name, ".pid"))
		}
	}
	return names, nil
}

func pidFilePath(dir, serviceName string) string {
	return filepath.Join(dir, serviceName+".pid")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/ -run "TestPIDDirPath|TestWriteReadPIDFile|TestReadPIDFile_NotFound|TestRemovePIDFile|TestListPIDFiles" -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/pidfile.go internal/runner/pidfile_test.go
git commit -m "feat(runner): add PID file I/O for process service tracking"
```

---

### Task 2: IsProcessAlive — cross-platform liveness check

**Files:**
- Modify: `internal/runner/pidfile.go`
- Modify: `internal/runner/pidfile_test.go`

- [ ] **Step 1: Write failing tests for IsProcessAlive**

Append to `internal/runner/pidfile_test.go`:

```go
func TestIsProcessAlive_RunningProcess(t *testing.T) {
	// Start a long-running subprocess
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	if !runner.IsProcessAlive(cmd.Process.Pid) {
		t.Error("expected alive for running process")
	}
}

func TestIsProcessAlive_DeadProcess(t *testing.T) {
	// Start and immediately kill a subprocess
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Kill()
	cmd.Wait()

	if runner.IsProcessAlive(pid) {
		t.Error("expected dead for killed process")
	}
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	// PID 0 and negative should return false
	if runner.IsProcessAlive(0) {
		t.Error("expected false for PID 0")
	}
	if runner.IsProcessAlive(-1) {
		t.Error("expected false for negative PID")
	}
}
```

Note: add `"os/exec"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -run "TestIsProcessAlive" -v`
Expected: compilation error — `IsProcessAlive` not defined

- [ ] **Step 3: Implement IsProcessAlive**

Add `"syscall"` to the existing import block in `internal/runner/pidfile.go`, then add the function:

```go
// IsProcessAlive checks whether a process with the given PID is still running.
// Uses signal 0 on Unix (Linux/macOS). Returns false for invalid PIDs.
// Note: on Linux, signal 0 succeeds for zombie processes (exited but not yet
// reaped). This is acceptable — stale PID files are cleaned up on next check.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/ -run "TestIsProcessAlive" -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/pidfile.go internal/runner/pidfile_test.go
git commit -m "feat(runner): add cross-platform IsProcessAlive liveness check"
```

---

### Task 3: ProcessRunner — PID file integration in Start/Stop

**Files:**
- Modify: `internal/runner/process.go`
- Modify: `internal/runner/process_test.go`

- [ ] **Step 1: Write failing tests for PID file creation on Start and removal on Stop**

Append to `internal/runner/process_test.go`:

```go
func TestProcessRunner_Start_CreatesPIDFile(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	svc := workspace.Service{Command: "sleep 60"}
	handle, err := r.Start(context.Background(), "api", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop(handle)

	info, err := runner.ReadPIDFile(pidDir, "api")
	if err != nil {
		t.Fatalf("PID file not created: %v", err)
	}
	if info.PID <= 0 {
		t.Errorf("expected positive PID, got %d", info.PID)
	}
	if info.Command != "sleep 60" {
		t.Errorf("expected command 'sleep 60', got %q", info.Command)
	}
}

func TestProcessRunner_Stop_RemovesPIDFile(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	svc := workspace.Service{Command: "sleep 60"}
	handle, err := r.Start(context.Background(), "api", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// PID file should exist
	if _, err := runner.ReadPIDFile(pidDir, "api"); err != nil {
		t.Fatalf("PID file not created: %v", err)
	}

	r.Stop(handle)

	// PID file should be gone
	if _, err := runner.ReadPIDFile(pidDir, "api"); err == nil {
		t.Error("PID file should have been removed after Stop")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -run "TestProcessRunner_Start_CreatesPIDFile|TestProcessRunner_Stop_RemovesPIDFile" -v`
Expected: compilation error — `SetPIDDir` not defined

- [ ] **Step 3: Implement SetPIDDir, update Start to write PID file, update Stop to remove PID file**

In `internal/runner/process.go`:

Add `pidDir string` field to `ProcessRunner` struct (line 29, alongside `logDir`).

Add `SetPIDDir` method:

```go
// SetPIDDir sets the directory for PID files.
// Must be called before Start(). When set, a PID file is written after
// each process starts and removed when it stops.
func (r *ProcessRunner) SetPIDDir(dir string) {
	r.pidDir = dir
}
```

In `Start()`, after `r.entries[name] = entry` (line 102) and before the return, add:

```go
if r.pidDir != "" {
	WritePIDFile(r.pidDir, name, PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   svc.Command,
		StartedAt: time.Now(),
	})
}
```

In `Stop()`, after `<-entry.done` (line 114), add:

```go
if r.pidDir != "" {
	RemovePIDFile(r.pidDir, handle.ID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/ -run "TestProcessRunner_Start_CreatesPIDFile|TestProcessRunner_Stop_RemovesPIDFile" -v`
Expected: all PASS

- [ ] **Step 5: Run all existing process tests to verify no regressions**

Run: `go test ./internal/runner/ -run "TestProcessRunner" -v`
Expected: all PASS (existing tests don't set pidDir, so PID file code is skipped)

- [ ] **Step 6: Commit**

```bash
git add internal/runner/process.go internal/runner/process_test.go
git commit -m "feat(runner): ProcessRunner writes/removes PID files on Start/Stop"
```

---

### Task 4: ProcessRunner — Reconnect and reconnected entry status

**Files:**
- Modify: `internal/runner/process.go`
- Modify: `internal/runner/process_test.go`

- [ ] **Step 1: Write failing tests for Reconnect and Status on reconnected entries**

Append to `internal/runner/process_test.go`:

```go
func TestProcessRunner_Reconnect_AliveProcess(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	// Start a real process out-of-band
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Write PID file manually (simulating a previous session)
	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	handle, err := r.Reconnect("worker")
	if err != nil {
		t.Fatal(err)
	}
	if handle.Type != "process" {
		t.Errorf("expected type process, got %s", handle.Type)
	}

	status, _ := r.Status(handle)
	if status != runner.StatusRunning {
		t.Errorf("expected running, got %s", status)
	}
}

func TestProcessRunner_Reconnect_DeadProcess(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	// Start and kill a process
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Kill()
	cmd.Wait()

	// Write stale PID file
	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	_, err := r.Reconnect("worker")
	if err == nil {
		t.Error("expected error for dead process")
	}

	// PID file should have been cleaned up
	if _, readErr := runner.ReadPIDFile(pidDir, "worker"); readErr == nil {
		t.Error("stale PID file should have been removed")
	}
}

func TestProcessRunner_Reconnect_NoPIDFile(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	_, err := r.Reconnect("nonexistent")
	if err == nil {
		t.Error("expected error when no PID file exists")
	}
}

func TestProcessRunner_Status_ReconnectedDies(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	// Start a real process
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	handle, err := r.Reconnect("worker")
	if err != nil {
		t.Fatal(err)
	}

	// Kill the process
	cmd.Process.Kill()
	cmd.Wait()
	time.Sleep(100 * time.Millisecond)

	status, _ := r.Status(handle)
	if status == runner.StatusRunning {
		t.Error("expected non-running status after process death")
	}
}
```

Note: add `"os/exec"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -run "TestProcessRunner_Reconnect|TestProcessRunner_Status_ReconnectedDies" -v`
Expected: compilation error — `Reconnect` not defined

- [ ] **Step 3: Implement reconnected field, Reconnect method, and updated Status**

In `internal/runner/process.go`:

Add `reconnected bool` and `pid int` fields to `processEntry`:

```go
type processEntry struct {
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	output      *syncBuffer
	logFile     *os.File
	done        chan struct{}
	err         error
	reconnected bool
	pid         int
}
```

Add `Reconnect` method:

```go
// Reconnect adopts a process service from a previous session by reading its
// PID file and checking liveness. Returns an error if the PID file doesn't
// exist or the process is dead (stale PID file is cleaned up).
func (r *ProcessRunner) Reconnect(serviceName string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pidDir == "" {
		return RunHandle{}, fmt.Errorf("pidDir not set")
	}

	info, err := ReadPIDFile(r.pidDir, serviceName)
	if err != nil {
		return RunHandle{}, fmt.Errorf("reading PID file for %s: %w", serviceName, err)
	}

	if !IsProcessAlive(info.PID) {
		RemovePIDFile(r.pidDir, serviceName)
		return RunHandle{}, fmt.Errorf("process %s (pid %d) is no longer running", serviceName, info.PID)
	}

	entry := &processEntry{
		reconnected: true,
		pid:         info.PID,
		done:        make(chan struct{}),
	}
	r.entries[serviceName] = entry
	return RunHandle{ID: serviceName, Type: "process"}, nil
}
```

Update `Status` to handle reconnected entries. Note: copy `reconnected` and `pid` to locals while holding the lock to avoid data races (these fields are set once in `Reconnect` but read here without the lock):

```go
func (r *ProcessRunner) Status(handle RunHandle) (ServiceStatus, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	if !ok {
		r.mu.Unlock()
		return StatusStopped, nil
	}
	reconnected := entry.reconnected
	pid := entry.pid
	r.mu.Unlock()

	if reconnected {
		if IsProcessAlive(pid) {
			return StatusRunning, nil
		}
		// Process died — clean up
		if r.pidDir != "" {
			RemovePIDFile(r.pidDir, handle.ID)
		}
		r.mu.Lock()
		delete(r.entries, handle.ID)
		r.mu.Unlock()
		return StatusStopped, nil
	}

	select {
	case <-entry.done:
		if entry.err != nil {
			return StatusCrashed, nil
		}
		return StatusStopped, nil
	default:
		return StatusRunning, nil
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/ -run "TestProcessRunner_Reconnect|TestProcessRunner_Status_ReconnectedDies" -v`
Expected: all PASS

- [ ] **Step 5: Run all runner tests for regressions**

Run: `go test ./internal/runner/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/runner/process.go internal/runner/process_test.go
git commit -m "feat(runner): add ProcessRunner.Reconnect for cross-session adoption"
```

---

### Task 5: ProcessRunner — Stop for reconnected entries

**Files:**
- Modify: `internal/runner/process.go`
- Modify: `internal/runner/process_test.go`

- [ ] **Step 1: Write failing test for Stop on a reconnected entry**

Append to `internal/runner/process_test.go`:

```go
func TestProcessRunner_Stop_ReconnectedEntry(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	// Start a real process out-of-band
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	handle, err := r.Reconnect("worker")
	if err != nil {
		t.Fatal(err)
	}

	// Stop should kill the process
	if err := r.Stop(handle); err != nil {
		t.Fatal(err)
	}

	// Process should be dead
	time.Sleep(100 * time.Millisecond)
	if runner.IsProcessAlive(pid) {
		t.Error("process should be dead after Stop")
	}

	// PID file should be removed
	if _, readErr := runner.ReadPIDFile(pidDir, "worker"); readErr == nil {
		t.Error("PID file should have been removed after Stop")
	}

	// Status should be stopped
	status, _ := r.Status(handle)
	if status != runner.StatusStopped {
		t.Errorf("expected stopped, got %s", status)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run "TestProcessRunner_Stop_ReconnectedEntry" -v`
Expected: FAIL — current `Stop` calls `entry.cancel()` which panics for reconnected entries (cancel is nil)

- [ ] **Step 3: Update Stop to handle reconnected entries**

Replace `Stop` in `internal/runner/process.go`:

```go
func (r *ProcessRunner) Stop(handle RunHandle) error {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil
	}

	if entry.reconnected {
		proc, err := os.FindProcess(entry.pid)
		if err == nil {
			proc.Signal(syscall.SIGTERM)
			// Wait up to 5 seconds for graceful shutdown
			deadline := time.After(5 * time.Second)
			for {
				select {
				case <-deadline:
					proc.Kill()
					goto cleanup
				case <-time.After(100 * time.Millisecond):
					if !IsProcessAlive(entry.pid) {
						goto cleanup
					}
				}
			}
		}
	cleanup:
		r.mu.Lock()
		delete(r.entries, handle.ID)
		r.mu.Unlock()
		if r.pidDir != "" {
			RemovePIDFile(r.pidDir, handle.ID)
		}
		return nil
	}

	entry.cancel()
	<-entry.done
	if r.pidDir != "" {
		RemovePIDFile(r.pidDir, handle.ID)
	}
	return nil
}
```

Note: add `"syscall"` to the existing import block in `process.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runner/ -run "TestProcessRunner_Stop_ReconnectedEntry" -v`
Expected: PASS

- [ ] **Step 5: Run all runner tests for regressions**

Run: `go test ./internal/runner/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/runner/process.go internal/runner/process_test.go
git commit -m "feat(runner): handle Stop for reconnected process entries"
```

---

### Task 6: Orchestrator — extend Reconnect for process services

**Files:**
- Modify: `internal/orchestrator/orchestrator.go`
- Modify: `internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write failing test for process reconnection in orchestrator**

Append to `internal/orchestrator/orchestrator_test.go`:

```go
func TestOrchestrator_Reconnect_ProcessServices(t *testing.T) {
	// Start a real process
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	wsRoot := t.TempDir()
	pidDir := runner.PIDDirPath(wsRoot)
	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	containerMock := &mockRunner{}
	processRunner := runner.NewProcessRunner()
	orch := orchestrator.New(containerMock, processRunner, nil)

	ws := workspace.Workspace{
		Name: "test", Root: wsRoot,
		Services: map[string]workspace.Service{
			"worker": {Command: "sleep 60"},
		},
	}

	if err := orch.Reconnect(ws); err != nil {
		t.Fatal(err)
	}

	statuses, _ := orch.Status(ws)
	if statuses["worker"] != runner.StatusRunning {
		t.Errorf("expected running, got %s", statuses["worker"])
	}
}

func TestOrchestrator_Reconnect_SkipsDeadProcesses(t *testing.T) {
	// Start and kill a process to get a dead PID
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Kill()
	cmd.Wait()

	wsRoot := t.TempDir()
	pidDir := runner.PIDDirPath(wsRoot)
	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	containerMock := &mockRunner{}
	processRunner := runner.NewProcessRunner()
	orch := orchestrator.New(containerMock, processRunner, nil)

	ws := workspace.Workspace{
		Name: "test", Root: wsRoot,
		Services: map[string]workspace.Service{
			"worker": {Command: "sleep 60"},
		},
	}

	// Should not error — just skips dead processes
	if err := orch.Reconnect(ws); err != nil {
		t.Fatal(err)
	}

	statuses, _ := orch.Status(ws)
	if statuses["worker"] != runner.StatusStopped {
		t.Errorf("expected stopped, got %s", statuses["worker"])
	}
}
```

Note: add `"time"` and `"os/exec"` to the test file imports (`runner` is already imported).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/orchestrator/ -run "TestOrchestrator_Reconnect_ProcessServices|TestOrchestrator_Reconnect_SkipsDeadProcesses" -v`
Expected: FAIL — `Reconnect` doesn't handle process services yet

- [ ] **Step 3: Extend Orchestrator.Reconnect to handle process services**

In `internal/orchestrator/orchestrator.go`, modify `Reconnect` (currently lines 337-373). After the existing container reconnection loop, add process reconnection:

```go
func (o *Orchestrator) Reconnect(ws workspace.Workspace) error {
	// Container reconnection (existing code)
	rc, ok := o.containerRunner.(runner.Reconnectable)
	if ok {
		prefix := rc.Prefix() + "_"
		containers, err := runner.FindContainers(prefix)
		if err != nil {
			return fmt.Errorf("finding containers for %s: %w", ws.Name, err)
		}

		o.mu.Lock()
		if o.handles[ws.Name] == nil {
			o.handles[ws.Name] = make(map[string]runner.RunHandle)
		}
		o.mu.Unlock()

		for _, containerName := range containers {
			if !strings.HasPrefix(containerName, prefix) {
				continue
			}
			if runner.ContainerStatus(containerName) != runner.StatusRunning {
				continue
			}
			serviceName := strings.TrimPrefix(containerName, prefix)
			if _, exists := ws.Services[serviceName]; !exists {
				continue
			}
			handle := rc.Adopt(serviceName)
			o.mu.Lock()
			o.handles[ws.Name][serviceName] = handle
			o.mu.Unlock()
		}
	}

	// Process reconnection
	pr, ok := o.processRunner.(*runner.ProcessRunner)
	if ok {
		pidDir := runner.PIDDirPath(ws.Root)
		pr.SetPIDDir(pidDir)
		pidServices, err := runner.ListPIDFiles(pidDir)
		if err != nil {
			return fmt.Errorf("listing PID files for %s: %w", ws.Name, err)
		}
		o.mu.Lock()
		if o.handles[ws.Name] == nil {
			o.handles[ws.Name] = make(map[string]runner.RunHandle)
		}
		o.mu.Unlock()

		for _, serviceName := range pidServices {
			if _, exists := ws.Services[serviceName]; !exists {
				continue
			}
			handle, err := pr.Reconnect(serviceName)
			if err != nil {
				// Dead process — already cleaned up by Reconnect
				continue
			}
			o.mu.Lock()
			o.handles[ws.Name][serviceName] = handle
			o.mu.Unlock()
		}
	}

	return nil
}
```

Note: The type assertion `o.processRunner.(*runner.ProcessRunner)` is needed because `Reconnect` is not on the `Runner` interface. This follows the same pattern as `o.containerRunner.(runner.Reconnectable)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/orchestrator/ -run "TestOrchestrator_Reconnect_ProcessServices|TestOrchestrator_Reconnect_SkipsDeadProcesses" -v`
Expected: all PASS

- [ ] **Step 5: Run all orchestrator tests for regressions**

Run: `go test ./internal/orchestrator/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/orchestrator/orchestrator.go internal/orchestrator/orchestrator_test.go
git commit -m "feat(orchestrator): extend Reconnect to adopt process services via PID files"
```

---

### Task 7: CLI status command — show process service status

**Files:**
- Modify: `internal/cli/status.go`

**Notes:**
- This removes the `hasProcessOnly`/`"unknown"` status branch from `showAllWorkspaces` — that was the bug this feature fixes.
- CLI command tests are listed as not-yet-implemented in CLAUDE.md. The `processStatus` helper logic is already validated by pidfile and IsProcessAlive tests in Tasks 1-2. Full CLI status tests are deferred.

- [ ] **Step 1: Update showWorkspaceDetail to check PID files for process services**

In `internal/cli/status.go`, replace the process service handling in `showWorkspaceDetail` (lines 78-85). Currently:

```go
for name, svc := range ws.Services {
	svcType := "process"
	status := "unknown"
	if svc.IsContainer() {
		svcType = "container"
		containerName := fmt.Sprintf("%s%s", prefix, name)
		status = string(runner.ContainerStatus(containerName))
	}
```

Replace with:

```go
for name, svc := range ws.Services {
	svcType := "process"
	var status string
	if svc.IsContainer() {
		svcType = "container"
		containerName := fmt.Sprintf("%s%s", prefix, name)
		status = string(runner.ContainerStatus(containerName))
	} else {
		status = string(processStatus(ws.Root, name))
	}
```

Add a helper function to `status.go`:

```go
// processStatus checks a process service's status via its PID file.
func processStatus(wsRoot, serviceName string) runner.ServiceStatus {
	pidDir := runner.PIDDirPath(wsRoot)
	info, err := runner.ReadPIDFile(pidDir, serviceName)
	if err != nil {
		return runner.StatusStopped
	}
	if runner.IsProcessAlive(info.PID) {
		return runner.StatusRunning
	}
	// Stale PID file — clean up
	runner.RemovePIDFile(pidDir, serviceName)
	return runner.StatusStopped
}
```

- [ ] **Step 2: Update showAllWorkspaces to include process service status in aggregate count**

In `internal/cli/status.go`, replace `showAllWorkspaces` (lines 29-68). The key change is counting process services alongside containers. Replace the running-count logic:

```go
func showAllWorkspaces(cctx *cliContext) error {
	entries := cctx.registry.List()
	if len(entries) == 0 {
		fmt.Println("No workspaces registered.")
		return nil
	}
	fmt.Printf("%-20s %-12s %-12s\n", "WORKSPACE", "STATUS", "SERVICES")
	for _, e := range entries {
		m, err := workspace.ParseManifest(filepath.Join(e.Path, "rook.yaml"))
		if err != nil {
			fmt.Printf("%-20s %-12s %-12s\n", e.Name, "error", "-")
			continue
		}
		ws, err := m.ToWorkspace(e.Path)
		if err != nil {
			fmt.Printf("%-20s %-12s %-12s\n", e.Name, "error", "-")
			continue
		}
		total := len(m.Services)
		prefix := fmt.Sprintf("rook_%s_", e.Name)
		containers, _ := runner.FindContainers(prefix)
		running := 0
		for _, c := range containers {
			if runner.ContainerStatus(c) == runner.StatusRunning {
				running++
			}
		}
		// Count running process services
		for name, svc := range ws.Services {
			if !svc.IsProcess() {
				continue
			}
			if processStatus(ws.Root, name) == runner.StatusRunning {
				running++
			}
		}
		status := "stopped"
		if running > 0 && running >= total {
			status = "running"
		} else if running > 0 {
			status = "partial"
		}
		fmt.Printf("%-20s %-12s %d/%d\n", e.Name, status, running, total)
	}
	return nil
}
```

Note: This requires `m.ToWorkspace(e.Path)` to get `ws.Root` and `ws.Services` with the `IsProcess()` method. The `workspace.Manifest` already has `ToWorkspace`. The import of `workspace` is already present.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/cli/`
Expected: compiles without errors

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/status.go
git commit -m "feat(cli): show process service status via PID file liveness check"
```

---

### Task 8: Wire PID directory into `rook up`

**Files:**
- Modify: `internal/cli/up.go`

**Note:** This wiring is validated indirectly by Task 3's `TestProcessRunner_Start_CreatesPIDFile` (which calls `SetPIDDir` directly). The CLI up command integration is a one-line addition mirroring the existing `SetLogDir` pattern.

- [ ] **Step 1: Add SetPIDDir call alongside SetLogDir in up command**

In `internal/cli/up.go`, after line 51 (`cctx.process.SetLogDir(logDirPath(ws.Root))`), add:

```go
cctx.process.SetPIDDir(runner.PIDDirPath(ws.Root))
```

Add `"github.com/andybarilla/rook/internal/runner"` to imports if not already present.

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/rook/`
Expected: compiles without errors

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat(cli): wire PID directory into rook up for process tracking"
```

---

### Task 9: Final integration verification

- [ ] **Step 1: Run the full test suite**

Run: `go test ./... -v`
Expected: all PASS

- [ ] **Step 2: Build the CLI**

Run: `make build-cli`
Expected: builds successfully

- [ ] **Step 3: Verify with `go vet`**

Run: `go vet ./...`
Expected: no issues
