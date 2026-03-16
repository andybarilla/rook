# CLI Command Wiring Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the 5 stub CLI commands (`up`, `down`, `restart`, `status`, `logs`) to the orchestrator, runners, and health checker so Rook can actually start, stop, and monitor services.

**Architecture:** A shared CLI context creates the orchestrator and runners. `rook up` runs in the foreground streaming logs (like `docker compose up`), with Ctrl+C for graceful shutdown. Other commands discover Docker containers by naming convention for standalone use. Health checks are integrated into the orchestrator's startup sequence.

**Tech Stack:** Go 1.22+, Cobra CLI

**Spec:** `docs/specs/2026-03-15-cli-wiring-design.md`

---

## File Structure

```
internal/
  cli/
    context.go              # Shared CLI context (registry, ports, orchestrator, runners)
    context_test.go         # Tests for context + loadWorkspace
    up.go                   # rook up (foreground + detach)
    down.go                 # rook down (Docker container discovery)
    restart.go              # rook restart
    status.go               # rook status (Docker container inspection)
    logs.go                 # rook logs (Docker log streaming)
    logmux.go               # Log multiplexer (color-coded interleaved output)
    logmux_test.go          # Log mux tests
  runner/
    docker.go               # Add FindContainers + StreamLogs
    process.go              # Add StreamLogs method
  orchestrator/
    orchestrator.go         # Add health check waiting to Up
    orchestrator_test.go    # Health check integration tests
```

---

## Chunk 1: Runner Extensions + CLI Context

### Task 1: Add FindContainers and StreamLogs to DockerRunner

**Files:**
- Modify: `internal/runner/docker.go`

- [ ] **Step 1: Add FindContainers function**

```go
// FindContainers returns container names matching the given prefix.
// Uses `docker ps -a --filter name=<prefix> --format {{.Names}}`.
func FindContainers(prefix string) ([]string, error) {
	cmd := exec.Command("docker", "ps", "-a",
		"--filter", fmt.Sprintf("name=%s", prefix),
		"--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
```

- [ ] **Step 2: Add StreamLogs to DockerRunner**

```go
// StreamLogs returns a streaming reader for a container's logs.
// The caller must close the returned ReadCloser and kill the process.
func (r *DockerRunner) StreamLogs(handle RunHandle) (io.ReadCloser, *exec.Cmd, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()
	if !ok {
		// Try by naming convention (for standalone discovery)
		containerName = r.containerName(handle.ID)
	}

	cmd := exec.Command("docker", "logs", "-f", "--follow", containerName)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("streaming logs for %s: %w", containerName, err)
	}
	return stdout, cmd, nil
}
```

- [ ] **Step 3: Add ContainerStatus for standalone inspection**

```go
// ContainerStatus checks the status of a container by name (not by handle).
func ContainerStatus(containerName string) ServiceStatus {
	output, err := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err != nil {
		return StatusStopped
	}
	switch strings.TrimSpace(string(output)) {
	case "running":
		return StatusRunning
	case "exited":
		out, _ := exec.Command("docker", "inspect", "-f", "{{.State.ExitCode}}", containerName).Output()
		if strings.TrimSpace(string(out)) != "0" {
			return StatusCrashed
		}
		return StatusStopped
	default:
		return StatusStopped
	}
}
```

- [ ] **Step 4: Verify existing tests still pass**

Run: `go test ./internal/runner/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/docker.go
git commit -m "feat: add FindContainers, StreamLogs, and ContainerStatus to docker runner"
```

---

### Task 2: Add StreamLogs to ProcessRunner

**Files:**
- Modify: `internal/runner/process.go`

- [ ] **Step 1: Modify ProcessRunner to use an io.Pipe for streaming**

The current `ProcessRunner` uses a `bytes.Buffer` for output. To support streaming, change the process to write to an `io.Pipe` writer, and provide a `StreamLogs` method that returns a new pipe reader tee'd from the output.

