# File-Backed Process Logs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `rook logs` stream process service logs from persistent log files, not just container logs.

**Architecture:** ProcessRunner tees stdout/stderr to `.rook/.cache/logs/<service>.log` via `io.MultiWriter`. A thread-safe `syncBuffer` replaces the existing racy `bytes.Buffer`. The `rook logs` command loads the workspace manifest, discovers process log files, and tails them alongside container logs.

**Tech Stack:** Go stdlib (`io`, `os`, `sync`, `context`, `time`)

**Spec:** `docs/superpowers/specs/2026-03-18-process-logs-design.md`

---

### Task 1: syncBuffer — thread-safe output buffer

Replaces the existing `bytes.Buffer` + `entry.mu` pattern with a self-contained thread-safe buffer. This is a prerequisite for the MultiWriter tee since writes come from the exec pipe copier goroutine.

**Files:**
- Create: `internal/runner/syncbuffer.go`
- Create: `internal/runner/syncbuffer_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/runner/syncbuffer_test.go`:

```go
package runner

import (
	"sync"
	"testing"
)

func TestSyncBuffer_WriteAndRead(t *testing.T) {
	var buf syncBuffer
	buf.Write([]byte("hello "))
	buf.Write([]byte("world"))

	data := buf.Bytes()
	if string(data) != "hello world" {
		t.Errorf("got %q", string(data))
	}
}

func TestSyncBuffer_Len(t *testing.T) {
	var buf syncBuffer
	if buf.Len() != 0 {
		t.Error("expected 0 length")
	}
	buf.Write([]byte("abc"))
	if buf.Len() != 3 {
		t.Errorf("expected 3, got %d", buf.Len())
	}
}

func TestSyncBuffer_ConcurrentAccess(t *testing.T) {
	var buf syncBuffer
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			buf.Write([]byte("x"))
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = buf.Bytes()
			_ = buf.Len()
		}
	}()

	wg.Wait()
	if buf.Len() != 1000 {
		t.Errorf("expected 1000 bytes, got %d", buf.Len())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestSyncBuffer -v`
Expected: FAIL — `syncBuffer` type not defined

- [ ] **Step 3: Write implementation**

In `internal/runner/syncbuffer.go`:

```go
package runner

import (
	"bytes"
	"sync"
)

// syncBuffer is a thread-safe bytes.Buffer for capturing process output.
// Writes and reads are serialized via a mutex.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// Bytes returns a copy of the buffer contents.
func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := make([]byte, b.buf.Len())
	copy(data, b.buf.Bytes())
	return data
}

// Len returns the number of bytes in the buffer.
func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runner/ -run TestSyncBuffer -v -race`
Expected: PASS with no race conditions

- [ ] **Step 5: Commit**

```bash
git add internal/runner/syncbuffer.go internal/runner/syncbuffer_test.go
git commit -m "feat: add syncBuffer for thread-safe process output capture"
```

---

### Task 2: Migrate ProcessRunner to syncBuffer

Replace the `bytes.Buffer` + `entry.mu` in ProcessRunner with `syncBuffer`. Remove the per-entry mutex. All existing tests must still pass.

**Files:**
- Modify: `internal/runner/process.go`

- [ ] **Step 1: Run existing tests to confirm they pass before changes**

Run: `go test ./internal/runner/ -run TestProcessRunner -v`
Expected: PASS

- [ ] **Step 2: Update processEntry and ProcessRunner**

In `internal/runner/process.go`, replace the `processEntry` struct and update `Start()`:

Change the imports — remove `"bytes"` (no longer needed directly):

```go
// Keep: context, fmt, io, os, os/exec, sync, time
// Remove: bytes
```

Replace `processEntry`:

```go
type processEntry struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	output *syncBuffer
	done   chan struct{}
	err    error
}
```

In `Start()`, replace:
```go
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
```
with:
```go
	var output syncBuffer
	cmd.Stdout = &output
	cmd.Stderr = &output
```

And update the entry creation:
```go
	entry := &processEntry{
		cmd:    cmd,
		cancel: cancel,
		output: &output,
		done:   make(chan struct{}),
	}
```

- [ ] **Step 3: Update Logs() — remove manual mutex**

Replace the `Logs` method:

```go
func (r *ProcessRunner) Logs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no logs for %s", handle.ID)
	}
	data := entry.output.Bytes()
	return io.NopCloser(bytes.NewReader(data)), nil
}
```

Note: re-add `"bytes"` to imports since `Logs()` still uses `bytes.NewReader`.

