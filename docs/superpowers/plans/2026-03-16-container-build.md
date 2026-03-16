# Container Build Support Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `build` field support so Rook can build Docker/Podman images from local Dockerfiles before running containers, enabling services like `api` and `worker` that have `build: .` in docker-compose.

**Architecture:** A `Build` field on `Service` marks it as a buildable container. `DockerRunner.Start` runs `podman build` (or `docker build`) before `podman run` when `Build` is set. A `ForceBuild` runtime flag (set by `--build` CLI flag) forces rebuilds. The compose discoverer extracts both `build` context paths and `command` overrides.

**Tech Stack:** Go 1.22+

**Spec:** `docs/specs/2026-03-16-container-build-design.md`

---

## File Structure

```
internal/
  workspace/
    workspace.go            # Add Build field, update IsContainer/IsProcess
    workspace_test.go       # Tests for Build-aware classification
  runner/
    docker.go               # Build-before-run logic in Start
  discovery/
    compose.go              # Extract build context + command from compose
    compose_test.go         # Tests for build + command extraction
  cli/
    up.go                   # Add --build flag, set ForceBuild on services
```

---

## Chunk 1: Workspace Types + Compose Discoverer

### Task 1: Add Build field and update IsContainer/IsProcess

**Files:**
- Modify: `internal/workspace/workspace.go`
- Modify: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write tests for Build-aware classification**

Add to `internal/workspace/workspace_test.go`:

```go
func TestServiceIsBuildContainer(t *testing.T) {
	svc := workspace.Service{Build: "."}
	if !svc.IsContainer() {
		t.Error("service with build should be a container")
	}
	if svc.IsProcess() {
		t.Error("service with build should not be a process")
	}
}

func TestServiceBuildWithCommand(t *testing.T) {
	svc := workspace.Service{Build: ".", Command: "./server -worker"}
	if !svc.IsContainer() {
		t.Error("service with build+command should be a container")
	}
	if svc.IsProcess() {
		t.Error("service with build+command should not be a process")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/workspace/... -timeout 10s -run Build`
