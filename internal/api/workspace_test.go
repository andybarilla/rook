package api_test

import (
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

func (r *stubRegistry) Register(name, path string) error          { return nil }
func (r *stubRegistry) Remove(name string)                        {}
func (r *stubRegistry) Get(name string) (registry.Entry, error)   { return registry.Entry{}, nil }
func (r *stubRegistry) List() []registry.Entry                    { return nil }

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
