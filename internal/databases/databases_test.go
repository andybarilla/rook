package databases_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/databases"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// --- Mock DBRunner ---

type mockDBRunner struct {
	started  map[databases.ServiceType]databases.ServiceConfig
	stopped  map[databases.ServiceType]bool
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

func (m *mockHost) Sites() []registry.Site { return m.sites }
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

func (h *loggingHost) Sites() []registry.Site { return h.sites }
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

func newTestPlugin(runner databases.DBRunner, configPath, dataDir string) *databases.Plugin {
	p := databases.NewPlugin(runner, configPath, dataDir)
	p.SetBinaryChecker(func(string) bool { return true })
	return p
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
	p := newTestPlugin(runner, configPath, dataDir)

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

	p := newTestPlugin(runner, configPath, dataDir)
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

	p := newTestPlugin(runner, configPath, dataDir)
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

	p := newTestPlugin(runner, configPath, dataDir)
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
	p := newTestPlugin(runner, configPath, dataDir)
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
	p := newTestPlugin(runner, configPath, dataDir)
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
	p := newTestPlugin(runner, configPath, dataDir)
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
	p := newTestPlugin(runner, configPath, dataDir)

	host := &loggingHost{}
	_ = p.Init(host)

	// StartSvc should return the error
	err := p.StartSvc(databases.MySQL)
	if err == nil {
		t.Error("expected error from StartSvc when runner fails")
	}
}

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

func TestServiceStatusesDetectsCrashedProcess(t *testing.T) {
	runner := newMockDBRunner()
	configPath, dataDir := tmpConfigDir(t)
	p := newTestPlugin(runner, configPath, dataDir)
	_ = p.Init(&mockHost{})

	// Start MySQL
	if err := p.StartSvc(databases.MySQL); err != nil {
		t.Fatalf("StartSvc: %v", err)
	}

	// Verify it shows as running
	infos := p.ServiceStatuses()
	for _, info := range infos {
		if info.Type == databases.MySQL && !info.Running {
			t.Fatal("MySQL should be running after StartSvc")
		}
	}

	// Simulate the process dying externally
	runner.statuses[databases.MySQL] = databases.StatusStopped

	// ServiceStatuses should now detect it as stopped
	infos = p.ServiceStatuses()
	for _, info := range infos {
		if info.Type == databases.MySQL && info.Running {
			t.Error("MySQL should show as stopped after process died")
		}
	}

	// ServiceStatus (aggregate) should also reflect stopped
	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Error("aggregate ServiceStatus should be Stopped when process crashed")
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
	p := newTestPlugin(runner, configPath, dataDir)
	_ = p.Init(host)

	// Start should not return error — it logs and continues
	if err := p.Start(); err != nil {
		t.Fatalf("Start should not error on autostart failure: %v", err)
	}

	if !host.logged {
		t.Error("expected autostart failure to be logged")
	}
}
