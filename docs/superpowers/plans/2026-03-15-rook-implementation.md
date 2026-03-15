# Rook Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local development workspace manager that eliminates port conflicts, automates environment file generation, and orchestrates service subsets via profiles.

**Architecture:** Interface-driven Go core library consumed by both a Cobra CLI and a Wails desktop GUI. Services run as either Docker containers or local processes. A global port registry prevents conflicts across workspaces.

**Tech Stack:** Go 1.24+, Cobra (CLI), Wails (GUI), React + Tailwind CSS v4 + shadcn/ui (frontend)

**Spec:** `docs/specs/2026-03-15-rook-design.md`

**Deferred to separate plan:** GUI (Wails desktop app, system tray, Dashboard, Workspace Detail, Discovery Wizard, React frontend). The GUI consumes the same core library built here and should be planned once the core is stable.

---

## File Structure

```
go.mod
go.sum

cmd/
  rook/main.go                    # CLI entry point

internal/
  workspace/
    workspace.go                  # Workspace, Service, Manifest types
    workspace_test.go
    manifest.go                   # YAML parsing/serialization
    manifest_test.go

  ports/
    allocator.go                  # PortAllocator interface + file-backed impl
    allocator_test.go

  registry/
    registry.go                   # Global workspace registry
    registry_test.go

  profile/
    resolver.go                   # Profile resolution (groups, wildcards, dedup)
    resolver_test.go

  envgen/
    generator.go                  # Template resolution, .env generation
    generator_test.go

  health/
    checker.go                    # HealthChecker interface + command/HTTP/TCP impls
    checker_test.go

  runner/
    runner.go                     # Runner interface + types
    docker.go                     # Docker container runner
    docker_test.go
    process.go                    # Local process runner
    process_test.go

  orchestrator/
    graph.go                      # Topological sort for dependency ordering
    graph_test.go
    orchestrator.go               # Service lifecycle, start/stop
    orchestrator_test.go

  discovery/
    discovery.go                  # Discoverer interface, composite runner
    compose.go                    # docker-compose discoverer
    compose_test.go
    devcontainer.go               # devcontainer.json discoverer
    devcontainer_test.go
    mise.go                       # mise.toml discoverer
    mise_test.go

  cli/
    root.go                       # Root command + global flags
    init.go                       # rook init
    up.go                         # rook up
    down.go                       # rook down
    status.go                     # rook status
    list.go                       # rook list
    ports.go                      # rook ports
    logs.go                       # rook logs
    env.go                        # rook env
    restart.go                    # rook restart
    discover.go                   # rook discover

test/
  e2e/
    init_test.go                  # End-to-end smoke tests
```

---

## Chunk 1: Project Setup + Workspace Model

### Task 1: Initialize Go Project

**Files:**
- Create: `go.mod`
- Create: `cmd/rook/main.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /home/andy/dev/andybarilla/rook
go mod init github.com/andybarilla/rook
```

- [ ] **Step 2: Create minimal CLI entry point**

```go
// cmd/rook/main.go
package main

import "fmt"

func main() {
	fmt.Println("rook")
}
```

- [ ] **Step 3: Verify it compiles and runs**

Run: `go run ./cmd/rook`
Expected: prints "rook"

- [ ] **Step 4: Initialize git and commit**

```bash
git init
echo -e "# Build\n/rook\n\n# IDE\n.idea/\n.vscode/\n\n# OS\n.DS_Store\n\n# Generated\n*.env\n" > .gitignore
git add go.mod cmd/rook/main.go .gitignore docs/
git commit -m "feat: initialize rook project with Go module and CLI stub"
```

---

### Task 2: Workspace and Service Types

**Files:**
- Create: `internal/workspace/workspace.go`
- Create: `internal/workspace/workspace_test.go`

- [ ] **Step 1: Write tests for workspace types**

```go
// internal/workspace/workspace_test.go
package workspace_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestServiceIsContainer(t *testing.T) {
	svc := workspace.Service{Image: "postgres:16-alpine"}
	if !svc.IsContainer() {
		t.Error("service with image should be a container")
	}
	if svc.IsProcess() {
		t.Error("service with image should not be a process")
	}
}

func TestServiceIsProcess(t *testing.T) {
	svc := workspace.Service{Command: "air"}
	if !svc.IsProcess() {
		t.Error("service with command should be a process")
	}
	if svc.IsContainer() {
		t.Error("service with command should not be a container")
	}
}

func TestWorkspaceServiceNames(t *testing.T) {
	ws := workspace.Workspace{
		Name: "test",
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air"},
		},
	}
	names := ws.ServiceNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 services, got %d", len(names))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/workspace/...`
Expected: FAIL — types not defined

- [ ] **Step 3: Implement workspace types**

```go
// internal/workspace/workspace.go
package workspace

import "sort"

// WorkspaceType distinguishes single-app from multi-service workspaces.
type WorkspaceType string

const (
	TypeSingle WorkspaceType = "single"
	TypeMulti  WorkspaceType = "multi"
)

// HealthcheckConfig holds structured healthcheck settings.
type HealthcheckConfig struct {
	Test     string `yaml:"test"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}

// Service represents a single runnable unit in a workspace.
type Service struct {
	Image       string            `yaml:"image,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	Path        string            `yaml:"path,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Ports       []int             `yaml:"ports,omitempty"`
	PinPort     int               `yaml:"pin_port,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Healthcheck any               `yaml:"healthcheck,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
}

func (s Service) IsContainer() bool { return s.Image != "" }
func (s Service) IsProcess() bool   { return s.Command != "" && s.Image == "" }

// Manifest is the rook.yaml file structure.
type Manifest struct {
	Name     string                `yaml:"name"`
	Type     WorkspaceType         `yaml:"type"`
	Root     string                `yaml:"root,omitempty"`
	Services map[string]Service    `yaml:"services"`
	Groups   map[string][]string   `yaml:"groups,omitempty"`
	Profiles map[string][]string   `yaml:"profiles,omitempty"`
}

// Workspace is a loaded, validated manifest with resolved paths.
type Workspace struct {
	Name     string
	Type     WorkspaceType
	Root     string
	Services map[string]Service
	Groups   map[string][]string
	Profiles map[string][]string
}