- [ ] **Step 4: Update StreamLogs() — remove manual mutex**

Replace the `StreamLogs` method:

```go
func (r *ProcessRunner) StreamLogs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no process for %s", handle.ID)
	}
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		lastLen := 0
		for {
			// Single Bytes() call avoids TOCTOU between Len() and Bytes()
			data := entry.output.Bytes()
			if len(data) > lastLen {
				pw.Write(data[lastLen:])
				lastLen = len(data)
			}
			select {
			case <-entry.done:
				data = entry.output.Bytes()
				if len(data) > lastLen {
					pw.Write(data[lastLen:])
				}
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	return pr, nil
}
```

- [ ] **Step 5: Run tests to verify migration is clean**

Run: `go test ./internal/runner/ -run TestProcessRunner -v -race`
Expected: PASS with no race conditions

- [ ] **Step 6: Commit**

```bash
git add internal/runner/process.go
git commit -m "refactor: migrate ProcessRunner to syncBuffer"
```

---

### Task 3: File-backed logging in ProcessRunner

Add `SetLogDir()` and file tee to `ProcessRunner.Start()`.

**Files:**
- Modify: `internal/runner/process.go`
- Modify: `internal/runner/process_test.go`

- [ ] **Step 1: Write the failing test for file logging**

Add to `internal/runner/process_test.go`:

```go
func TestProcessRunner_FileLogging(t *testing.T) {
	logDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetLogDir(logDir)

	svc := workspace.Service{Command: "echo hello-from-log"}
	handle, err := r.Start(context.Background(), "log-svc", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	r.Stop(handle)

	logPath := filepath.Join(logDir, "log-svc.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "--- rook up") {
		t.Error("expected session separator in log file")
	}
	if !strings.Contains(content, "hello-from-log") {
		t.Errorf("expected process output in log file, got: %s", content)
	}
}
```

Add imports `"os"`, `"path/filepath"`, `"strings"` to the test file.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestProcessRunner_FileLogging -v`
Expected: FAIL — `SetLogDir` method not defined

- [ ] **Step 3: Write the failing test for append across sessions**

Add to `internal/runner/process_test.go`:

```go
func TestProcessRunner_FileLogging_AppendsSessions(t *testing.T) {
	logDir := t.TempDir()

	// First session
	r1 := runner.NewProcessRunner()
	r1.SetLogDir(logDir)
	svc := workspace.Service{Command: "echo session-one"}
	h1, _ := r1.Start(context.Background(), "app", svc, nil, t.TempDir())
	time.Sleep(200 * time.Millisecond)
	r1.Stop(h1)

	// Second session
	r2 := runner.NewProcessRunner()
	r2.SetLogDir(logDir)
	svc2 := workspace.Service{Command: "echo session-two"}
	h2, _ := r2.Start(context.Background(), "app", svc2, nil, t.TempDir())
	time.Sleep(200 * time.Millisecond)
	r2.Stop(h2)

	data, _ := os.ReadFile(filepath.Join(logDir, "app.log"))
	content := string(data)
	if !strings.Contains(content, "session-one") {
		t.Error("expected first session output")
	}
	if !strings.Contains(content, "session-two") {
		t.Error("expected second session output")
	}
	if strings.Count(content, "--- rook up") != 2 {
		t.Errorf("expected 2 session separators, got %d", strings.Count(content, "--- rook up"))
	}
}
```

- [ ] **Step 4: Implement SetLogDir and file tee**

In `internal/runner/process.go`, add the `logDir` field and `SetLogDir`:

```go
type ProcessRunner struct {
	mu      sync.Mutex
	entries map[string]*processEntry
	logDir  string
}

// SetLogDir sets the directory for persistent process log files.
// Must be called before Start(). When set, process output is teed to
// <logDir>/<service>.log in addition to the in-memory buffer.
func (r *ProcessRunner) SetLogDir(dir string) {
	r.logDir = dir
}
```

Add `logFile` to `processEntry`:

```go
type processEntry struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	output  *syncBuffer
	logFile *os.File
	done    chan struct{}
	err     error
}
```

Add `"path/filepath"` and `"time"` to imports (time is already there).

In `Start()`, after creating the `syncBuffer` and before setting `cmd.Stdout`, add file tee logic:

```go
	var output syncBuffer
	var logFile *os.File

	if r.logDir != "" {
		if err := os.MkdirAll(r.logDir, 0755); err != nil {
			cancel()
			return RunHandle{}, fmt.Errorf("creating log dir: %w", err)
		}
		logPath := filepath.Join(r.logDir, name+".log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			cancel()
			return RunHandle{}, fmt.Errorf("opening log file for %s: %w", name, err)
		}
		fmt.Fprintf(f, "--- rook up %s ---\n", time.Now().Format(time.RFC3339))
		logFile = f
		cmd.Stdout = io.MultiWriter(&output, f)
		cmd.Stderr = io.MultiWriter(&output, f)
	} else {
		cmd.Stdout = &output
		cmd.Stderr = &output
	}