Actually, the simplest approach: change `Start` to use `io.Pipe` and keep a reference to the reader side. Add a `StreamLogs` method:

```go
// StreamLogs returns a reader that streams the process's stdout/stderr.
// It replays existing output then streams new output.
// The caller must close the returned ReadCloser.
func (r *ProcessRunner) StreamLogs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no process for %s", handle.ID)
	}

	// Return a reader over the current buffer snapshot
	// For streaming, we'll use a polling approach since bytes.Buffer
	// doesn't support blocking reads
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		lastLen := 0
		for {
			entry.mu.Lock()
			data := entry.output.Bytes()
			currentLen := len(data)
			entry.mu.Unlock()

			if currentLen > lastLen {
				pw.Write(data[lastLen:currentLen])
				lastLen = currentLen
			}

			select {
			case <-entry.done:
				// Final flush
				entry.mu.Lock()
				data = entry.output.Bytes()
				entry.mu.Unlock()
				if len(data) > lastLen {
					pw.Write(data[lastLen:])
				}
				return
			default:
				// Poll every 100ms
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return pr, nil
}
```

Add `"time"` to the imports.

- [ ] **Step 2: Verify existing tests still pass**

Run: `go test ./internal/runner/... -timeout 10s`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/runner/process.go
git commit -m "feat: add StreamLogs to ProcessRunner for live log streaming"
```

---

### Task 3: Health check integration in orchestrator

**Files:**
- Modify: `internal/orchestrator/orchestrator.go`
- Modify: `internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write test for health check waiting**

Add to `internal/orchestrator/orchestrator_test.go`:

```go
func TestOrchestrator_Up_WaitsForHealthCheck(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16", Healthcheck: "echo ok"},
			"app":      {Command: "air", DependsOn: []string{"postgres"}},
		},
		Profiles: map[string][]string{"default": {"postgres", "app"}},
	}

	orch := orchestrator.New(mock, mock, nil)
	err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	// postgres should start before app (dependency order preserved)
	if len(mock.started) != 2 {
		t.Fatalf("expected 2 started, got %d", len(mock.started))
	}
	if mock.started[0] != "postgres" {
		t.Errorf("expected postgres first, got %s", mock.started[0])
	}
}
```

- [ ] **Step 2: Add health check waiting to the Up method**

In `internal/orchestrator/orchestrator.go`, in the `Up` method, after starting each service in the loop (after `r.Start`), add:

```go
	// Wait for health check if defined
	if svc.Healthcheck != nil {
		check, cfg, err := health.ParseFromService(svc.Healthcheck)
		if err == nil {
			hctx, hcancel := context.WithTimeout(ctx, cfg.Timeout)
			if waitErr := health.WaitUntilHealthy(hctx, check, cfg.Interval); waitErr != nil {
				hcancel()
				return fmt.Errorf("health check failed for %s: %w", name, waitErr)
			}
			hcancel()
		}
	}
```

Add `"github.com/andybarilla/rook/internal/health"` to imports.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/orchestrator/... -timeout 30s`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/orchestrator/
git commit -m "feat: add health check waiting to orchestrator Up method"
```

---

### Task 4: Shared CLI context

**Files:**
- Create: `internal/cli/context.go`
- Create: `internal/cli/context_test.go`

- [ ] **Step 1: Write context tests**

```go
// internal/cli/context_test.go
package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCLIContext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	ctx, err := newCLIContext()
	if err != nil {
		t.Fatal(err)
	}
	if ctx.registry == nil {
		t.Error("expected registry")
	}
	if ctx.portAlloc == nil {
		t.Error("expected port allocator")
	}
}

func TestResolveWorkspaceName_FromArgs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	ctx, _ := newCLIContext()
	// Register a workspace
	ctx.registry.Register("myws", "/some/path")

	name, err := ctx.resolveWorkspaceName([]string{"myws"})
	if err != nil {
		t.Fatal(err)
	}
	if name != "myws" {
		t.Errorf("expected myws, got %s", name)
	}
}

func TestResolveWorkspaceName_FromCurrentDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Create rook.yaml in a temp dir
	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte("name: localws\nservices: {}"), 0644)

	// Change to that dir
	origDir, _ := os.Getwd()
	os.Chdir(wsDir)
	defer os.Chdir(origDir)

	ctx, _ := newCLIContext()
	name, err := ctx.resolveWorkspaceName(nil)
	if err != nil {
		t.Fatal(err)
	}
	if name != "localws" {
		t.Errorf("expected localws, got %s", name)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/... -timeout 10s`
