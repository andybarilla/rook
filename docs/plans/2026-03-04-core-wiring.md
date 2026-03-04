# Core Wiring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire together Registry, Plugin Manager, Caddy Manager, SSL Plugin, and PHP Plugin into a running application with a testable `Core` struct and thin Wails `App` layer.

**Architecture:** `internal/core/core.go` contains a `Core` struct that orchestrates the full lifecycle via dependency injection. Dependencies (CaddyRunner, FPMRunner, CertStore) are passed via a `Config` struct. `App` in `app.go` creates `Core` on startup and exposes Wails-bound methods that delegate to it. Stub implementations of CaddyRunner and FPMRunner are included for testing; real process management is deferred.

**Tech Stack:** Go 1.23. No new external dependencies.

---

## Task 1: Core Struct + Lifecycle

**Files:**
- Create: `internal/core/core.go`
- Create: `internal/core/core_test.go`

**Step 1: Write failing tests**

Create `internal/core/core_test.go`:

```go
package core_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/core"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// --- Stubs ---

type stubCaddyRunner struct {
	runCalls  int
	stopCalls int
	lastCfg   []byte
	runErr    error
}

func (s *stubCaddyRunner) Run(cfgJSON []byte) error {
	s.runCalls++
	s.lastCfg = cfgJSON
	return s.runErr
}

func (s *stubCaddyRunner) Stop() error {
	s.stopCalls++
	return nil
}

type stubFPMRunner struct {
	started map[string]bool
	stopped map[string]bool
}

func newStubFPMRunner() *stubFPMRunner {
	return &stubFPMRunner{started: map[string]bool{}, stopped: map[string]bool{}}
}

func (s *stubFPMRunner) StartPool(version string) error {
	s.started[version] = true
	return nil
}

func (s *stubFPMRunner) StopPool(version string) error {
	s.stopped[version] = true
	return nil
}

func (s *stubFPMRunner) PoolSocket(version string) string {
	return fmt.Sprintf("/tmp/php-fpm-%s.sock", version)
}

type stubCertStore struct {
	caInstalled bool
	certs       map[string]bool
}

func newStubCertStore() *stubCertStore {
	return &stubCertStore{certs: map[string]bool{}}
}

func (s *stubCertStore) InstallCA() error {
	s.caInstalled = true
	return nil
}

func (s *stubCertStore) GenerateCert(domain string) error {
	s.certs[domain] = true
	return nil
}

func (s *stubCertStore) CertPath(domain string) string {
	return "/tmp/certs/" + domain + ".pem"
}

func (s *stubCertStore) KeyPath(domain string) string {
	return "/tmp/certs/" + domain + "-key.pem"
}

func (s *stubCertStore) HasCert(domain string) bool {
	return s.certs[domain]
}

// --- Helpers ---

func tmpSitesFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "sites.json")
}

func testConfig(t *testing.T) (core.Config, *stubCaddyRunner, *stubFPMRunner, *stubCertStore) {
	t.Helper()
	runner := &stubCaddyRunner{}
	fpm := newStubFPMRunner()
	certs := newStubCertStore()
	cfg := core.Config{
		SitesFile:   tmpSitesFile(t),
		Logger:      log.New(os.Stderr, "", 0),
		CaddyRunner: runner,
		FPMRunner:   fpm,
		CertStore:   certs,
	}
	return cfg, runner, fpm, certs
}

// --- Tests ---

func TestNewCoreAndStartStop(t *testing.T) {
	cfg, runner, _, _ := testConfig(t)

	c := core.NewCore(cfg)
	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if runner.runCalls != 1 {
		t.Errorf("caddy runCalls = %d, want 1", runner.runCalls)
	}

	if err := c.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if runner.stopCalls != 1 {
		t.Errorf("caddy stopCalls = %d, want 1", runner.stopCalls)
	}
}

func TestStartLoadsSitesAndStartsPlugins(t *testing.T) {
	cfg, _, fpm, certs := testConfig(t)

	// Pre-populate sites file
	sitesJSON := `[{"path":"/tmp","domain":"app.test","php_version":"8.3","tls":true}]`
	os.MkdirAll(filepath.Dir(cfg.SitesFile), 0o755)
	os.WriteFile(cfg.SitesFile, []byte(sitesJSON), 0o644)

	c := core.NewCore(cfg)
	if err := c.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	// PHP plugin should have started pool for 8.3
	if !fpm.started["8.3"] {
		t.Error("expected FPM pool 8.3 to be started")
	}

	// SSL plugin should have installed CA and generated cert
	if !certs.caInstalled {
		t.Error("expected CA to be installed")
	}
	if !certs.certs["app.test"] {
		t.Error("expected cert for app.test to be generated")
	}
}

func TestSitesReturnsList(t *testing.T) {
	cfg, _, _, _ := testConfig(t)

	sitesJSON := `[{"path":"/tmp","domain":"app.test"}]`
	os.MkdirAll(filepath.Dir(cfg.SitesFile), 0o755)
	os.WriteFile(cfg.SitesFile, []byte(sitesJSON), 0o644)

	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	sites := c.Sites()
	if len(sites) != 1 {
		t.Fatalf("Sites() len = %d, want 1", len(sites))
	}
	if sites[0].Domain != "app.test" {
		t.Errorf("domain = %q, want app.test", sites[0].Domain)
	}
}

func TestPluginsReturnsInfo(t *testing.T) {
	cfg, _, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	plugins := c.Plugins()
	if len(plugins) != 2 {
		t.Fatalf("Plugins() len = %d, want 2", len(plugins))
	}

	ids := map[string]bool{}
	for _, p := range plugins {
		ids[p.ID] = true
	}
	if !ids["flock-ssl"] {
		t.Error("expected flock-ssl plugin")
	}
	if !ids["flock-php"] {
		t.Error("expected flock-php plugin")
	}
}

func TestStartErrorOnCaddyFailure(t *testing.T) {
	cfg, runner, _, _ := testConfig(t)
	runner.runErr = fmt.Errorf("caddy failed")

	c := core.NewCore(cfg)
	err := c.Start()
	if err == nil {
		t.Error("expected error from Start when Caddy fails")
	}
}

func TestAddSiteReloadsCaddy(t *testing.T) {
	cfg, runner, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	initialRuns := runner.runCalls

	dir := t.TempDir()
	err := c.AddSite(registry.Site{Path: dir, Domain: "new.test"})
	if err != nil {
		t.Fatalf("AddSite: %v", err)
	}

	if runner.runCalls != initialRuns+1 {
		t.Errorf("caddy runCalls = %d, want %d (reload after AddSite)", runner.runCalls, initialRuns+1)
	}
}

func TestRemoveSiteReloadsCaddy(t *testing.T) {
	cfg, runner, _, _ := testConfig(t)

	dir := t.TempDir()
	sitesJSON := fmt.Sprintf(`[{"path":%q,"domain":"app.test"}]`, dir)
	os.MkdirAll(filepath.Dir(cfg.SitesFile), 0o755)
	os.WriteFile(cfg.SitesFile, []byte(sitesJSON), 0o644)

	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	initialRuns := runner.runCalls

	err := c.RemoveSite("app.test")
	if err != nil {
		t.Fatalf("RemoveSite: %v", err)
	}

	if runner.runCalls != initialRuns+1 {
		t.Errorf("caddy runCalls = %d, want %d (reload after RemoveSite)", runner.runCalls, initialRuns+1)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/core/... -v`
