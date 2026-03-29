package api_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/api"
	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/workspace"
	"gopkg.in/yaml.v3"
)

// stubRegistry implements registry.Registry for testing.
type stubRegistry struct{}

func (r *stubRegistry) Register(name, path string) error        { return nil }
func (r *stubRegistry) Remove(name string)                      {}
func (r *stubRegistry) Get(name string) (registry.Entry, error) { return registry.Entry{}, nil }
func (r *stubRegistry) List() []registry.Entry                  { return nil }

// stubPortAlloc implements ports.PortAllocator for testing.
type stubPortAlloc struct{}

func (s *stubPortAlloc) Allocate(workspace, service string) (int, error) {
	return 10000, nil
}
func (s *stubPortAlloc) AllocatePinned(workspace, service string, port int) (int, error) {
	return port, nil
}
func (s *stubPortAlloc) Release(workspace, service string) error { return nil }
func (s *stubPortAlloc) Get(workspace, service string) ports.LookupResult {
	return ports.LookupResult{}
}
func (s *stubPortAlloc) All() []ports.PortEntry { return nil }

func newTestAPI() *api.WorkspaceAPI {
	reg := &stubRegistry{}
	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	return api.NewWorkspaceAPI(reg, alloc, orch, nil)
}

func TestListWorkspaces_Empty(t *testing.T) {
	a := newTestAPI()
	result := a.ListWorkspaces()
	if len(result) != 0 {
		t.Fatalf("expected 0 workspaces, got %d", len(result))
	}
}

func TestGetPorts_Empty(t *testing.T) {
	a := newTestAPI()
	result := a.GetPorts()
	if len(result) != 0 {
		t.Fatalf("expected 0 ports, got %d", len(result))
	}
}

func TestGetLogs(t *testing.T) {
	a := newTestAPI()
	a.BufferLog("ws1", "svc1", "hello")
	a.BufferLog("ws1", "svc1", "world")

	lines, err := a.GetLogs("ws1", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestPreviewManifest(t *testing.T) {
	a := newTestAPI()
	m := &workspace.Manifest{
		Name: "test",
		Type: workspace.TypeMulti,
		Services: map[string]workspace.Service{
			"web": {Command: "npm start", Ports: []int{3000}},
		},
	}

	result, err := a.PreviewManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	// Verify it's valid YAML by parsing it back
	var parsed workspace.Manifest
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid YAML: %v", err)
	}
	if parsed.Name != "test" {
		t.Fatalf("expected name 'test', got %q", parsed.Name)
	}
}

func TestSettingsTypes_Exist(t *testing.T) {
	s := api.Settings{AutoRebuild: true}
	if !s.AutoRebuild {
		t.Error("AutoRebuild should be true")
	}

	bs := api.BuildStatus{
		Name:     "web",
		HasBuild: true,
		Status:   "needs_rebuild",
		Reasons:  []string{"Dockerfile modified"},
	}
	if bs.Name != "web" {
		t.Error("BuildStatus name mismatch")
	}

	bcr := api.BuildCheckResult{
		Services: []api.BuildStatus{bs},
		HasStale: true,
	}
	if !bcr.HasStale {
		t.Error("BuildCheckResult.HasStale should be true")
	}
}

func TestGetSettings_ReturnsDefaults(t *testing.T) {
	a := newTestAPI()
	s := a.GetSettings()
	if !s.AutoRebuild {
		t.Error("expected AutoRebuild to be true by default")
	}
}

func TestSaveSettings_PersistsSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	reg := &stubRegistry{}
	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, nil, path)

	// Modify and save
	s := a.GetSettings()
	s.AutoRebuild = false
	if err := a.SaveSettings(s); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	// Verify file was written
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	// Load fresh and verify
	a2 := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, nil, path)
	loaded := a2.GetSettings()
	if loaded.AutoRebuild {
		t.Error("expected AutoRebuild to be false after save")
	}
}

func TestCheckBuilds_MethodExists(t *testing.T) {
	// This test verifies that the CheckBuilds method exists and has the correct signature
	a := newTestAPI()

	// This should compile - if the method doesn't exist, this will fail
	var _ func(string) (*api.BuildCheckResult, error) = a.CheckBuilds
}

func TestCheckBuilds_EmptyWorkspace(t *testing.T) {
	// This test verifies that CheckBuilds works with an empty workspace
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	reg, err := registry.NewFileRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	// Register a workspace
	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	// Create rook.yaml
	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{}, // empty services
	}
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644); err != nil {
		t.Fatal(err)
	}

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)

	// Create API
	a := api.NewWorkspaceAPI(reg, alloc, orch, nil)

	result, err := a.CheckBuilds("myproject")
	if err != nil {
		t.Fatalf("CheckBuilds failed: %v", err)
	}

	if len(result.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(result.Services))
	}
	if result.HasStale {
		t.Error("expected HasStale to be false for empty workspace")
	}
}