Expected: FAIL

- [ ] **Step 3: Implement context.go**

```go
// internal/cli/context.go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

type cliContext struct {
	registry  *registry.FileRegistry
	portAlloc *ports.FileAllocator
	process   *runner.ProcessRunner
}

func newCLIContext() (*cliContext, error) {
	cfgDir := configDir()
	os.MkdirAll(cfgDir, 0755)

	reg, err := registry.NewFileRegistry(filepath.Join(cfgDir, "workspaces.json"))
	if err != nil {
		return nil, fmt.Errorf("loading registry: %w", err)
	}

	alloc, err := ports.NewFileAllocator(filepath.Join(cfgDir, "ports.json"), 10000, 60000)
	if err != nil {
		return nil, fmt.Errorf("loading port allocator: %w", err)
	}

	return &cliContext{
		registry:  reg,
		portAlloc: alloc,
		process:   runner.NewProcessRunner(),
	}, nil
}

// newOrchestrator creates an orchestrator with a workspace-scoped DockerRunner.
func (c *cliContext) newOrchestrator(wsName string) *orchestrator.Orchestrator {
	docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))
	return orchestrator.New(docker, c.process, c.portAlloc)
}

// resolveWorkspaceName gets the workspace name from args or current directory.
func (c *cliContext) resolveWorkspaceName(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	// Try current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	manifestPath := filepath.Join(cwd, "rook.yaml")
	m, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return "", fmt.Errorf("no workspace specified and no rook.yaml in current directory")
	}
	return m.Name, nil
}

// loadWorkspace resolves and loads a workspace by name.
func (c *cliContext) loadWorkspace(name string) (*workspace.Workspace, error) {
	entry, err := c.registry.Get(name)
	if err != nil {
		return nil, err
	}

	m, err := workspace.ParseManifest(filepath.Join(entry.Path, "rook.yaml"))
	if err != nil {
		return nil, err
	}

	return m.ToWorkspace(entry.Path)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cli/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/context.go internal/cli/context_test.go
git commit -m "feat: add shared CLI context with workspace resolution"
```

---

## Chunk 2: Log Multiplexer + Up Command

### Task 5: Log multiplexer

**Files:**
- Create: `internal/cli/logmux.go`
- Create: `internal/cli/logmux_test.go`

- [ ] **Step 1: Write logmux tests**

```go
// internal/cli/logmux_test.go
package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLogMux_FormatsLines(t *testing.T) {
	var out bytes.Buffer
	mux := newLogMux(&out)

	r := io.NopCloser(strings.NewReader("line 1\nline 2\n"))
	done := make(chan struct{})
	go func() {
		mux.addStream("postgres", r, 0)
		close(done)
	}()
	<-done

	output := out.String()
	if !strings.Contains(output, "[postgres") {
		t.Errorf("expected [postgres prefix, got:\n%s", output)
	}
	if !strings.Contains(output, "line 1") {
		t.Errorf("expected 'line 1' in output")
	}
}

func TestLogMux_MultipleServices(t *testing.T) {
	var out bytes.Buffer
	mux := newLogMux(&out)

	r1 := io.NopCloser(strings.NewReader("pg ready\n"))
	r2 := io.NopCloser(strings.NewReader("app started\n"))

	done := make(chan struct{}, 2)
	go func() { mux.addStream("postgres", r1, 0); done <- struct{}{} }()
	go func() { mux.addStream("app", r2, 1); done <- struct{}{} }()
	<-done
	<-done

	output := out.String()
	if !strings.Contains(output, "pg ready") {
		t.Error("missing postgres output")
	}
	if !strings.Contains(output, "app started") {
		t.Error("missing app output")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/... -timeout 10s -run LogMux`
