# GUI Log Streaming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire up log streaming from runners through the orchestrator into the GUI's existing LogBuffer/event infrastructure so logs appear in the frontend.

**Architecture:** Orchestrator gets a `StreamServiceLogs` method that type-asserts runners and returns an `io.ReadCloser`. WorkspaceAPI spawns goroutines per service that read lines and call `BufferLog()`. A `cmdReadCloser` wrapper in the runner package handles Docker's `*exec.Cmd` lifecycle.

**Tech Stack:** Go, Wails v2 events

**Spec:** `docs/superpowers/specs/2026-03-24-gui-log-streaming-design.md`

---

### Task 1: Add `cmdReadCloser` wrapper to runner package

**Files:**
- Create: `internal/runner/cmdreadcloser.go`
- Create: `internal/runner/cmdreadcloser_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/runner/cmdreadcloser_test.go
package runner

import (
	"io"
	"os/exec"
	"strings"
	"testing"
)

func TestCmdReadCloser_Read(t *testing.T) {
	inner := io.NopCloser(strings.NewReader("hello\nworld\n"))
	crc := &cmdReadCloser{ReadCloser: inner}
	data, err := io.ReadAll(crc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\nworld\n" {
		t.Fatalf("got %q", string(data))
	}
}

func TestCmdReadCloser_CloseWaitsCmd(t *testing.T) {
	cmd := exec.Command("echo", "done")
	cmd.Start()
	inner := io.NopCloser(strings.NewReader(""))
	crc := &cmdReadCloser{ReadCloser: inner, cmd: cmd}
	if err := crc.Close(); err != nil {
		t.Fatal(err)
	}
	// cmd.Wait already called by Close — calling again should error
	if err := cmd.Wait(); err == nil {
		t.Fatal("expected error from double Wait")
	}
}

func TestCmdReadCloser_CloseNilCmd(t *testing.T) {
	inner := io.NopCloser(strings.NewReader(""))
	crc := &cmdReadCloser{ReadCloser: inner}
	if err := crc.Close(); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestCmdReadCloser -v`
Expected: FAIL — `cmdReadCloser` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/runner/cmdreadcloser.go
package runner

import (
	"io"
	"os/exec"
)

// cmdReadCloser wraps an io.ReadCloser with an optional exec.Cmd.
// Close() closes the reader and waits for the command to exit.
type cmdReadCloser struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *cmdReadCloser) Close() error {
	err := c.ReadCloser.Close()
	if c.cmd != nil {
		c.cmd.Wait()
	}
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/ -run TestCmdReadCloser -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/cmdreadcloser.go internal/runner/cmdreadcloser_test.go
git commit -m "feat: add cmdReadCloser wrapper for Docker log streaming cleanup"
```

---

### Task 2: Add `StreamServiceLogs` to Orchestrator

**Files:**
- Modify: `internal/orchestrator/orchestrator.go` (add method at end)
- Modify: `internal/orchestrator/orchestrator_test.go` (add tests)

- [ ] **Step 1: Write the failing test**

The existing `mockRunner` in orchestrator_test.go doesn't support streaming. Since `StreamServiceLogs` type-asserts to concrete runner types (`*runner.DockerRunner` / `*runner.ProcessRunner`), we can't easily test it with mocks. Instead, test the error paths (no workspace, no handle, unknown type).

```go
// Add to internal/orchestrator/orchestrator_test.go

func TestOrchestrator_StreamServiceLogs_NoWorkspace(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	_, err := orch.StreamServiceLogs("nonexistent", "svc")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
}

func TestOrchestrator_StreamServiceLogs_NoHandle(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"svc": {Command: "echo hi"},
		},
	}
	// Start a service so the workspace exists in handles
	if err := orch.Up(context.Background(), ws, ""); err != nil {
		t.Fatal(err)
	}
	// Ask for a service that wasn't started
	_, err := orch.StreamServiceLogs("test", "other")
	if err == nil {
		t.Fatal("expected error for missing handle")
	}
}

