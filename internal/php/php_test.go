package php_test

import (
	"fmt"
	"testing"

	"github.com/andybarilla/flock/internal/php"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// --- Mock FPMRunner ---

type mockFPMRunner struct {
	startCalls map[string]int
	stopCalls  map[string]int
	running    map[string]bool
	startErr   error
}

func newMockFPMRunner() *mockFPMRunner {
	return &mockFPMRunner{
		startCalls: map[string]int{},
		stopCalls:  map[string]int{},
		running:    map[string]bool{},
	}
}

func (m *mockFPMRunner) StartPool(version string) error {
	m.startCalls[version]++
	if m.startErr != nil {
		return m.startErr
	}
	m.running[version] = true
	return nil
}

func (m *mockFPMRunner) StopPool(version string) error {
	m.stopCalls[version]++
	m.running[version] = false
	return nil
}

func (m *mockFPMRunner) PoolSocket(version string) string {
	return fmt.Sprintf("/tmp/php-fpm-%s.sock", version)
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
	p := php.NewPlugin(newMockFPMRunner())
	if p.ID() != "flock-php" {
		t.Errorf("ID = %q, want flock-php", p.ID())
	}
	if p.Name() != "Flock PHP" {
		t.Errorf("Name = %q, want Flock PHP", p.Name())
	}
}

func TestHandlesPHPSite(t *testing.T) {
	p := php.NewPlugin(newMockFPMRunner())

	phpSite := registry.Site{Path: "/app", Domain: "app.test", PHPVersion: "8.3"}
	if !p.Handles(phpSite) {
		t.Error("expected Handles to return true for PHP site")
	}

	staticSite := registry.Site{Path: "/docs", Domain: "docs.test"}
	if p.Handles(staticSite) {
		t.Error("expected Handles to return false for non-PHP site")
	}
}

func TestStartStartsPoolsForPHPSites(t *testing.T) {
	runner := newMockFPMRunner()
	p := php.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app1", Domain: "app1.test", PHPVersion: "8.3"},
		{Path: "/app2", Domain: "app2.test", PHPVersion: "8.2"},
		{Path: "/app3", Domain: "app3.test", PHPVersion: "8.3"}, // duplicate version
		{Path: "/docs", Domain: "docs.test"},                     // no PHP
	}}
	_ = p.Init(host)
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if runner.startCalls["8.3"] != 1 {
		t.Errorf("startCalls[8.3] = %d, want 1", runner.startCalls["8.3"])
	}
	if runner.startCalls["8.2"] != 1 {
		t.Errorf("startCalls[8.2] = %d, want 1", runner.startCalls["8.2"])
	}
}

func TestStartSkipsNonPHPSites(t *testing.T) {
	runner := newMockFPMRunner()
	p := php.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/docs", Domain: "docs.test"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	if len(runner.startCalls) != 0 {
		t.Errorf("expected no start calls, got %v", runner.startCalls)
	}
}

func TestUpstreamForReturnsSocket(t *testing.T) {
	runner := newMockFPMRunner()
	p := php.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app", Domain: "app.test", PHPVersion: "8.3"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	site := registry.Site{Path: "/app", Domain: "app.test", PHPVersion: "8.3"}
	upstream, err := p.UpstreamFor(site)
	if err != nil {
		t.Fatalf("UpstreamFor: %v", err)
	}
	if upstream != "unix//tmp/php-fpm-8.3.sock" {
		t.Errorf("upstream = %q, want unix//tmp/php-fpm-8.3.sock", upstream)
	}
}

func TestUpstreamForErrorsIfPoolNotRunning(t *testing.T) {
	runner := newMockFPMRunner()
	p := php.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{}}
	_ = p.Init(host)

	site := registry.Site{Path: "/app", Domain: "app.test", PHPVersion: "8.3"}
	_, err := p.UpstreamFor(site)
	if err == nil {
		t.Error("expected error for non-running pool")
	}
}

func TestStopStopsAllPools(t *testing.T) {
	runner := newMockFPMRunner()
	p := php.NewPlugin(runner)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app1", Domain: "app1.test", PHPVersion: "8.3"},
		{Path: "/app2", Domain: "app2.test", PHPVersion: "8.2"},
	}}
	_ = p.Init(host)
	_ = p.Start()

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if runner.stopCalls["8.3"] != 1 {
		t.Errorf("stopCalls[8.3] = %d, want 1", runner.stopCalls["8.3"])
	}
	if runner.stopCalls["8.2"] != 1 {
		t.Errorf("stopCalls[8.2] = %d, want 1", runner.stopCalls["8.2"])
	}
}

func TestServiceStatus(t *testing.T) {
	runner := newMockFPMRunner()
	p := php.NewPlugin(runner)

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Errorf("initial status = %d, want ServiceStopped", p.ServiceStatus())
	}

	host := &mockHost{sites: []registry.Site{
		{Path: "/app", Domain: "app.test", PHPVersion: "8.3"},
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

func TestStartLogsPoolFailure(t *testing.T) {
	runner := newMockFPMRunner()
	runner.startErr = fmt.Errorf("php-fpm not found")
	p := php.NewPlugin(runner)

	logged := false
	host := &loggingHost{
		sites: []registry.Site{
			{Path: "/app", Domain: "app.test", PHPVersion: "8.3"},
		},
		onLog: func(pluginID, msg string, args ...any) {
			logged = true
		},
	}
	_ = p.Init(host)
	// Start should not return error — it logs and continues
	if err := p.Start(); err != nil {
		t.Fatalf("Start should not error on pool failure: %v", err)
	}

	if !logged {
		t.Error("expected pool failure to be logged")
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