func TestResetPorts_ClearsPortsFile(t *testing.T) {
	dir := t.TempDir()
	portsPath := filepath.Join(dir, "ports.json")
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	// Create ports file with some content
	os.WriteFile(portsPath, []byte(`[{"workspace":"test","service":"web","port":3000}]`), 0644)

	reg, _ := registry.NewFileRegistry(registryPath)
	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, nil, settingsPath, portsPath)

	err := a.ResetPorts()
	if err != nil {
		t.Fatalf("ResetPorts failed: %v", err)
	}

	// Verify ports file was deleted
	if _, err := os.Stat(portsPath); !os.IsNotExist(err) {
		t.Error("expected ports file to be deleted")
	}
}

func TestDiscoverWorkspace_NoChanges(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, nil, settingsPath)

	diff, err := a.DiscoverWorkspace("myproject")
	if err != nil {
		t.Fatalf("DiscoverWorkspace failed: %v", err)
	}

	if diff.HasChanges {
		t.Error("expected no changes for empty workspace")
	}
}

func TestDiscoverWorkspace_DetectsNewServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "3000:80"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, discoverers, settingsPath)

	diff, err := a.DiscoverWorkspace("myproject")
	if err != nil {
		t.Fatalf("DiscoverWorkspace failed: %v", err)
	}

	if !diff.HasChanges {
		t.Error("expected changes detected")
	}
	if len(diff.NewServices) != 1 {
		t.Errorf("expected 1 new service, got %d", len(diff.NewServices))
	}
	if diff.NewServices[0].Name != "web" {
		t.Errorf("expected new service 'web', got %s", diff.NewServices[0].Name)
	}
}

func TestDiscoverWorkspace_DetectsRemovedServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name: "myproject",
		Type: workspace.TypeMulti,
		Services: map[string]workspace.Service{
			"old-service": {Image: "nginx:old"},
		},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIWithSettings(reg, alloc, orch, discoverers, settingsPath)

	diff, err := a.DiscoverWorkspace("myproject")
	if err != nil {
		t.Fatalf("DiscoverWorkspace failed: %v", err)
	}

	if !diff.HasChanges {
		t.Error("expected changes detected")
	}
	if len(diff.RemovedServices) != 1 {
		t.Errorf("expected 1 removed service, got %d", len(diff.RemovedServices))
	}
	if diff.RemovedServices[0].Name != "old-service" {
		t.Errorf("expected removed service 'old-service', got %s", diff.RemovedServices[0].Name)
	}
}

func TestApplyDiscovery_AddsServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	portsPath := filepath.Join(dir, "ports.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "3000:80"
  db:
    image: postgres:15
    ports:
      - "5432:5432"
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, discoverers, settingsPath, portsPath)

	err := a.ApplyDiscovery("myproject", []string{"web"}, []string{})
	if err != nil {
		t.Fatalf("ApplyDiscovery failed: %v", err)
	}

	manifestPath := filepath.Join(wsDir, "rook.yaml")
	updated, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("parsing updated manifest: %v", err)
	}

	if _, exists := updated.Services["web"]; !exists {
		t.Error("expected web service to be added to manifest")
	}
}

func TestApplyDiscovery_RemovesServices(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	portsPath := filepath.Join(dir, "ports.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name: "myproject",
		Type: workspace.TypeMulti,
		Services: map[string]workspace.Service{
			"old-service": {Image: "nginx:old"},
		},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, discoverers, settingsPath, portsPath)

	err := a.ApplyDiscovery("myproject", []string{}, []string{"old-service"})
	if err != nil {
		t.Fatalf("ApplyDiscovery failed: %v", err)
	}

	manifestPath := filepath.Join(wsDir, "rook.yaml")
	updated, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("parsing updated manifest: %v", err)
	}

	if _, exists := updated.Services["old-service"]; exists {
		t.Error("expected old-service to be removed from manifest")
	}
}

func TestApplyDiscovery_RejectsInvalidServiceNames(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")
	portsPath := filepath.Join(dir, "ports.json")
	settingsPath := filepath.Join(dir, "settings.json")

	reg, _ := registry.NewFileRegistry(registryPath)

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	composeContent := `
services:
  web:
    image: nginx:latest
`
	os.WriteFile(filepath.Join(wsDir, "docker-compose.yml"), []byte(composeContent), 0644)

	manifest := &workspace.Manifest{
		Name:     "myproject",
		Type:     workspace.TypeMulti,
		Services: map[string]workspace.Service{},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc, _ := ports.NewFileAllocator(portsPath, 10000, 60000)
	orch := orchestrator.New(nil, nil, alloc)
	discoverers := []discovery.Discoverer{discovery.NewComposeDiscoverer()}
	a := api.NewWorkspaceAPIFull(reg, alloc, orch, discoverers, settingsPath, portsPath)

	err := a.ApplyDiscovery("myproject", []string{"nonexistent"}, []string{})
	if err == nil {
		t.Error("expected error for nonexistent service name")
	}
}

func TestStartLogStream_BuffersLines(t *testing.T) {
	a := newTestAPI()
	r := io.NopCloser(strings.NewReader("line one\nline two\nline three\n"))
	a.StreamFromReader("ws", "svc", r)

	time.Sleep(100 * time.Millisecond)

	logs, err := a.GetLogs("ws", "svc", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(logs))
	}
	if logs[0].Line != "line one" {
		t.Fatalf("expected 'line one', got %q", logs[0].Line)
	}
	if logs[2].Line != "line three" {
		t.Fatalf("expected 'line three', got %q", logs[2].Line)
	}
}