func TestOrchestrator_StreamServiceLogs_UnsupportedRunner(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"svc": {Command: "echo hi"},
		},
	}
	if err := orch.Up(context.Background(), ws, ""); err != nil {
		t.Fatal(err)
	}
	// mockRunner doesn't implement StreamLogs, so this should error
	_, err := orch.StreamServiceLogs("test", "svc")
	if err == nil {
		t.Fatal("expected error for unsupported runner type")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/orchestrator/ -run TestOrchestrator_StreamServiceLogs -v`
Expected: FAIL — `StreamServiceLogs` not defined

- [ ] **Step 3: Write implementation**

Add to `internal/orchestrator/orchestrator.go`:

```go
// StreamServiceLogs returns a streaming log reader for a running service.
// Callers must Close() the returned reader when done.
func (o *Orchestrator) StreamServiceLogs(wsName, serviceName string) (io.ReadCloser, error) {
	o.mu.Lock()
	handles, ok := o.handles[wsName]
	if !ok {
		o.mu.Unlock()
		return nil, fmt.Errorf("no workspace %q in handles", wsName)
	}
	handle, ok := handles[serviceName]
	o.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no handle for service %q in workspace %q", serviceName, wsName)
	}

	switch handle.Type {
	case "docker":
		dr, ok := o.containerRunner.(*runner.DockerRunner)
		if !ok {
			return nil, fmt.Errorf("container runner does not support log streaming")
		}
		reader, cmd, err := dr.StreamLogs(handle)
		if err != nil {
			return nil, err
		}
		return runner.NewCmdReadCloser(reader, cmd), nil
	case "process":
		pr, ok := o.processRunner.(*runner.ProcessRunner)
		if !ok {
			return nil, fmt.Errorf("process runner does not support log streaming")
		}
		return pr.StreamLogs(handle)
	default:
		return nil, fmt.Errorf("unknown runner type %q for service %q", handle.Type, serviceName)
	}
}
```

This requires exporting the `cmdReadCloser` constructor. Update `internal/runner/cmdreadcloser.go`: rename the type to keep it unexported but add a constructor:

```go
// NewCmdReadCloser wraps an io.ReadCloser with an exec.Cmd that is waited on Close.
func NewCmdReadCloser(r io.ReadCloser, cmd *exec.Cmd) io.ReadCloser {
	return &cmdReadCloser{ReadCloser: r, cmd: cmd}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/orchestrator/ -run TestOrchestrator_StreamServiceLogs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/orchestrator.go internal/orchestrator/orchestrator_test.go internal/runner/cmdreadcloser.go
git commit -m "feat: add StreamServiceLogs to orchestrator"
```

---

### Task 3: Add log streaming helpers to WorkspaceAPI

**Files:**
- Modify: `internal/api/workspace.go` (add fields, helpers)
- Modify: `internal/api/workspace_test.go` (add tests)

- [ ] **Step 1: Write the failing test**

We need a mock orchestrator approach. Since `WorkspaceAPI` takes `*orchestrator.Orchestrator` (concrete type), we can't easily mock `StreamServiceLogs`. Instead, test `startLogStream` indirectly by testing that `BufferLog` is called. But first, let's test the exported `StartLogStream` and `StopLogStream` methods more directly.

Actually, the simplest testable approach: extract the goroutine logic into a helper that takes an `io.ReadCloser` and test that it populates the buffer.

```go
// Add to internal/api/workspace_test.go

func TestStartLogStream_BuffersLines(t *testing.T) {
	a := newTestAPI()
	// Manually feed a reader into the streaming helper
	r := io.NopCloser(strings.NewReader("line one\nline two\nline three\n"))
	a.StreamFromReader("ws", "svc", r)

	// Give goroutine time to process
	time.Sleep(100 * time.Millisecond)

	logs, err := a.GetLogs("ws", "svc", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(logs))
	}
	if logs[0].Line != "line one" {
		t.Fatalf("expected 'line one', got %q", logs[0].Line)
	}
	if logs[2].Line != "line three" {
		t.Fatalf("expected 'line three', got %q", logs[2].Line)
	}
}