Expected: FAIL or unexpected pass (Build field doesn't exist yet)

- [ ] **Step 3: Add Build field and update methods**

In `internal/workspace/workspace.go`, add to the `Service` struct:

```go
Build      string `yaml:"build,omitempty"`
ForceBuild bool   `yaml:"-"`
```

Update the methods:

```go
func (s Service) IsContainer() bool { return s.Image != "" || s.Build != "" }
func (s Service) IsProcess() bool   { return s.Command != "" && s.Image == "" && s.Build == "" }
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/workspace/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/
git commit -m "feat: add Build field to Service with container classification"
```

---

### Task 2: Compose discoverer — extract build and command

**Files:**
- Modify: `internal/discovery/compose.go`
- Modify: `internal/discovery/compose_test.go`

- [ ] **Step 1: Write tests for build and command extraction**

Add to `internal/discovery/compose_test.go`:

```go
func TestComposeDiscoverer_BuildContext(t *testing.T) {
	dir := t.TempDir()
	compose := `
services:
  api:
    build: .
    ports:
      - "8080:8080"
  worker:
    build: .
    command: ["./server", "-worker"]
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	d := discovery.NewComposeDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}

	api, ok := result.Services["api"]
	if !ok {
		t.Fatal("expected api service")
	}
	if api.Build != "." {
		t.Errorf("expected build '.', got '%s'", api.Build)
	}
	if api.IsProcess() {
		t.Error("api with build should not be a process")
	}
	if !api.IsContainer() {
		t.Error("api with build should be a container")
	}

	worker, ok := result.Services["worker"]
	if !ok {
		t.Fatal("expected worker service")
	}
	if worker.Build != "." {
		t.Errorf("expected build '.', got '%s'", worker.Build)
	}
	if worker.Command != "./server -worker" {
		t.Errorf("expected command './server -worker', got '%s'", worker.Command)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/... -timeout 10s -run BuildContext`
Expected: FAIL — Build and Command not populated

- [ ] **Step 3: Update compose discoverer**

In `internal/discovery/compose.go`, in the `Discover` method's service loop, after the existing field extraction, add:

```go
		// Extract build context (simple string form only)
		if cs.Build != nil {
			if buildStr, ok := cs.Build.(string); ok {
				svc.Build = buildStr
			}
		}

		// Extract command
		if cs.Command != nil {
			switch v := cs.Command.(type) {
			case string:
				svc.Command = v
			case []any:
				parts := make([]string, len(v))
				for i, p := range v {
					parts[i] = fmt.Sprintf("%v", p)
				}
				svc.Command = strings.Join(parts, " ")
			}
		}
```

Add `"strings"` to the imports if not already present.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/discovery/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/
git commit -m "feat: extract build context and command from docker-compose"
```

---

## Chunk 2: DockerRunner Build + CLI Flag

### Task 3: DockerRunner.Start — build before run

**Files:**
- Modify: `internal/runner/docker.go`

- [ ] **Step 1: Add build logic to Start method**

In `internal/runner/docker.go`, modify the `Start` method. After the adoption check (container already running) and before creating the new container, add build handling. Replace the section from the adoption check through the `docker run` args construction:

```go
func (r *DockerRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	containerName := r.containerName(name)

	// Check if container already exists
	output, err := exec.Command(ContainerRuntime, "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err == nil {
		state := strings.TrimSpace(string(output))
		if state == "running" {
			r.containers[name] = containerName
			return RunHandle{ID: name, Type: "docker"}, nil
		}
		exec.Command(ContainerRuntime, "rm", "-f", containerName).Run()
	}

	// Determine the image to use
	imageTag := svc.Image
	if svc.Build != "" {
		// Build the image
		imageTag = fmt.Sprintf("rook-%s-%s:latest", r.prefix, name)
		// Remove the "rook_" prefix from the tag since r.prefix is already "rook_<workspace>"
		// Actually, use a clean tag derived from the prefix
		// r.prefix is "rook_<workspace>", so imageTag becomes "rook-rook_<workspace>-<service>"
		// Better: derive workspace name from prefix
		wsName := strings.TrimPrefix(r.prefix, "rook_")
		imageTag = fmt.Sprintf("rook-%s-%s:latest", wsName, name)

		needsBuild := svc.ForceBuild
		if !needsBuild {
			// Check if image exists
			if err := exec.Command(ContainerRuntime, "image", "inspect", imageTag).Run(); err != nil {
				needsBuild = true
			}
		}

		if needsBuild {
			if workDir == "" {
				return RunHandle{}, fmt.Errorf("cannot build %s: workspace root is empty", name)
			}
			buildCtx := filepath.Join(workDir, svc.Build)
			fmt.Fprintf(os.Stderr, "Building %s from %s...\n", name, buildCtx)
			buildCmd := exec.CommandContext(ctx, ContainerRuntime, "build", "-t", imageTag, buildCtx)
			buildCmd.Stdout = os.Stderr
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				return RunHandle{}, fmt.Errorf("building %s: %w", name, err)
			}
		}
	}

	// Create new container
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

	args = append(args, imageTag)

	if svc.Command != "" {
		args = append(args, "sh", "-c", svc.Command)
	}

	cmd := exec.CommandContext(ctx, ContainerRuntime, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if _, err := cmd.Output(); err != nil {
		return RunHandle{}, fmt.Errorf("%s run %s: %s: %w", ContainerRuntime, containerName, stderr.String(), err)
	}

	r.containers[name] = containerName
	return RunHandle{ID: name, Type: "docker"}, nil
}
```

Add `"os"` and `"path/filepath"` to the imports.

- [ ] **Step 2: Verify existing tests pass**

Run: `go test ./internal/runner/... -timeout 10s`
Expected: PASS

- [ ] **Step 3: Verify full suite**

Run: `go test ./internal/... ./test/... -timeout 30s`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/runner/docker.go
git commit -m "feat: add container build support to DockerRunner.Start"
```

---

### Task 4: Add --build flag to rook up

**Files:**
- Modify: `internal/cli/up.go`

- [ ] **Step 1: Add --build flag and ForceBuild propagation**

In `internal/cli/up.go`, add a `build` variable alongside `detach`:

```go
var detach bool
var build bool
```

At the bottom of the function, register the flag:

```go
cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Start services and exit immediately")
cmd.Flags().BoolVar(&build, "build", false, "Force rebuild of services with build context")
```

After loading the workspace and before calling `orch.Up()`, set `ForceBuild` on services:

```go
			// Set ForceBuild on services if --build flag is set
			if build {
				for name, svc := range ws.Services {
					if svc.Build != "" {
						svc.ForceBuild = true
						ws.Services[name] = svc
					}
				}
			}
```

This goes right before the `orch := cctx.newOrchestrator(wsName)` line.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/rook`
Expected: compiles

- [ ] **Step 3: Verify --build flag shows in help**

Run: `go run ./cmd/rook up --help`
Expected: shows `--build` flag

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/... ./test/... -timeout 30s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat: add --build flag to rook up for forcing image rebuilds"
```

---

## Summary

| Task | What it delivers |
|------|-----------------|
| 1 | `Build` field on Service, updated IsContainer/IsProcess |
| 2 | Compose discoverer extracts build context + command |
| 3 | DockerRunner builds images before running containers |
| 4 | `--build` flag on `rook up` |