Expected: FAIL

- [ ] **Step 3: Implement logmux.go**

```go
// internal/cli/logmux.go
package cli

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// ANSI color codes for log output
var logColors = []string{
	"\033[32m", // green
	"\033[33m", // yellow
	"\033[34m", // blue
	"\033[35m", // purple
	"\033[36m", // cyan
	"\033[31m", // red
}

const colorReset = "\033[0m"

// logMux multiplexes log streams from multiple services to a single writer.
type logMux struct {
	mu  sync.Mutex
	out io.Writer
}

func newLogMux(out io.Writer) *logMux {
	return &logMux{out: out}
}

// addStream reads lines from r and writes them to out with a colored service prefix.
// colorIdx determines the color (cycles through logColors).
// Blocks until r is exhausted or closed.
func (m *logMux) addStream(service string, r io.ReadCloser, colorIdx int) {
	defer r.Close()
	color := logColors[colorIdx%len(logColors)]
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m.mu.Lock()
		fmt.Fprintf(m.out, "%s[%-12s]%s %s\n", color, service, colorReset, line)
		m.mu.Unlock()
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/cli/... -timeout 10s -run LogMux`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/logmux.go internal/cli/logmux_test.go
git commit -m "feat: add log multiplexer for color-coded interleaved service output"
```

---

### Task 6: Wire `rook up` command

**Files:**
- Modify: `internal/cli/up.go`

- [ ] **Step 1: Implement the up command**

```go
// internal/cli/up.go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	var detach bool

	cmd := &cobra.Command{
		Use:   "up [workspace] [profile]",
		Short: "Start services",
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

			// Determine profile
			profile := "all"
			if len(args) > 1 {
				profile = args[1]
			} else if _, ok := ws.Profiles["default"]; ok {
				profile = "default"
			}

			// Generate .env files
			portMap := make(map[string]int)
			for name := range ws.Services {
				if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
					portMap[name] = result.Port
				}
			}

			for name, svc := range ws.Services {
				if len(svc.Environment) > 0 {
					resolved, err := envgen.ResolveTemplates(svc.Environment, portMap, false)
					if err != nil {
						return fmt.Errorf("resolving env for %s: %w", name, err)
					}
					envPath := fmt.Sprintf("%s/.env.%s", ws.Root, name)
					if err := envgen.WriteEnvFile(envPath, resolved); err != nil {
						return fmt.Errorf("writing env for %s: %w", name, err)
					}
				}
			}

			// Create workspace-scoped orchestrator and start
			orch := cctx.newOrchestrator(wsName)
			fmt.Printf("Starting %s (profile: %s)...\n", wsName, profile)
			if err := orch.Up(ctx, *ws, profile); err != nil {
				return err
			}

			// Get service order for port display
			services, _ := orchestrator.TopoSort(ws.Services, ws.ServiceNames())
			for _, name := range services {
				if port, ok := portMap[name]; ok {
					fmt.Printf("  %-20s :%d\n", name, port)
				} else {
					fmt.Printf("  %-20s (no port)\n", name)
				}
			}

			if detach {
				fmt.Println("Services started in detach mode.")
				return nil
			}

			// Foreground mode: stream logs
			fmt.Println("\nStreaming logs (Ctrl+C to stop)...\n")
			mux := newLogMux(os.Stdout)
			docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))

			var wg sync.WaitGroup
			statuses, _ := orch.Status(*ws)
			colorIdx := 0
			for _, name := range services {
				if statuses[name] != runner.StatusRunning {
					continue
				}
				svc := ws.Services[name]
				svcName := name
				idx := colorIdx
				colorIdx++

				if svc.IsContainer() {
					handle := runner.RunHandle{ID: svcName, Type: "docker"}
					reader, logCmd, err := docker.StreamLogs(handle)
					if err != nil {
						fmt.Fprintf(os.Stderr, "warning: cannot stream logs for %s: %v\n", svcName, err)
						continue
					}
					wg.Add(1)
					go func() {
						defer wg.Done()
						mux.addStream(svcName, reader, idx)
						if logCmd != nil {
							logCmd.Wait()
						}
					}()
				} else {
					reader, err := cctx.process.StreamLogs(runner.RunHandle{ID: svcName, Type: "process"})
					if err != nil {
						fmt.Fprintf(os.Stderr, "warning: cannot stream logs for %s: %v\n", svcName, err)
						continue
					}
					wg.Add(1)
					go func() {
						defer wg.Done()
						mux.addStream(svcName, reader, idx)
					}()
				}
			}

			// Wait for signal
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nShutting down...")
			cancel() // cancel context stops process services
			orch.Down(context.Background(), *ws)

			wg.Wait()
			fmt.Println("All services stopped.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Start services and exit immediately")
	return cmd
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/rook`
Expected: compiles

- [ ] **Step 3: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat: wire rook up with foreground log streaming and signal handling"
```