func TestStopLogStream_CancelsReader(t *testing.T) {
	a := newTestAPI()
	// Create a reader that blocks forever
	pr, pw := io.Pipe()
	a.StreamFromReader("ws", "svc", pr)

	// Write one line
	pw.Write([]byte("hello\n"))
	time.Sleep(50 * time.Millisecond)

	// Stop should cancel the goroutine
	a.StopLogStream("ws", "svc")
	time.Sleep(50 * time.Millisecond)

	// Writing more should not panic (goroutine exited)
	pw.Close()

	logs, _ := a.GetLogs("ws", "svc", 10)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logs))
	}
}
```

Add `"io"`, `"strings"`, and `"time"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/api/ -run TestStartLogStream -v && go test ./internal/api/ -run TestStopLogStream -v`
Expected: FAIL — `StreamFromReader` / `StopLogStream` not defined

- [ ] **Step 3: Write implementation**

Add new fields to `WorkspaceAPI` struct in `internal/api/workspace.go`:

```go
type WorkspaceAPI struct {
	registry       registry.Registry
	portAlloc      ports.PortAllocator
	orch           *orchestrator.Orchestrator
	discoverers    []discovery.Discoverer
	logBuffer      *LogBuffer
	emitter        EventEmitter
	activeProfiles map[string]string
	settingsPath   string
	portsPath      string
	logMu          sync.Mutex
	logCancels     map[string]context.CancelFunc
}
```

Initialize `logCancels` in all constructors:

```go
logCancels: make(map[string]context.CancelFunc),
```

Add `"bufio"` and `"sync"` to imports.

Add helper methods:

```go
func logKey(ws, svc string) string {
	return ws + "/" + svc
}

// StreamFromReader starts a goroutine that reads lines from r and buffers them.
// Exported for testing. In production, use startLogStream instead.
func (w *WorkspaceAPI) StreamFromReader(ws, svc string, r io.ReadCloser) {
	ctx, cancel := context.WithCancel(context.Background())

	w.logMu.Lock()
	// Cancel any existing stream for this service
	if existing, ok := w.logCancels[logKey(ws, svc)]; ok {
		existing()
	}
	w.logCancels[logKey(ws, svc)] = cancel
	w.logMu.Unlock()

	go func() {
		defer r.Close()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				w.BufferLog(ws, svc, scanner.Text())
			}
		}
		// Clean up cancel func on natural EOF
		w.logMu.Lock()
		delete(w.logCancels, logKey(ws, svc))
		w.logMu.Unlock()
	}()
}

// startLogStream starts streaming logs for a service via the orchestrator.
func (w *WorkspaceAPI) startLogStream(ws, svc string) {
	reader, err := w.orch.StreamServiceLogs(ws, svc)
	if err != nil {
		return // silently skip — service may not support streaming
	}
	w.StreamFromReader(ws, svc, reader)
}

// StopLogStream cancels an active log stream for a service.
func (w *WorkspaceAPI) StopLogStream(ws, svc string) {
	w.logMu.Lock()
	if cancel, ok := w.logCancels[logKey(ws, svc)]; ok {
		cancel()
		delete(w.logCancels, logKey(ws, svc))
	}
	w.logMu.Unlock()
}

// stopAllLogStreams cancels all active log streams for a workspace.
func (w *WorkspaceAPI) stopAllLogStreams(ws string) {
	w.logMu.Lock()
	prefix := ws + "/"
	for key, cancel := range w.logCancels {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			cancel()
			delete(w.logCancels, key)
		}
	}
	w.logMu.Unlock()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/api/ -run "TestStartLogStream|TestStopLogStream" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/workspace.go internal/api/workspace_test.go