// ServiceNames returns sorted service names.
func (w Workspace) ServiceNames() []string {
	names := make([]string, 0, len(w.Services))
	for name := range w.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/workspace/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "feat: add workspace and service domain types"
```

---

### Task 3: Manifest Parsing

**Files:**
- Create: `internal/workspace/manifest.go`
- Create: `internal/workspace/manifest_test.go`
- Modify: `go.mod` (add gopkg.in/yaml.v3)

- [ ] **Step 1: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 2: Write manifest parsing tests**

```go
// internal/workspace/manifest_test.go
package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestParseManifest_SingleWorkspace(t *testing.T) {
	yaml := `
name: skeetr
type: single
services:
  postgres:
    image: postgres:16-alpine
    healthcheck: pg_isready -U skeetr
    volumes:
      - pg-data:/var/lib/postgresql/data
  app:
    command: air
    ports: [8080]
    depends_on: [postgres]
    environment:
      DATABASE_URL: "postgres://skeetr:skeetr@{{.Host.postgres}}:{{.Port.postgres}}/skeetr"
groups:
  infra:
    - postgres
profiles:
  default:
    - infra
    - app
`
	dir := t.TempDir()
	path := filepath.Join(dir, "rook.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := workspace.ParseManifest(path)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if m.Name != "skeetr" {
		t.Errorf("expected name skeetr, got %s", m.Name)
	}
	if m.Type != workspace.TypeSingle {
		t.Errorf("expected type single, got %s", m.Type)
	}
	if len(m.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(m.Services))
	}
	pg := m.Services["postgres"]
	if !pg.IsContainer() {
		t.Error("postgres should be a container service")
	}
	app := m.Services["app"]
	if !app.IsProcess() {
		t.Error("app should be a process service")
	}
	if len(app.DependsOn) != 1 || app.DependsOn[0] != "postgres" {
		t.Errorf("app depends_on wrong: %v", app.DependsOn)
	}
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rook.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.ParseManifest(path)
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestParseManifest_MissingFile(t *testing.T) {
	_, err := workspace.ParseManifest("/nonexistent/rook.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestManifestToWorkspace(t *testing.T) {
	m := workspace.Manifest{
		Name: "test",
		Type: workspace.TypeSingle,
		Services: map[string]workspace.Service{
			"app": {Command: "air", Ports: []int{8080}},
		},
	}
	ws, err := m.ToWorkspace("/some/path")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Root != "/some/path" {
		t.Errorf("expected root /some/path, got %s", ws.Root)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/workspace/...`
Expected: FAIL — ParseManifest and ToWorkspace not defined

- [ ] **Step 4: Implement manifest parsing**

```go
// internal/workspace/manifest.go
package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParseManifest reads and parses a rook.yaml file.
func ParseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &m, nil
}

// WriteManifest writes a manifest to a rook.yaml file.
func WriteManifest(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ToWorkspace converts a parsed manifest into a Workspace with resolved paths.
func (m *Manifest) ToWorkspace(manifestDir string) (*Workspace, error) {
	root := manifestDir
	if m.Root != "" {
		expanded := m.Root
		if expanded[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("expanding home dir: %w", err)
			}
			expanded = filepath.Join(home, expanded[1:])
		}
		root = expanded
	}

	if m.Name == "" {
		return nil, fmt.Errorf("manifest missing required field: name")
	}

	return &Workspace{
		Name:     m.Name,
		Type:     m.Type,
		Root:     root,
		Services: m.Services,
		Groups:   m.Groups,
		Profiles: m.Profiles,
	}, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/workspace/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/workspace/manifest.go internal/workspace/manifest_test.go go.mod go.sum
git commit -m "feat: add manifest YAML parsing and workspace conversion"
```

---

## Chunk 2: Port Allocator + Registry

### Task 4: Port Allocator

**Files:**
- Create: `internal/ports/allocator.go`
- Create: `internal/ports/allocator_test.go`

- [ ] **Step 1: Write port allocator tests**

```go
// internal/ports/allocator_test.go
package ports_test

import (
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/ports"
)

func TestAllocate_AssignsFromRange(t *testing.T) {
	dir := t.TempDir()
	a, err := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	if err != nil {
		t.Fatal(err)
	}

	port, err := a.Allocate("ws1", "postgres", 0)
	if err != nil {
		t.Fatal(err)
	}
	if port < 10000 || port > 10010 {
		t.Errorf("port %d outside range 10000-10010", port)
	}
}

func TestAllocate_PreferredPort(t *testing.T) {
	dir := t.TempDir()
	a, err := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	if err != nil {
		t.Fatal(err)
	}

	port, err := a.Allocate("ws1", "app", 10005)
	if err != nil {
		t.Fatal(err)
	}
	if port != 10005 {
		t.Errorf("expected preferred port 10005, got %d", port)
	}
}

func TestAllocate_StablePorts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ports.json")

	a1, _ := ports.NewFileAllocator(path, 10000, 10010)
	port1, _ := a1.Allocate("ws1", "postgres", 0)

	a2, _ := ports.NewFileAllocator(path, 10000, 10010)
	port2, _ := a2.Get("ws1", "postgres")
	if !port2.OK {
		t.Fatal("expected port to persist across reloads")
	}
	if port1 != port2.Port {
		t.Errorf("port changed: %d -> %d", port1, port2.Port)
	}
}

func TestAllocate_NoConflicts(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10002)

	a.Allocate("ws1", "a", 0)
	a.Allocate("ws1", "b", 0)
	a.Allocate("ws1", "c", 0)

	_, err := a.Allocate("ws1", "d", 0)
	if err == nil {
		t.Fatal("expected error when port range exhausted")
	}
}

func TestAllocate_PinnedConflict(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)

	_, err := a.AllocatePinned("ws1", "app", 8080)
	if err != nil {
		t.Fatal(err)
	}

	_, err = a.AllocatePinned("ws2", "app", 8080)
	if err == nil {
		t.Fatal("expected error for pinned port conflict")
	}
}

func TestRelease(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10001)

	a.Allocate("ws1", "a", 0)
	a.Allocate("ws1", "b", 0)

	err := a.Release("ws1", "a")
	if err != nil {
		t.Fatal(err)
	}

	_, err = a.Allocate("ws1", "c", 0)
	if err != nil {
		t.Fatal("should be able to allocate after release")
	}
}

func TestAll(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	a.Allocate("ws1", "postgres", 0)
	a.Allocate("ws2", "redis", 0)

	all := a.All()
	if len(all) != 2 {
		t.Errorf("expected 2 entries, got %d", len(all))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ports/...`
Expected: FAIL

- [ ] **Step 3: Implement port allocator**

```go
// internal/ports/allocator.go
package ports

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// PortEntry is a single port allocation record.
type PortEntry struct {
	Workspace string `json:"workspace"`
	Service   string `json:"service"`
	Port      int    `json:"port"`
	Pinned    bool   `json:"pinned,omitempty"`
}

// LookupResult is returned by Get.
type LookupResult struct {
	Port int
	OK   bool
}

// PortAllocator manages global port assignments.
type PortAllocator interface {
	Allocate(workspace, service string, preferred int) (int, error)
	AllocatePinned(workspace, service string, port int) (int, error)
	Release(workspace, service string) error
	Get(workspace, service string) LookupResult
	All() []PortEntry
}

// FileAllocator is a file-backed PortAllocator.
type FileAllocator struct {
	mu       sync.Mutex
	path     string
	minPort  int
	maxPort  int
	entries  []PortEntry
	used     map[int]bool
}

func NewFileAllocator(path string, minPort, maxPort int) (*FileAllocator, error) {
	a := &FileAllocator{
		path:    path,
		minPort: minPort,
		maxPort: maxPort,
		used:    make(map[int]bool),
	}
	if err := a.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return a, nil
}

func (a *FileAllocator) load() error {
	data, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &a.entries); err != nil {
		return err
	}
	for _, e := range a.entries {
		a.used[e.Port] = true
	}
	return nil
}

func (a *FileAllocator) save() error {
	data, err := json.MarshalIndent(a.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.path, data, 0644)
}

func (a *FileAllocator) key(workspace, service string) string {
	return workspace + "." + service
}

func (a *FileAllocator) findIndex(workspace, service string) int {
	for i, e := range a.entries {
		if e.Workspace == workspace && e.Service == service {
			return i
		}
	}
	return -1
}

func (a *FileAllocator) Allocate(workspace, service string, preferred int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if idx := a.findIndex(workspace, service); idx >= 0 {
		return a.entries[idx].Port, nil
	}

	if preferred > 0 && !a.used[preferred] {
		return a.assign(workspace, service, preferred, false)
	}

	for p := a.minPort; p <= a.maxPort; p++ {
		if !a.used[p] {
			return a.assign(workspace, service, p, false)
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", a.minPort, a.maxPort)
}

func (a *FileAllocator) AllocatePinned(workspace, service string, port int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if idx := a.findIndex(workspace, service); idx >= 0 {
		return a.entries[idx].Port, nil
	}

	if a.used[port] {
		for _, e := range a.entries {
			if e.Port == port {
				return 0, fmt.Errorf("port %d already pinned by %s.%s", port, e.Workspace, e.Service)
			}
		}
	}

	return a.assign(workspace, service, port, true)
}

func (a *FileAllocator) assign(workspace, service string, port int, pinned bool) (int, error) {
	entry := PortEntry{Workspace: workspace, Service: service, Port: port, Pinned: pinned}
	a.entries = append(a.entries, entry)
	a.used[port] = true
	if err := a.save(); err != nil {
		return 0, err
	}
	return port, nil
}

func (a *FileAllocator) Release(workspace, service string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	idx := a.findIndex(workspace, service)
	if idx < 0 {
		return nil
	}

	port := a.entries[idx].Port
	a.entries = append(a.entries[:idx], a.entries[idx+1:]...)
	delete(a.used, port)
	return a.save()
}

func (a *FileAllocator) Get(workspace, service string) LookupResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	idx := a.findIndex(workspace, service)
	if idx < 0 {
		return LookupResult{}
	}
	return LookupResult{Port: a.entries[idx].Port, OK: true}
}

func (a *FileAllocator) All() []PortEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]PortEntry, len(a.entries))
	copy(out, a.entries)
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ports/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ports/
git commit -m "feat: add file-backed port allocator with pinning support"
```

---

### Task 5: Workspace Registry

**Files:**
- Create: `internal/registry/registry.go`
- Create: `internal/registry/registry_test.go`

- [ ] **Step 1: Write registry tests**

```go
// internal/registry/registry_test.go
package registry_test

import (
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/registry"
)

func TestRegisterAndList(t *testing.T) {
	dir := t.TempDir()
	r, err := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}

	err = r.Register("skeetr", "/home/user/dev/skeetr")
	if err != nil {
		t.Fatal(err)
	}

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(list))
	}
	if list[0].Name != "skeetr" {
		t.Errorf("expected name skeetr, got %s", list[0].Name)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	r.Register("skeetr", "/path/a")
	err := r.Register("skeetr", "/path/b")
	if err == nil {
		t.Fatal("expected error for duplicate workspace name")
	}
}

