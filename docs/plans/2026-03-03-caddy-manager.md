# Caddy Manager Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an embedded Caddy manager that generates JSON config from sites + plugin upstreams and manages Caddy lifecycle with hot-reload support.

**Architecture:** Single `internal/caddy` package. `CaddyRunner` and `UpstreamResolver` interfaces for testability. `BuildConfig` function generates Caddy JSON from site list. `Manager` struct orchestrates lifecycle via Start/Reload/Stop.

**Tech Stack:** Go 1.23, `encoding/json` for config generation. No Caddy dependency in this task — the `CaddyRunner` interface abstracts it away. Real Caddy wiring happens in the core wiring task.

---

## Task 1: Caddy Manager

**Files:**
- Create: `internal/caddy/caddy.go`
- Create: `internal/caddy/caddy_test.go`

**Step 1: Write failing tests**

Create `internal/caddy/caddy_test.go`:

```go
package caddy_test

import (
	"encoding/json"
	"testing"

	flockCaddy "github.com/andybarilla/flock/internal/caddy"
	"github.com/andybarilla/flock/internal/registry"
)

// --- Mock CaddyRunner ---

type mockRunner struct {
	runCalls  int
	stopCalls int
	lastCfg   []byte
	runErr    error
	stopErr   error
}

func (m *mockRunner) Run(cfgJSON []byte) error {
	m.runCalls++
	m.lastCfg = cfgJSON
	return m.runErr
}

func (m *mockRunner) Stop() error {
	m.stopCalls++
	return m.stopErr
}

// --- Mock UpstreamResolver ---

type mockResolver struct {
	upstreams map[string]string
}

func (m *mockResolver) ResolveUpstream(site registry.Site) (string, error) {
	return m.upstreams[site.Domain], nil
}

// --- Helper to dig into generated JSON ---

func parseConfig(t *testing.T, cfgJSON []byte) map[string]any {
	t.Helper()
	var cfg map[string]any
	if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return cfg
}

func getRoutes(t *testing.T, cfg map[string]any) []any {
	t.Helper()
	apps := cfg["apps"].(map[string]any)
	http := apps["http"].(map[string]any)
	servers := http["servers"].(map[string]any)
	flock := servers["flock"].(map[string]any)
	return flock["routes"].([]any)
}

func getRouteHandler(t *testing.T, route any) map[string]any {
	t.Helper()
	r := route.(map[string]any)
	handles := r["handle"].([]any)
	return handles[0].(map[string]any)
}

func getRouteHost(t *testing.T, route any) string {
	t.Helper()
	r := route.(map[string]any)
	matches := r["match"].([]any)
	match := matches[0].(map[string]any)
	hosts := match["host"].([]any)
	return hosts[0].(string)
}

// --- Tests ---

func TestBuildConfigStaticSite(t *testing.T) {
	resolver := &mockResolver{upstreams: map[string]string{}}
	sites := []registry.Site{
		{Path: "/home/user/static", Domain: "static.test", TLS: false},
	}

	cfgJSON, err := flockCaddy.BuildConfig(sites, resolver)
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}

	cfg := parseConfig(t, cfgJSON)
	routes := getRoutes(t, cfg)
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	handler := getRouteHandler(t, routes[0])
	if handler["handler"] != "file_server" {
		t.Errorf("handler = %q, want file_server", handler["handler"])
	}
	if handler["root"] != "/home/user/static" {
		t.Errorf("root = %q, want /home/user/static", handler["root"])
	}

	host := getRouteHost(t, routes[0])
	if host != "static.test" {
		t.Errorf("host = %q, want static.test", host)
	}
}

func TestBuildConfigProxiedSite(t *testing.T) {
	resolver := &mockResolver{upstreams: map[string]string{
		"myapp.test": "unix//tmp/php-fpm.sock",
	}}
	sites := []registry.Site{
		{Path: "/home/user/myapp", Domain: "myapp.test", TLS: true},
	}

	cfgJSON, err := flockCaddy.BuildConfig(sites, resolver)
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}

	cfg := parseConfig(t, cfgJSON)
	routes := getRoutes(t, cfg)
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	handler := getRouteHandler(t, routes[0])
	if handler["handler"] != "reverse_proxy" {
		t.Errorf("handler = %q, want reverse_proxy", handler["handler"])
	}
	upstreams := handler["upstreams"].([]any)
	upstream := upstreams[0].(map[string]any)
	if upstream["dial"] != "unix//tmp/php-fpm.sock" {
		t.Errorf("dial = %q, want unix//tmp/php-fpm.sock", upstream["dial"])
	}
}

func TestBuildConfigMixedSites(t *testing.T) {
	resolver := &mockResolver{upstreams: map[string]string{
		"app.test": "localhost:9000",
	}}
	sites := []registry.Site{
		{Path: "/home/user/app", Domain: "app.test", TLS: true},
		{Path: "/home/user/docs", Domain: "docs.test", TLS: false},
	}

	cfgJSON, err := flockCaddy.BuildConfig(sites, resolver)
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}

	cfg := parseConfig(t, cfgJSON)
	routes := getRoutes(t, cfg)
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	// First route: proxied
	h0 := getRouteHandler(t, routes[0])
	if h0["handler"] != "reverse_proxy" {
		t.Errorf("route[0] handler = %q, want reverse_proxy", h0["handler"])
	}

	// Second route: static
	h1 := getRouteHandler(t, routes[1])
	if h1["handler"] != "file_server" {
		t.Errorf("route[1] handler = %q, want file_server", h1["handler"])
	}
}

func TestBuildConfigAdminDisabled(t *testing.T) {
	resolver := &mockResolver{upstreams: map[string]string{}}
	cfgJSON, err := flockCaddy.BuildConfig(nil, resolver)
	if err != nil {
		t.Fatalf("BuildConfig: %v", err)
	}

	cfg := parseConfig(t, cfgJSON)
	admin := cfg["admin"].(map[string]any)
	if admin["disabled"] != true {
		t.Errorf("admin.disabled = %v, want true", admin["disabled"])
	}
}

func TestStartCallsRunnerRun(t *testing.T) {
	runner := &mockRunner{}
	resolver := &mockResolver{upstreams: map[string]string{}}
	m := flockCaddy.NewManager(runner, resolver)

	sites := []registry.Site{
		{Path: "/tmp/app", Domain: "app.test", TLS: false},
	}
	if err := m.Start(sites); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if runner.runCalls != 1 {
		t.Errorf("runCalls = %d, want 1", runner.runCalls)
	}
	if len(runner.lastCfg) == 0 {
		t.Error("expected non-empty config JSON")
	}
}

func TestReloadCallsRunnerRunAgain(t *testing.T) {
	runner := &mockRunner{}
	resolver := &mockResolver{upstreams: map[string]string{}}
	m := flockCaddy.NewManager(runner, resolver)

	sites := []registry.Site{
		{Path: "/tmp/app", Domain: "app.test", TLS: false},
	}
	_ = m.Start(sites)
	_ = m.Reload(sites)

	if runner.runCalls != 2 {
		t.Errorf("runCalls = %d, want 2", runner.runCalls)
	}
}

func TestStopCallsRunnerStop(t *testing.T) {
	runner := &mockRunner{}
	resolver := &mockResolver{upstreams: map[string]string{}}
	m := flockCaddy.NewManager(runner, resolver)

	_ = m.Start([]registry.Site{{Path: "/tmp/app", Domain: "app.test", TLS: false}})
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if runner.stopCalls != 1 {
		t.Errorf("stopCalls = %d, want 1", runner.stopCalls)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/caddy/... -v`