git commit -m "feat: add log streaming helpers to WorkspaceAPI"
```

---

### Task 4: Wire log streaming into service lifecycle methods

**Files:**
- Modify: `internal/api/workspace.go:283-322` (StartService, StopService, RestartService)
- Modify: `internal/api/workspace.go:245-268` (StartWorkspace)
- Modify: `internal/api/workspace.go:270-281` (StopWorkspace)

- [ ] **Step 1: Wire StartService**

After the successful `orch.StartService` call and status event emit, add:

```go
func (w *WorkspaceAPI) StartService(ws, svc string) error {
	wks, err := w.loadWorkspace(ws)
	if err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusStarting})
	if err := w.orch.StartService(context.Background(), *wks, svc); err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusRunning})
	w.startLogStream(ws, svc)
	return nil
}
```

- [ ] **Step 2: Wire StopService**

Stop the log stream before stopping the service:

```go
func (w *WorkspaceAPI) StopService(ws, svc string) error {
	wks, err := w.loadWorkspace(ws)
	if err != nil {
		return err
	}
	w.StopLogStream(ws, svc)
	if err := w.orch.StopService(context.Background(), *wks, svc); err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusStopped})
	return nil
}
```

- [ ] **Step 3: Wire RestartService**

Stop old stream, restart, start new stream:

```go
func (w *WorkspaceAPI) RestartService(ws, svc string) error {
	wks, err := w.loadWorkspace(ws)
	if err != nil {
		return err
	}
	w.StopLogStream(ws, svc)
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusStarting})
	if err := w.orch.RestartService(context.Background(), *wks, svc); err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusRunning})
	w.startLogStream(ws, svc)
	return nil
}
```

- [ ] **Step 4: Wire StartWorkspace**

After `orch.Up` succeeds, get status and stream for each running service:

```go
func (w *WorkspaceAPI) StartWorkspace(name, profile string, forceBuild bool) error {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return err
	}
	if forceBuild {
		for svcName, svc := range ws.Services {
			if svc.Build != "" {
				svc.ForceBuild = true
				ws.Services[svcName] = svc
			}
		}
	}
	if err := w.orch.Up(context.Background(), *ws, profile); err != nil {
		return err
	}
	w.activeProfiles[name] = profile

	// Start log streaming for all running services
	statuses, _ := w.orch.Status(*ws)
	for svcName, status := range statuses {
		if status == runner.StatusRunning {
			w.startLogStream(name, svcName)
		}
	}
	return nil
}
```

- [ ] **Step 5: Wire StopWorkspace**

Stop all log streams before stopping services:

```go
func (w *WorkspaceAPI) StopWorkspace(name string) error {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return err
	}
	w.stopAllLogStreams(name)
	if err := w.orch.Down(context.Background(), *ws); err != nil {
		return err
	}
	delete(w.activeProfiles, name)
	return nil
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/api/ -v && go test ./internal/orchestrator/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/api/workspace.go
git commit -m "feat: wire log streaming into service lifecycle methods"
```

---

### Task 5: Add `ReconnectWorkspace` method and wire into GUI startup

**Files:**
- Modify: `internal/api/workspace.go` (add ReconnectWorkspace)
- Modify: `cmd/rook-gui/main.go:63-65` (call ReconnectWorkspace in OnStartup)

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/api/workspace_test.go

func TestReconnectWorkspace_ErrorsWithNoRegistry(t *testing.T) {
	a := newTestAPI()
	err := a.ReconnectWorkspace("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered workspace")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -run TestReconnectWorkspace -v`
Expected: FAIL — `ReconnectWorkspace` not defined

- [ ] **Step 3: Write ReconnectWorkspace**

Add to `internal/api/workspace.go`:

```go
// ReconnectWorkspace discovers already-running services and starts log streaming for them.
func (w *WorkspaceAPI) ReconnectWorkspace(name string) error {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return err
	}
	if err := w.orch.Reconnect(*ws); err != nil {
		return err
	}
	statuses, _ := w.orch.Status(*ws)
	for svcName, status := range statuses {
		if status == runner.StatusRunning {
			w.startLogStream(name, svcName)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run TestReconnectWorkspace -v`
Expected: PASS

- [ ] **Step 5: Wire into GUI startup**

Modify `cmd/rook-gui/main.go` OnStartup to reconnect all workspaces:

```go
OnStartup: func(ctx context.Context) {
	wsAPI.SetEmitter(&wailsEmitter{ctx: ctx})
	// Reconnect already-running services and start log streaming
	for _, entry := range reg.List() {
		wsAPI.ReconnectWorkspace(entry.Name)
	}
},
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/api/workspace.go internal/api/workspace_test.go cmd/rook-gui/main.go
git commit -m "feat: add ReconnectWorkspace and wire into GUI startup"
```

---

### Task 6: Build and manual test

**Files:** None (verification only)

- [ ] **Step 1: Build CLI and GUI**

Run: `make build-cli && make build-gui`
Expected: Both build without errors

- [ ] **Step 2: Manual test with a workspace**

1. Start a service via CLI: `rook up <workspace>`
2. Open the GUI
3. Navigate to the workspace detail page
4. Verify logs appear in the LogViewer for running services
5. Stop a service via GUI — verify log stream stops cleanly
6. Start a service via GUI — verify new logs appear
7. Start a service that crashes — verify crash logs appear

- [ ] **Step 3: Commit any fixes**

If manual testing reveals issues, fix and commit.