func TestGetByName(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	r.Register("skeetr", "/some/path")

	entry, err := r.Get("skeetr")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Path != "/some/path" {
		t.Errorf("expected path /some/path, got %s", entry.Path)
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))

	_, err := r.Get("nope")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	r.Register("skeetr", "/path")
	r.Remove("skeetr")

	list := r.List()
	if len(list) != 0 {
		t.Fatalf("expected 0 workspaces, got %d", len(list))
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspaces.json")

	r1, _ := registry.NewFileRegistry(path)
	r1.Register("skeetr", "/path")

	r2, _ := registry.NewFileRegistry(path)
	list := r2.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 workspace after reload, got %d", len(list))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/registry/...`
Expected: FAIL

- [ ] **Step 3: Implement registry**

```go
// internal/registry/registry.go
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Entry represents a registered workspace.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Registry manages the global list of registered workspaces.
type Registry interface {
	Register(name, path string) error
	Remove(name string)
	Get(name string) (Entry, error)
	List() []Entry
}

// FileRegistry is a file-backed registry.
type FileRegistry struct {
	mu      sync.Mutex
	path    string
	entries []Entry
}

func NewFileRegistry(path string) (*FileRegistry, error) {
	r := &FileRegistry{path: path}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return r, nil
}

func (r *FileRegistry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &r.entries)
}

func (r *FileRegistry) save() error {
	data, err := json.MarshalIndent(r.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

func (r *FileRegistry) Register(name, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.entries {
		if e.Name == name {
			return fmt.Errorf("workspace %q already registered", name)
		}
	}

	r.entries = append(r.entries, Entry{Name: name, Path: path})
	return r.save()
}

func (r *FileRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, e := range r.entries {
		if e.Name == name {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			r.save()
			return
		}
	}
}

func (r *FileRegistry) Get(name string) (Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.entries {
		if e.Name == name {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("workspace %q not found", name)
}

func (r *FileRegistry) List() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/registry/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/registry/
git commit -m "feat: add file-backed workspace registry"
```

---

## Chunk 3: Profile Resolution + Environment Generation

### Task 6: Profile Resolver

**Files:**
- Create: `internal/profile/resolver.go`
- Create: `internal/profile/resolver_test.go`

- [ ] **Step 1: Write profile resolver tests**

```go
// internal/profile/resolver_test.go
package profile_test

import (
	"sort"
	"testing"

	"github.com/andybarilla/rook/internal/profile"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestResolve_DefaultProfile(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air"},
		},
		Groups: map[string][]string{
			"infra": {"postgres"},
		},
		Profiles: map[string][]string{
			"default": {"infra", "app"},
		},
	}

	services, err := profile.Resolve(ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(services)
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d: %v", len(services), services)
	}
	if services[0] != "app" || services[1] != "postgres" {
		t.Errorf("unexpected services: %v", services)
	}
}

func TestResolve_WildcardProfile(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{
			"postgres": {}, "redis": {}, "app": {},
		},
		Groups:   map[string][]string{"infra": {"postgres", "redis"}},
		Profiles: map[string][]string{"all": {"infra", "*"}},
	}

	services, err := profile.Resolve(ws, "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 3 {
		t.Errorf("expected 3 services, got %d: %v", len(services), services)
	}
}

func TestResolve_ImplicitAllProfile(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{
			"a": {}, "b": {}, "c": {},
		},
	}

	services, err := profile.Resolve(ws, "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 3 {
		t.Fatalf("expected 3 services for implicit all, got %d", len(services))
	}
}

func TestResolve_UnknownProfile(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{"a": {}},
	}

	_, err := profile.Resolve(ws, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestResolve_Deduplication(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{
			"postgres": {}, "app": {},
		},
		Groups:   map[string][]string{"infra": {"postgres"}},
		Profiles: map[string][]string{"dupe": {"infra", "postgres", "app"}},
	}

	services, err := profile.Resolve(ws, "dupe")
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 {
		t.Errorf("expected 2 deduplicated services, got %d: %v", len(services), services)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/profile/...`
Expected: FAIL

- [ ] **Step 3: Implement profile resolver**

```go
// internal/profile/resolver.go
package profile

import (
	"fmt"

	"github.com/andybarilla/rook/internal/workspace"
)

// Resolve expands a profile name into a deduplicated list of service names.
func Resolve(ws workspace.Workspace, profileName string) ([]string, error) {
	var entries []string

	if profileName == "all" {
		if p, ok := ws.Profiles["all"]; ok {
			entries = p
		} else {
			return ws.ServiceNames(), nil
		}
	} else {
		p, ok := ws.Profiles[profileName]
		if !ok {
			return nil, fmt.Errorf("unknown profile: %q", profileName)
		}
		entries = p
	}

	seen := make(map[string]bool)
	var result []string

	for _, entry := range entries {
		if entry == "*" {
			for _, name := range ws.ServiceNames() {
				if !seen[name] {
					seen[name] = true
					result = append(result, name)
				}
			}
			continue
		}

		if group, ok := ws.Groups[entry]; ok {
			for _, name := range group {
				if !seen[name] {
					seen[name] = true
					result = append(result, name)
				}
			}
			continue
		}

		if _, ok := ws.Services[entry]; ok {
			if !seen[entry] {
				seen[entry] = true
				result = append(result, entry)
			}
			continue
		}

		return nil, fmt.Errorf("profile %q references unknown service or group: %q", profileName, entry)
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/profile/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/profile/
git commit -m "feat: add profile resolver with group expansion and wildcards"
```

---

### Task 7: Environment Generator

**Files:**
- Create: `internal/envgen/generator.go`
- Create: `internal/envgen/generator_test.go`

- [ ] **Step 1: Write envgen tests**

```go
// internal/envgen/generator_test.go
package envgen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestResolveTemplates_PortAndHost(t *testing.T) {
	env := map[string]string{
		"DATABASE_URL": "postgres://u:p@{{.Host.postgres}}:{{.Port.postgres}}/db",
	}

	portMap := map[string]int{"postgres": 12345}

	result, err := envgen.ResolveTemplates(env, portMap, false)
	if err != nil {
		t.Fatal(err)
	}

	expected := "postgres://u:p@localhost:12345/db"
	if result["DATABASE_URL"] != expected {
		t.Errorf("expected %s, got %s", expected, result["DATABASE_URL"])
	}
}

func TestResolveTemplates_DevcontainerContext(t *testing.T) {
	env := map[string]string{
		"REDIS_URL": "redis://{{.Host.redis}}:{{.Port.redis}}",
	}

	svc := workspace.Service{Image: "redis:7", Ports: []int{6379}}
	portMap := map[string]int{"redis": 12346}

	result, err := envgen.ResolveTemplatesWithServices(env, portMap, true,
		map[string]workspace.Service{"redis": svc})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result["REDIS_URL"], "redis") {
		t.Errorf("devcontainer should use service name as host, got: %s", result["REDIS_URL"])
	}
	if !strings.Contains(result["REDIS_URL"], "6379") {
		t.Errorf("devcontainer should use internal port, got: %s", result["REDIS_URL"])
	}
}

func TestWriteEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	vars := map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "12345",
	}

	err := envgen.WriteEnvFile(path, vars)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, "DB_HOST=localhost") {
		t.Errorf("expected DB_HOST=localhost in output, got:\n%s", content)
	}
	if !strings.Contains(content, "DB_PORT=12345") {
		t.Errorf("expected DB_PORT=12345 in output, got:\n%s", content)
	}
}

func TestResolveTemplates_NoTemplates(t *testing.T) {
	env := map[string]string{"STATIC": "value"}
	result, err := envgen.ResolveTemplates(env, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result["STATIC"] != "value" {
		t.Errorf("expected value, got %s", result["STATIC"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/envgen/...`
Expected: FAIL

- [ ] **Step 3: Implement environment generator**

```go
// internal/envgen/generator.go
package envgen

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/andybarilla/rook/internal/workspace"
)

type templateData struct {
	Port map[string]string
	Host map[string]string
}

// ResolveTemplates resolves {{.Port.x}} and {{.Host.x}} in environment values.
func ResolveTemplates(env map[string]string, portMap map[string]int, devcontainer bool) (map[string]string, error) {
	return ResolveTemplatesWithServices(env, portMap, devcontainer, nil)
}

// ResolveTemplatesWithServices resolves templates with full service context (needed for devcontainer internal ports).
func ResolveTemplatesWithServices(env map[string]string, portMap map[string]int, devcontainer bool, services map[string]workspace.Service) (map[string]string, error) {
	data := templateData{
		Port: make(map[string]string),
		Host: make(map[string]string),
	}

	for name, port := range portMap {
		if devcontainer && services != nil {
			if svc, ok := services[name]; ok && svc.IsContainer() && len(svc.Ports) > 0 {
				data.Port[name] = strconv.Itoa(svc.Ports[0])
			} else {
				data.Port[name] = strconv.Itoa(port)
			}
		} else {
			data.Port[name] = strconv.Itoa(port)
		}

		if devcontainer {
			data.Host[name] = name
		} else {
			data.Host[name] = "localhost"
		}
	}

	result := make(map[string]string, len(env))
	for k, v := range env {
		if !strings.Contains(v, "{{") {
			result[k] = v
			continue
		}

		tmpl, err := template.New(k).Parse(v)
		if err != nil {
			return nil, fmt.Errorf("parsing template for %s: %w", k, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("executing template for %s: %w", k, err)
		}
		result[k] = buf.String()
	}

	return result, nil
}

// WriteEnvFile writes environment variables to a .env file.
func WriteEnvFile(path string, vars map[string]string) error {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString("# Generated by Rook — do not edit\n")
	for _, k := range keys {
		fmt.Fprintf(&buf, "%s=%s\n", k, vars[k])
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/envgen/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/envgen/
git commit -m "feat: add environment generator with template resolution and devcontainer support"
```

---

## Chunk 4: Health Checks + Service Runners

### Task 8: Health Checker

**Files:**
- Create: `internal/health/checker.go`
- Create: `internal/health/checker_test.go`

- [ ] **Step 1: Write health checker tests**

```go
// internal/health/checker_test.go
package health_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/health"
)

func TestParseCheck_Command(t *testing.T) {
	check, err := health.Parse("pg_isready -U skeetr")
	if err != nil {
		t.Fatal(err)
	}
	if check.Type != health.TypeCommand {
		t.Errorf("expected command type, got %s", check.Type)
	}
}

func TestParseCheck_HTTP(t *testing.T) {
	check, err := health.Parse("http://localhost:8080/health")
	if err != nil {
		t.Fatal(err)
	}
	if check.Type != health.TypeHTTP {
		t.Errorf("expected http type, got %s", check.Type)
	}
	if check.Target != "http://localhost:8080/health" {
		t.Errorf("unexpected target: %s", check.Target)
	}
}

func TestParseCheck_TCP(t *testing.T) {
	check, err := health.Parse("tcp://localhost:5432")
	if err != nil {
		t.Fatal(err)
	}
	if check.Type != health.TypeTCP {
		t.Errorf("expected tcp type, got %s", check.Type)
	}
}

func TestHTTPCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	check := health.Check{Type: health.TypeHTTP, Target: srv.URL}
	err := health.Run(context.Background(), check)
	if err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestHTTPCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	check := health.Check{Type: health.TypeHTTP, Target: srv.URL}
	err := health.Run(context.Background(), check)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestTCPCheck_Healthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	check := health.Check{Type: health.TypeTCP, Target: ln.Addr().String()}
	err = health.Run(context.Background(), check)
	if err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestTCPCheck_Unhealthy(t *testing.T) {
	check := health.Check{Type: health.TypeTCP, Target: "127.0.0.1:1"}
	err := health.Run(context.Background(), check)
	if err == nil {
		t.Error("expected error for closed port")
	}
}

func TestWaitForHealthy_Timeout(t *testing.T) {
	check := health.Check{Type: health.TypeTCP, Target: "127.0.0.1:1"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := health.WaitUntilHealthy(ctx, check, 50*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestWaitForHealthy_EventuallyHealthy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	// Start listener after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		ln2, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Printf("test listener error: %v\n", err)
			return
		}
		defer ln2.Close()
		time.Sleep(2 * time.Second) // keep it alive
	}()

	check := health.Check{Type: health.TypeTCP, Target: addr}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = health.WaitUntilHealthy(ctx, check, 50*time.Millisecond)
	if err != nil {
		t.Errorf("expected eventual health, got error: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/health/...`
Expected: FAIL

- [ ] **Step 3: Implement health checker**

```go
// internal/health/checker.go
package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type CheckType string

const (
	TypeCommand CheckType = "command"
	TypeHTTP    CheckType = "http"
	TypeTCP     CheckType = "tcp"
)

type Check struct {
	Type   CheckType
	Target string // command string, URL, or host:port
}

// Parse determines check type from a healthcheck string.
func Parse(s string) (Check, error) {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return Check{Type: TypeHTTP, Target: s}, nil
	}
	if strings.HasPrefix(s, "tcp://") {
		return Check{Type: TypeTCP, Target: strings.TrimPrefix(s, "tcp://")}, nil
	}
	return Check{Type: TypeCommand, Target: s}, nil
}

// Run executes a single health check. Returns nil if healthy.
func Run(ctx context.Context, check Check) error {
	switch check.Type {
	case TypeHTTP:
		return runHTTP(ctx, check.Target)
	case TypeTCP:
		return runTCP(ctx, check.Target)
	case TypeCommand:
		return runCommand(ctx, check.Target)
	default:
		return fmt.Errorf("unknown health check type: %s", check.Type)
	}
}

func runHTTP(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

func runTCP(ctx context.Context, addr string) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func runCommand(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	return cmd.Run()
}

// Config holds healthcheck timing settings.
type Config struct {
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

// DefaultConfig returns the default healthcheck config (2s interval, 30s timeout, 15 retries).
func DefaultConfig() Config {
	return Config{Interval: 2 * time.Second, Timeout: 30 * time.Second, Retries: 15}
}

// ParseFromService handles both string and structured healthcheck forms from a service definition.
func ParseFromService(hc any) (Check, Config, error) {
	cfg := DefaultConfig()

	switch v := hc.(type) {
	case string:
		check, err := Parse(v)
		return check, cfg, err
	case map[string]any:
		if test, ok := v["test"].(string); ok {
			check, err := Parse(test)
			if err != nil {
				return Check{}, cfg, err
			}
			if interval, ok := v["interval"].(string); ok {
				if d, err := time.ParseDuration(interval); err == nil {
					cfg.Interval = d
				}
			}
			if timeout, ok := v["timeout"].(string); ok {
				if d, err := time.ParseDuration(timeout); err == nil {
					cfg.Timeout = d
				}
			}
			if retries, ok := v["retries"].(int); ok {
				cfg.Retries = retries
			}
			return check, cfg, nil
		}
		return Check{}, cfg, fmt.Errorf("structured healthcheck missing 'test' field")
	case nil:
		return Check{}, cfg, fmt.Errorf("no healthcheck defined")
	default:
		return Check{}, cfg, fmt.Errorf("unsupported healthcheck type: %T", hc)
	}
}

// WaitUntilHealthy polls the check at the given interval until healthy or context expires.
func WaitUntilHealthy(ctx context.Context, check Check, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Try immediately first
	if err := Run(ctx, check); err == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timed out: %w", ctx.Err())
		case <-ticker.C:
			if err := Run(ctx, check); err == nil {
				return nil
			}
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/health/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/health/
git commit -m "feat: add health checker with HTTP, TCP, and command support"
```

---

### Task 9: Runner Interface + Process Runner

**Files:**
- Create: `internal/runner/runner.go`
- Create: `internal/runner/process.go`
- Create: `internal/runner/process_test.go`

- [ ] **Step 1: Write process runner tests**

```go
// internal/runner/process_test.go
package runner_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestProcessRunner_StartAndStop(t *testing.T) {
	r := runner.NewProcessRunner()

	svc := workspace.Service{
		Command: "sleep 60",
	}

	ctx := context.Background()
	handle, err := r.Start(ctx, "test-svc", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	status, err := r.Status(handle)
	if err != nil {
		t.Fatal(err)
	}
	if status != runner.StatusRunning {
		t.Errorf("expected running, got %s", status)
	}

	if err := r.Stop(handle); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	status, _ = r.Status(handle)
	if status == runner.StatusRunning {
		t.Error("expected not running after stop")
	}
}

func TestProcessRunner_Logs(t *testing.T) {
	r := runner.NewProcessRunner()

	svc := workspace.Service{
		Command: "echo hello-from-process",
	}

	ctx := context.Background()
	handle, err := r.Start(ctx, "echo-svc", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	reader, err := r.Logs(handle)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	data, _ := io.ReadAll(reader)
	if len(data) == 0 {
		t.Error("expected log output")
	}
}

func TestProcessRunner_WorkingDir(t *testing.T) {
	r := runner.NewProcessRunner()
	dir := t.TempDir()

	svc := workspace.Service{
		Command: "pwd",
	}

	handle, err := r.Start(context.Background(), "pwd-svc", svc, nil, dir)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(200 * time.Millisecond)

	reader, _ := r.Logs(handle)
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	output := string(data)

	if len(output) == 0 {
		t.Error("expected pwd output")
	}

	_ = r.Stop(handle)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/...`
Expected: FAIL

- [ ] **Step 3: Implement runner interface and process runner**

```go
// internal/runner/runner.go
package runner

import (
	"context"
	"io"

	"github.com/andybarilla/rook/internal/workspace"
)

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusStopped ServiceStatus = "stopped"
	StatusCrashed ServiceStatus = "crashed"
)

type RunHandle struct {
	ID   string
	Type string // "process" or "docker"
}

type PortMap map[string]int

type Runner interface {
	Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error)
	Stop(handle RunHandle) error
	Status(handle RunHandle) (ServiceStatus, error)
	Logs(handle RunHandle) (io.ReadCloser, error)
}
```

```go
// internal/runner/process.go
package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/andybarilla/rook/internal/workspace"
)

type processEntry struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	output *bytes.Buffer
	mu     sync.Mutex
	done   chan struct{}
	err    error
}

type ProcessRunner struct {
	mu      sync.Mutex
	entries map[string]*processEntry
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{entries: make(map[string]*processEntry)}
}

func (r *ProcessRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	procCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(procCtx, "sh", "-c", svc.Command)
	cmd.Dir = workDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	cmd.Env = os.Environ()
	for k, v := range svc.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	entry := &processEntry{
		cmd:    cmd,
		cancel: cancel,
		output: &output,
		done:   make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return RunHandle{}, fmt.Errorf("starting %s: %w", name, err)
	}

	go func() {
		entry.err = cmd.Wait()
		close(entry.done)
	}()

	handle := RunHandle{ID: name, Type: "process"}
	r.entries[name] = entry
	return handle, nil
}

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

func (r *ProcessRunner) Status(handle RunHandle) (ServiceStatus, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()

	if !ok {
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

func (r *ProcessRunner) Logs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	entry, ok := r.entries[handle.ID]
	r.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no logs for %s", handle.ID)
	}

	entry.mu.Lock()
	data := make([]byte, entry.output.Len())
	copy(data, entry.output.Bytes())
	entry.mu.Unlock()

	return io.NopCloser(bytes.NewReader(data)), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/... -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/
git commit -m "feat: add runner interface and process runner implementation"
```

---

### Task 10: Docker Runner

**Files:**
- Create: `internal/runner/docker.go`
- Create: `internal/runner/docker_test.go`

- [ ] **Step 1: Write docker runner tests**

Note: Docker tests need Docker available. Use build tags or skip if unavailable.

```go
// internal/runner/docker_test.go
package runner_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

func dockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func TestDockerRunner_StartAndStop(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	r := runner.NewDockerRunner("rook-test")

	svc := workspace.Service{
		Image: "alpine:latest",
		Ports: []int{8080},
	}
	ports := runner.PortMap{"test-container": 18080}

	ctx := context.Background()
	handle, err := r.Start(ctx, "test-container", svc, ports, "")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	status, err := r.Status(handle)
	if err != nil {
		t.Fatal(err)
	}
	if status != runner.StatusRunning {
		t.Logf("status: %s (alpine may exit immediately, that's ok)", status)
	}

	if err := r.Stop(handle); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/... -run Docker -timeout 30s`
Expected: FAIL (or skip)

- [ ] **Step 3: Implement docker runner**

```go
// internal/runner/docker.go
package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/andybarilla/rook/internal/workspace"
)

// DockerRunner manages Docker containers.
type DockerRunner struct {
	mu        sync.Mutex
	prefix    string
	containers map[string]string // handle ID -> container name
}

func NewDockerRunner(prefix string) *DockerRunner {
	return &DockerRunner{
		prefix:     prefix,
		containers: make(map[string]string),
	}
}

func (r *DockerRunner) containerName(name string) string {
	return fmt.Sprintf("%s_%s", r.prefix, name)
}

func (r *DockerRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	containerName := r.containerName(name)

	// Remove any existing container with same name
	exec.CommandContext(ctx, "docker", "rm", "-f", containerName).Run()

	args := []string{"run", "-d", "--name", containerName}

	// Port mappings
	if port, ok := ports[name]; ok && len(svc.Ports) > 0 {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, svc.Ports[0]))
	}

	// Environment
	for k, v := range svc.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Volumes
	for _, vol := range svc.Volumes {
		args = append(args, "-v", vol)
	}

	args = append(args, svc.Image)

	// If there's also a command on a container service, append it
	if svc.Command != "" {
		args = append(args, "sh", "-c", svc.Command)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return RunHandle{}, fmt.Errorf("docker run %s: %s: %w", containerName, stderr.String(), err)
	}

	containerID := strings.TrimSpace(string(output))
	_ = containerID

	handle := RunHandle{ID: name, Type: "docker"}
	r.containers[name] = containerName
	return handle, nil
}

func (r *DockerRunner) Stop(handle RunHandle) error {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()

	if !ok {
		return nil
	}

	cmd := exec.Command("docker", "stop", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stopping container %s: %w", containerName, err)
	}

	cmd = exec.Command("docker", "rm", containerName)
	cmd.Run() // best effort

	return nil
}

func (r *DockerRunner) Status(handle RunHandle) (ServiceStatus, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()

	if !ok {
		return StatusStopped, nil
	}

	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return StatusStopped, nil
	}

	state := strings.TrimSpace(string(output))
	switch state {
	case "running":
		return StatusRunning, nil
	case "exited":
		// Check exit code
		cmd = exec.Command("docker", "inspect", "-f", "{{.State.ExitCode}}", containerName)
		out, _ := cmd.Output()
		if strings.TrimSpace(string(out)) != "0" {
			return StatusCrashed, nil
		}
		return StatusStopped, nil
	default:
		return StatusStopped, nil
	}
}

func (r *DockerRunner) Logs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("no container for %s", handle.ID)
	}

	cmd := exec.Command("docker", "logs", containerName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("getting logs for %s: %w", containerName, err)
	}

	return io.NopCloser(bytes.NewReader(output)), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/runner/... -timeout 30s`
Expected: PASS (docker tests may skip if Docker unavailable)

- [ ] **Step 5: Commit**

```bash
git add internal/runner/docker.go internal/runner/docker_test.go
git commit -m "feat: add Docker container runner"
```

---

## Chunk 5: Orchestrator

### Task 11: Dependency Graph + Topological Sort

**Files:**
- Create: `internal/orchestrator/graph.go`
- Create: `internal/orchestrator/graph_test.go`

- [ ] **Step 1: Write graph tests**

```go
// internal/orchestrator/graph_test.go
package orchestrator_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestTopoSort_Simple(t *testing.T) {
	services := map[string]workspace.Service{
		"postgres": {},
		"app":      {DependsOn: []string{"postgres"}},
	}

	order, err := orchestrator.TopoSort(services, []string{"postgres", "app"})
	if err != nil {
		t.Fatal(err)
	}

	pgIdx, appIdx := -1, -1
	for i, name := range order {
		if name == "postgres" { pgIdx = i }
		if name == "app" { appIdx = i }
	}
	if pgIdx > appIdx {
		t.Errorf("postgres (idx %d) should come before app (idx %d)", pgIdx, appIdx)
	}
}

func TestTopoSort_CircularDependency(t *testing.T) {
	services := map[string]workspace.Service{
		"a": {DependsOn: []string{"b"}},
		"b": {DependsOn: []string{"a"}},
	}

	_, err := orchestrator.TopoSort(services, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for circular dependency")
	}
}

func TestTopoSort_Diamond(t *testing.T) {
	services := map[string]workspace.Service{
		"db":    {},
		"cache": {},
		"api":   {DependsOn: []string{"db", "cache"}},
		"web":   {DependsOn: []string{"api"}},
	}

	order, err := orchestrator.TopoSort(services, []string{"db", "cache", "api", "web"})
	if err != nil {
		t.Fatal(err)
	}

	indexOf := func(name string) int {
		for i, n := range order {
			if n == name { return i }
		}
		return -1
	}

	if indexOf("db") > indexOf("api") {
		t.Error("db should come before api")
	}
	if indexOf("cache") > indexOf("api") {
		t.Error("cache should come before api")
	}
	if indexOf("api") > indexOf("web") {
		t.Error("api should come before web")
	}
}

func TestTopoSort_NoDeps(t *testing.T) {
	services := map[string]workspace.Service{
		"a": {}, "b": {}, "c": {},
	}
	order, err := orchestrator.TopoSort(services, []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Errorf("expected 3, got %d", len(order))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/orchestrator/...`
Expected: FAIL

- [ ] **Step 3: Implement topological sort**

```go
// internal/orchestrator/graph.go
package orchestrator

import (
	"fmt"

	"github.com/andybarilla/rook/internal/workspace"
)

// TopoSort returns services in dependency order (dependencies first).
func TopoSort(services map[string]workspace.Service, targets []string) ([]string, error) {
	targetSet := make(map[string]bool, len(targets))
	for _, t := range targets {
		targetSet[t] = true
	}

	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)

	state := make(map[string]int)
	var order []string

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("circular dependency detected involving %q", name)
		}

		state[name] = visiting

		svc, ok := services[name]
		if !ok {
			return fmt.Errorf("unknown service: %q", name)
		}

		for _, dep := range svc.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}

		state[name] = visited
		order = append(order, name)
		return nil
	}

	for _, name := range targets {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/orchestrator/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/
git commit -m "feat: add topological sort for service dependency ordering"
```

---

### Task 12: Orchestrator

**Files:**
- Create: `internal/orchestrator/orchestrator.go`
- Create: `internal/orchestrator/orchestrator_test.go`

- [ ] **Step 1: Write orchestrator tests using mock runner**

```go
// internal/orchestrator/orchestrator_test.go
package orchestrator_test

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

type mockRunner struct {
	mu       sync.Mutex
	started  []string
	stopped  []string
}

func (m *mockRunner) Start(ctx context.Context, name string, svc workspace.Service, ports runner.PortMap, workDir string) (runner.RunHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = append(m.started, name)
	return runner.RunHandle{ID: name, Type: "mock"}, nil
}

func (m *mockRunner) Stop(handle runner.RunHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = append(m.stopped, handle.ID)
	return nil
}

func (m *mockRunner) Status(handle runner.RunHandle) (runner.ServiceStatus, error) {
	return runner.StatusRunning, nil
}

func (m *mockRunner) Logs(handle runner.RunHandle) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("mock logs")), nil
}

func TestOrchestrator_Up(t *testing.T) {
	mock := &mockRunner{}

	ws := workspace.Workspace{
		Name: "test",
		Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air", DependsOn: []string{"postgres"}},
		},
		Profiles: map[string][]string{
			"default": {"postgres", "app"},
		},
	}

	orch := orchestrator.New(mock, mock, nil)
	err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}

	if len(mock.started) != 2 {
		t.Fatalf("expected 2 started, got %d: %v", len(mock.started), mock.started)
	}

	// postgres should start before app
	if mock.started[0] != "postgres" {
		t.Errorf("expected postgres first, got %s", mock.started[0])
	}
}

func TestOrchestrator_Up_WithPorts(t *testing.T) {
	mock := &mockRunner{}

	dir := t.TempDir()
	alloc, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10100)

	ws := workspace.Workspace{
		Name: "test",
		Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"app": {Command: "air", Ports: []int{8080}},
		},
		Profiles: map[string][]string{
			"default": {"app"},
		},
	}

	orch := orchestrator.New(mock, mock, alloc)
	err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}

	result := alloc.Get("test", "app")
	if !result.OK {
		t.Fatal("expected port to be allocated for app")
	}
	if result.Port < 10000 || result.Port > 10100 {
		t.Errorf("port %d outside expected range", result.Port)
	}
}

func TestOrchestrator_IncrementalProfileSwitch(t *testing.T) {
	mock := &mockRunner{}

	ws := workspace.Workspace{
		Name: "test",
		Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"redis":    {Image: "redis:7"},
			"app":      {Command: "air"},
		},
		Profiles: map[string][]string{
			"backend":  {"postgres", "app"},
			"full":     {"postgres", "redis", "app"},
		},
	}

	orch := orchestrator.New(mock, mock, nil)

	// Start backend profile
	orch.Up(context.Background(), ws, "backend")
	if len(mock.started) != 2 {
		t.Fatalf("expected 2 started, got %d", len(mock.started))
	}

	// Switch to full — postgres and app should stay, redis should start
	mock.started = nil // reset to track new starts
	orch.Up(context.Background(), ws, "full")
	if len(mock.started) != 1 || mock.started[0] != "redis" {
		t.Errorf("expected only redis to start, got %v", mock.started)
	}
}

func TestOrchestrator_Down(t *testing.T) {
	mock := &mockRunner{}

	ws := workspace.Workspace{
		Name: "test",
		Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air"},
		},
		Profiles: map[string][]string{
			"default": {"postgres", "app"},
		},
	}

	orch := orchestrator.New(mock, mock, nil)
	orch.Up(context.Background(), ws, "default")
	err := orch.Down(context.Background(), ws)
	if err != nil {
		t.Fatal(err)
	}

	if len(mock.stopped) != 2 {
		t.Fatalf("expected 2 stopped, got %d", len(mock.stopped))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/orchestrator/...`
Expected: FAIL

- [ ] **Step 3: Implement orchestrator**

```go
// internal/orchestrator/orchestrator.go
package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/profile"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

type Orchestrator struct {
	mu             sync.Mutex
	containerRunner runner.Runner
	processRunner   runner.Runner
	portAllocator   ports.PortAllocator
	handles        map[string]map[string]runner.RunHandle // workspace -> service -> handle
}

func New(containerRunner, processRunner runner.Runner, portAllocator ports.PortAllocator) *Orchestrator {
	return &Orchestrator{
		containerRunner: containerRunner,
		processRunner:   processRunner,
		portAllocator:   portAllocator,
		handles:        make(map[string]map[string]runner.RunHandle),
	}
}

func (o *Orchestrator) Up(ctx context.Context, ws workspace.Workspace, profileName string) error {
	services, err := profile.Resolve(ws, profileName)
	if err != nil {
		return fmt.Errorf("resolving profile: %w", err)
	}

	order, err := TopoSort(ws.Services, services)
	if err != nil {
		return fmt.Errorf("sorting dependencies: %w", err)
	}

	// Build port map
	portMap := make(runner.PortMap)
	if o.portAllocator != nil {
		for _, name := range order {
			svc := ws.Services[name]
			if len(svc.Ports) > 0 {
				preferred := svc.Ports[0]
				if svc.PinPort > 0 {
					port, err := o.portAllocator.AllocatePinned(ws.Name, name, svc.PinPort)
					if err != nil {
						return fmt.Errorf("pinning port for %s: %w", name, err)
					}
					portMap[name] = port
				} else {
					port, err := o.portAllocator.Allocate(ws.Name, name, preferred)
					if err != nil {
						return fmt.Errorf("allocating port for %s: %w", name, err)
					}
					portMap[name] = port
				}
			}
		}
	}

	// Incremental profile switching: compute diff against currently running services
	o.mu.Lock()
	if o.handles[ws.Name] == nil {
		o.handles[ws.Name] = make(map[string]runner.RunHandle)
	}
	// Snapshot the current handles to avoid data races during iteration
	currentHandles := make(map[string]runner.RunHandle, len(o.handles[ws.Name]))
	for k, v := range o.handles[ws.Name] {
		currentHandles[k] = v
	}
	o.mu.Unlock()

	desiredSet := make(map[string]bool, len(order))
	for _, name := range order {
		desiredSet[name] = true
	}

	// Stop services that are running but not in the new profile
	for name, handle := range currentHandles {
		if !desiredSet[name] {
			var r runner.Runner
			if handle.Type == "process" {
				r = o.processRunner
			} else {
				r = o.containerRunner
			}
			r.Stop(handle)
			o.mu.Lock()
			delete(o.handles[ws.Name], name)
			o.mu.Unlock()
		}
	}

	// Start services that are in the new profile but not already running
	for _, name := range order {
		o.mu.Lock()
		_, alreadyRunning := o.handles[ws.Name][name]
		o.mu.Unlock()

		if alreadyRunning {
			continue
		}

		svc := ws.Services[name]

		var r runner.Runner
		if svc.IsContainer() {
			r = o.containerRunner
		} else {
			r = o.processRunner
		}

		handle, err := r.Start(ctx, name, svc, portMap, ws.Root)
		if err != nil {
			return fmt.Errorf("starting %s: %w", name, err)
		}

		o.mu.Lock()
		o.handles[ws.Name][name] = handle
		o.mu.Unlock()
	}

	return nil
}

func (o *Orchestrator) Down(ctx context.Context, ws workspace.Workspace) error {
	o.mu.Lock()
	handles, ok := o.handles[ws.Name]
	o.mu.Unlock()

	if !ok {
		return nil
	}

	var errs []error
	for name, handle := range handles {
		var r runner.Runner
		if handle.Type == "process" {
			r = o.processRunner
		} else {
			r = o.containerRunner
		}

		if err := r.Stop(handle); err != nil {
			errs = append(errs, fmt.Errorf("stopping %s: %w", name, err))
		}
	}

	o.mu.Lock()
	delete(o.handles, ws.Name)
	o.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping services: %v", errs)
	}
	return nil
}

func (o *Orchestrator) Status(ws workspace.Workspace) (map[string]runner.ServiceStatus, error) {
	o.mu.Lock()
	handles, ok := o.handles[ws.Name]
	o.mu.Unlock()

	result := make(map[string]runner.ServiceStatus)
	if !ok {
		for name := range ws.Services {
			result[name] = runner.StatusStopped
		}
		return result, nil
	}

	for name, handle := range handles {
		var r runner.Runner
		if handle.Type == "process" {
			r = o.processRunner
		} else {
			r = o.containerRunner
		}
		status, _ := r.Status(handle)
		result[name] = status
	}

	for name := range ws.Services {
		if _, ok := result[name]; !ok {
			result[name] = runner.StatusStopped
		}
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/orchestrator/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/orchestrator/orchestrator.go internal/orchestrator/orchestrator_test.go
git commit -m "feat: add orchestrator with dependency-ordered startup and shutdown"
```

---

## Chunk 6: CLI Commands

### Task 13: Root Command + Init

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/init.go`
- Modify: `cmd/rook/main.go`

- [ ] **Step 1: Add cobra dependency**

```bash
go get github.com/spf13/cobra
```

- [ ] **Step 2: Implement root command**

```go
// internal/cli/root.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rook",
		Short: "Local development workspace manager",
	}

	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	cmd.AddCommand(
		newInitCmd(),
		newDiscoverCmd(),
		newUpCmd(),
		newDownCmd(),
		newRestartCmd(),
		newStatusCmd(),
		newListCmd(),
		newPortsCmd(),
		newLogsCmd(),
		newEnvCmd(),
	)

	return cmd
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.ExpandEnv("$HOME/.config")
	}
	return dir + "/rook"
}
```

- [ ] **Step 3: Implement init command**

```go
// internal/cli/init.go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <path>",
		Short: "Initialize a workspace from a project directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			manifestPath := filepath.Join(dir, "rook.yaml")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				return fmt.Errorf("no rook.yaml found in %s (auto-discovery not yet implemented — create rook.yaml manually)", dir)
			}

			m, err := workspace.ParseManifest(manifestPath)
			if err != nil {
				return err
			}

			cfgDir := configDir()
			os.MkdirAll(cfgDir, 0755)

			reg, err := registry.NewFileRegistry(filepath.Join(cfgDir, "workspaces.json"))
			if err != nil {
				return err
			}

			if err := reg.Register(m.Name, dir); err != nil {
				return err
			}

			alloc, err := ports.NewFileAllocator(
				filepath.Join(cfgDir, "ports.json"), 10000, 60000)
			if err != nil {
				return err
			}

			for name, svc := range m.Services {
				if svc.PinPort > 0 {
					allocated, err := alloc.AllocatePinned(m.Name, name, svc.PinPort)
					if err != nil {
						return fmt.Errorf("pinning port for %s: %w", name, err)
					}
					fmt.Printf("  %s.%s -> :%d (pinned)\n", m.Name, name, allocated)
				} else {
					for _, port := range svc.Ports {
						allocated, err := alloc.Allocate(m.Name, name, port)
						if err != nil {
							return fmt.Errorf("allocating port for %s: %w", name, err)
						}
						fmt.Printf("  %s.%s -> :%d\n", m.Name, name, allocated)
					}
				}
			}

			fmt.Printf("Workspace %q registered from %s\n", m.Name, dir)
			return nil
		},
	}
}
```

- [ ] **Step 4: Implement list, ports, status, up, down commands**

```go
// internal/cli/list.go
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/registry"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := registry.NewFileRegistry(filepath.Join(configDir(), "workspaces.json"))
			if err != nil {
				return err
			}

			entries := reg.List()
			if jsonOutput {
				printJSON(entries)
				return nil
			}

			if len(entries) == 0 {
				fmt.Println("No workspaces registered. Run 'rook init <path>' to add one.")
				return nil
			}

			for _, e := range entries {
				fmt.Printf("%-20s %s\n", e.Name, e.Path)
			}
			return nil
		},
	}
}
```

```go
// internal/cli/ports.go
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/ports"
	"github.com/spf13/cobra"
)

func newPortsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ports",
		Short: "Show global port allocation table",
		RunE: func(cmd *cobra.Command, args []string) error {
			alloc, err := ports.NewFileAllocator(
				filepath.Join(configDir(), "ports.json"), 10000, 60000)
			if err != nil {
				return err
			}

			all := alloc.All()
			if jsonOutput {
				printJSON(all)
				return nil
			}

			if len(all) == 0 {
				fmt.Println("No ports allocated.")
				return nil
			}

			fmt.Printf("%-20s %-20s %s\n", "WORKSPACE", "SERVICE", "PORT")
			for _, e := range all {
				pinned := ""
				if e.Pinned {
					pinned = " (pinned)"
				}
				fmt.Printf("%-20s %-20s %d%s\n", e.Workspace, e.Service, e.Port, pinned)
			}
			return nil
		},
	}
}
```

```go
// internal/cli/status.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show all workspaces and running services",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Status display not yet implemented (requires running daemon)")
			return nil
		},
	}
}
```

```go
// internal/cli/up.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up [workspace] [profile]",
		Short: "Start services",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Up command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
```

```go
// internal/cli/down.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down [workspace]",
		Short: "Stop all services in workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Down command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
```

```go
// internal/cli/restart.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart [workspace] [service]",
		Short: "Restart a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Restart command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