Expected: FAIL — package does not exist.

**Step 3: Implement Caddy manager**

Create `internal/caddy/caddy.go`:

```go
package caddy

import (
	"encoding/json"
	"fmt"

	"github.com/andybarilla/flock/internal/registry"
)

type CaddyRunner interface {
	Run(cfgJSON []byte) error
	Stop() error
}

type UpstreamResolver interface {
	ResolveUpstream(site registry.Site) (string, error)
}

type Manager struct {
	runner   CaddyRunner
	resolver UpstreamResolver
	running  bool
}

func NewManager(runner CaddyRunner, resolver UpstreamResolver) *Manager {
	return &Manager{runner: runner, resolver: resolver}
}

func (m *Manager) Start(sites []registry.Site) error {
	cfgJSON, err := BuildConfig(sites, m.resolver)
	if err != nil {
		return fmt.Errorf("build config: %w", err)
	}
	if err := m.runner.Run(cfgJSON); err != nil {
		return fmt.Errorf("caddy run: %w", err)
	}
	m.running = true
	return nil
}

func (m *Manager) Reload(sites []registry.Site) error {
	return m.Start(sites)
}

func (m *Manager) Stop() error {
	if err := m.runner.Stop(); err != nil {
		return fmt.Errorf("caddy stop: %w", err)
	}
	m.running = false
	return nil
}

func BuildConfig(sites []registry.Site, resolver UpstreamResolver) ([]byte, error) {
	routes := make([]map[string]any, 0, len(sites))

	for _, site := range sites {
		upstream, err := resolver.ResolveUpstream(site)
		if err != nil {
			return nil, fmt.Errorf("resolve upstream for %q: %w", site.Domain, err)
		}

		var handler map[string]any
		if upstream != "" {
			handler = map[string]any{
				"handler": "reverse_proxy",
				"upstreams": []map[string]any{
					{"dial": upstream},
				},
			}
		} else {
			handler = map[string]any{
				"handler": "file_server",
				"root":    site.Path,
			}
		}

		route := map[string]any{
			"match": []map[string]any{
				{"host": []string{site.Domain}},
			},
			"handle": []map[string]any{handler},
		}
		routes = append(routes, route)
	}

	cfg := map[string]any{
		"admin": map[string]any{"disabled": true},
		"apps": map[string]any{
			"http": map[string]any{
				"servers": map[string]any{
					"flock": map[string]any{
						"listen": []string{":80", ":443"},
						"routes": routes,
					},
				},
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/caddy/... -v`
Expected: PASS — all 7 tests pass.

**Step 5: Run full test suite**

Run: `go test ./internal/... -v`
Expected: PASS — all tests (config + plugin + registry + caddy) pass.

**Step 6: Commit**

```bash
git add internal/caddy/
git commit -m "feat: add Caddy manager with config generation and lifecycle management"
```
