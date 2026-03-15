package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func TestParseManifest_SingleWorkspace(t *testing.T) {
	yaml := `
name: skeetr
type: single
services:
  postgres:
    image: postgres:16-alpine
    healthcheck: pg_isready -U skeetr
    volumes:
      - pg-data:/var/lib/postgresql/data
  app:
    command: air
    ports: [8080]
    depends_on: [postgres]
    environment:
      DATABASE_URL: "postgres://skeetr:skeetr@{{.Host.postgres}}:{{.Port.postgres}}/skeetr"
groups:
  infra:
    - postgres
profiles:
  default:
    - infra
    - app
`
	dir := t.TempDir()
	path := filepath.Join(dir, "rook.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := workspace.ParseManifest(path)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if m.Name != "skeetr" {
		t.Errorf("expected name skeetr, got %s", m.Name)
	}
	if m.Type != workspace.TypeSingle {
		t.Errorf("expected type single, got %s", m.Type)
	}
	if len(m.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(m.Services))
	}
	pg := m.Services["postgres"]
	if !pg.IsContainer() {
		t.Error("postgres should be a container service")
	}
	app := m.Services["app"]
	if !app.IsProcess() {
		t.Error("app should be a process service")
	}
	if len(app.DependsOn) != 1 || app.DependsOn[0] != "postgres" {
		t.Errorf("app depends_on wrong: %v", app.DependsOn)
	}
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rook.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.ParseManifest(path)
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestParseManifest_MissingFile(t *testing.T) {
	_, err := workspace.ParseManifest("/nonexistent/rook.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestManifestToWorkspace(t *testing.T) {
	m := workspace.Manifest{
		Name: "test",
		Type: workspace.TypeSingle,
		Services: map[string]workspace.Service{
			"app": {Command: "air", Ports: []int{8080}},
		},
	}
	ws, err := m.ToWorkspace("/some/path")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Root != "/some/path" {
		t.Errorf("expected root /some/path, got %s", ws.Root)
	}
}
