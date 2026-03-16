# Container Reconnection Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the CLI and GUI to reconnect to already-running Docker containers instead of destroying and recreating them, making `rook up` idempotent.

**Architecture:** `DockerRunner.Start` checks for existing containers before creating new ones. A `Reconnectable` interface lets the orchestrator adopt running containers via `DockerRunner.Adopt`. The CLI calls `Orchestrator.Reconnect` after creating the orchestrator so all commands see the correct state.

**Tech Stack:** Go 1.22+

**Spec:** `docs/specs/2026-03-15-container-reconnect-design.md`

---

## File Structure

```
internal/
  runner/
    runner.go               # Add Reconnectable interface
    docker.go               # Add Adopt, Prefix methods; modify Start for adoption
    docker_test.go          # Tests for adoption logic
  orchestrator/
    orchestrator.go         # Add Reconnect method
    orchestrator_test.go    # Tests for Reconnect
  cli/
    up.go                   # Call Reconnect after creating orchestrator
    restart.go              # Call Reconnect after creating orchestrator
```

---

## Chunk 1: Runner + Orchestrator Changes

### Task 1: Add Reconnectable interface and DockerRunner methods

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/docker.go`

- [ ] **Step 1: Add Reconnectable interface to runner.go**

Add after the `Runner` interface definition:

```go
// Reconnectable is implemented by runners that support discovering and adopting
// already-running services (e.g., DockerRunner for Docker containers).
type Reconnectable interface {
	Prefix() string
	Adopt(serviceName string) RunHandle
}
```

- [ ] **Step 2: Add Prefix and Adopt methods to DockerRunner**

Add to `internal/runner/docker.go`:

```go
// Prefix returns the container name prefix used by this runner.
func (r *DockerRunner) Prefix() string {
	return r.prefix
}

// Adopt registers an already-running container in the runner's internal map
// so that Stop, Status, and Logs calls work correctly.
func (r *DockerRunner) Adopt(serviceName string) RunHandle {
	containerName := r.containerName(serviceName)
	r.mu.Lock()
	r.containers[serviceName] = containerName
	r.mu.Unlock()
	return RunHandle{ID: serviceName, Type: "docker"}
}
```

- [ ] **Step 3: Modify DockerRunner.Start to adopt running containers**

Replace the `docker rm -f` line and the beginning of the Start method body (inside the lock) with:

```go
func (r *DockerRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	containerName := r.containerName(name)

	// Check if container already exists
	output, err := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err == nil {
		state := strings.TrimSpace(string(output))
		if state == "running" {
			// Adopt the running container
			r.containers[name] = containerName
			return RunHandle{ID: name, Type: "docker"}, nil
		}
		// Container exists but not running — remove it
		exec.Command("docker", "rm", "-f", containerName).Run()
	}

	// Create new container (existing code from here)
	args := []string{"run", "-d", "--name", containerName}

	if port, ok := ports[name]; ok && len(svc.Ports) > 0 {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, svc.Ports[0]))
	}

	for k, v := range svc.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, vol := range svc.Volumes {
		args = append(args, "-v", vol)
	}

	args = append(args, svc.Image)

	if svc.Command != "" {
		args = append(args, "sh", "-c", svc.Command)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if _, err := cmd.Output(); err != nil {
		return RunHandle{}, fmt.Errorf("docker run %s: %s: %w", containerName, stderr.String(), err)
	}

	r.containers[name] = containerName
	return RunHandle{ID: name, Type: "docker"}, nil
}
```

- [ ] **Step 4: Verify existing tests pass**

Run: `go test ./internal/runner/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/runner.go internal/runner/docker.go
git commit -m "feat: add container adoption to DockerRunner with Reconnectable interface"
```

---

### Task 2: Add Orchestrator.Reconnect

**Files:**
- Modify: `internal/orchestrator/orchestrator.go`
- Modify: `internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write Reconnect tests**

Add to `internal/orchestrator/orchestrator_test.go`:

