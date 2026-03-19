# Shared Build / Image Alias (`build_from`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow multiple services to share a single built Docker image via `build_from`, eliminating redundant builds.

**Architecture:** Add `BuildFrom` field to `Service`, update `IsContainer()`/`IsProcess()` classification, add manifest validation, modify the runner to resolve `build_from` to the source service's image tag, update topo-sort to include build-order edges, update discovery to auto-detect duplicate build configs, and skip `build_from` services in build cache and rebuild prompts.

**Tech Stack:** Go 1.22+, stdlib `testing`, `gopkg.in/yaml.v3`

**Spec:** `docs/superpowers/specs/2026-03-18-shared-build-alias-design.md`

---

### Task 1: Add `BuildFrom` field and update service classification

**Files:**
- Modify: `internal/workspace/workspace.go:19-38`
- Test: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write failing tests for `BuildFrom` field and classification**

```go
// In internal/workspace/workspace_test.go

func TestService_IsContainer_WithBuildFrom(t *testing.T) {
	svc := Service{BuildFrom: "server"}
	if !svc.IsContainer() {
		t.Error("service with BuildFrom should be a container")
	}
}

func TestService_IsProcess_WithBuildFrom(t *testing.T) {
	svc := Service{BuildFrom: "server", Command: "run.sh"}
	if svc.IsProcess() {
		t.Error("service with BuildFrom should not be a process even with command")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/workspace/ -run "TestService_Is.*BuildFrom" -v`
Expected: FAIL — `BuildFrom` field doesn't exist

- [ ] **Step 3: Add `BuildFrom` field and update classification methods**

In `internal/workspace/workspace.go`, add to `Service` struct after `Dockerfile` (line 32):

```go
BuildFrom  string `yaml:"build_from,omitempty"`
```

Update `IsContainer()` (line 37):

```go
func (s Service) IsContainer() bool { return s.Image != "" || s.Build != "" || s.BuildFrom != "" }
```

Update `IsProcess()` (line 38):

```go
func (s Service) IsProcess() bool { return s.Command != "" && s.Image == "" && s.Build == "" && s.BuildFrom == "" }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/workspace/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "feat: add BuildFrom field to Service struct"
```

---

### Task 2: Add manifest validation for `build_from`

**Files:**
- Modify: `internal/workspace/workspace.go`
- Test: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write failing tests for validation**

```go
// In internal/workspace/workspace_test.go

func TestManifest_Validate_BuildFromValid(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"server": {Build: ".", Ports: []int{8080}},
			"worker": {BuildFrom: "server", Command: "work"},
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("valid build_from should not error: %v", err)
	}
}

func TestManifest_Validate_BuildFromMutuallyExclusiveWithBuild(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"server": {Build: "."},
			"worker": {BuildFrom: "server", Build: "."},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from with build should error")
	}
}

func TestManifest_Validate_BuildFromMutuallyExclusiveWithImage(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"server": {Build: "."},
			"worker": {BuildFrom: "server", Image: "nginx"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from with image should error")
	}
}

func TestManifest_Validate_BuildFromTargetMissing(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"worker": {BuildFrom: "nonexistent"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from referencing missing service should error")
	}
}

func TestManifest_Validate_BuildFromTargetHasNoBuild(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"server": {Image: "nginx"},
			"worker": {BuildFrom: "server"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("build_from referencing service without build should error")
	}
}

func TestManifest_Validate_BuildFromChainDisallowed(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"server":  {Build: "."},
			"worker":  {BuildFrom: "server"},
			"worker2": {BuildFrom: "worker"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Error("chained build_from should error")
	}
}

func TestManifest_Validate_NoBuildFrom(t *testing.T) {
	m := &Manifest{
		Services: map[string]Service{
			"web": {Image: "nginx", Ports: []int{80}},
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("manifest without build_from should validate: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/workspace/ -run "TestManifest_Validate" -v`
Expected: FAIL — `Validate` method doesn't exist

- [ ] **Step 3: Implement `Validate()` method**

Add to `internal/workspace/workspace.go`:

```go
func (m *Manifest) Validate() error {
	for name, svc := range m.Services {
		if svc.BuildFrom == "" {
			continue
		}
		if svc.Build != "" {
			return fmt.Errorf("service %q: build_from is mutually exclusive with build", name)
		}
		if svc.Image != "" {
			return fmt.Errorf("service %q: build_from is mutually exclusive with image", name)
		}
		target, ok := m.Services[svc.BuildFrom]
		if !ok {
			return fmt.Errorf("service %q: build_from references unknown service %q", name, svc.BuildFrom)
		}
		if target.Build == "" {
			return fmt.Errorf("service %q: build_from target %q has no build context", name, svc.BuildFrom)
		}
		if target.BuildFrom != "" {
			return fmt.Errorf("service %q: build_from target %q is itself a build_from (chaining not allowed)", name, svc.BuildFrom)
		}
	}
	return nil
}
```

Add `"fmt"` to the imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/workspace/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "feat: add Validate() for build_from rules"
```

---

### Task 3: Wire validation into manifest parsing

**Files:**
- Modify: `internal/workspace/manifest.go`
- Test: `internal/workspace/manifest_test.go`

- [ ] **Step 1: Write failing test for validation during parse**

```go
// In internal/workspace/manifest_test.go

func TestParseManifest_ValidatesBuildFrom(t *testing.T) {
	dir := t.TempDir()
	content := `name: test
type: single
services:
  worker:
    build_from: nonexistent
    command: run.sh
`
	path := filepath.Join(dir, "rook.yaml")
	os.WriteFile(path, []byte(content), 0644)

	_, err := ParseManifest(path)
	if err == nil {
		t.Error("ParseManifest should return validation error for invalid build_from")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/workspace/ -run "TestParseManifest_ValidatesBuildFrom" -v`
Expected: FAIL — parse succeeds without validation

- [ ] **Step 3: Add validation call to `ParseManifest`**

In `internal/workspace/manifest.go`, after the YAML unmarshal and before returning, add:

```go
if err := m.Validate(); err != nil {
    return nil, fmt.Errorf("validating manifest: %w", err)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/workspace/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/manifest.go internal/workspace/manifest_test.go
git commit -m "feat: wire build_from validation into manifest parsing"
```

---

### Task 4: Update `DockerRunner.Start()` to handle `build_from`

**Files:**
- Modify: `internal/runner/docker.go:58-106`
- Test: `internal/runner/docker_test.go`

Note: `DockerRunner.Start()` shells out to the container runtime, so unit tests should verify the image tag resolution logic rather than running actual containers. The existing test file uses `package runner_test` (black-box testing), so the `resolveImageTag` helper must be exported as `ResolveImageTag` or tested via a white-box test file. Use a separate white-box test file for this.

- [ ] **Step 1: Write failing test for `build_from` image tag resolution**

Create `internal/runner/imagetag_test.go` with `package runner` (white-box test):

```go
// In internal/runner/imagetag_test.go
package runner

import (
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestResolveImageTag_BuildFrom(t *testing.T) {
	r := NewDockerRunner("rook_myapp")
	tag := r.resolveImageTag("worker", workspace.Service{BuildFrom: "server"})
	want := "rook-myapp-server:latest"
	if tag != want {
		t.Errorf("got %q, want %q", tag, want)
	}
}

func TestResolveImageTag_Build(t *testing.T) {
	r := NewDockerRunner("rook_myapp")
	tag := r.resolveImageTag("api", workspace.Service{Build: "."})
	want := "rook-myapp-api:latest"
	if tag != want {
		t.Errorf("got %q, want %q", tag, want)
	}
}

func TestResolveImageTag_Image(t *testing.T) {
	r := NewDockerRunner("rook_myapp")
	tag := r.resolveImageTag("db", workspace.Service{Image: "postgres:16"})
	want := "postgres:16"
	if tag != want {
		t.Errorf("got %q, want %q", tag, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -run "TestResolveImageTag" -v`
Expected: FAIL — `resolveImageTag` doesn't exist

- [ ] **Step 3: Extract `resolveImageTag` and update `Start()`**

In `internal/runner/docker.go`, add the helper:

```go
// resolveImageTag determines the container image tag for a service.
// For build_from services, uses the referenced service's image tag.
func (r *DockerRunner) resolveImageTag(name string, svc workspace.Service) string {
	if svc.BuildFrom != "" {
		wsName := strings.TrimPrefix(r.prefix, "rook_")
		return fmt.Sprintf("rook-%s-%s:latest", wsName, svc.BuildFrom)
	}
	if svc.Build != "" {
		wsName := strings.TrimPrefix(r.prefix, "rook_")
		return fmt.Sprintf("rook-%s-%s:latest", wsName, name)
	}
	return svc.Image
}
```

Then refactor `Start()` (lines 75-106) to use it:

```go
	// Determine the image to use
	imageTag := r.resolveImageTag(name, svc)

	if svc.BuildFrom != "" {
		// build_from: use referenced service's image, no build step
		if err := exec.Command(ContainerRuntime, "image", "inspect", imageTag).Run(); err != nil {
			return RunHandle{}, fmt.Errorf("image for %s not found (build_from: %s) — ensure %s starts first", name, svc.BuildFrom, svc.BuildFrom)
		}
		fmt.Fprintf(os.Stderr, "%s: using image from %s\n", name, svc.BuildFrom)
	} else if svc.Build != "" {
		needsBuild := svc.ForceBuild
		if !needsBuild {
			if err := exec.Command(ContainerRuntime, "image", "inspect", imageTag).Run(); err != nil {
				needsBuild = true
			}
		}

		if needsBuild {
			if workDir == "" {
				return RunHandle{}, fmt.Errorf("cannot build %s: workspace root is empty", name)
			}
			buildCtx := filepath.Join(workDir, svc.Build)
			buildArgs := []string{"build", "-t", imageTag}
			if svc.Dockerfile != "" {
				buildArgs = append(buildArgs, "-f", filepath.Join(workDir, svc.Dockerfile))
			}
			buildArgs = append(buildArgs, buildCtx)
			fmt.Fprintf(os.Stderr, "Building %s from %s...\n", name, buildCtx)
			buildCmd := exec.CommandContext(ctx, ContainerRuntime, buildArgs...)
			buildCmd.Stdout = os.Stderr
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				return RunHandle{}, fmt.Errorf("building %s: %w", name, err)
			}
		}
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/ -run "TestResolveImageTag" -v`
Expected: PASS

Run: `go test ./internal/runner/ -v`
Expected: All existing tests still pass

- [ ] **Step 5: Commit**

```bash
git add internal/runner/docker.go internal/runner/docker_test.go
git commit -m "feat: handle build_from in DockerRunner.Start()"
```

---

### Task 5: Add `build_from` edges to topo-sort

**Files:**
- Modify: `internal/orchestrator/graph.go:12-50`
- Test: `internal/orchestrator/graph_test.go`

- [ ] **Step 1: Write failing tests for `build_from` ordering**

```go
// In internal/orchestrator/graph_test.go

func TestTopoSort_BuildFromOrdering(t *testing.T) {
	services := map[string]workspace.Service{
		"server": {Build: ".", Ports: []int{8080}},
		"worker": {BuildFrom: "server", Command: "work"},
	}
	order, err := TopoSort(services, []string{"server", "worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// server must come before worker
	serverIdx := indexOf(order, "server")
	workerIdx := indexOf(order, "worker")
	if serverIdx > workerIdx {
		t.Errorf("server (idx %d) should come before worker (idx %d), got %v", serverIdx, workerIdx, order)
	}
}

func TestTopoSort_BuildFromPullsInSource(t *testing.T) {
	services := map[string]workspace.Service{
		"server": {Build: ".", Ports: []int{8080}},
		"worker": {BuildFrom: "server", Command: "work"},
	}
	// Only request worker — server should be pulled in
	order, err := TopoSort(services, []string{"worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsStr(order, "server") {
		t.Errorf("build_from source should be pulled into order, got %v", order)
	}
}

func TestTopoSort_BuildFromWithDependsOn(t *testing.T) {
	services := map[string]workspace.Service{
		"postgres": {Image: "postgres:16", Ports: []int{5432}},
		"server":   {Build: ".", DependsOn: []string{"postgres"}},
		"worker":   {BuildFrom: "server", DependsOn: []string{"postgres"}},
	}
	order, err := TopoSort(services, []string{"server", "worker", "postgres"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pgIdx := indexOf(order, "postgres")
	serverIdx := indexOf(order, "server")
	workerIdx := indexOf(order, "worker")
	if pgIdx > serverIdx {
		t.Errorf("postgres should come before server")
	}
	if serverIdx > workerIdx {
		t.Errorf("server should come before worker (build_from dependency)")
	}
}

// helpers — add only if they don't already exist in the test file
func indexOf(order []string, name string) int {
	for i, n := range order {
		if n == name {
			return i
		}
	}
	return -1
}

func containsStr(order []string, name string) bool {
	return indexOf(order, name) >= 0
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/orchestrator/ -run "TestTopoSort_BuildFrom" -v`
Expected: FAIL — `BuildFrom` not followed as edge

- [ ] **Step 3: Add `BuildFrom` edges to `TopoSort`**

**Note:** Adding `build_from` as a topo-sort edge means the consumer waits for the source to fully start (including health checks), not just for the image to be built. This is stricter than the spec's "build-order only" intent, but is acceptable since the image must exist before the consumer can start, and the current architecture doesn't separate "build phase" from "start phase."

In `internal/orchestrator/graph.go`, inside the `visit` function (after the `DependsOn` loop at line 38), add:

```go
		if svc.BuildFrom != "" {
			if err := visit(svc.BuildFrom); err != nil {
				return err
			}
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/orchestrator/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/graph.go internal/orchestrator/graph_test.go
git commit -m "feat: add build_from edges to topo-sort"
```

---

### Task 6: Skip `build_from` services in build cache and rebuild prompts

**Files:**
- Modify: `internal/cli/up.go:67-81` (staleness check), `internal/cli/up.go:287-294` (--build flag), `internal/cli/up.go:305-327` (cache update)
- Modify: `internal/cli/check_builds.go:47-64`
- Test: `internal/cli/up_test.go`, `internal/cli/check_builds_test.go`

- [ ] **Step 1: Write failing tests**

The key behavior: services with `BuildFrom` set should be skipped in staleness checks, `--build` flag marking, and cache updates.

**Already handled (no code changes needed):**
- `up.go` staleness check (line 69): `svc.Build == ""` skips `build_from` services since they have `Build == ""`
- `up.go` `--build` flag (line 289): `svc.Build != ""` only marks services with actual builds
- `up.go` cache update (line 307): `svc.Build == ""` skips `build_from` services
- `buildcache/detect.go` (line 23): `svc.Build == ""` returns early for `build_from` services

Write a test confirming this implicit skip behavior works.

For `check_builds.go`, `build_from` services have `Build == ""` so they'll show as "no build context" — which is acceptable but could say "uses image from server" instead. Add a condition:

```go
// In check_builds.go printCheckBuildsText, add before the Build == "" check:
if svc.BuildFrom != "" {
    fmt.Printf("%s: uses image from %s\n", name, svc.BuildFrom)
    continue
}
```

Similarly for JSON output, add status `"build_from"` with reason being the source service name.

- [ ] **Step 2: Run tests to verify behavior**

Run: `go test ./internal/cli/ -run "TestCheckBuilds" -v`

- [ ] **Step 3: Update `check_builds.go` for `build_from` display**

Update `printCheckBuildsText` and `printCheckBuildsJSON` to handle `BuildFrom`.

- [ ] **Step 4: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/up.go internal/cli/check_builds.go internal/cli/up_test.go internal/cli/check_builds_test.go
git commit -m "feat: skip build_from services in build cache and check-builds"
```

---

### Task 7: Auto-detect duplicate builds in discovery

**Files:**
- Modify: `internal/discovery/compose.go:58-207`
- Test: `internal/discovery/compose_test.go`

- [ ] **Step 1: Write failing test for duplicate build detection**

```go
// In internal/discovery/compose_test.go

func TestComposeDiscoverer_BuildFromDedup(t *testing.T) {
	dir := t.TempDir()
	compose := `services:
  server:
    build: .
    command: ./server
    ports:
      - "8080:8080"
  worker:
    build: .
    command: ./worker
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	d := NewComposeDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First alphabetically (server) should keep build
	server := result.Services["server"]
	if server.Build != "." {
		t.Errorf("server should keep build, got %q", server.Build)
	}
	if server.BuildFrom != "" {
		t.Errorf("server should not have build_from, got %q", server.BuildFrom)
	}

	// Second (worker) should get build_from
	worker := result.Services["worker"]
	if worker.Build != "" {
		t.Errorf("worker should have build cleared, got %q", worker.Build)
	}
	if worker.BuildFrom != "server" {
		t.Errorf("worker should have build_from=server, got %q", worker.BuildFrom)
	}
}

func TestComposeDiscoverer_BuildFromDedup_DifferentDockerfile(t *testing.T) {
	dir := t.TempDir()
	compose := `services:
  api:
    build:
      context: .
      dockerfile: Dockerfile.api
  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	d := NewComposeDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Different dockerfiles — both should keep their builds
	api := result.Services["api"]
	worker := result.Services["worker"]
	if api.BuildFrom != "" || worker.BuildFrom != "" {
		t.Error("different dockerfiles should not trigger build_from dedup")
	}
}

func TestComposeDiscoverer_BuildFromDedup_SameDockerfile(t *testing.T) {
	dir := t.TempDir()
	compose := `services:
  api:
    build:
      context: .
      dockerfile: Dockerfile.go
  worker:
    build:
      context: .
      dockerfile: Dockerfile.go
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	d := NewComposeDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Same context + dockerfile — first alphabetically keeps build
	api := result.Services["api"]
	worker := result.Services["worker"]
	if api.Build == "" {
		t.Error("api (first alpha) should keep build")
	}
	if worker.BuildFrom != "api" {
		t.Errorf("worker should have build_from=api, got %q", worker.BuildFrom)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/ -run "TestComposeDiscoverer_BuildFromDedup" -v`
Expected: FAIL — no dedup logic

- [ ] **Step 3: Implement dedup in `Discover()`**

After the main service loop (after line 197 in `compose.go`, before the devcontainer merge), add dedup logic:

```go
	// Deduplicate identical builds: services with the same (build, dockerfile)
	// tuple get build_from pointing to the first service alphabetically.
	type buildKey struct{ build, dockerfile string }
	buildOwners := make(map[buildKey]string)

	// Collect sorted service names for deterministic ordering
	sortedNames := make([]string, 0, len(result.Services))
	for name := range result.Services {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		svc := result.Services[name]
		if svc.Build == "" {
			continue
		}
		key := buildKey{build: svc.Build, dockerfile: svc.Dockerfile}
		if owner, exists := buildOwners[key]; exists {
			// Duplicate — set build_from, clear build/dockerfile
			svc.BuildFrom = owner
			svc.Build = ""
			svc.Dockerfile = ""
			result.Services[name] = svc
		} else {
			buildOwners[key] = name
		}
	}
```

Add `"sort"` to imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/compose.go internal/discovery/compose_test.go
git commit -m "feat: auto-detect duplicate builds and set build_from in discovery"
```

---

### Task 8: Update CLAUDE.md and final verification

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: All packages pass (except `cmd/rook-gui` which requires frontend build — pre-existing)

- [ ] **Step 2: Update CLAUDE.md**

Remove "Shared build / image alias" from "What's Not Yet Implemented" section.

Add to "Key Patterns" section:

```
- Shared builds (`build_from`): when multiple services share the same build context and Dockerfile, discovery auto-sets `build_from` on duplicates; the runner reuses the source service's image tag without rebuilding
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for build_from feature"
```