```

```go
// internal/cli/logs.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <workspace> [service]",
		Short: "Tail logs (all or specific service)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Logs command not yet wired to orchestrator (requires running daemon)")
			return nil
		},
	}
}
```

```go
// internal/cli/env.go
package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env <workspace>",
		Short: "Print generated environment variables",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, err := registry.NewFileRegistry(filepath.Join(configDir(), "workspaces.json"))
			if err != nil {
				return err
			}

			entry, err := reg.Get(args[0])
			if err != nil {
				return err
			}

			m, err := workspace.ParseManifest(filepath.Join(entry.Path, "rook.yaml"))
			if err != nil {
				return err
			}

			alloc, err := ports.NewFileAllocator(filepath.Join(configDir(), "ports.json"), 10000, 60000)
			if err != nil {
				return err
			}

			portMap := make(map[string]int)
			for name := range m.Services {
				if result := alloc.Get(m.Name, name); result.OK {
					portMap[name] = result.Port
				}
			}

			for name, svc := range m.Services {
				resolved, err := envgen.ResolveTemplates(svc.Environment, portMap, false)
				if err != nil {
					return err
				}
				for k, v := range resolved {
					fmt.Printf("%s.%s: %s=%s\n", m.Name, name, k, v)
				}
			}
			return nil
		},
	}
}
```

```go
// internal/cli/discover.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover <workspace>",
		Short: "Re-scan workspace and show changes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Discover command not yet fully implemented")
			return nil
		},
	}
}
```

- [ ] **Step 5: Update main.go**

```go
// cmd/rook/main.go
package main