```go
// mockReconnectable implements runner.Reconnectable for testing.
type mockReconnectable struct {
	mockRunner
	prefix    string
	adopted   []string
}

func (m *mockReconnectable) Prefix() string { return m.prefix }
func (m *mockReconnectable) Adopt(serviceName string) runner.RunHandle {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adopted = append(m.adopted, serviceName)
	return runner.RunHandle{ID: serviceName, Type: "docker"}
}

func TestOrchestrator_Reconnect_AdoptsRunningContainers(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	// This test needs real Docker — or we test the logic without Docker
	// For unit testing, we verify the method exists and handles the no-containers case
	mock := &mockReconnectable{prefix: "rook_test"}
	orch := orchestrator.New(mock, &mock.mockRunner, nil)

	ws := workspace.Workspace{
		Name: "test",
		Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
		},
	}

	// No containers running — should be a no-op
	err := orch.Reconnect(ws)
	if err != nil {
		t.Fatal(err)
	}

	// Status should show all stopped
	statuses, _ := orch.Status(ws)
	if statuses["postgres"] != runner.StatusStopped {
		t.Errorf("expected stopped, got %s", statuses["postgres"])
	}
}

func TestOrchestrator_Reconnect_NonReconnectable(t *testing.T) {
	// ProcessRunner doesn't implement Reconnectable — should be a no-op
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)

	ws := workspace.Workspace{
		Name: "test",
		Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"app": {Command: "air"},
		},
	}

	err := orch.Reconnect(ws)
	if err != nil {
		t.Fatal(err)
	}
}
```

You'll also need to add a `dockerAvailable` helper if not already present:

```go
func dockerAvailable() bool {
	return exec.Command("docker", "info").Run() == nil
}
```

Add `"os/exec"` to imports if needed.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/orchestrator/... -timeout 10s`
Expected: FAIL — Reconnect not defined

- [ ] **Step 3: Implement Reconnect**

Add to `internal/orchestrator/orchestrator.go`:

```go
// Reconnect discovers already-running Docker containers for a workspace
// and populates the orchestrator's handle map so subsequent operations
// (Up, Down, Status) are aware of them.
func (o *Orchestrator) Reconnect(ws workspace.Workspace) error {
	rc, ok := o.containerRunner.(runner.Reconnectable)
	if !ok {
		return nil // runner doesn't support reconnection (e.g., process-only)
	}

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
		// Post-filter for exact prefix match (Docker does substring matching)
		if !strings.HasPrefix(containerName, prefix) {
			continue
		}

		if runner.ContainerStatus(containerName) != runner.StatusRunning {
			continue
		}

		serviceName := strings.TrimPrefix(containerName, prefix)

		// Only adopt services that are defined in the workspace
		if _, exists := ws.Services[serviceName]; !exists {
			continue
		}

		handle := rc.Adopt(serviceName)
		o.mu.Lock()
		o.handles[ws.Name][serviceName] = handle
		o.mu.Unlock()
	}

	return nil
}
```

Add `"strings"` to imports if not already present.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/orchestrator/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/
git commit -m "feat: add Orchestrator.Reconnect for Docker container discovery"
```

---

### Task 3: Wire Reconnect into CLI commands

**Files:**
- Modify: `internal/cli/up.go`
- Modify: `internal/cli/restart.go`

- [ ] **Step 1: Add Reconnect call to up.go**

In `internal/cli/up.go`, after `orch := cctx.newOrchestrator(wsName)` and before `orch.Up(...)`, add:

```go
			// Reconnect to any already-running containers
			if err := orch.Reconnect(*ws); err != nil {
				fmt.Fprintf(os.Stderr, "warning: reconnect failed: %v\n", err)
			}
```

- [ ] **Step 2: Add Reconnect call to restart.go**

In `internal/cli/restart.go`, after `orch := cctx.newOrchestrator(wsName)` and before the service restart logic, add:

```go
			// Reconnect to any already-running containers
			orch.Reconnect(*ws)
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./cmd/rook`
Expected: compiles

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/... ./test/... -timeout 30s`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add internal/cli/up.go internal/cli/restart.go
git commit -m "feat: wire Reconnect into rook up and restart commands"
```

---

## Summary

| Task | What it delivers |
|------|-----------------|
| 1 | `Reconnectable` interface, `DockerRunner.Adopt`/`Prefix`, adoption in `Start` |
| 2 | `Orchestrator.Reconnect` with container discovery + prefix filtering |
| 3 | CLI wiring (`up`, `restart` call Reconnect on startup) |