---

## Chunk 3: Down, Status, Restart, Logs Commands

### Task 7: Wire `rook down` command

**Files:**
- Modify: `internal/cli/down.go`

- [ ] **Step 1: Implement down command**

```go
// internal/cli/down.go
package cli

import (
	"fmt"
	"strings"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down [workspace]",
		Short: "Stop all services in workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return fmt.Errorf("finding containers: %w", err)
			}

			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				return nil
			}

			for _, name := range containers {
				fmt.Printf("Stopping %s...\n", name)
				runner.StopContainer(name)
			}

			fmt.Printf("Stopped %d container(s) for %s.\n", len(containers), wsName)
			return nil
		},
	}
}
```

- [ ] **Step 2: Add StopContainer helper to docker.go**

```go
// StopContainer stops and removes a container by name.
func StopContainer(name string) {
	exec.Command("docker", "stop", name).Run()
	exec.Command("docker", "rm", name).Run()
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./cmd/rook`
Expected: compiles

- [ ] **Step 4: Commit**

```bash
git add internal/cli/down.go internal/runner/docker.go
git commit -m "feat: wire rook down with Docker container discovery"
```

---

### Task 8: Wire `rook status` command

**Files:**
- Modify: `internal/cli/status.go`

- [ ] **Step 1: Implement status command**

```go
// internal/cli/status.go
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [workspace]",
		Short: "Show all workspaces and running services",
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
	}
}

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

		total := len(m.Services)
		prefix := fmt.Sprintf("rook_%s_", e.Name)
		containers, _ := runner.FindContainers(prefix)

		running := 0
		for _, c := range containers {
			if runner.ContainerStatus(c) == runner.StatusRunning {
				running++
			}
		}

		hasProcessServices := false
		for _, svc := range m.Services {
			if svc.IsProcess() {
				hasProcessServices = true
				break
			}
		}

		status := "stopped"
		if running > 0 && running >= total {
			status = "running"
		} else if running > 0 {
			status = "partial"
		} else if hasProcessServices && len(containers) == 0 {
			status = "unknown"
		}

		fmt.Printf("%-20s %-12s %d/%d\n", e.Name, status, running, total)
	}
	return nil
}

func showWorkspaceDetail(cctx *cliContext, wsName string) error {
	ws, err := cctx.loadWorkspace(wsName)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("rook_%s_", wsName)

	fmt.Printf("%-20s %-12s %-12s %-8s\n", "SERVICE", "TYPE", "STATUS", "PORT")
	for name, svc := range ws.Services {
		svcType := "process"
		status := "unknown"
		if svc.IsContainer() {
			svcType = "container"
			containerName := fmt.Sprintf("%s%s", prefix, name)
			status = string(runner.ContainerStatus(containerName))
		}

		port := "-"
		if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
			port = fmt.Sprintf("%d", result.Port)
		}

		fmt.Printf("%-20s %-12s %-12s %-8s\n", name, svcType, status, port)
	}

	if jsonOutput {
		// JSON output handled separately
	}

	return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/cli/status.go
git commit -m "feat: wire rook status with Docker container inspection"
```