```

Update the entry creation to include `logFile`:

```go
	entry := &processEntry{
		cmd:     cmd,
		cancel:  cancel,
		output:  &output,
		logFile: logFile,
		done:    make(chan struct{}),
	}
```

Handle `cmd.Start()` failure — close the log file:

```go
	if err := cmd.Start(); err != nil {
		cancel()
		if logFile != nil {
			logFile.Close()
		}
		return RunHandle{}, fmt.Errorf("starting %s: %w", name, err)
	}
```

Close the log file in the done goroutine:

```go
	go func() {
		entry.err = cmd.Wait()
		if entry.logFile != nil {
			entry.logFile.Close()
		}
		close(entry.done)
	}()
```

In `Stop()`, close the log file after the process exits:

```go
func (r *ProcessRunner) Stop(handle RunHandle) error {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	entry.cancel()
	<-entry.done
	return nil
}
```

Note: `Stop()` doesn't need to close the file because `<-entry.done` waits for the done goroutine which already closes it.

- [ ] **Step 5: Run all tests**

Run: `go test ./internal/runner/ -v -race`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/runner/process.go internal/runner/process_test.go
git commit -m "feat: add file-backed logging to ProcessRunner"
```

---

### Task 4: Wire SetLogDir in up.go

Add the `logDirPath` helper and call `SetLogDir` before starting services.

**Files:**
- Modify: `internal/cli/up.go`

- [ ] **Step 1: Add logDirPath helper**

In `internal/cli/up.go`, add after `resolvedDirPath`:

```go
// logDirPath returns the path to the process log files directory.
func logDirPath(wsRoot string) string {
	return filepath.Join(wsRoot, ".rook", ".cache", "logs")
}
```

- [ ] **Step 2: Add SetLogDir call**

In `up.go`, inside `RunE`, after `ws, err := cctx.loadWorkspace(wsName)` and the error check (after line 47), add:

```go
			// Set log directory for process services
			cctx.process.SetLogDir(logDirPath(ws.Root))
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat: wire process log directory in rook up"
```

---

### Task 5: tailFile — file tailing reader

A standalone function that tails a file and returns an `io.ReadCloser` compatible with `logMux.addStream()`.

**Files:**
- Create: `internal/cli/tailfile.go`
- Create: `internal/cli/tailfile_test.go`

- [ ] **Step 1: Write the failing test for basic tailing**

In `internal/cli/tailfile_test.go`:

```go
package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTailFile_ReadsExistingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("existing line\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := tailFile(path, ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Read with a timeout
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _ := reader.Read(buf)
		done <- string(buf[:n])
	}()

	select {
	case content := <-done:
		if !strings.Contains(content, "existing line") {
			t.Errorf("expected existing content, got %q", content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout reading existing content")
	}
}

func TestTailFile_FollowsNewWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("initial\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, err := tailFile(path, ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Read initial content
	buf := make([]byte, 1024)
	reader.Read(buf)

	// Append new content
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("appended line\n")
	f.Close()

	// Read new content with timeout
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 1024)
		n, _ := reader.Read(buf)
		done <- string(buf[:n])
	}()

	select {
	case content := <-done:
		if !strings.Contains(content, "appended line") {
			t.Errorf("expected appended content, got %q", content)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout reading appended content")
	}
}

func TestTailFile_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	os.WriteFile(path, []byte("data\n"), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	reader, err := tailFile(path, ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Read existing content first
	buf := make([]byte, 1024)
	reader.Read(buf)

	// Cancel and verify reader returns EOF
	cancel()
	time.Sleep(300 * time.Millisecond) // wait for poll cycle

	_, err = io.ReadAll(reader)
	if err != nil && err != io.EOF {
		t.Errorf("expected EOF or nil after cancel, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestTailFile -v`
Expected: FAIL — `tailFile` not defined

- [ ] **Step 3: Implement tailFile**

In `internal/cli/tailfile.go`:

