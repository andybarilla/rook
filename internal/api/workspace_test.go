package api_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/api"
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

func (s *stubPortAlloc) Allocate(workspace, service string, preferred int) (int, error) {
	return preferred, nil
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