---

### Task 9: Wire `rook restart` command

**Files:**
- Modify: `internal/cli/restart.go`

- [ ] **Step 1: Implement restart command**

```go
// internal/cli/restart.go
package cli

import (
	"context"
	"fmt"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [workspace] [service]",
		Short: "Restart a service",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			orch := cctx.newOrchestrator(wsName)

			// Specific service
			if len(args) > 1 {
				svcName := args[1]
				if _, ok := ws.Services[svcName]; !ok {
					return fmt.Errorf("unknown service: %s", svcName)
				}

				// Stop container if running (standalone mode)
				containerName := fmt.Sprintf("rook_%s_%s", wsName, svcName)
				runner.StopContainer(containerName)

				fmt.Printf("Restarting %s...\n", svcName)
				if err := orch.StartService(context.Background(), *ws, svcName); err != nil {
					return err
				}
				fmt.Printf("Restarted %s.\n", svcName)
				return nil
			}

			// All services — find running containers
			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, _ := runner.FindContainers(prefix)

			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				return nil
			}

			for _, c := range containers {
				runner.StopContainer(c)
			}

			// Re-start via orchestrator
			profile := "all"
			if _, ok := ws.Profiles["default"]; ok {
				profile = "default"
			}

			fmt.Printf("Restarting %s (profile: %s)...\n", wsName, profile)
			if err := orch.Up(context.Background(), *ws, profile); err != nil {
				return err
			}
			fmt.Println("Restarted.")
			return nil
		},
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/cli/restart.go
git commit -m "feat: wire rook restart with Docker container discovery and orchestrator"
```

---

### Task 10: Wire `rook logs` command

**Files:**
- Modify: `internal/cli/logs.go`

- [ ] **Step 1: Implement logs command**

```go
// internal/cli/logs.go
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
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
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			// Single service
			if len(args) > 1 {
				svcName := args[1]
				containerName := fmt.Sprintf("rook_%s_%s", wsName, svcName)
				return streamSingleContainer(containerName)
			}

			// All containers
			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return err
			}
			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				return nil
			}

			mux := newLogMux(os.Stdout)
			var wg sync.WaitGroup

			for i, containerName := range containers {
				svcName := strings.TrimPrefix(containerName, prefix)
				idx := i

				logCmd := exec.Command("docker", "logs", "-f", "--follow", containerName)
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

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nStopped tailing logs.")
			return nil
		},
	}
}

func streamSingleContainer(containerName string) error {
	cmd := exec.Command("docker", "logs", "-f", "--follow", containerName)
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
```

- [ ] **Step 2: Verify everything compiles**

Run: `go build ./cmd/rook`
Expected: compiles

- [ ] **Step 3: Commit**

```bash
git add internal/cli/logs.go
git commit -m "feat: wire rook logs with Docker log streaming and multiplexing"
```

---

### Task 11: Final verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -timeout 30s`
Expected: All packages pass

- [ ] **Step 2: Build and smoke test**

```bash
go build -o bin/rook ./cmd/rook
./bin/rook --help
./bin/rook status
./bin/rook up --help
```

Expected: Help text shows all commands, status shows workspaces, up shows -d flag

- [ ] **Step 3: Commit any fixes**

```bash
git add -A && git commit -m "fix: final CLI wiring adjustments"
```

---

## Summary

| Chunk | Tasks | What it delivers |
|-------|-------|-----------------|
| 1: Foundations | 1-4 | Docker discovery/streaming, process streaming, health check integration, CLI context |
| 2: Up Command | 5-6 | Log multiplexer, `rook up` with foreground streaming + detach mode |
| 3: Other Commands | 7-11 | `rook down`, `rook status`, `rook restart`, `rook logs`, final verification |
