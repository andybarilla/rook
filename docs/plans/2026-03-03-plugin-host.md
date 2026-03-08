# Plugin Interfaces + Host Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the plugin system — interfaces for three plugin types, a Host interface for plugin-to-core communication, and a Manager that handles plugin lifecycle and upstream resolution.

**Architecture:** Single `internal/plugin` package. Plugin/RuntimePlugin/ServicePlugin interfaces and status types in `plugin.go`, Manager struct with lifecycle methods and Host implementation in `manager.go`. Manager receives a `SiteSource` interface (satisfied by `registry.Registry`) to decouple from the concrete registry.

**Tech Stack:** Go 1.23, standard library only (log, fmt)

---

## Task 1: Plugin Interfaces + Host

**Files:**
- Create: `internal/plugin/plugin.go`
- Create: `internal/plugin/manager.go`
- Create: `internal/plugin/manager_test.go`

**Step 1: Write failing tests**

Create `internal/plugin/manager_test.go`:

```go
package plugin_test

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/andybarilla/rook/internal/plugin"
	"github.com/andybarilla/rook/internal/registry"
)

// --- Mock SiteSource ---

type mockSiteSource struct {
	sites []registry.Site
}

func (m *mockSiteSource) List() []registry.Site {
	out := make([]registry.Site, len(m.sites))
	copy(out, m.sites)
	return out
}

func (m *mockSiteSource) Get(domain string) (registry.Site, bool) {
	for _, s := range m.sites {
		if s.Domain == domain {
			return s, true
		}
	}
	return registry.Site{}, false
}

// --- Mock Plugin ---

type mockPlugin struct {
	id        string
	name      string
	initErr   error
	startErr  error
	stopErr   error
	initCalls int
	startCalls int
	stopCalls  int
	host      plugin.Host
}

func (p *mockPlugin) ID() string   { return p.id }
func (p *mockPlugin) Name() string { return p.name }
func (p *mockPlugin) Init(h plugin.Host) error {
	p.initCalls++
	p.host = h
	return p.initErr
}
func (p *mockPlugin) Start() error {
	p.startCalls++
	return p.startErr
}
func (p *mockPlugin) Stop() error {
	p.stopCalls++
	return p.stopErr
}

// --- Mock RuntimePlugin ---

type mockRuntimePlugin struct {
	mockPlugin
	handles  bool
	upstream string
	upErr    error
}

func (p *mockRuntimePlugin) Handles(site registry.Site) bool {
	return p.handles
}

func (p *mockRuntimePlugin) UpstreamFor(site registry.Site) (string, error) {
	return p.upstream, p.upErr
}

// --- Tests ---

func newManager(sites []registry.Site) *plugin.Manager {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	src := &mockSiteSource{sites: sites}
	return plugin.NewManager(src, logger)
}

func TestInitAllStartAllSucceeds(t *testing.T) {
	m := newManager(nil)
	p1 := &mockPlugin{id: "a", name: "Alpha"}
	p2 := &mockPlugin{id: "b", name: "Beta"}
	m.Register(p1)
	m.Register(p2)

	m.InitAll()
	m.StartAll()

	if p1.initCalls != 1 || p1.startCalls != 1 {
		t.Errorf("p1: init=%d start=%d, want 1,1", p1.initCalls, p1.startCalls)
	}
	if p2.initCalls != 1 || p2.startCalls != 1 {
		t.Errorf("p2: init=%d start=%d, want 1,1", p2.initCalls, p2.startCalls)
	}

	infos := m.Plugins()
	for _, info := range infos {
		if info.Status != plugin.PluginReady {
			t.Errorf("plugin %q status = %v, want PluginReady", info.ID, info.Status)
		}
	}
}

func TestInitAllMarksDegradedOnError(t *testing.T) {
	m := newManager(nil)
	good := &mockPlugin{id: "good", name: "Good"}
	bad := &mockPlugin{id: "bad", name: "Bad", initErr: errors.New("init failed")}
	after := &mockPlugin{id: "after", name: "After"}
	m.Register(good)
	m.Register(bad)
	m.Register(after)

	m.InitAll()

	infos := m.Plugins()
	if infos[0].Status != plugin.PluginReady {
		t.Errorf("good: status = %v, want PluginReady", infos[0].Status)
	}
	if infos[1].Status != plugin.PluginDegraded {
		t.Errorf("bad: status = %v, want PluginDegraded", infos[1].Status)
	}
	if infos[2].Status != plugin.PluginReady {
		t.Errorf("after: status = %v, want PluginReady", infos[2].Status)
	}
}

func TestStartAllSkipsDegradedAndMarksFailing(t *testing.T) {
	m := newManager(nil)
	degraded := &mockPlugin{id: "degraded", name: "Degraded", initErr: errors.New("init failed")}
	failStart := &mockPlugin{id: "failstart", name: "FailStart", startErr: errors.New("start failed")}
	m.Register(degraded)
	m.Register(failStart)

	m.InitAll()
	m.StartAll()

	if degraded.startCalls != 0 {
		t.Errorf("degraded plugin should not have Start called, got %d", degraded.startCalls)
	}
	infos := m.Plugins()
	if infos[1].Status != plugin.PluginDegraded {
		t.Errorf("failstart: status = %v, want PluginDegraded", infos[1].Status)
	}
}

func TestStopAllReverseOrder(t *testing.T) {
	m := newManager(nil)
	var order []string
	p1 := &mockPlugin{id: "first", name: "First"}
	p2 := &mockPlugin{id: "second", name: "Second"}

	// Override Stop to track order
	origStop1 := p1.Stop
	_ = origStop1
	p1Wrapper := &orderTrackingPlugin{mockPlugin: p1, order: &order}
	p2Wrapper := &orderTrackingPlugin{mockPlugin: p2, order: &order}

	m.Register(p1Wrapper)
	m.Register(p2Wrapper)
	m.InitAll()
	m.StartAll()
	m.StopAll()

	if len(order) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(order))
	}
	if order[0] != "second" || order[1] != "first" {
		t.Errorf("stop order = %v, want [second, first]", order)
	}
}

type orderTrackingPlugin struct {
	mockPlugin *mockPlugin
	order      *[]string
}

func (p *orderTrackingPlugin) ID() string              { return p.mockPlugin.id }
func (p *orderTrackingPlugin) Name() string            { return p.mockPlugin.name }
func (p *orderTrackingPlugin) Init(h plugin.Host) error { return p.mockPlugin.Init(h) }
func (p *orderTrackingPlugin) Start() error            { return p.mockPlugin.Start() }
func (p *orderTrackingPlugin) Stop() error {
	*p.order = append(*p.order, p.mockPlugin.id)
	return p.mockPlugin.Stop()
}

func TestResolveUpstreamFirstMatch(t *testing.T) {
	m := newManager(nil)
	p1 := &mockRuntimePlugin{
		mockPlugin: mockPlugin{id: "skip", name: "Skip"},
		handles:    false,
	}
	p2 := &mockRuntimePlugin{
		mockPlugin: mockPlugin{id: "php", name: "PHP"},
		handles:    true,
		upstream:   "fastcgi://unix//tmp/php-fpm.sock",
	}
	m.Register(p1)
	m.Register(p2)
	m.InitAll()

	site := registry.Site{Domain: "myapp.test", Path: "/tmp", TLS: true}
	upstream, err := m.ResolveUpstream(site)
	if err != nil {
		t.Fatalf("ResolveUpstream: %v", err)
	}
	if upstream != "fastcgi://unix//tmp/php-fpm.sock" {
		t.Errorf("upstream = %q, want fastcgi://unix//tmp/php-fpm.sock", upstream)
	}
}

func TestResolveUpstreamNoMatch(t *testing.T) {
	m := newManager(nil)
	p := &mockRuntimePlugin{
		mockPlugin: mockPlugin{id: "php", name: "PHP"},
		handles:    false,
	}
	m.Register(p)
	m.InitAll()

	site := registry.Site{Domain: "static.test", Path: "/tmp", TLS: false}
	upstream, err := m.ResolveUpstream(site)
	if err != nil {
		t.Fatalf("ResolveUpstream: %v", err)
	}
	if upstream != "" {
		t.Errorf("upstream = %q, want empty string", upstream)
	}
}

func TestPluginsReturnsInfo(t *testing.T) {
	m := newManager(nil)
	m.Register(&mockPlugin{id: "a", name: "Alpha"})
	m.Register(&mockPlugin{id: "b", name: "Beta", initErr: errors.New("fail")})
	m.InitAll()

	infos := m.Plugins()
	if len(infos) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(infos))
	}
	if infos[0].ID != "a" || infos[0].Name != "Alpha" || infos[0].Status != plugin.PluginReady {
		t.Errorf("infos[0] = %+v", infos[0])
	}
	if infos[1].ID != "b" || infos[1].Name != "Beta" || infos[1].Status != plugin.PluginDegraded {
		t.Errorf("infos[1] = %+v", infos[1])
	}
}

func TestHostSitesAndGetSite(t *testing.T) {
	sites := []registry.Site{
		{Path: "/tmp/app", Domain: "app.test", TLS: true},
	}
	m := newManager(sites)
	p := &mockPlugin{id: "test", name: "Test"}
	m.Register(p)
	m.InitAll()

	// Host was passed to plugin via Init
	h := p.host
	if h == nil {
		t.Fatal("host not set on plugin")
	}

	got := h.Sites()
	if len(got) != 1 || got[0].Domain != "app.test" {
		t.Errorf("Sites() = %v, want [app.test]", got)
	}

	site, ok := h.GetSite("app.test")
	if !ok {
		t.Fatal("GetSite: expected to find app.test")
	}
	if site.Domain != "app.test" {
		t.Errorf("GetSite domain = %q, want app.test", site.Domain)
	}

	_, ok = h.GetSite("nope.test")
	if ok {
		t.Fatal("GetSite: expected not found for nope.test")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/plugin/... -v`
