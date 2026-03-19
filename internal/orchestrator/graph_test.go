package orchestrator_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestTopoSort_Simple(t *testing.T) {
	services := map[string]workspace.Service{
		"postgres": {},
		"app":      {DependsOn: []string{"postgres"}},
	}
	order, err := orchestrator.TopoSort(services, []string{"postgres", "app"})
	if err != nil {
		t.Fatal(err)
	}
	pgIdx, appIdx := -1, -1
	for i, name := range order {
		if name == "postgres" {
			pgIdx = i
		}
		if name == "app" {
			appIdx = i
		}
	}
	if pgIdx > appIdx {
		t.Errorf("postgres should come before app")
	}
}

func TestTopoSort_CircularDependency(t *testing.T) {
	services := map[string]workspace.Service{
		"a": {DependsOn: []string{"b"}},
		"b": {DependsOn: []string{"a"}},
	}
	_, err := orchestrator.TopoSort(services, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for circular")
	}
}

func TestTopoSort_Diamond(t *testing.T) {
	services := map[string]workspace.Service{
		"db":    {},
		"cache": {},
		"api":   {DependsOn: []string{"db", "cache"}},
		"web":   {DependsOn: []string{"api"}},
	}
	order, _ := orchestrator.TopoSort(services, []string{"db", "cache", "api", "web"})
	indexOf := func(name string) int {
		for i, n := range order {
			if n == name {
				return i
			}
		}
		return -1
	}
	if indexOf("db") > indexOf("api") {
		t.Error("db before api")
	}
	if indexOf("cache") > indexOf("api") {
		t.Error("cache before api")
	}
	if indexOf("api") > indexOf("web") {
		t.Error("api before web")
	}
}

func TestTopoSort_NoDeps(t *testing.T) {
	services := map[string]workspace.Service{"a": {}, "b": {}, "c": {}}
	order, _ := orchestrator.TopoSort(services, []string{"a", "b", "c"})
	if len(order) != 3 {
		t.Errorf("expected 3, got %d", len(order))
	}
}

func TestTopoSort_BuildFromOrdering(t *testing.T) {
	services := map[string]workspace.Service{
		"server": {Build: ".", Ports: []int{8080}},
		"worker": {BuildFrom: "server", Command: "work"},
	}
	order, err := orchestrator.TopoSort(services, []string{"server", "worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	serverIdx := indexOf(order, "server")
	workerIdx := indexOf(order, "worker")
	if serverIdx > workerIdx {
		t.Errorf("server (idx %d) should come before worker (idx %d), got %v", serverIdx, workerIdx, order)
	}
}

func TestTopoSort_BuildFromPullsInSource(t *testing.T) {
	services := map[string]workspace.Service{
		"server": {Build: ".", Ports: []int{8080}},
		"worker": {BuildFrom: "server", Command: "work"},
	}
	order, err := orchestrator.TopoSort(services, []string{"worker"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsStr(order, "server") {
		t.Errorf("build_from source should be pulled into order, got %v", order)
	}
}

func TestTopoSort_BuildFromWithDependsOn(t *testing.T) {
	services := map[string]workspace.Service{
		"postgres": {Image: "postgres:16", Ports: []int{5432}},
		"server":   {Build: ".", DependsOn: []string{"postgres"}},
		"worker":   {BuildFrom: "server", DependsOn: []string{"postgres"}},
	}
	order, err := orchestrator.TopoSort(services, []string{"server", "worker", "postgres"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pgIdx := indexOf(order, "postgres")
	serverIdx := indexOf(order, "server")
	workerIdx := indexOf(order, "worker")
	if pgIdx > serverIdx {
		t.Errorf("postgres should come before server")
	}
	if serverIdx > workerIdx {
		t.Errorf("server should come before worker (build_from dependency)")
	}
}

func indexOf(order []string, name string) int {
	for i, n := range order {
		if n == name {
			return i
		}
	}
	return -1
}

func containsStr(order []string, name string) bool {
	return indexOf(order, name) >= 0
}
