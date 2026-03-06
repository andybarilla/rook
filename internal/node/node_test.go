package node_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/andybarilla/flock/internal/node"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// --- Mock NodeRunner ---

type mockNodeRunner struct {
	startCalls map[string]int // siteDir -> port
	stopCalls  map[string]int // siteDir -> count
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
	if p.ID() != "flock-node" {
		t.Errorf("ID = %q, want flock-node", p.ID())
	}
	if p.Name() != "Flock Node" {
		t.Errorf("Name = %q, want Flock Node", p.Name())
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
		{Path: "/docs", Domain: "docs.test"},                  // no Node
		{Path: "/php", Domain: "php.test", PHPVersion: "8.3"}, // PHP only
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