import (
	"os"

	"github.com/andybarilla/rook/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Verify it builds and basic commands work**

Run: `go build ./cmd/rook && ./rook --help && ./rook list && ./rook ports`
Expected: Help text, "No workspaces registered", "No ports allocated"

- [ ] **Step 7: Commit**

```bash
git add cmd/rook/main.go internal/cli/ go.mod go.sum
git commit -m "feat: add CLI with init, list, ports, status, up, down commands"
```

---

## Chunk 7: Auto-Discovery

### Task 14: Discoverer Interface + Docker Compose Discoverer

**Files:**
- Create: `internal/discovery/discovery.go`
- Create: `internal/discovery/compose.go`
- Create: `internal/discovery/compose_test.go`

- [ ] **Step 1: Write compose discoverer tests**

```go
// internal/discovery/compose_test.go
package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/discovery"
)

func TestComposeDiscoverer_Detect(t *testing.T) {
	dir := t.TempDir()

	d := discovery.NewComposeDiscoverer()

	if d.Detect(dir) {
		t.Error("should not detect without docker-compose.yml")
	}

	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)
	if !d.Detect(dir) {
		t.Error("should detect with docker-compose.yml")
	}
}

func TestComposeDiscoverer_Discover(t *testing.T) {
	dir := t.TempDir()
	compose := `
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
    volumes:
      - pg-data:/var/lib/postgresql/data
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	d := discovery.NewComposeDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(result.Services))
	}

	pg, ok := result.Services["postgres"]
	if !ok {
		t.Fatal("expected postgres service")
	}
	if pg.Image != "postgres:16-alpine" {
		t.Errorf("expected image postgres:16-alpine, got %s", pg.Image)
	}
	if len(pg.Ports) == 0 || pg.Ports[0] != 5432 {
		t.Errorf("expected port 5432, got %v", pg.Ports)
	}
}