Expected: FAIL — package does not exist.

**Step 3: Implement Core**

Create `internal/core/core.go`:

```go
package core

import (
	"fmt"
	"log"

	"github.com/andybarilla/flock/internal/caddy"
	"github.com/andybarilla/flock/internal/php"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
	"github.com/andybarilla/flock/internal/ssl"
)

type Config struct {
	SitesFile   string
	Logger      *log.Logger
	CaddyRunner caddy.CaddyRunner
	FPMRunner   php.FPMRunner
	CertStore   ssl.CertStore
}

type Core struct {
	registry  *registry.Registry
	pluginMgr *plugin.Manager
	caddyMgr  *caddy.Manager
	sslPlugin *ssl.Plugin
	phpPlugin *php.Plugin
	logger    *log.Logger
}

func NewCore(cfg Config) *Core {
	reg := registry.New(cfg.SitesFile)
	pluginMgr := plugin.NewManager(reg, cfg.Logger)

	sslPlugin := ssl.NewPlugin(cfg.CertStore)
	phpPlugin := php.NewPlugin(cfg.FPMRunner)
	pluginMgr.Register(sslPlugin)
	pluginMgr.Register(phpPlugin)

	caddyMgr := caddy.NewManager(cfg.CaddyRunner, pluginMgr, sslPlugin)

	c := &Core{
		registry:  reg,
		pluginMgr: pluginMgr,
		caddyMgr:  caddyMgr,
		sslPlugin: sslPlugin,
		phpPlugin: phpPlugin,
		logger:    cfg.Logger,
	}

	reg.OnChange(func(e registry.ChangeEvent) {
		if err := c.caddyMgr.Reload(c.registry.List()); err != nil {
			c.logger.Printf("caddy reload failed: %v", err)
		}
	})

	return c
}

func (c *Core) Start() error {
	if err := c.registry.Load(); err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	c.pluginMgr.InitAll()
	c.pluginMgr.StartAll()

	if err := c.caddyMgr.Start(c.registry.List()); err != nil {
		return fmt.Errorf("start caddy: %w", err)
	}

	return nil
}

func (c *Core) Stop() error {
	if err := c.caddyMgr.Stop(); err != nil {
		c.logger.Printf("caddy stop failed: %v", err)
	}
	c.pluginMgr.StopAll()
	return nil
}

func (c *Core) Sites() []registry.Site {
	return c.registry.List()
}

func (c *Core) AddSite(site registry.Site) error {
	return c.registry.Add(site)
}

func (c *Core) RemoveSite(domain string) error {
	return c.registry.Remove(domain)
}

func (c *Core) Plugins() []plugin.PluginInfo {
	return c.pluginMgr.Plugins()
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/core/... -v`
Expected: PASS — all 7 tests pass.