```go
package cli

import (
	"context"
	"io"
	"os"
	"time"
)

// tailFile opens a file and returns a reader that streams its content,
// including existing data and any new data appended after opening.
// The reader closes when the context is cancelled.
func tailFile(path string, ctx context.Context) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		defer f.Close()
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				if _, writeErr := pw.Write(buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil && err != io.EOF {
				return
			}
			// At EOF — poll for new data
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}()
	return pr, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cli/ -run TestTailFile -v -race`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/tailfile.go internal/cli/tailfile_test.go
git commit -m "feat: add tailFile for tailing process log files"
```

---

### Task 6: Update rook logs to handle process services

Rewrite `logs.go` to load the workspace, handle process log files alongside containers, and support single-service mode for processes.

**Files:**
- Modify: `internal/cli/logs.go`

- [ ] **Step 1: Rewrite logs.go**

Replace the contents of `internal/cli/logs.go`:

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs [workspace] [service]",
		Short: "Tail logs (all or specific service)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}

			// Single-service mode
			if len(args) > 1 {
				svcName := args[1]
				if svc, ok := ws.Services[svcName]; ok && svc.IsProcess() {
					logPath := filepath.Join(logDirPath(ws.Root), svcName+".log")
					return streamSingleProcessLog(logPath, ctx)
				}
				containerName := fmt.Sprintf("rook_%s_%s", wsName, svcName)
				return streamSingleContainer(containerName)
			}

			// Multi-service mode
			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return err
			}

			// Find process services with log files
			type processLog struct {
				name string
				path string
			}
			var processLogs []processLog
			logDir := logDirPath(ws.Root)
			for name, svc := range ws.Services {
				if !svc.IsProcess() {
					continue
				}
				logPath := filepath.Join(logDir, name+".log")
				if _, err := os.Stat(logPath); err == nil {
					processLogs = append(processLogs, processLog{name: name, path: logPath})
				}
			}

			if len(containers) == 0 && len(processLogs) == 0 {
				fmt.Printf("No running services found for %s.\n", wsName)
				return nil
			}

			mux := newLogMux(os.Stdout)
			var wg sync.WaitGroup
			colorIdx := 0

			// Stream container logs
			for _, containerName := range containers {
				svcName := strings.TrimPrefix(containerName, prefix)
				idx := colorIdx
				colorIdx++
				cName := containerName

				logCmd := exec.Command(runner.ContainerRuntime, "logs", "-f", "--follow", cName)
				stdout, err := logCmd.StdoutPipe()
				if err != nil {
					continue
				}
				logCmd.Stderr = logCmd.Stdout
				if err := logCmd.Start(); err != nil {
					continue
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					mux.addStream(svcName, stdout, idx)
					logCmd.Wait()
				}()
			}

			// Stream process log files
			for _, pl := range processLogs {
				idx := colorIdx
				colorIdx++
				name := pl.name

				reader, err := tailFile(pl.path, ctx)
				if err != nil {
					continue
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					mux.addStream(name, reader, idx)
				}()
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			cancel()
			fmt.Println("\nStopped tailing logs.")
			return nil
		},
	}
}

func streamSingleContainer(containerName string) error {
	cmd := exec.Command(runner.ContainerRuntime, "logs", "-f", "--follow", containerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("tailing logs for %s: %w", containerName, err)
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cmd.Process.Kill()
	return nil
}

func streamSingleProcessLog(logPath string, ctx context.Context) error {
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("no log file found at %s", logPath)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	reader, err := tailFile(logPath, ctx)
	if err != nil {
		return fmt.Errorf("tailing log file: %w", err)
	}

	// Stream to stdout
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cancel()
	<-done
	return nil
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/cli/...`
Expected: no errors

- [ ] **Step 3: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/logs.go
git commit -m "feat: rook logs streams process service log files"
```

---

### Task 7: Full integration test and final verification

Run the full test suite, verify no regressions.

**Files:**
- No new files

- [ ] **Step 1: Run all tests with race detection**

Run: `go test ./internal/... ./cmd/rook/... ./test/... -v -race -timeout 60s`
Expected: PASS

- [ ] **Step 2: Run go vet**

Run: `go vet ./internal/... ./cmd/rook/...`
Expected: no issues

- [ ] **Step 3: Manual smoke test (if possible)**

If a workspace is available:
```bash
# Start with a process service
bin/rook up <workspace>
# In another terminal
bin/rook logs <workspace>
# Verify process service logs appear alongside container logs
```

- [ ] **Step 4: Final commit if any fixups needed**

Only if tests revealed issues that needed fixing.