func TestStopLogStream_CancelsReader(t *testing.T) {
	a := newTestAPI()
	pr, pw := io.Pipe()
	a.StreamFromReader("ws", "svc", pr)

	pw.Write([]byte("hello\n"))
	time.Sleep(50 * time.Millisecond)

	a.StopLogStream("ws", "svc")
	time.Sleep(50 * time.Millisecond)

	pw.Close()

	logs, _ := a.GetLogs("ws", "svc", 10)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logs))
	}
}

func TestReconnectWorkspace_ErrorsWithNoRegistry(t *testing.T) {
	a := newTestAPI()
	err := a.ReconnectWorkspace("nonexistent")
	if err == nil {
		t.Fatal("expected error for unregistered workspace")
	}
}

func TestGetManifest_ReturnsFullManifest(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	reg, err := registry.NewFileRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	manifest := &workspace.Manifest{
		Name: "myproject",
		Type: workspace.TypeMulti,
		Services: map[string]workspace.Service{
			"web": {
				Command:    "npm start",
				Ports:      []int{3000},
				DependsOn:  []string{"db"},
				EnvFile:    ".env",
				WorkingDir: "/app",
				Environment: map[string]string{
					"DB_URL": "postgres://{{.Host.db}}:{{.Port.db}}/app",
				},
			},
			"db": {
				Image:   "postgres:16",
				Volumes: []string{"pg-data:/var/lib/postgresql/data"},
				Healthcheck: "pg_isready -U app",
			},
		},
		Groups: map[string][]string{
			"infra": {"db"},
		},
		Profiles: map[string][]string{
			"default": {"*"},
		},
	}
	manifestData, _ := yaml.Marshal(manifest)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	a := api.NewWorkspaceAPI(reg, alloc, orch, nil)

	got, err := a.GetManifest("myproject")
	if err != nil {
		t.Fatalf("GetManifest failed: %v", err)
	}

	if got.Name != "myproject" {
		t.Errorf("expected name 'myproject', got %q", got.Name)
	}
	if len(got.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(got.Services))
	}
	web := got.Services["web"]
	if web.Command != "npm start" {
		t.Errorf("expected command 'npm start', got %q", web.Command)
	}
	if web.EnvFile != ".env" {
		t.Errorf("expected env_file '.env', got %q", web.EnvFile)
	}
	if web.WorkingDir != "/app" {
		t.Errorf("expected working_dir '/app', got %q", web.WorkingDir)
	}
	db := got.Services["db"]
	if len(db.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(db.Volumes))
	}
	if len(got.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(got.Groups))
	}
	if len(got.Profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(got.Profiles))
	}
}

func TestGetManifest_SaveManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "workspaces.json")

	reg, err := registry.NewFileRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	wsDir := filepath.Join(dir, "myproject")
	os.MkdirAll(wsDir, 0755)

	original := &workspace.Manifest{
		Name: "myproject",
		Type: workspace.TypeSingle,
		Services: map[string]workspace.Service{
			"app": {Command: "go run .", Ports: []int{8080}},
		},
	}
	manifestData, _ := yaml.Marshal(original)
	os.WriteFile(filepath.Join(wsDir, "rook.yaml"), manifestData, 0644)

	reg.Register("myproject", wsDir)

	alloc := &stubPortAlloc{}
	orch := orchestrator.New(nil, nil, nil)
	a := api.NewWorkspaceAPI(reg, alloc, orch, nil)

	// Get, modify, save, get again
	got, err := a.GetManifest("myproject")
	if err != nil {
		t.Fatal(err)
	}

	svc := got.Services["app"]
	svc.Ports = []int{8080, 9090}
	got.Services["app"] = svc
	got.Services["db"] = workspace.Service{Image: "postgres:16"}

	if err := a.SaveManifest("myproject", got); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	got2, err := a.GetManifest("myproject")
	if err != nil {
		t.Fatal(err)
	}

	if len(got2.Services) != 2 {
		t.Errorf("expected 2 services after save, got %d", len(got2.Services))
	}
	if len(got2.Services["app"].Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(got2.Services["app"].Ports))
	}
	if got2.Services["db"].Image != "postgres:16" {
		t.Errorf("expected db image 'postgres:16', got %q", got2.Services["db"].Image)
	}
}

func TestStartWorkspace_AcceptsForceBuild(t *testing.T) {
	// Test that the signature accepts the forceBuild parameter
	a := newTestAPI()

	// This should compile - if the signature doesn't have forceBuild, it won't
	_ = func() {
		var _ func(string, string, bool) error = a.StartWorkspace
	}
}
