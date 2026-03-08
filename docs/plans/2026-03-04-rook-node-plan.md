# rook-node Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a built-in Node.js runtime plugin that manages per-site `npm start` processes with auto-assigned ports and HTTP reverse proxy upstream.

**Architecture:** Mirror rook-php's pattern — a `NodeRunner` interface injected into a `Plugin` struct implementing `RuntimePlugin` + `ServicePlugin`. Each Node-enabled site gets a process on port 3100+. Caddy reverse-proxies HTTP traffic to it.

**Tech Stack:** Go, standard library `os/exec`, existing plugin/registry interfaces.

---

### Task 1: Add NodeVersion to Site struct

**Files:**
- Modify: `internal/registry/site.go:3-8`
- Modify: `internal/registry/registry_test.go`

**Step 1: Write the failing test**

Add to `internal/registry/registry_test.go`:

```go
func TestNodeVersionPersistence(t *testing.T) {
	dir := realDir(t)
	path := tempFile(t)

	r1 := registry.New(path)
	_ = r1.Add(registry.Site{Path: dir, Domain: "nodeapp.test", NodeVersion: "system"})

	r2 := registry.New(path)
	if err := r2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	sites := r2.List()
	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}
	if sites[0].NodeVersion != "system" {
		t.Errorf("NodeVersion = %q, want %q", sites[0].NodeVersion, "system")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/registry/ -run TestNodeVersionPersistence -v`
Expected: FAIL — `NodeVersion` field doesn't exist on `Site`

**Step 3: Write minimal implementation**

In `internal/registry/site.go`, add `NodeVersion` field to the `Site` struct:

```go
type Site struct {
	Path        string `json:"path"`
	Domain      string `json:"domain"`
	PHPVersion  string `json:"php_version,omitempty"`
	NodeVersion string `json:"node_version,omitempty"`
	TLS         bool   `json:"tls"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/registry/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/registry/site.go internal/registry/registry_test.go
git commit -m "feat(registry): add NodeVersion field to Site struct"
```

---

### Task 2: Node plugin — NodeRunner interface and Plugin struct

**Files:**
- Create: `internal/node/node.go`
- Create: `internal/node/node_test.go`

**Step 1: Write the failing tests**

Create `internal/node/node_test.go`:

```go
package node_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/andybarilla/rook/internal/node"
	"github.com/andybarilla/rook/internal/plugin"
	"github.com/andybarilla/rook/internal/registry"
)

// --- Mock NodeRunner ---

type mockNodeRunner struct {
	startCalls map[string]int    // siteDir -> port
	stopCalls  map[string]int    // siteDir -> count
	running    map[string]bool
	startErr   error
}

func newMockNodeRunner() *mockNodeRunner {
	return &mockNodeRunner{
		startCalls: map[string]int{},
		stopCalls:  map[string]int{},
		running:    map[string]bool{},
	}
}

func (m *mockNodeRunner) StartApp(siteDir string, port int) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.startCalls[siteDir] = port
	m.running[siteDir] = true
	return nil
}

func (m *mockNodeRunner) StopApp(siteDir string) error {
	m.stopCalls[siteDir]++
	m.running[siteDir] = false
	return nil
}

func (m *mockNodeRunner) IsRunning(siteDir string) bool {
	return m.running[siteDir]
}

func (m *mockNodeRunner) AppPort(siteDir string) int {
	return m.startCalls[siteDir]
}

// --- Mock Host ---

type mockHost struct {
	sites []registry.Site
}

func (m *mockHost) Sites() []registry.Site {
	return m.sites
}

func (m *mockHost) GetSite(domain string) (registry.Site, bool) {
	for _, s := range m.sites {
		if s.Domain == domain {
			return s, true
		}
	}
	return registry.Site{}, false
}

func (m *mockHost) Log(pluginID string, msg string, args ...any) {}

// --- Tests ---

func TestPluginIDAndName(t *testing.T) {
	p := node.NewPlugin(newMockNodeRunner())
	if p.ID() != "rook-node" {
		t.Errorf("ID = %q, want rook-node", p.ID())
	}
	if p.Name() != "Rook Node" {
		t.Errorf("Name = %q, want Rook Node", p.Name())
	}
}

func TestHandlesNodeSite(t *testing.T) {
	p := node.NewPlugin(newMockNodeRunner())

	nodeSite := registry.Site{Path: "/app", Domain: "app.test", NodeVersion: "system"}
	if !p.Handles(nodeSite) {
		t.Error("expected Handles to return true for Node site")
	}

	staticSite := registry.Site{Path: "/docs", Domain: "docs.test"}
	if p.Handles(staticSite) {
		t.Error("expected Handles to return false for non-Node site")
	}
}

