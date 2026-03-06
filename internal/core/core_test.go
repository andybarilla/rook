package core_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/core"
	"github.com/andybarilla/flock/internal/databases"
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

type stubDBRunner struct {
	started map[databases.ServiceType]databases.ServiceConfig
	stopped map[databases.ServiceType]bool
}

func newStubDBRunner() *stubDBRunner {
	return &stubDBRunner{
		started: map[databases.ServiceType]databases.ServiceConfig{},
		stopped: map[databases.ServiceType]bool{},
	}
}

func (s *stubDBRunner) Start(svc databases.ServiceType, cfg databases.ServiceConfig) error {
	s.started[svc] = cfg
	return nil
}

func (s *stubDBRunner) Stop(svc databases.ServiceType) error {
	s.stopped[svc] = true
	return nil
}

func (s *stubDBRunner) Status(svc databases.ServiceType) databases.ServiceStatus {
	if _, ok := s.started[svc]; ok {
		if !s.stopped[svc] {
			return databases.StatusRunning
		}
	}
	return databases.StatusStopped
}

type stubNodeRunner struct {
	started map[string]int // siteDir -> port
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

// --- Helpers ---

func tmpSitesFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "sites.json")
}

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

// --- Tests ---

func TestNewCoreAndStartStop(t *testing.T) {
	cfg, runner, _, _, _, _ := testConfig(t)

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
	cfg, _, fpm, certs, _, _ := testConfig(t)

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
	cfg, _, _, _, _, _ := testConfig(t)

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
	cfg, _, _, _, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	plugins := c.Plugins()
	if len(plugins) != 4 {
		t.Fatalf("Plugins() len = %d, want 4", len(plugins))
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
	if !ids["flock-node"] {
		t.Error("expected flock-node plugin")
	}
	if !ids["flock-databases"] {
		t.Error("expected flock-databases plugin")
	}
}

func TestPluginsIncludesDatabases(t *testing.T) {
	cfg, _, _, _, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	services := c.DatabaseServices()
	if len(services) != 3 {
		t.Fatalf("DatabaseServices() len = %d, want 3", len(services))
	}
}

func TestExternalPluginsRegistered(t *testing.T) {
	cfg, _, _, _, _, _ := testConfig(t)

	// Create a plugins directory with a fake plugin manifest
	pluginsDir := filepath.Join(t.TempDir(), "plugins")
	os.MkdirAll(filepath.Join(pluginsDir, "test-ext"), 0o755)

	manifest := `{"id":"test-ext","name":"Test External","version":"0.1.0","executable":"test-ext","capabilities":["runtime"]}`
	os.WriteFile(filepath.Join(pluginsDir, "test-ext", "plugin.json"), []byte(manifest), 0o644)

	// Create a fake executable (won't be called since we won't start)
	os.WriteFile(filepath.Join(pluginsDir, "test-ext", "test-ext"), []byte("#!/bin/sh\n"), 0o755)

	cfg.PluginsDir = pluginsDir

	c := core.NewCore(cfg)
	plugins := c.Plugins()

	// 4 built-in + 1 external
	if len(plugins) != 5 {
		t.Fatalf("expected 5 plugins, got %d: %v", len(plugins), plugins)
	}

	found := false
	for _, p := range plugins {
		if p.ID == "test-ext" {
			found = true
			break
		}
	}
	if !found {
		t.Error("external plugin 'test-ext' not found in plugins list")
	}
}

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

func TestStartErrorOnCaddyFailure(t *testing.T) {
	cfg, runner, _, _, _, _ := testConfig(t)
	runner.runErr = fmt.Errorf("caddy failed")

	c := core.NewCore(cfg)
	err := c.Start()
	if err == nil {
		t.Error("expected error from Start when Caddy fails")
	}
}

func TestAddSiteReloadsCaddy(t *testing.T) {
	cfg, runner, _, _, _, _ := testConfig(t)
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
	cfg, runner, _, _, _, _ := testConfig(t)

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