func TestComposeDiscoverer_DependsOn(t *testing.T) {
	dir := t.TempDir()
	compose := `
services:
  postgres:
    image: postgres:16
  app:
    build: .
    depends_on:
      - postgres
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)

	d := discovery.NewComposeDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}

	app := result.Services["app"]
	if len(app.DependsOn) != 1 || app.DependsOn[0] != "postgres" {
		t.Errorf("expected depends_on [postgres], got %v", app.DependsOn)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/...`
Expected: FAIL

- [ ] **Step 3: Implement discoverer interface and compose discoverer**

```go
// internal/discovery/discovery.go
package discovery

import "github.com/andybarilla/rook/internal/workspace"

// DiscoveryResult holds what a discoverer found.
type DiscoveryResult struct {
	Source   string
	Services map[string]workspace.Service
	Groups   map[string][]string
}

// Discoverer extracts workspace config from a project directory.
type Discoverer interface {
	Name() string
	Detect(dir string) bool
	Discover(dir string) (*DiscoveryResult, error)
}

// RunAll runs all discoverers and merges results.
func RunAll(dir string, discoverers []Discoverer) (*DiscoveryResult, error) {
	merged := &DiscoveryResult{
		Services: make(map[string]workspace.Service),
		Groups:   make(map[string][]string),
	}

	for _, d := range discoverers {
		if !d.Detect(dir) {
			continue
		}
		result, err := d.Discover(dir)
		if err != nil {
			return nil, err
		}
		for name, svc := range result.Services {
			merged.Services[name] = svc
		}
		for name, group := range result.Groups {
			merged.Groups[name] = group
		}
		if merged.Source == "" {
			merged.Source = d.Name()
		} else {
			merged.Source += ", " + d.Name()
		}
	}

	return merged, nil
}
```

```go
// internal/discovery/compose.go
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andybarilla/rook/internal/workspace"
	"gopkg.in/yaml.v3"
)

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string            `yaml:"image"`
	Build       any               `yaml:"build"`
	Ports       []string          `yaml:"ports"`
	Environment any               `yaml:"environment"`
	Volumes     []string          `yaml:"volumes"`
	DependsOn   any               `yaml:"depends_on"`
	Healthcheck *composeHealth    `yaml:"healthcheck"`
	Command     any               `yaml:"command"`
}

type composeHealth struct {
	Test     any    `yaml:"test"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}

type ComposeDiscoverer struct{}

func NewComposeDiscoverer() *ComposeDiscoverer {
	return &ComposeDiscoverer{}
}

func (d *ComposeDiscoverer) Name() string { return "docker-compose" }

func (d *ComposeDiscoverer) Detect(dir string) bool {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func (d *ComposeDiscoverer) Discover(dir string) (*DiscoveryResult, error) {
	var path string
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		return nil, fmt.Errorf("no compose file found in %s", dir)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}

	result := &DiscoveryResult{
		Source:   "docker-compose",
		Services: make(map[string]workspace.Service),
	}

	for name, cs := range cf.Services {
		svc := workspace.Service{
			Image:   cs.Image,
			Volumes: cs.Volumes,
		}

		// Parse ports — extract container port from "host:container" format
		for _, p := range cs.Ports {
			parts := strings.Split(p, ":")
			portStr := parts[len(parts)-1]
			// Strip protocol suffix like /tcp
			portStr = strings.Split(portStr, "/")[0]
			port, err := strconv.Atoi(portStr)
			if err == nil {
				svc.Ports = append(svc.Ports, port)
			}
		}

		// Parse environment
		svc.Environment = parseEnvironment(cs.Environment)

		// Parse depends_on
		svc.DependsOn = parseDependsOn(cs.DependsOn)

		result.Services[name] = svc
	}

	return result, nil
}

