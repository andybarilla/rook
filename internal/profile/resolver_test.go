package profile_test

import (
	"sort"
	"testing"

	"github.com/andybarilla/rook/internal/profile"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestResolve_DefaultProfile(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{"postgres": {Image: "postgres:16"}, "app": {Command: "air"}},
		Groups:   map[string][]string{"infra": {"postgres"}},
		Profiles: map[string][]string{"default": {"infra", "app"}},
	}
	services, err := profile.Resolve(ws, "default")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(services)
	if len(services) != 2 {
		t.Fatalf("expected 2, got %d", len(services))
	}
	if services[0] != "app" || services[1] != "postgres" {
		t.Errorf("unexpected: %v", services)
	}
}

func TestResolve_WildcardProfile(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{"postgres": {}, "redis": {}, "app": {}},
		Groups:   map[string][]string{"infra": {"postgres", "redis"}},
		Profiles: map[string][]string{"all": {"infra", "*"}},
	}
	services, _ := profile.Resolve(ws, "all")
	if len(services) != 3 {
		t.Errorf("expected 3, got %d", len(services))
	}
}

func TestResolve_ImplicitAllProfile(t *testing.T) {
	ws := workspace.Workspace{Services: map[string]workspace.Service{"a": {}, "b": {}, "c": {}}}
	services, _ := profile.Resolve(ws, "all")
	if len(services) != 3 {
		t.Fatalf("expected 3, got %d", len(services))
	}
}

func TestResolve_UnknownProfile(t *testing.T) {
	ws := workspace.Workspace{Services: map[string]workspace.Service{"a": {}}}
	_, err := profile.Resolve(ws, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolve_Deduplication(t *testing.T) {
	ws := workspace.Workspace{
		Services: map[string]workspace.Service{"postgres": {}, "app": {}},
		Groups:   map[string][]string{"infra": {"postgres"}},
		Profiles: map[string][]string{"dupe": {"infra", "postgres", "app"}},
	}
	services, _ := profile.Resolve(ws, "dupe")
	if len(services) != 2 {
		t.Errorf("expected 2 deduped, got %d", len(services))
	}
}
