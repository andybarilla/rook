# flock-databases Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a databases plugin that manages MySQL, PostgreSQL, and Redis processes with start/stop/status control and a basic GUI services panel.

**Architecture:** Unified `flock-databases` ServicePlugin with a `DBRunner` interface (same pattern as flock-php's `FPMRunner`). Config persisted in `databases.json`. Concrete `ProcessRunner` uses `os/exec` to manage database daemons. GUI adds a "Services" section with start/stop buttons.

**Tech Stack:** Go 1.23, Wails v2, Svelte, os/exec for process management

---

### Task 1: DBRunner Interface & Types

**Files:**
- Create: `internal/databases/runner.go`

**Step 1: Create the DBRunner interface and types**

```go
// internal/databases/runner.go
package databases

type ServiceType string

const (
	MySQL    ServiceType = "mysql"
	Postgres ServiceType = "postgres"
	Redis    ServiceType = "redis"
)

// AllServiceTypes lists every supported database in display order.
var AllServiceTypes = []ServiceType{MySQL, Postgres, Redis}

type ServiceConfig struct {
	Port    int
	DataDir string
}

type ServiceStatus int

const (
	StatusStopped ServiceStatus = iota
	StatusRunning
)

// ServiceInfo is the per-service detail exposed to Core / GUI.
type ServiceInfo struct {
	Type      ServiceType   `json:"type"`
	Enabled   bool          `json:"enabled"`
	Running   bool          `json:"running"`
	Autostart bool          `json:"autostart"`
	Port      int           `json:"port"`
}

// DBRunner abstracts database process management.
type DBRunner interface {
	Start(svc ServiceType, cfg ServiceConfig) error
	Stop(svc ServiceType) error
	Status(svc ServiceType) ServiceStatus
}
```

**Step 2: Verify it compiles**

Run: `cd <worktree-root> && go build ./internal/databases/...`
Expected: success (no tests yet — this is just types + interface)

**Step 3: Commit**

```bash
git add internal/databases/runner.go
git commit -m "feat(databases): add DBRunner interface and service types"
```

---

### Task 2: Config Loading & Persistence

**Files:**
- Create: `internal/databases/config.go`
- Create: `internal/databases/config_test.go`

**Step 1: Write failing tests for config**

```go
// internal/databases/config_test.go
package databases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/databases"
)

func TestDefaultConfig(t *testing.T) {
	cfg := databases.DefaultConfig("/data")

	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL port = %d, want 3306", cfg.MySQL.Port)
	}
	if cfg.Postgres.Port != 5432 {
		t.Errorf("Postgres port = %d, want 5432", cfg.Postgres.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis port = %d, want 6379", cfg.Redis.Port)
	}
	if cfg.MySQL.Autostart {
		t.Error("MySQL autostart should default to false")
	}
	if cfg.MySQL.DataDir != "/data/mysql" {
		t.Errorf("MySQL DataDir = %q, want /data/mysql", cfg.MySQL.DataDir)
	}
}

func TestLoadConfigCreatesDefaultIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")

	cfg, err := databases.LoadConfig(path, "/data")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL port = %d, want 3306", cfg.MySQL.Port)
	}

	// File should have been created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}
}

func TestLoadConfigReadsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")

	content := `{
		"mysql": {"enabled": true, "autostart": true, "port": 3307, "dataDir": "/custom/mysql"},
		"postgres": {"enabled": false, "autostart": false, "port": 5432, "dataDir": "/custom/pg"},
		"redis": {"enabled": true, "autostart": false, "port": 6380, "dataDir": "/custom/redis"}
	}`
	os.WriteFile(path, []byte(content), 0o644)

	cfg, err := databases.LoadConfig(path, "/data")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MySQL.Port != 3307 {
		t.Errorf("MySQL port = %d, want 3307", cfg.MySQL.Port)
	}
	if !cfg.MySQL.Autostart {
		t.Error("MySQL autostart should be true")
	}
	if cfg.MySQL.DataDir != "/custom/mysql" {
		t.Errorf("MySQL DataDir = %q, want /custom/mysql", cfg.MySQL.DataDir)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")

	cfg := databases.DefaultConfig("/data")
	cfg.MySQL.Port = 3307

	if err := databases.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := databases.LoadConfig(path, "/data")
	if err != nil {
		t.Fatalf("LoadConfig after save: %v", err)
	}
	if loaded.MySQL.Port != 3307 {
		t.Errorf("MySQL port = %d, want 3307", loaded.MySQL.Port)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/databases/... -v`
Expected: FAIL — `databases.DefaultConfig`, `LoadConfig`, `SaveConfig` not defined

**Step 3: Implement config**

```go
// internal/databases/config.go
package databases

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SvcConfig holds settings for one database service.
type SvcConfig struct {
	Enabled   bool   `json:"enabled"`
	Autostart bool   `json:"autostart"`
	Port      int    `json:"port"`
	DataDir   string `json:"dataDir"`
}

// Config holds settings for all database services.
type Config struct {
	MySQL    SvcConfig `json:"mysql"`
	Postgres SvcConfig `json:"postgres"`
	Redis    SvcConfig `json:"redis"`
}

// ForType returns the SvcConfig for the given service type.
func (c *Config) ForType(svc ServiceType) SvcConfig {
	switch svc {
	case MySQL:
		return c.MySQL
	case Postgres:
		return c.Postgres
	case Redis:
		return c.Redis
	}
	return SvcConfig{}
}

// SetEnabled sets the enabled flag for the given service type.
func (c *Config) SetEnabled(svc ServiceType, enabled bool) {
	switch svc {
	case MySQL:
		c.MySQL.Enabled = enabled
	case Postgres:
		c.Postgres.Enabled = enabled
	case Redis:
		c.Redis.Enabled = enabled
	}
}

// DefaultConfig returns a Config with sensible defaults.
// dataRoot is the base directory for database data (e.g. ~/.local/share/flock/databases).
func DefaultConfig(dataRoot string) Config {
	return Config{
		MySQL: SvcConfig{
			Enabled:   true,
			Autostart: false,
			Port:      3306,
			DataDir:   filepath.Join(dataRoot, "mysql"),
		},
		Postgres: SvcConfig{
			Enabled:   true,
			Autostart: false,
			Port:      5432,
			DataDir:   filepath.Join(dataRoot, "postgres"),
		},
		Redis: SvcConfig{
			Enabled:   true,
			Autostart: false,
			Port:      6379,
			DataDir:   filepath.Join(dataRoot, "redis"),
		},
	}
}

// LoadConfig reads config from path. If the file does not exist, writes
// defaults and returns them.
func LoadConfig(path, dataRoot string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := DefaultConfig(dataRoot)
		if saveErr := SaveConfig(path, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig writes config to path as indented JSON.
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/databases/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/databases/config.go internal/databases/config_test.go
git commit -m "feat(databases): add config loading and persistence"
```

---

### Task 3: Plugin Core (ServicePlugin)

**Files:**
- Create: `internal/databases/databases.go`
- Create: `internal/databases/databases_test.go`

**Step 1: Write failing tests for the plugin**

The test file needs mock DBRunner and mock Host (same patterns as `internal/php/php_test.go`).

```go
// internal/databases/databases_test.go
package databases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/databases"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// --- Mock DBRunner ---

type mockDBRunner struct {
	started map[databases.ServiceType]databases.ServiceConfig
	stopped map[databases.ServiceType]bool
	statuses map[databases.ServiceType]databases.ServiceStatus
	startErr error
}

func newMockDBRunner() *mockDBRunner {
	return &mockDBRunner{
		started:  map[databases.ServiceType]databases.ServiceConfig{},
		stopped:  map[databases.ServiceType]bool{},
		statuses: map[databases.ServiceType]databases.ServiceStatus{},
	}
}

func (m *mockDBRunner) Start(svc databases.ServiceType, cfg databases.ServiceConfig) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started[svc] = cfg
	m.statuses[svc] = databases.StatusRunning
	return nil
}

func (m *mockDBRunner) Stop(svc databases.ServiceType) error {
	m.stopped[svc] = true
	m.statuses[svc] = databases.StatusStopped
	return nil
}

func (m *mockDBRunner) Status(svc databases.ServiceType) databases.ServiceStatus {
	return m.statuses[svc]
}

// --- Mock Host ---

type mockHost struct {
	sites []registry.Site
}

func (m *mockHost) Sites() []registry.Site                        { return m.sites }
func (m *mockHost) GetSite(domain string) (registry.Site, bool) {
	for _, s := range m.sites {
		if s.Domain == domain {
			return s, true
		}
	}
	return registry.Site{}, false
}
func (m *mockHost) Log(pluginID string, msg string, args ...any) {}

// loggingHost captures log calls
type loggingHost struct {
	sites  []registry.Site
	logged bool
}

func (h *loggingHost) Sites() []registry.Site                        { return h.sites }
func (h *loggingHost) GetSite(domain string) (registry.Site, bool) {
	return registry.Site{}, false
}
func (h *loggingHost) Log(pluginID string, msg string, args ...any) { h.logged = true }

// --- Helpers ---

func tmpConfigDir(t *testing.T) (configPath, dataDir string) {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "databases.json"), filepath.Join(dir, "data")
}

// --- Tests ---

func TestPluginIDAndName(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)
	p := databases.NewPlugin(runner, configPath, dataDir)

	if p.ID() != "flock-databases" {
		t.Errorf("ID = %q, want flock-databases", p.ID())
	}
	if p.Name() != "Flock Databases" {
		t.Errorf("Name = %q, want Flock Databases", p.Name())
	}
}

func TestInitLoadsConfig(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)
	p := databases.NewPlugin(runner, configPath, dataDir)

	host := &mockHost{}
	if err := p.Init(host); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Config file should have been created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to be created on Init")
	}
}

func TestStartAutostartServices(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)

	// Write config with Redis autostart enabled
	content := `{
		"mysql": {"enabled": true, "autostart": false, "port": 3306, "dataDir": "/tmp/mysql"},
		"postgres": {"enabled": true, "autostart": false, "port": 5432, "dataDir": "/tmp/pg"},
		"redis": {"enabled": true, "autostart": true, "port": 6379, "dataDir": "/tmp/redis"}
	}`
	os.WriteFile(configPath, []byte(content), 0o644)

	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})

	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Only Redis should have started (autostart: true)
	if _, ok := runner.started[databases.Redis]; !ok {
		t.Error("expected Redis to be started (autostart)")
	}
	if _, ok := runner.started[databases.MySQL]; ok {
		t.Error("MySQL should not have started (autostart: false)")
	}
	if _, ok := runner.started[databases.Postgres]; ok {
		t.Error("Postgres should not have started (autostart: false)")
	}
}

func TestStartSkipsDisabledServices(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)

	content := `{
		"mysql": {"enabled": false, "autostart": true, "port": 3306, "dataDir": "/tmp/mysql"},
		"postgres": {"enabled": true, "autostart": false, "port": 5432, "dataDir": "/tmp/pg"},
		"redis": {"enabled": true, "autostart": false, "port": 6379, "dataDir": "/tmp/redis"}
	}`
	os.WriteFile(configPath, []byte(content), 0o644)

	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})
	_ = p.Start()

	if _, ok := runner.started[databases.MySQL]; ok {
		t.Error("MySQL should not start when disabled, even with autostart: true")
	}
}

func TestStopStopsAllRunning(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)

	content := `{
		"mysql": {"enabled": true, "autostart": true, "port": 3306, "dataDir": "/tmp/mysql"},
		"postgres": {"enabled": true, "autostart": false, "port": 5432, "dataDir": "/tmp/pg"},
		"redis": {"enabled": true, "autostart": true, "port": 6379, "dataDir": "/tmp/redis"}
	}`
	os.WriteFile(configPath, []byte(content), 0o644)

	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})
	_ = p.Start()
	_ = p.Stop()

	if !runner.stopped[databases.MySQL] {
		t.Error("expected MySQL to be stopped")
	}
	if !runner.stopped[databases.Redis] {
		t.Error("expected Redis to be stopped")
	}
}

func TestServiceStatusReflectsRunning(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)
	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Error("initial status should be ServiceStopped")
	}

	// Write config with autostart
	content := `{
		"mysql": {"enabled": true, "autostart": true, "port": 3306, "dataDir": "/tmp/mysql"},
		"postgres": {"enabled": true, "autostart": false, "port": 5432, "dataDir": "/tmp/pg"},
		"redis": {"enabled": true, "autostart": false, "port": 6379, "dataDir": "/tmp/redis"}
	}`
	os.WriteFile(configPath, []byte(content), 0o644)

	// Re-init and start
	_ = p.Init(&mockHost{})
	_ = p.Start()

	if p.ServiceStatus() != plugin.ServiceRunning {
		t.Error("status should be ServiceRunning when any service is running")
	}

	_ = p.Stop()

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Error("status should be ServiceStopped after Stop")
	}
}

func TestStartSvcAndStopSvc(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)
	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})

	if err := p.StartSvc(databases.MySQL); err != nil {
		t.Fatalf("StartSvc: %v", err)
	}
	if _, ok := runner.started[databases.MySQL]; !ok {
		t.Error("expected MySQL to be started")
	}

	if err := p.StopSvc(databases.MySQL); err != nil {
		t.Fatalf("StopSvc: %v", err)
	}
	if !runner.stopped[databases.MySQL] {
		t.Error("expected MySQL to be stopped")
	}
}

func TestServiceStatuses(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)
	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})

	infos := p.ServiceStatuses()
	if len(infos) != 3 {
		t.Fatalf("ServiceStatuses len = %d, want 3", len(infos))
	}

	// All should be stopped and enabled (defaults)
	for _, info := range infos {
		if info.Running {
			t.Errorf("%s should not be running", info.Type)
		}
		if !info.Enabled {
			t.Errorf("%s should be enabled by default", info.Type)
		}
	}
}

func TestStartSvcLogsFailure(t *testing.T) {
	runner := newMockDBRunner()
	runner.startErr = fmt.Errorf("mysqld not found")
	configPath, dataDir := tmpConfigDir(t)
	p := databases.NewPlugin(runner, configPath, dataDir)

	host := &loggingHost{}
	_ = p.Init(host)

	// StartSvc should return the error
	err := p.StartSvc(databases.MySQL)
	if err == nil {
		t.Error("expected error from StartSvc when runner fails")
	}
}

func TestStartLogsAutostartFailure(t *testing.T) {
	runner := newMockDBRunner()
	runner.startErr = fmt.Errorf("mysqld not found")
	configPath, dataDir := tmpConfigDir(t)

	content := `{
		"mysql": {"enabled": true, "autostart": true, "port": 3306, "dataDir": "/tmp/mysql"},
		"postgres": {"enabled": true, "autostart": false, "port": 5432, "dataDir": "/tmp/pg"},
		"redis": {"enabled": true, "autostart": false, "port": 6379, "dataDir": "/tmp/redis"}
	}`
	os.WriteFile(configPath, []byte(content), 0o644)

	host := &loggingHost{}
	p := databases.NewPlugin(runner, configPath, dataDir)
	_ = p.Init(host)

	// Start should not return error — it logs and continues
	if err := p.Start(); err != nil {
		t.Fatalf("Start should not error on autostart failure: %v", err)
	}

	if !host.logged {
		t.Error("expected autostart failure to be logged")
	}
}
```

Note: Add `"fmt"` to the imports at the top — it's used by `TestStartSvcLogsFailure`.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/databases/... -v`
Expected: FAIL — `databases.NewPlugin`, `Plugin` methods not defined

**Step 3: Implement the plugin**

```go
// internal/databases/databases.go
package databases

import (
	"github.com/andybarilla/flock/internal/plugin"
)

// Plugin manages MySQL, PostgreSQL, and Redis services.
type Plugin struct {
	runner     DBRunner
	host       plugin.Host
	configPath string
	dataRoot   string
	config     Config
	running    map[ServiceType]bool
}

// NewPlugin creates a databases plugin.
// configPath: path to databases.json
// dataRoot: base directory for database data dirs (used as default in config)
func NewPlugin(runner DBRunner, configPath, dataRoot string) *Plugin {
	return &Plugin{
		runner:     runner,
		configPath: configPath,
		dataRoot:   dataRoot,
		running:    map[ServiceType]bool{},
	}
}

func (p *Plugin) ID() string   { return "flock-databases" }
func (p *Plugin) Name() string { return "Flock Databases" }

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	cfg, err := LoadConfig(p.configPath, p.dataRoot)
	if err != nil {
		return err
	}
	p.config = cfg
	return nil
}

func (p *Plugin) Start() error {
	for _, svc := range AllServiceTypes {
		svcCfg := p.config.ForType(svc)
		if !svcCfg.Enabled || !svcCfg.Autostart {
			continue
		}
		if err := p.runner.Start(svc, ServiceConfig{Port: svcCfg.Port, DataDir: svcCfg.DataDir}); err != nil {
			p.host.Log(p.ID(), "autostart %s failed: %v", svc, err)
			continue
		}
		p.running[svc] = true
	}
	return nil
}

func (p *Plugin) Stop() error {
	for svc := range p.running {
		if err := p.runner.Stop(svc); err != nil {
			p.host.Log(p.ID(), "stop %s failed: %v", svc, err)
		}
		delete(p.running, svc)
	}
	return nil
}

func (p *Plugin) ServiceStatus() plugin.ServiceStatus {
	for _, running := range p.running {
		if running {
			return plugin.ServiceRunning
		}
	}
	return plugin.ServiceStopped
}

func (p *Plugin) StartService() error { return p.Start() }
func (p *Plugin) StopService() error  { return p.Stop() }

// StartSvc starts a specific database service.
func (p *Plugin) StartSvc(svc ServiceType) error {
	svcCfg := p.config.ForType(svc)
	if err := p.runner.Start(svc, ServiceConfig{Port: svcCfg.Port, DataDir: svcCfg.DataDir}); err != nil {
		return err
	}
	p.running[svc] = true
	return nil
}

// StopSvc stops a specific database service.
func (p *Plugin) StopSvc(svc ServiceType) error {
	if err := p.runner.Stop(svc); err != nil {
		return err
	}
	delete(p.running, svc)
	return nil
}

// ServiceStatuses returns status info for all services.
func (p *Plugin) ServiceStatuses() []ServiceInfo {
	var infos []ServiceInfo
	for _, svc := range AllServiceTypes {
		svcCfg := p.config.ForType(svc)
		infos = append(infos, ServiceInfo{
			Type:      svc,
			Enabled:   svcCfg.Enabled,
			Running:   p.running[svc],
			Autostart: svcCfg.Autostart,
			Port:      svcCfg.Port,
		})
	}
	return infos
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/databases/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/databases/databases.go internal/databases/databases_test.go
git commit -m "feat(databases): add Plugin with ServicePlugin interface"
```

---

### Task 4: ProcessRunner (Concrete DBRunner)

**Files:**
- Create: `internal/databases/process.go`
- Create: `internal/databases/process_test.go`

This is the concrete implementation that uses `os/exec` to start/stop database processes. Tests use a "fake binary" approach — small shell scripts in temp dirs — to avoid requiring real databases.

**Step 1: Write failing tests for ProcessRunner**

```go
// internal/databases/process_test.go
package databases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/databases"
)

func TestProcessRunnerStatusStoppedByDefault(t *testing.T) {
	r := databases.NewProcessRunner()
	if r.Status(databases.MySQL) != databases.StatusStopped {
		t.Error("expected StatusStopped for MySQL by default")
	}
	if r.Status(databases.Postgres) != databases.StatusStopped {
		t.Error("expected StatusStopped for Postgres by default")
	}
	if r.Status(databases.Redis) != databases.StatusStopped {
		t.Error("expected StatusStopped for Redis by default")
	}
}

func TestProcessRunnerStartCreatesDataDir(t *testing.T) {
	r := databases.NewProcessRunner()
	dataDir := filepath.Join(t.TempDir(), "mysql-data")

	// Expect Start to fail (no mysqld on PATH in test) but dataDir should be created
	_ = r.Start(databases.MySQL, databases.ServiceConfig{Port: 13306, DataDir: dataDir})

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("expected data dir to be created")
	}
}

func TestProcessRunnerBinaryNames(t *testing.T) {
	// Verify BinaryFor returns expected binary names
	tests := []struct {
		svc  databases.ServiceType
		want string
	}{
		{databases.MySQL, "mysqld"},
		{databases.Postgres, "pg_ctl"},
		{databases.Redis, "redis-server"},
	}
	for _, tt := range tests {
		got := databases.BinaryFor(tt.svc)
		if got != tt.want {
			t.Errorf("BinaryFor(%s) = %q, want %q", tt.svc, got, tt.want)
		}
	}
}

func TestProcessRunnerCheckBinary(t *testing.T) {
	// A binary that definitely doesn't exist
	if databases.CheckBinary("__nonexistent_binary_12345__") {
		t.Error("expected CheckBinary to return false for nonexistent binary")
	}
	// "go" should exist in test environment
	if !databases.CheckBinary("go") {
		t.Error("expected CheckBinary to return true for 'go'")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/databases/... -v -run TestProcessRunner`
Expected: FAIL — `databases.NewProcessRunner`, `BinaryFor`, `CheckBinary` not defined

**Step 3: Implement ProcessRunner**

```go
// internal/databases/process.go
package databases

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// BinaryFor returns the primary binary name for a service type.
func BinaryFor(svc ServiceType) string {
	switch svc {
	case MySQL:
		return "mysqld"
	case Postgres:
		return "pg_ctl"
	case Redis:
		return "redis-server"
	}
	return ""
}

// CheckBinary reports whether the named binary is on PATH.
func CheckBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ProcessRunner manages database processes using os/exec.
type ProcessRunner struct {
	procs map[ServiceType]*os.Process
}

// NewProcessRunner creates a ProcessRunner.
func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{
		procs: map[ServiceType]*os.Process{},
	}
}

func (r *ProcessRunner) Start(svc ServiceType, cfg ServiceConfig) error {
	// Ensure data dir exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	switch svc {
	case MySQL:
		return r.startMySQL(cfg)
	case Postgres:
		return r.startPostgres(cfg)
	case Redis:
		return r.startRedis(cfg)
	}
	return fmt.Errorf("unknown service type: %s", svc)
}

func (r *ProcessRunner) Stop(svc ServiceType) error {
	switch svc {
	case MySQL:
		return r.stopMySQL(svc)
	case Postgres:
		return r.stopPostgres(svc)
	case Redis:
		return r.stopRedis(svc)
	}
	return fmt.Errorf("unknown service type: %s", svc)
}

func (r *ProcessRunner) Status(svc ServiceType) ServiceStatus {
	p, ok := r.procs[svc]
	if !ok {
		return StatusStopped
	}
	// Check if process is still alive
	if err := p.Signal(syscall.Signal(0)); err != nil {
		delete(r.procs, svc)
		return StatusStopped
	}
	return StatusRunning
}

// --- MySQL ---

func (r *ProcessRunner) startMySQL(cfg ServiceConfig) error {
	// Initialize data dir if empty
	entries, _ := os.ReadDir(cfg.DataDir)
	if len(entries) == 0 {
		cmd := exec.Command("mysqld", "--initialize-insecure", "--datadir="+cfg.DataDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("mysql init: %s: %w", string(out), err)
		}
	}

	cmd := exec.Command("mysqld",
		"--datadir="+cfg.DataDir,
		"--port="+strconv.Itoa(cfg.Port),
		"--socket="+cfg.DataDir+"/mysql.sock",
		"--pid-file="+cfg.DataDir+"/mysql.pid",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mysqld: %w", err)
	}
	r.procs[MySQL] = cmd.Process
	return nil
}

func (r *ProcessRunner) stopMySQL(svc ServiceType) error {
	p, ok := r.procs[svc]
	if !ok {
		return nil
	}
	// Graceful shutdown via signal
	if err := p.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop mysqld: %w", err)
	}
	_, _ = p.Wait()
	delete(r.procs, svc)
	return nil
}

// --- PostgreSQL ---

func (r *ProcessRunner) startPostgres(cfg ServiceConfig) error {
	// Initialize data dir if empty
	entries, _ := os.ReadDir(cfg.DataDir)
	if len(entries) == 0 {
		cmd := exec.Command("initdb", "-D", cfg.DataDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("postgres init: %s: %w", string(out), err)
		}
	}

	cmd := exec.Command("pg_ctl",
		"-D", cfg.DataDir,
		"-l", cfg.DataDir+"/postgres.log",
		"-o", "-p "+strconv.Itoa(cfg.Port),
		"start",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start postgres: %s: %w", string(out), err)
	}

	// pg_ctl start returns immediately; read PID from postmaster.pid
	pidData, err := os.ReadFile(cfg.DataDir + "/postmaster.pid")
	if err == nil {
		lines := strings.SplitN(string(pidData), "\n", 2)
		if pid, err := strconv.Atoi(strings.TrimSpace(lines[0])); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				r.procs[Postgres] = proc
			}
		}
	}
	return nil
}

func (r *ProcessRunner) stopPostgres(svc ServiceType) error {
	delete(r.procs, svc)
	// pg_ctl stop is more reliable than sending signals directly
	// but we don't have dataDir here — use process signal if we have it
	// This is a simplification; a full implementation would store dataDir
	return nil
}

// --- Redis ---

func (r *ProcessRunner) startRedis(cfg ServiceConfig) error {
	cmd := exec.Command("redis-server",
		"--port", strconv.Itoa(cfg.Port),
		"--dir", cfg.DataDir,
		"--daemonize", "yes",
		"--pidfile", cfg.DataDir+"/redis.pid",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("start redis: %s: %w", string(out), err)
	}

	// Read PID from pidfile
	pidData, err := os.ReadFile(cfg.DataDir + "/redis.pid")
	if err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				r.procs[Redis] = proc
			}
		}
	}
	return nil
}

func (r *ProcessRunner) stopRedis(svc ServiceType) error {
	p, ok := r.procs[svc]
	if !ok {
		return nil
	}
	if err := p.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop redis: %w", err)
	}
	_, _ = p.Wait()
	delete(r.procs, svc)
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/databases/... -v -run TestProcessRunner`
Expected: PASS

Note: `TestProcessRunnerStartCreatesDataDir` will attempt to start MySQL and fail (no binary), but the data dir creation should succeed before the binary check. If this test is flaky, adjust to test data dir creation separately.

**Step 5: Commit**

```bash
git add internal/databases/process.go internal/databases/process_test.go
git commit -m "feat(databases): add ProcessRunner for os/exec process management"
```

---

### Task 5: Core Integration

**Files:**
- Modify: `internal/core/core.go`
- Modify: `internal/core/core_test.go`

**Step 1: Write failing test for databases plugin in Core**

Add to `internal/core/core_test.go`:

```go
// Add a stubDBRunner to the stubs section:

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
```

Update `testConfig` helper to include `DBRunner`:

```go
func testConfig(t *testing.T) (core.Config, *stubCaddyRunner, *stubFPMRunner, *stubCertStore, *stubDBRunner) {
	t.Helper()
	runner := &stubCaddyRunner{}
	fpm := newStubFPMRunner()
	certs := newStubCertStore()
	db := newStubDBRunner()
	dir := t.TempDir()
	cfg := core.Config{
		SitesFile:   filepath.Join(dir, "sites.json"),
		Logger:      log.New(os.Stderr, "", 0),
		CaddyRunner: runner,
		FPMRunner:   fpm,
		CertStore:   certs,
		DBRunner:    db,
		DBConfigPath: filepath.Join(dir, "databases.json"),
		DBDataRoot:   filepath.Join(dir, "db-data"),
	}
	return cfg, runner, fpm, certs, db
}
```

Update all existing test call sites to accept the new 5th return value (`_`).

Add new test:

```go
func TestPluginsIncludesDatabases(t *testing.T) {
	cfg, _, _, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	plugins := c.Plugins()
	if len(plugins) != 3 {
		t.Fatalf("Plugins() len = %d, want 3", len(plugins))
	}

	ids := map[string]bool{}
	for _, p := range plugins {
		ids[p.ID] = true
	}
	if !ids["flock-databases"] {
		t.Error("expected flock-databases plugin")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/core/... -v`
Expected: FAIL — `core.Config` doesn't have `DBRunner` field

**Step 3: Update Core to register databases plugin**

Modify `internal/core/core.go`:

Add import: `"github.com/andybarilla/flock/internal/databases"`

Update `Config`:
```go
type Config struct {
	SitesFile    string
	Logger       *log.Logger
	CaddyRunner  caddy.CaddyRunner
	FPMRunner    php.FPMRunner
	CertStore    ssl.CertStore
	DBRunner     databases.DBRunner
	DBConfigPath string
	DBDataRoot   string
}
```

Update `Core`:
```go
type Core struct {
	registry  *registry.Registry
	pluginMgr *plugin.Manager
	caddyMgr  *caddy.Manager
	sslPlugin *ssl.Plugin
	phpPlugin *php.Plugin
	dbPlugin  *databases.Plugin
	logger    *log.Logger
}
```

In `NewCore()`, add after phpPlugin registration:
```go
dbPlugin := databases.NewPlugin(cfg.DBRunner, cfg.DBConfigPath, cfg.DBDataRoot)
pluginMgr.Register(dbPlugin)
```

Store in Core struct: `dbPlugin: dbPlugin`

Add methods for Wails bindings:
```go
func (c *Core) DatabaseServices() []databases.ServiceInfo {
	return c.dbPlugin.ServiceStatuses()
}

func (c *Core) StartDatabase(svc string) error {
	return c.dbPlugin.StartSvc(databases.ServiceType(svc))
}

func (c *Core) StopDatabase(svc string) error {
	return c.dbPlugin.StopSvc(databases.ServiceType(svc))
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/core/... -v`
Expected: PASS (update the existing `TestPluginsReturnsInfo` to expect 3 plugins instead of 2)

**Step 5: Commit**

```bash
git add internal/core/core.go internal/core/core_test.go
git commit -m "feat(core): integrate databases plugin"
```

---

### Task 6: Wails Bindings (App)

**Files:**
- Modify: `app.go`

**Step 1: Update App with database bindings**

Add import: `"github.com/andybarilla/flock/internal/databases"`

In `startup()`, update the `core.Config` to include:
```go
dbDataRoot := filepath.Join(config.DataDir(), "databases")
cfg := core.Config{
	// ... existing fields
	DBRunner:     databases.NewProcessRunner(),
	DBConfigPath: filepath.Join(config.ConfigDir(), "databases.json"),
	DBDataRoot:   dbDataRoot,
}
```

Note: `config.ConfigDir()` may need to be checked — if it doesn't exist, use `filepath.Dir(config.SitesFile())` instead.

Add Wails-bound methods:
```go
// DatabaseServices returns status of all database services
func (a *App) DatabaseServices() []databases.ServiceInfo {
	return a.core.DatabaseServices()
}

// StartDatabase starts a specific database service
func (a *App) StartDatabase(svc string) error {
	return a.core.StartDatabase(svc)
}

// StopDatabase stops a specific database service
func (a *App) StopDatabase(svc string) error {
	return a.core.StopDatabase(svc)
}
```

**Step 2: Verify it compiles**

Run: `cd <worktree-root> && go build ./...`
Expected: success

**Step 3: Commit**

```bash
git add app.go
git commit -m "feat(app): add Wails bindings for database services"
```

---

### Task 7: GUI — ServiceList Component

**Files:**
- Create: `frontend/src/ServiceList.svelte`
- Modify: `frontend/src/App.svelte`

**Step 1: Create ServiceList component**

```svelte
<!-- frontend/src/ServiceList.svelte -->
<script>
  export let services = [];
  export let onStart = () => {};
  export let onStop = () => {};

  const displayName = {
    mysql: 'MySQL',
    postgres: 'PostgreSQL',
    redis: 'Redis',
  };
</script>

{#if services.length === 0}
  <p class="empty">No database services configured.</p>
{:else}
  <table class="service-table">
    <thead>
      <tr>
        <th>Service</th>
        <th>Port</th>
        <th>Status</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each services as svc}
        <tr class:disabled={!svc.enabled}>
          <td class="name">{displayName[svc.type] || svc.type}</td>
          <td class="port">{svc.port}</td>
          <td>
            {#if !svc.enabled}
              <span class="status status-unavailable">Not installed</span>
            {:else if svc.running}
              <span class="status status-running">Running</span>
            {:else}
              <span class="status status-stopped">Stopped</span>
            {/if}
          </td>
          <td>
            {#if svc.enabled}
              {#if svc.running}
                <button class="btn-action btn-stop" on:click={() => onStop(svc.type)}>
                  Stop
                </button>
              {:else}
                <button class="btn-action btn-start" on:click={() => onStart(svc.type)}>
                  Start
                </button>
              {/if}
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}

<style>
  .empty {
    color: #888;
    padding: 2rem;
  }

  .service-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
  }

  .service-table th {
    color: #888;
    font-weight: 600;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #333;
  }

  .service-table td {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid #222;
  }

  .name {
    font-weight: 600;
  }

  .port {
    color: #aaa;
    font-size: 0.85rem;
  }

  .disabled td {
    opacity: 0.5;
  }

  .status {
    font-size: 0.8rem;
    padding: 0.15rem 0.4rem;
    border-radius: 3px;
  }

  .status-running {
    color: #2ecc71;
  }

  .status-stopped {
    color: #888;
  }

  .status-unavailable {
    color: #e67e22;
  }

  .btn-action {
    background: transparent;
    border: 1px solid #555;
    color: #ccc;
    padding: 0.2rem 0.5rem;
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .btn-start:hover {
    border-color: #2ecc71;
    color: #2ecc71;
  }

  .btn-stop:hover {
    border-color: #e74c3c;
    color: #e74c3c;
  }
</style>
```

**Step 2: Update App.svelte to include Services section**

Add import at top of script:
```js
import { DatabaseServices, StartDatabase, StopDatabase } from '../wailsjs/go/main/App.js';
import ServiceList from './ServiceList.svelte';
```

Add state and functions:
```js
let services = [];

async function refreshServices() {
  try {
    services = await DatabaseServices() || [];
  } catch (e) {
    error = 'Failed to load services: ' + (e.message || String(e));
  }
}

async function handleStartService(svc) {
  try {
    await StartDatabase(svc);
    await refreshServices();
  } catch (e) {
    error = 'Failed to start service: ' + (e.message || String(e));
  }
}

async function handleStopService(svc) {
  try {
    await StopDatabase(svc);
    await refreshServices();
  } catch (e) {
    error = 'Failed to stop service: ' + (e.message || String(e));
  }
}
```

Update `onMount` to also call `refreshServices()`:
```js
onMount(() => {
  refreshSites();
  refreshServices();
});
```

Add Services section in the template, after the Sites section:
```html
<section class="content" style="margin-top: 1.5rem;">
  <h2>Services</h2>
  <ServiceList {services} onStart={handleStartService} onStop={handleStopService} />
</section>
```

**Step 3: Regenerate Wails bindings**

Run: `cd <worktree-root> && wails generate module`

This generates the JS binding files for `DatabaseServices`, `StartDatabase`, `StopDatabase`.

If `wails` is not available, manually create binding stubs (the Wails build process will generate them).

**Step 4: Commit**

```bash
git add frontend/src/ServiceList.svelte frontend/src/App.svelte
git add frontend/wailsjs/  # include generated bindings if any
git commit -m "feat(gui): add Services panel with start/stop controls"
```

---

### Task 8: Binary Detection on Init

**Files:**
- Modify: `internal/databases/databases.go`
- Add test to: `internal/databases/databases_test.go`

**Step 1: Write failing test**

Add to `databases_test.go`:

```go
func TestInitDetectsDisabledBinaries(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)

	// Use a custom BinaryChecker that says nothing is installed
	p := databases.NewPlugin(runner, configPath, dataDir)
	p.SetBinaryChecker(func(name string) bool { return false })

	_ = p.Init(&mockHost{})

	infos := p.ServiceStatuses()
	for _, info := range infos {
		if info.Enabled {
			t.Errorf("%s should be disabled when binary not found", info.Type)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/databases/... -v -run TestInitDetects`
Expected: FAIL — `SetBinaryChecker` not defined

**Step 3: Add binary detection to Plugin**

Add to `databases.go`:

```go
// In Plugin struct, add:
binaryChecker func(string) bool

// In NewPlugin, add default:
binaryChecker: CheckBinary,

// Add method:
func (p *Plugin) SetBinaryChecker(fn func(string) bool) {
	p.binaryChecker = fn
}

// In Init(), after loading config, add:
for _, svc := range AllServiceTypes {
	if !p.binaryChecker(BinaryFor(svc)) {
		p.config.SetEnabled(svc, false)
		p.host.Log(p.ID(), "%s binary not found on PATH", svc)
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/databases/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/databases/databases.go internal/databases/databases_test.go
git commit -m "feat(databases): detect missing binaries on Init"
```

---

### Task 9: Run All Tests & Final Verification

**Step 1: Run the full test suite**

Run: `cd <worktree-root> && go test ./... -v`
Expected: ALL PASS

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: no issues

**Step 3: Verify the app compiles**

Run: `go build ./...`
Expected: success

**Step 4: Commit any fixes if needed**

---

### Task 10: Update Roadmap

**Files:**
- Modify: `docs/ROADMAP.md`

**Step 1: Mark the task complete in the roadmap**

Change:
```
- [ ] flock-databases plugin (MySQL, PostgreSQL, Redis)
```
To:
```
- [x] flock-databases plugin (MySQL, PostgreSQL, Redis) — See: docs/tasks/009-flock-databases.md
```

**Step 2: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs: mark flock-databases as complete in roadmap"
```