func parseEnvironment(env any) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string)

	switch v := env.(type) {
	case map[string]any:
		for k, val := range v {
			result[k] = fmt.Sprintf("%v", val)
		}
	case []any:
		for _, item := range v {
			s := fmt.Sprintf("%v", item)
			parts := strings.SplitN(s, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func parseDependsOn(deps any) []string {
	if deps == nil {
		return nil
	}

	switch v := deps.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case map[string]any:
		result := make([]string, 0, len(v))
		for k := range v {
			result = append(result, k)
		}
		return result
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/
git commit -m "feat: add discoverer interface and docker-compose discoverer"
```

---

### Task 15: Mise Discoverer

**Files:**
- Create: `internal/discovery/mise.go`
- Create: `internal/discovery/mise_test.go`

- [ ] **Step 1: Write mise discoverer tests**

```go
// internal/discovery/mise_test.go
package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/discovery"
)

func TestMiseDiscoverer_Detect(t *testing.T) {
	dir := t.TempDir()
	d := discovery.NewMiseDiscoverer()

	if d.Detect(dir) {
		t.Error("should not detect without mise.toml")
	}

	os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("[tools]"), 0644)
	if !d.Detect(dir) {
		t.Error("should detect with mise.toml")
	}
}

func TestMiseDiscoverer_DetectsToolVersions(t *testing.T) {
	dir := t.TempDir()
	d := discovery.NewMiseDiscoverer()

	os.WriteFile(filepath.Join(dir, ".tool-versions"), []byte("go 1.24"), 0644)
	if !d.Detect(dir) {
		t.Error("should detect with .tool-versions")
	}
}

func TestMiseDiscoverer_Discover(t *testing.T) {
	dir := t.TempDir()
	mise := `
[tools]
go = "1.24"
node = "22"
`
	os.WriteFile(filepath.Join(dir, "mise.toml"), []byte(mise), 0644)

	d := discovery.NewMiseDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Source != "mise" {
		t.Errorf("expected source mise, got %s", result.Source)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/... -run Mise`
Expected: FAIL

- [ ] **Step 3: Implement mise discoverer**

```go
// internal/discovery/mise.go
package discovery

import (
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
)

type MiseDiscoverer struct{}

func NewMiseDiscoverer() *MiseDiscoverer { return &MiseDiscoverer{} }

func (d *MiseDiscoverer) Name() string { return "mise" }

func (d *MiseDiscoverer) Detect(dir string) bool {
	for _, name := range []string{"mise.toml", ".mise.toml", ".tool-versions"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func (d *MiseDiscoverer) Discover(dir string) (*DiscoveryResult, error) {
	// Mise provides runtime version info — not services.
	// The discovery result is informational for the user, not service definitions.
	return &DiscoveryResult{
		Source:   "mise",
		Services: make(map[string]workspace.Service),
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/... -run Mise`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/mise.go internal/discovery/mise_test.go
git commit -m "feat: add mise discoverer for runtime version detection"
```

---

### Task 16: Devcontainer Discoverer

**Files:**
- Create: `internal/discovery/devcontainer.go`
- Create: `internal/discovery/devcontainer_test.go`

- [ ] **Step 1: Write devcontainer discoverer tests**

```go
// internal/discovery/devcontainer_test.go
package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/discovery"
)

func TestDevcontainerDiscoverer_Detect(t *testing.T) {
	dir := t.TempDir()
	d := discovery.NewDevcontainerDiscoverer()

	if d.Detect(dir) {
		t.Error("should not detect without .devcontainer")
	}

	os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0755)
	os.WriteFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"), []byte("{}"), 0644)
	if !d.Detect(dir) {
		t.Error("should detect with .devcontainer/devcontainer.json")
	}
}

func TestDevcontainerDiscoverer_ForwardedPorts(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0755)
	dc := `{
		"forwardPorts": [5432, 6379, 8080]
	}`
	os.WriteFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"), []byte(dc), 0644)

	d := discovery.NewDevcontainerDiscoverer()
	result, err := d.Discover(dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Source != "devcontainer" {
		t.Errorf("expected source devcontainer, got %s", result.Source)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/discovery/... -run Devcontainer`
Expected: FAIL

- [ ] **Step 3: Implement devcontainer discoverer**

```go
// internal/discovery/devcontainer.go
package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
)

type devcontainerJSON struct {
	ForwardPorts []int `json:"forwardPorts"`
}

type DevcontainerDiscoverer struct{}

func NewDevcontainerDiscoverer() *DevcontainerDiscoverer { return &DevcontainerDiscoverer{} }

func (d *DevcontainerDiscoverer) Name() string { return "devcontainer" }

func (d *DevcontainerDiscoverer) Detect(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".devcontainer", "devcontainer.json"))
	return err == nil
}

func (d *DevcontainerDiscoverer) Discover(dir string) (*DiscoveryResult, error) {
	path := filepath.Join(dir, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var dc devcontainerJSON
	if err := json.Unmarshal(data, &dc); err != nil {
		return nil, err
	}

	// Devcontainer provides forwarded port info — informational for rook init
	return &DiscoveryResult{
		Source:   "devcontainer",
		Services: make(map[string]workspace.Service),
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/discovery/... -run Devcontainer`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/discovery/devcontainer.go internal/discovery/devcontainer_test.go
git commit -m "feat: add devcontainer discoverer"
```

---

## Chunk 8: Integration + End-to-End

### Task 17: Wire Init Command with Discovery

**Files:**
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Update init command to run discovery when no rook.yaml exists**

```go
// Replace the existing init command body to add discovery fallback:
// After checking for rook.yaml, if missing, run discoverers and generate one.

// In the RunE function, replace the os.IsNotExist check:
manifestPath := filepath.Join(dir, "rook.yaml")
if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
    // Run auto-discovery
    discoverers := []discovery.Discoverer{
        discovery.NewComposeDiscoverer(),
        discovery.NewDevcontainerDiscoverer(),
        discovery.NewMiseDiscoverer(),
    }

    result, err := discovery.RunAll(dir, discoverers)
    if err != nil {
        return fmt.Errorf("discovery failed: %w", err)
    }

    if len(result.Services) == 0 {
        return fmt.Errorf("no services discovered in %s — create a rook.yaml manually", dir)
    }

    fmt.Printf("Discovered from %s:\n", result.Source)
    for name, svc := range result.Services {
        if svc.IsContainer() {
            fmt.Printf("  %s (container: %s)\n", name, svc.Image)
        } else {
            fmt.Printf("  %s (process)\n", name)
        }
    }

    wsName := filepath.Base(dir)
    m := &workspace.Manifest{
        Name:     wsName,
        Type:     workspace.TypeSingle,
        Services: result.Services,
        Groups:   result.Groups,
    }

    if err := workspace.WriteManifest(manifestPath, m); err != nil {
        return fmt.Errorf("writing manifest: %w", err)
    }
    fmt.Printf("Generated %s\n", manifestPath)
}
```

- [ ] **Step 2: Add discovery import to init.go**

Add `"github.com/andybarilla/rook/internal/discovery"` to imports.

- [ ] **Step 3: Verify build succeeds**

Run: `go build ./cmd/rook`
Expected: builds without errors

- [ ] **Step 4: Commit**

```bash
git add internal/cli/init.go
git commit -m "feat: wire auto-discovery into init command"
```

---

### Task 18: End-to-End Smoke Test

**Files:**
- Create: `test/e2e/init_test.go`

- [ ] **Step 1: Write e2e test for init with existing rook.yaml**

```go
// test/e2e/init_test.go
package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWithManifest(t *testing.T) {
	// Build the binary
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "rook")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/rook")
	build.Dir = findProjectRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	// Create a project dir with rook.yaml
	projectDir := t.TempDir()
	manifest := `
name: test-project
type: single
services:
  postgres:
    image: postgres:16-alpine
    ports: [5432]
  app:
    command: echo hello
    ports: [8080]
    depends_on: [postgres]
`
	os.WriteFile(filepath.Join(projectDir, "rook.yaml"), []byte(manifest), 0644)

	// Set custom config dir so we don't pollute real config
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Run init
	cmd := exec.Command(binPath, "init", projectDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %s\n%s", err, output)
	}

	out := string(output)
	if !strings.Contains(out, "test-project") {
		t.Errorf("expected workspace name in output:\n%s", out)
	}

	// Run list
	cmd = exec.Command(binPath, "list")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %s\n%s", err, output)
	}

	if !strings.Contains(string(output), "test-project") {
		t.Errorf("expected test-project in list output:\n%s", string(output))
	}

	// Run ports
	cmd = exec.Command(binPath, "ports")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ports failed: %s\n%s", err, output)
	}

	if !strings.Contains(string(output), "postgres") {
		t.Errorf("expected postgres in ports output:\n%s", string(output))
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
```

- [ ] **Step 2: Run e2e test**

Run: `go test ./test/e2e/... -timeout 30s -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add test/e2e/
git commit -m "test: add end-to-end smoke test for init, list, ports commands"
```

---

## Summary

| Chunk | Tasks | What it delivers |
|-------|-------|-----------------|
| 1: Foundation | 1-3 | Go project, workspace types, manifest parsing |
| 2: Port Allocator + Registry | 4-5 | Global port management, workspace registration |
| 3: Profile + Envgen | 6-7 | Profile resolution, .env generation |
| 4: Health + Runners | 8-10 | Health checks, process runner, docker runner |
| 5: Orchestrator | 11-12 | Dependency ordering, service lifecycle |
| 6: CLI | 13 | All CLI commands via Cobra |
| 7: Discovery | 14-16 | Compose, mise, devcontainer discoverers |
| 8: Integration | 17-18 | Wired init + e2e smoke test |

**GUI (Wails + React frontend) is intentionally deferred** — it consumes the same core library and should be built once the core is stable and tested. It warrants its own spec-and-plan cycle.
