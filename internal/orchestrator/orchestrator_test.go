package orchestrator_test

import (
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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
	if err := orch.Up(context.Background(), ws, "default"); err != nil {
		t.Fatal(err)
	}
	if len(mock.started) != 2 {
		t.Fatalf("expected 2, got %d", len(mock.started))
	}
	if mock.started[0] != "postgres" {
		t.Errorf("expected postgres first, got %s", mock.started[0])
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
	result := alloc.Get("test", "app")
	if !result.OK {
		t.Fatal("expected port allocated")
	}
	if result.Port == 0 {
		t.Errorf("expected non-zero port, got %d", result.Port)
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
	orch.Up(context.Background(), ws, "full")
	if len(mock.started) != 1 || mock.started[0] != "redis" {
		t.Errorf("expected only redis to start, got %v", mock.started)
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
	err := orch.Up(context.Background(), ws, "default")
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
	if exec.Command("docker", "info").Run() != nil {
		t.Skip("docker not available")
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
