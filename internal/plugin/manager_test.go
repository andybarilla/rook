package plugin_test

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
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
	id         string
	name       string
	initErr    error
	startErr   error
	stopErr    error
	initCalls  int
	startCalls int
	stopCalls  int
	host       plugin.Host
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

func (p *orderTrackingPlugin) ID() string               { return p.mockPlugin.id }
func (p *orderTrackingPlugin) Name() string             { return p.mockPlugin.name }
func (p *orderTrackingPlugin) Init(h plugin.Host) error { return p.mockPlugin.Init(h) }
func (p *orderTrackingPlugin) Start() error             { return p.mockPlugin.Start() }
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