Expected: FAIL — package does not exist.

**Step 3: Implement interfaces and types**

Create `internal/plugin/plugin.go`:

```go
package plugin

import (
	"github.com/andybarilla/rook/internal/registry"
)

type Plugin interface {
	ID() string
	Name() string
	Init(host Host) error
	Start() error
	Stop() error
}

type RuntimePlugin interface {
	Plugin
	Handles(site registry.Site) bool
	UpstreamFor(site registry.Site) (string, error)
}

type ServicePlugin interface {
	Plugin
	ServiceStatus() ServiceStatus
	StartService() error
	StopService() error
}

type Host interface {
	Sites() []registry.Site
	GetSite(domain string) (registry.Site, bool)
	Log(pluginID string, msg string, args ...any)
}

type SiteSource interface {
	List() []registry.Site
	Get(domain string) (registry.Site, bool)
}

type ServiceStatus int

const (
	ServiceStopped ServiceStatus = iota
	ServiceRunning
	ServiceDegraded
)

type PluginStatus int

const (
	PluginReady PluginStatus = iota
	PluginDegraded
)

type PluginInfo struct {
	ID     string
	Name   string
	Status PluginStatus
}
```

**Step 4: Implement Manager**

Create `internal/plugin/manager.go`:

```go
package plugin

import (
	"fmt"
	"log"

	"github.com/andybarilla/rook/internal/registry"
)

type pluginEntry struct {
	plugin Plugin
	status PluginStatus
}

type Manager struct {
	registry  SiteSource
	logger    *log.Logger
	plugins   []pluginEntry
}

func NewManager(src SiteSource, logger *log.Logger) *Manager {
	return &Manager{registry: src, logger: logger}
}

func (m *Manager) Register(p Plugin) {
	m.plugins = append(m.plugins, pluginEntry{plugin: p, status: PluginReady})
}

func (m *Manager) InitAll() {
	for i := range m.plugins {
		e := &m.plugins[i]
		if err := e.plugin.Init(m); err != nil {
			m.logger.Printf("plugin %q init failed: %v", e.plugin.ID(), err)
			e.status = PluginDegraded
		}
	}
}

func (m *Manager) StartAll() {
	for i := range m.plugins {
		e := &m.plugins[i]
		if e.status == PluginDegraded {
			continue
		}
		if err := e.plugin.Start(); err != nil {
			m.logger.Printf("plugin %q start failed: %v", e.plugin.ID(), err)
			e.status = PluginDegraded
		}
	}
}

func (m *Manager) StopAll() {
	for i := len(m.plugins) - 1; i >= 0; i-- {
		e := &m.plugins[i]
		if err := e.plugin.Stop(); err != nil {
			m.logger.Printf("plugin %q stop failed: %v", e.plugin.ID(), err)
		}
	}
}

func (m *Manager) ResolveUpstream(site registry.Site) (string, error) {
	for _, e := range m.plugins {
		rp, ok := e.plugin.(RuntimePlugin)
		if !ok {
			continue
		}
		if rp.Handles(site) {
			return rp.UpstreamFor(site)
		}
	}
	return "", nil
}

func (m *Manager) Plugins() []PluginInfo {
	out := make([]PluginInfo, len(m.plugins))
	for i, e := range m.plugins {
		out[i] = PluginInfo{
			ID:     e.plugin.ID(),
			Name:   e.plugin.Name(),
			Status: e.status,
		}
	}
	return out
}

// Host interface implementation

func (m *Manager) Sites() []registry.Site {
	return m.registry.List()
}

func (m *Manager) GetSite(domain string) (registry.Site, bool) {
	return m.registry.Get(domain)
}

func (m *Manager) Log(pluginID string, msg string, args ...any) {
	m.logger.Printf("[%s] %s", pluginID, fmt.Sprintf(msg, args...))
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/plugin/... -v`
Expected: PASS — all 8 tests pass.

**Step 6: Run full test suite**

Run: `go test ./internal/... -v`
Expected: PASS — all tests (config + registry + plugin) pass.

**Step 7: Commit**

```bash
git add internal/plugin/
git commit -m "feat: add plugin interfaces and host manager with lifecycle and upstream resolution"
```
