package orchestrator_test

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

type mockRunner struct {
	mu      sync.Mutex
	started []string
	stopped []string
}

func (m *mockRunner) Start(ctx context.Context, name string, svc workspace.Service, ports runner.PortMap, workDir string) (runner.RunHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = append(m.started, name)
	return runner.RunHandle{ID: name, Type: "mock"}, nil
}

func (m *mockRunner) Stop(handle runner.RunHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = append(m.stopped, handle.ID)
	return nil
}

func (m *mockRunner) Status(handle runner.RunHandle) (runner.ServiceStatus, error) {
	return runner.StatusRunning, nil
}

func (m *mockRunner) Logs(handle runner.RunHandle) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("mock")), nil
}

func TestOrchestrator_Up(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air", DependsOn: []string{"postgres"}},
		},
		Profiles: map[string][]string{"default": {"postgres", "app"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	result, err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.started) != 2 {
		t.Fatalf("expected 2, got %d", len(mock.started))
	}
	if mock.started[0] != "postgres" {
		t.Errorf("expected postgres first, got %s", mock.started[0])
	}
	if len(result.Started) != 2 {
		t.Errorf("expected 2 started, got %d", len(result.Started))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(result.Skipped))
	}
}

func TestOrchestrator_Up_WithPorts(t *testing.T) {
	mock := &mockRunner{}
	dir := t.TempDir()
	alloc, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10100)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{"app": {Command: "air", Ports: []int{8080}}},
		Profiles: map[string][]string{"default": {"app"}},
	}
	orch := orchestrator.New(mock, mock, alloc)
	orch.Up(context.Background(), ws, "default")
	allocResult := alloc.Get("test", "app")
	if !allocResult.OK {
		t.Fatal("expected port allocated")
	}
	if allocResult.Port == 0 {
		t.Errorf("expected non-zero port, got %d", allocResult.Port)
	}
}

func TestOrchestrator_IncrementalProfileSwitch(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"}, "redis": {Image: "redis:7"}, "app": {Command: "air"},
		},
		Profiles: map[string][]string{
			"backend": {"postgres", "app"}, "full": {"postgres", "redis", "app"},
		},
	}
	orch := orchestrator.New(mock, mock, nil)
	orch.Up(context.Background(), ws, "backend")
	if len(mock.started) != 2 {
		t.Fatalf("expected 2, got %d", len(mock.started))
	}
	mock.started = nil
	result, _ := orch.Up(context.Background(), ws, "full")
	if len(mock.started) != 1 || mock.started[0] != "redis" {
		t.Errorf("expected only redis to start, got %v", mock.started)
	}
	if len(result.Started) != 1 || result.Started[0] != "redis" {
		t.Errorf("expected redis in result.Started, got %v", result.Started)
	}
	if len(result.Skipped) != 2 {
		t.Errorf("expected 2 skipped (postgres, app), got %d", len(result.Skipped))
	}
}

func TestOrchestrator_StartService(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air", DependsOn: []string{"postgres"}},
		},
	}
	orch := orchestrator.New(mock, mock, nil)
	err := orch.StartService(context.Background(), ws, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.started) != 1 || mock.started[0] != "postgres" {
		t.Errorf("expected [postgres], got %v", mock.started)
	}
}