**Step 5: Run full test suite**

Run: `go test ./internal/... -v`
Expected: PASS — all tests across caddy, config, core, php, plugin, registry, and ssl packages pass.

**Step 6: Commit**

```bash
git add internal/core/
git commit -m "feat: add Core struct wiring registry, plugins, and Caddy lifecycle"
```

---

## Task 2: App Integration

**Files:**
- Modify: `app.go`

**Step 1: Write failing tests**

No test file for `app.go` — it's a thin Wails binding layer. We test the App manually via `Core` tests above. The changes are structural only: add `Core` field, create it in `startup()`, delegate Wails methods to it.

**Step 2: Update app.go**

Replace `app.go` with:

```go
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/andybarilla/flock/internal/config"
	"github.com/andybarilla/flock/internal/core"
	"github.com/andybarilla/flock/internal/registry"
)

// App struct
type App struct {
	ctx  context.Context
	core *core.Core
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	logDir := config.DataDir()
	os.MkdirAll(logDir, 0o755)
	logFile, err := os.OpenFile(config.LogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logFile = os.Stderr
	}
	logger := log.New(logFile, "[flock] ", log.LstdFlags)

	cfg := core.Config{
		SitesFile:   config.SitesFile(),
		Logger:      logger,
		CaddyRunner: &loggingCaddyRunner{logger: logger},
		FPMRunner:   &loggingFPMRunner{logger: logger},
		CertStore:   &loggingCertStore{logger: logger, certsDir: filepath.Join(config.DataDir(), "certs")},
	}

	a.core = core.NewCore(cfg)
	if err := a.core.Start(); err != nil {
		logger.Printf("core start failed: %v", err)
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.core != nil {
		a.core.Stop()
	}
}

// ListSites returns all registered sites
func (a *App) ListSites() []registry.Site {
	return a.core.Sites()
}

// AddSite registers a new site
func (a *App) AddSite(path, domain, phpVersion string, tls bool) error {
	return a.core.AddSite(registry.Site{
		Path:       path,
		Domain:     domain,
		PHPVersion: phpVersion,
		TLS:        tls,
	})
}

// RemoveSite removes a registered site
func (a *App) RemoveSite(domain string) error {
	return a.core.RemoveSite(domain)
}
```

**Step 3: Create stub implementations**

Create `stubs.go` in root package:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// loggingCaddyRunner logs Caddy operations without running real Caddy.
type loggingCaddyRunner struct {
	logger *log.Logger
}

func (r *loggingCaddyRunner) Run(cfgJSON []byte) error {
	r.logger.Printf("caddy: loading config (%d bytes)", len(cfgJSON))
	return nil
}

func (r *loggingCaddyRunner) Stop() error {
	r.logger.Println("caddy: stopped")
	return nil
}

// loggingFPMRunner logs FPM operations without running real php-fpm.
type loggingFPMRunner struct {
	logger *log.Logger
}

func (r *loggingFPMRunner) StartPool(version string) error {
	r.logger.Printf("php-fpm: starting pool for PHP %s", version)
	return nil
}

func (r *loggingFPMRunner) StopPool(version string) error {
	r.logger.Printf("php-fpm: stopping pool for PHP %s", version)
	return nil
}

func (r *loggingFPMRunner) PoolSocket(version string) string {
	return fmt.Sprintf("/tmp/php-fpm-%s.sock", version)
}

// loggingCertStore logs cert operations without generating real certs.
type loggingCertStore struct {
	logger   *log.Logger
	certsDir string
}

func (s *loggingCertStore) InstallCA() error {
	s.logger.Println("ssl: CA install (stub)")
	return nil
}

func (s *loggingCertStore) GenerateCert(domain string) error {
	s.logger.Printf("ssl: generating cert for %s (stub)", domain)
	os.MkdirAll(s.certsDir, 0o755)
	// Create empty placeholder files so HasCert returns true
	os.WriteFile(s.CertPath(domain), []byte("stub"), 0o644)
	os.WriteFile(s.KeyPath(domain), []byte("stub"), 0o644)
	return nil
}

func (s *loggingCertStore) CertPath(domain string) string {
	return filepath.Join(s.certsDir, domain+".pem")
}

func (s *loggingCertStore) KeyPath(domain string) string {
	return filepath.Join(s.certsDir, domain+"-key.pem")
}

func (s *loggingCertStore) HasCert(domain string) bool {
	_, err := os.Stat(s.CertPath(domain))
	return err == nil
}
```

**Step 4: Update main.go to wire shutdown**

Modify `main.go` to add `OnShutdown`:

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "flock",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:        app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
```

**Step 5: Run internal tests to verify nothing broke**

Run: `go test ./internal/... -v`
Expected: PASS — all tests still pass. (Root package can't be tested without Wails frontend build.)

**Step 6: Commit**

```bash
git add app.go stubs.go main.go
git commit -m "feat: wire App to Core with stub CaddyRunner, FPMRunner, and CertStore"
```