func TestStartStartsAppsForNodeSites(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app1", Domain: "app1.test", NodeVersion: "system"},
		{Path: "/app2", Domain: "app2.test", NodeVersion: "system"},
		{Path: "/docs", Domain: "docs.test"},                              // no Node
		{Path: "/php", Domain: "php.test", PHPVersion: "8.3"},             // PHP only
	}}
	_ = p.Init(host)
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if len(runner.startCalls) != 2 {
		t.Errorf("expected 2 start calls, got %d", len(runner.startCalls))
	}
	if _, ok := runner.startCalls["/app1"]; !ok {
		t.Error("expected /app1 to be started")
	}
	if _, ok := runner.startCalls["/app2"]; !ok {
		t.Error("expected /app2 to be started")
	}
}

func TestStartSkipsNonNodeSites(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/docs", Domain: "docs.test"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	if len(runner.startCalls) != 0 {
		t.Errorf("expected no start calls, got %v", runner.startCalls)
	}
}

func TestStartAssignsSequentialPorts(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app-b", Domain: "b.test", NodeVersion: "system"},
		{Path: "/app-a", Domain: "a.test", NodeVersion: "system"},
		{Path: "/app-c", Domain: "c.test", NodeVersion: "system"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	// Ports should be assigned sorted by domain
	ports := []int{runner.startCalls["/app-a"], runner.startCalls["/app-b"], runner.startCalls["/app-c"]}
	expected := []int{3100, 3101, 3102}
	sort.Ints(ports) // just in case
	for i, port := range ports {
		if port != expected[i] {
			t.Errorf("port[%d] = %d, want %d", i, port, expected[i])
		}
	}
}

func TestStartLogsAndContinuesOnFailure(t *testing.T) {
	runner := newMockNodeRunner()
	runner.startErr = fmt.Errorf("npm not found")
	p := node.NewPlugin(runner)

	logged := false
	host := &loggingHost{
		sites: []registry.Site{
			{Path: "/app", Domain: "app.test", NodeVersion: "system"},
		},
		onLog: func(pluginID, msg string, args ...any) {
			logged = true
		},
	}
	_ = p.Init(host)

	if err := p.Start(); err != nil {
		t.Fatalf("Start should not error on app failure: %v", err)
	}
	if !logged {
		t.Error("expected app failure to be logged")
	}
}

func TestStopStopsAllApps(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app1", Domain: "app1.test", NodeVersion: "system"},
		{Path: "/app2", Domain: "app2.test", NodeVersion: "system"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if runner.stopCalls["/app1"] != 1 {
		t.Errorf("stopCalls[/app1] = %d, want 1", runner.stopCalls["/app1"])
	}
	if runner.stopCalls["/app2"] != 1 {
		t.Errorf("stopCalls[/app2] = %d, want 1", runner.stopCalls["/app2"])
	}
}

func TestUpstreamForReturnsHTTPUpstream(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app", Domain: "app.test", NodeVersion: "system"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	site := registry.Site{Path: "/app", Domain: "app.test", NodeVersion: "system"}
	upstream, err := p.UpstreamFor(site)
	if err != nil {
		t.Fatalf("UpstreamFor: %v", err)
	}

	port := runner.startCalls["/app"]
	expected := fmt.Sprintf("http://127.0.0.1:%d", port)
	if upstream != expected {
		t.Errorf("upstream = %q, want %q", upstream, expected)
	}
}

func TestUpstreamForErrorsIfAppNotRunning(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{}}
	_ = p.Init(host)

	site := registry.Site{Path: "/app", Domain: "app.test", NodeVersion: "system"}
	_, err := p.UpstreamFor(site)
	if err == nil {
		t.Error("expected error for non-running app")
	}
}

func TestServiceStatus(t *testing.T) {
	runner := newMockNodeRunner()
	p := node.NewPlugin(runner)

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Errorf("initial status = %d, want ServiceStopped", p.ServiceStatus())
	}

	host := &mockHost{sites: []registry.Site{
		{Path: "/app", Domain: "app.test", NodeVersion: "system"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	if p.ServiceStatus() != plugin.ServiceRunning {
		t.Errorf("after start status = %d, want ServiceRunning", p.ServiceStatus())
	}

	_ = p.Stop()

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Errorf("after stop status = %d, want ServiceStopped", p.ServiceStatus())
	}
}

// loggingHost captures log calls
type loggingHost struct {
	sites []registry.Site
	onLog func(pluginID, msg string, args ...any)
}

func (h *loggingHost) Sites() []registry.Site { return h.sites }
func (h *loggingHost) GetSite(domain string) (registry.Site, bool) {
	for _, s := range h.sites {
		if s.Domain == domain {
			return s, true
		}
	}
	return registry.Site{}, false
}
func (h *loggingHost) Log(pluginID string, msg string, args ...any) {
	if h.onLog != nil {
		h.onLog(pluginID, msg, args...)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/node/ -v`
Expected: FAIL — package `node` doesn't exist

**Step 3: Write minimal implementation**

Create `internal/node/node.go`:

```go
package node

import (
	"fmt"
	"sort"

	"github.com/andybarilla/rook/internal/plugin"
	"github.com/andybarilla/rook/internal/registry"
)

type NodeRunner interface {
	StartApp(siteDir string, port int) error
	StopApp(siteDir string) error
	IsRunning(siteDir string) bool
	AppPort(siteDir string) int
}

type Plugin struct {
	runner   NodeRunner
	host     plugin.Host
	basePort int
	portMap  map[string]int  // domain -> port
	apps     map[string]bool // siteDir -> running
	status   plugin.ServiceStatus
}

func NewPlugin(runner NodeRunner) *Plugin {
	return &Plugin{
		runner:   runner,
		basePort: 3100,
		portMap:  map[string]int{},
		apps:     map[string]bool{},
		status:   plugin.ServiceStopped,
	}
}

func (p *Plugin) ID() string   { return "rook-node" }
func (p *Plugin) Name() string { return "Rook Node" }

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	return nil
}

func (p *Plugin) Start() error {
	type nodeSite struct {
		domain  string
		siteDir string
	}

	var sites []nodeSite
	for _, site := range p.host.Sites() {
		if site.NodeVersion != "" {
			sites = append(sites, nodeSite{domain: site.Domain, siteDir: site.Path})
		}
	}

	sort.Slice(sites, func(i, j int) bool {
		return sites[i].domain < sites[j].domain
	})

	for i, s := range sites {
		port := p.basePort + i
		if err := p.runner.StartApp(s.siteDir, port); err != nil {
			p.host.Log(p.ID(), "failed to start app for %s: %v", s.domain, err)
			continue
		}
		p.portMap[s.domain] = port
		p.apps[s.siteDir] = true
	}

	if len(p.apps) > 0 {
		p.status = plugin.ServiceRunning
	}
	return nil
}

func (p *Plugin) Stop() error {
	for siteDir := range p.apps {
		if err := p.runner.StopApp(siteDir); err != nil {
			p.host.Log(p.ID(), "failed to stop app at %s: %v", siteDir, err)
		}
		delete(p.apps, siteDir)
	}
	p.portMap = map[string]int{}
	p.status = plugin.ServiceStopped
	return nil
}

func (p *Plugin) Handles(site registry.Site) bool {
	return site.NodeVersion != ""
}

func (p *Plugin) UpstreamFor(site registry.Site) (string, error) {
	port, ok := p.portMap[site.Domain]
	if !ok {
		return "", fmt.Errorf("no running app for %s", site.Domain)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

func (p *Plugin) ServiceStatus() plugin.ServiceStatus {
	return p.status
}

func (p *Plugin) StartService() error { return p.Start() }
func (p *Plugin) StopService() error  { return p.Stop() }
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/node/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/node/node.go internal/node/node_test.go
git commit -m "feat(node): add Node.js plugin with NodeRunner interface"
```

---

### Task 3: ProcessRunner concrete implementation

**Files:**
- Create: `internal/node/process.go`
- Create: `internal/node/process_test.go`

**Step 1: Write the failing test**

Create `internal/node/process_test.go`:

```go
package node_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/node"
)

func TestProcessRunnerStartAndStop(t *testing.T) {
	// Create a temp dir with a package.json that runs a simple HTTP server
	dir := t.TempDir()
	packageJSON := `{"name":"test","scripts":{"start":"node server.js"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0o644)

	// Simple Node server that listens on PORT env var
	serverJS := `
const http = require('http');
const port = process.env.PORT || 3000;
const server = http.createServer((req, res) => {
	res.writeHead(200);
	res.end('ok');
});
server.listen(port, '127.0.0.1');
`
	os.WriteFile(filepath.Join(dir, "server.js"), []byte(serverJS), 0o644)

	runner := node.NewProcessRunner()
	port := 13100 // use high port to avoid conflicts

	err := runner.StartApp(dir, port)
	if err != nil {
		t.Fatalf("StartApp: %v", err)
	}

	// Give the process a moment to start
	time.Sleep(500 * time.Millisecond)

	if !runner.IsRunning(dir) {
		t.Error("expected app to be running")
	}
	if runner.AppPort(dir) != port {
		t.Errorf("AppPort = %d, want %d", runner.AppPort(dir), port)
	}

	err = runner.StopApp(dir)
	if err != nil {
		t.Fatalf("StopApp: %v", err)
	}

	// Give the process a moment to stop
	time.Sleep(200 * time.Millisecond)

	if runner.IsRunning(dir) {
		t.Error("expected app to be stopped")
	}
}

func TestProcessRunnerStopNonexistent(t *testing.T) {
	runner := node.NewProcessRunner()
	err := runner.StopApp("/no/such/dir")
	if err == nil {
		t.Error("expected error stopping nonexistent app")
	}
}

func TestProcessRunnerIsRunningFalseByDefault(t *testing.T) {
	runner := node.NewProcessRunner()
	if runner.IsRunning("/no/such/dir") {
		t.Error("expected IsRunning to be false for unknown dir")
	}
}

func TestProcessRunnerAppPortZeroByDefault(t *testing.T) {
	runner := node.NewProcessRunner()
	if runner.AppPort("/no/such/dir") != 0 {
		t.Error("expected AppPort to be 0 for unknown dir")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/node/ -run TestProcessRunner -v`
Expected: FAIL — `NewProcessRunner` doesn't exist

**Step 3: Write minimal implementation**

Create `internal/node/process.go`:

```go
package node

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type appEntry struct {
	cmd  *exec.Cmd
	port int
}

type ProcessRunner struct {
	mu   sync.Mutex
	apps map[string]*appEntry // siteDir -> entry
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{
		apps: map[string]*appEntry{},
	}
}

func (r *ProcessRunner) StartApp(siteDir string, port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.Command("npm", "start")
	cmd.Dir = siteDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start npm in %s: %w", siteDir, err)
	}

	r.apps[siteDir] = &appEntry{cmd: cmd, port: port}

	// Goroutine to clean up when process exits
	go func() {
		cmd.Wait()
		r.mu.Lock()
		defer r.mu.Unlock()
		if entry, ok := r.apps[siteDir]; ok && entry.cmd == cmd {
			delete(r.apps, siteDir)
		}
	}()

	return nil
}

func (r *ProcessRunner) StopApp(siteDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.apps[siteDir]
	if !ok {
		return fmt.Errorf("no running app at %s", siteDir)
	}

	if err := entry.cmd.Process.Signal(os.Interrupt); err != nil {
		// If interrupt fails, force kill
		entry.cmd.Process.Kill()
	}

	delete(r.apps, siteDir)
	return nil
}

func (r *ProcessRunner) IsRunning(siteDir string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.apps[siteDir]
	return ok
}

func (r *ProcessRunner) AppPort(siteDir string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.apps[siteDir]
	if !ok {
		return 0
	}
	return entry.port
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/node/ -run TestProcessRunner -v`
Expected: ALL PASS (note: `TestProcessRunnerStartAndStop` requires `node` and `npm` on PATH)

**Step 5: Run all node tests**

Run: `go test ./internal/node/ -v`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/node/process.go internal/node/process_test.go
git commit -m "feat(node): add ProcessRunner using os/exec for npm start"
```

---

### Task 4: Wire node plugin into Core

**Files:**
- Modify: `internal/core/core.go`
- Modify: `internal/core/core_test.go`
- Modify: `app.go`

**Step 1: Write the failing test**

Add to `internal/core/core_test.go`:

First add a `stubNodeRunner`:

```go
type stubNodeRunner struct {
	started map[string]int  // siteDir -> port
	stopped map[string]bool
}

func newStubNodeRunner() *stubNodeRunner {
	return &stubNodeRunner{started: map[string]int{}, stopped: map[string]bool{}}
}

func (s *stubNodeRunner) StartApp(siteDir string, port int) error {
	s.started[siteDir] = port
	return nil
}

func (s *stubNodeRunner) StopApp(siteDir string) error {
	s.stopped[siteDir] = true
	return nil
}

func (s *stubNodeRunner) IsRunning(siteDir string) bool {
	if _, ok := s.started[siteDir]; ok {
		return !s.stopped[siteDir]
	}
	return false
}

func (s *stubNodeRunner) AppPort(siteDir string) int {
	return s.started[siteDir]
}
```

Update `testConfig` to include `NodeRunner`:

```go
func testConfig(t *testing.T) (core.Config, *stubCaddyRunner, *stubFPMRunner, *stubCertStore, *stubDBRunner, *stubNodeRunner) {
	t.Helper()
	runner := &stubCaddyRunner{}
	fpm := newStubFPMRunner()
	certs := newStubCertStore()
	db := newStubDBRunner()
	nodeRunner := newStubNodeRunner()
	dir := t.TempDir()
	cfg := core.Config{
		SitesFile:    tmpSitesFile(t),
		Logger:       log.New(os.Stderr, "", 0),
		CaddyRunner:  runner,
		FPMRunner:    fpm,
		CertStore:    certs,
		DBRunner:     db,
		DBConfigPath: filepath.Join(dir, "databases.json"),
		DBDataRoot:   filepath.Join(dir, "db-data"),
		NodeRunner:   nodeRunner,
	}
	return cfg, runner, fpm, certs, db, nodeRunner
}
```

Add the new test:

```go
func TestNodePluginStartsForNodeSites(t *testing.T) {
	cfg, _, _, _, _, nodeRunner := testConfig(t)

	dir := t.TempDir()
	sitesJSON := fmt.Sprintf(`[{"path":%q,"domain":"nodeapp.test","node_version":"system"}]`, dir)
	os.MkdirAll(filepath.Dir(cfg.SitesFile), 0o755)
	os.WriteFile(cfg.SitesFile, []byte(sitesJSON), 0o644)

	c := core.NewCore(cfg)
	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	if _, ok := nodeRunner.started[dir]; !ok {
		t.Error("expected Node app to be started for nodeapp.test")
	}
}
```

Update all existing test call sites that call `testConfig` to accept the 6th return value (add `_` for the new `stubNodeRunner` where unused).

**Step 2: Run test to verify it fails**

Run: `go test ./internal/core/ -run TestNodePluginStartsForNodeSites -v`
Expected: FAIL — `core.Config` has no `NodeRunner` field

**Step 3: Write minimal implementation**

Modify `internal/core/core.go`:

Add import:
```go
"github.com/andybarilla/rook/internal/node"
```

Add to `Config`:
```go
NodeRunner node.NodeRunner
```

Add `nodePlugin` field to `Core`:
```go
nodePlugin *node.Plugin
```

In `NewCore`, create and register the node plugin after PHP:
```go
nodePlugin := node.NewPlugin(cfg.NodeRunner)
pluginMgr.Register(sslPlugin)
pluginMgr.Register(phpPlugin)
pluginMgr.Register(nodePlugin)
pluginMgr.Register(dbPlugin)
```

Store it:
```go
nodePlugin: nodePlugin,
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/core/ -v`
Expected: ALL PASS

Note: `TestPluginsReturnsInfo` needs updating — it now expects 4 plugins (ssl, php, node, databases) instead of 3. Update the expected count and add `rook-node` check.

**Step 5: Update app.go**

In `app.go`, add the node import and instantiate the process runner:

```go
import "github.com/andybarilla/rook/internal/node"
```

Add to the `cfg` in `startup()`:
```go
NodeRunner: node.NewProcessRunner(),
```

**Step 6: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

**Step 7: Commit**

```bash
git add internal/core/core.go internal/core/core_test.go app.go
git commit -m "feat(core): wire rook-node plugin into core startup"
```

---

### Task 5: Update AddSite to accept NodeVersion

**Files:**
- Modify: `app.go:70-77`

**Step 1: Update AddSite signature**

Change `AddSite` in `app.go` to accept `nodeVersion`:

```go
func (a *App) AddSite(path, domain, phpVersion, nodeVersion string, tls bool) error {
	return a.core.AddSite(registry.Site{
		Path:        path,
		Domain:      domain,
		PHPVersion:  phpVersion,
		NodeVersion: nodeVersion,
		TLS:         tls,
	})
}
```

**Step 2: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add app.go
git commit -m "feat(app): add NodeVersion parameter to AddSite"
```

---

### Task 6: Create task file and update roadmap

**Files:**
- Create: `docs/tasks/011-rook-node.md`
- Modify: `docs/ROADMAP.md`

**Step 1: Create the task file**

Create `docs/tasks/011-rook-node.md` following the `000-sample.md` template, referencing the design doc and listing all implementation steps with acceptance criteria.

**Step 2: Update roadmap**

No roadmap changes yet — the task will be marked complete after the PR is merged.

**Step 3: Commit**

```bash
git add docs/tasks/011-rook-node.md
git commit -m "docs: add rook-node task file"
```