func TestOrchestrator_StopService(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air"},
		},
		Profiles: map[string][]string{"default": {"postgres", "app"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	orch.Up(context.Background(), ws, "default")
	mock.stopped = nil
	err := orch.StopService(context.Background(), ws, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.stopped) != 1 || mock.stopped[0] != "postgres" {
		t.Errorf("expected [postgres], got %v", mock.stopped)
	}
}

func TestOrchestrator_RestartService(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{"app": {Command: "air"}},
		Profiles: map[string][]string{"default": {"app"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	orch.Up(context.Background(), ws, "default")
	mock.started = nil
	mock.stopped = nil
	err := orch.RestartService(context.Background(), ws, "app")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.stopped) != 1 {
		t.Errorf("expected 1 stopped, got %d", len(mock.stopped))
	}
	if len(mock.started) != 1 {
		t.Errorf("expected 1 started, got %d", len(mock.started))
	}
}

func TestOrchestrator_StartService_Unknown(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{Name: "test", Root: t.TempDir(), Services: map[string]workspace.Service{}}
	orch := orchestrator.New(mock, mock, nil)
	err := orch.StartService(context.Background(), ws, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOrchestrator_Up_WaitsForHealthCheck(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16", Healthcheck: "echo ok"},
			"app":      {Command: "air", DependsOn: []string{"postgres"}},
		},
		Profiles: map[string][]string{"default": {"postgres", "app"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	_, err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.started) != 2 {
		t.Fatalf("expected 2, got %d", len(mock.started))
	}
	if mock.started[0] != "postgres" {
		t.Errorf("expected postgres first")
	}
}

func TestOrchestrator_Down(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{"postgres": {Image: "postgres:16"}, "app": {Command: "air"}},
		Profiles: map[string][]string{"default": {"postgres", "app"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	orch.Up(context.Background(), ws, "default")
	orch.Down(context.Background(), ws)
	if len(mock.stopped) != 2 {
		t.Fatalf("expected 2 stopped, got %d", len(mock.stopped))
	}
}

func TestOrchestrator_Up_SkipsRunning(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"postgres": {Image: "postgres:16"},
			"app":      {Command: "air", DependsOn: []string{"postgres"}},
		},
		Profiles: map[string][]string{"default": {"postgres", "app"}},
	}
	orch := orchestrator.New(mock, mock, nil)

	// First Up: should start both
	result1, err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(result1.Started) != 2 {
		t.Errorf("expected 2 started on first Up, got %d", len(result1.Started))
	}
	if len(result1.Skipped) != 0 {
		t.Errorf("expected 0 skipped on first Up, got %d", len(result1.Skipped))
	}

	mock.started = nil

	// Second Up: should skip both
	result2, err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(result2.Started) != 0 {
		t.Errorf("expected 0 started on second Up, got %d", len(result2.Started))
	}
	if len(result2.Skipped) != 2 {
		t.Errorf("expected 2 skipped on second Up, got %d", len(result2.Skipped))
	}
	if len(mock.started) != 0 {
		t.Errorf("mock should have 0 starts on second Up, got %d", len(mock.started))
	}
}

// mockReconnectable implements runner.Reconnectable for testing.
type mockReconnectable struct {
	mockRunner
	prefix  string
	adopted []string
}

func (m *mockReconnectable) Prefix() string { return m.prefix }
func (m *mockReconnectable) Adopt(serviceName string) runner.RunHandle {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adopted = append(m.adopted, serviceName)
	return runner.RunHandle{ID: serviceName, Type: "docker"}
}

func TestOrchestrator_Reconnect_NonReconnectable(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{"app": {Command: "air"}},
	}
	err := orch.Reconnect(ws)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrchestrator_Reconnect_NoContainers(t *testing.T) {
	runner.DetectRuntime()
	if exec.Command(runner.ContainerRuntime, "info").Run() != nil {
		t.Skip("container runtime not available")
	}
	mock := &mockReconnectable{prefix: "rook_test"}
	orch := orchestrator.New(mock, &mock.mockRunner, nil)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{"postgres": {Image: "postgres:16"}},
	}
	err := orch.Reconnect(ws)
	if err != nil {
		t.Fatal(err)
	}
	statuses, _ := orch.Status(ws)
	if statuses["postgres"] != runner.StatusStopped {
		t.Errorf("expected stopped, got %s", statuses["postgres"])
	}
}

func TestOrchestrator_Reconnect_ProcessServices(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	wsRoot := t.TempDir()
	pidDir := runner.PIDDirPath(wsRoot)
	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	containerMock := &mockRunner{}
	processRunner := runner.NewProcessRunner()
	orch := orchestrator.New(containerMock, processRunner, nil)

	ws := workspace.Workspace{
		Name: "test", Root: wsRoot,
		Services: map[string]workspace.Service{
			"worker": {Command: "sleep 60"},
		},
	}

	if err := orch.Reconnect(ws); err != nil {
		t.Fatal(err)
	}

	statuses, _ := orch.Status(ws)
	if statuses["worker"] != runner.StatusRunning {
		t.Errorf("expected running, got %s", statuses["worker"])
	}
}

type crashingMockRunner struct {
	mockRunner
	crashNames map[string]bool
	logOutput  string
}

func (m *crashingMockRunner) Status(handle runner.RunHandle) (runner.ServiceStatus, error) {
	if m.crashNames[handle.ID] {
		return runner.StatusCrashed, nil
	}
	return runner.StatusRunning, nil
}

func (m *crashingMockRunner) Logs(handle runner.RunHandle) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.logOutput)), nil
}

func TestOrchestrator_Up_DetectsProcessCrash(t *testing.T) {
	crash := &crashingMockRunner{
		crashNames: map[string]bool{"worker": true},
		logOutput:  "Error: missing DATABASE_URL",
	}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"worker": {Command: "node worker.js"},
		},
		Profiles: map[string][]string{"default": {"worker"}},
	}
	orch := orchestrator.New(crash, crash, nil)
	_, err := orch.Up(context.Background(), ws, "default")
	if err == nil {
		t.Fatal("expected error for crashed process")
	}
	if !strings.Contains(err.Error(), "crashed immediately") {
		t.Errorf("expected crash message, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "missing DATABASE_URL") {
		t.Errorf("expected log output in error, got: %s", err.Error())
	}
}

func TestOrchestrator_Up_DetectsContainerCrash(t *testing.T) {
	crash := &crashingMockRunner{
		crashNames: map[string]bool{"db": true},
		logOutput:  "FATAL: password authentication failed",
	}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"db": {Image: "postgres:16"},
		},
		Profiles: map[string][]string{"default": {"db"}},
	}
	orch := orchestrator.New(crash, crash, nil)
	_, err := orch.Up(context.Background(), ws, "default")
	if err == nil {
		t.Fatal("expected error for crashed container")
	}
	if !strings.Contains(err.Error(), "crashed immediately") {
		t.Errorf("expected crash message, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "password authentication failed") {
		t.Errorf("expected log output in error, got: %s", err.Error())
	}
}

func TestOrchestrator_Up_HealthyProcessPasses(t *testing.T) {
	mock := &mockRunner{}
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"worker": {Command: "node worker.js"},
		},
		Profiles: map[string][]string{"default": {"worker"}},
	}
	orch := orchestrator.New(mock, mock, nil)
	_, err := orch.Up(context.Background(), ws, "default")
	if err != nil {
		t.Fatalf("healthy process should not error: %v", err)
	}
}

func TestOrchestrator_StreamServiceLogs_NoWorkspace(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	_, err := orch.StreamServiceLogs("nonexistent", "svc")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
}

func TestOrchestrator_StreamServiceLogs_NoHandle(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"svc": {Command: "echo hi"},
		},
		Profiles: map[string][]string{"default": {"svc"}},
	}
	if _, err := orch.Up(context.Background(), ws, "default"); err != nil {
		t.Fatal(err)
	}
	_, err := orch.StreamServiceLogs("test", "other")
	if err == nil {
		t.Fatal("expected error for missing handle")
	}
}

func TestOrchestrator_StreamServiceLogs_UnsupportedRunner(t *testing.T) {
	mock := &mockRunner{}
	orch := orchestrator.New(mock, mock, nil)
	ws := workspace.Workspace{
		Name: "test", Root: t.TempDir(),
		Services: map[string]workspace.Service{
			"svc": {Command: "echo hi"},
		},
		Profiles: map[string][]string{"default": {"svc"}},
	}
	if _, err := orch.Up(context.Background(), ws, "default"); err != nil {
		t.Fatal(err)
	}
	_, err := orch.StreamServiceLogs("test", "svc")
	if err == nil {
		t.Fatal("expected error for unsupported runner type")
	}
}

func TestOrchestrator_Reconnect_SkipsDeadProcesses(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Kill()
	cmd.Wait()

	wsRoot := t.TempDir()
	pidDir := runner.PIDDirPath(wsRoot)
	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	containerMock := &mockRunner{}
	processRunner := runner.NewProcessRunner()
	orch := orchestrator.New(containerMock, processRunner, nil)

	ws := workspace.Workspace{
		Name: "test", Root: wsRoot,
		Services: map[string]workspace.Service{
			"worker": {Command: "sleep 60"},
		},
	}

	if err := orch.Reconnect(ws); err != nil {
		t.Fatal(err)
	}

	statuses, _ := orch.Status(ws)
	if statuses["worker"] != runner.StatusStopped {
		t.Errorf("expected stopped, got %s", statuses["worker"])
	}
}
