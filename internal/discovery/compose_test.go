package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestComposeDiscoverer_Detect(t *testing.T) {
	d := NewComposeDiscoverer()

	t.Run("detects docker-compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}"), 0644)
		if !d.Detect(dir) {
			t.Error("expected Detect to return true for docker-compose.yml")
		}
	})

	t.Run("detects compose.yml", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "compose.yml"), []byte("services: {}"), 0644)
		if !d.Detect(dir) {
			t.Error("expected Detect to return true for compose.yml")
		}
	})

	t.Run("returns false without file", func(t *testing.T) {
		dir := t.TempDir()
		if d.Detect(dir) {
			t.Error("expected Detect to return false when no compose file exists")
		}
	})
}

func TestComposeDiscoverer_Discover(t *testing.T) {
	d := NewComposeDiscoverer()

	composeContent := `
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    environment:
      - FOO=bar
      - BAZ=qux
  db:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data
`

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(composeContent), 0644)

	result, err := d.Discover(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Source != "docker-compose" {
		t.Errorf("expected source 'docker-compose', got %q", result.Source)
	}

	if len(result.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(result.Services))
	}

	web := result.Services["web"]
	if web.Image != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got %q", web.Image)
	}
	if len(web.Ports) != 1 || web.Ports[0] != 80 {
		t.Errorf("expected ports [80], got %v", web.Ports)
	}
	if web.Environment["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %q", web.Environment["FOO"])
	}
	if web.Environment["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %q", web.Environment["BAZ"])
	}

	db := result.Services["db"]
	if db.Image != "postgres:15" {
		t.Errorf("expected image 'postgres:15', got %q", db.Image)
	}
	if len(db.Ports) != 1 || db.Ports[0] != 5432 {
		t.Errorf("expected ports [5432], got %v", db.Ports)
	}
	if db.Environment["POSTGRES_USER"] != "admin" {
		t.Errorf("expected POSTGRES_USER=admin, got %q", db.Environment["POSTGRES_USER"])
	}
	if len(db.Volumes) != 1 || db.Volumes[0] != "pgdata:/var/lib/postgresql/data" {
		t.Errorf("expected volume 'pgdata:/var/lib/postgresql/data', got %v", db.Volumes)
	}
}

func TestComposeDiscoverer_DependsOn(t *testing.T) {
	d := NewComposeDiscoverer()

	t.Run("list format", func(t *testing.T) {
		content := `
services:
  web:
    image: nginx
    depends_on:
      - db
      - redis
  db:
    image: postgres
  redis:
    image: redis
`
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(content), 0644)

		result, err := d.Discover(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		web := result.Services["web"]
		sort.Strings(web.DependsOn)
		if len(web.DependsOn) != 2 {
			t.Fatalf("expected 2 depends_on, got %d", len(web.DependsOn))
		}
		if web.DependsOn[0] != "db" || web.DependsOn[1] != "redis" {
			t.Errorf("expected depends_on [db, redis], got %v", web.DependsOn)
		}
	})

	t.Run("build context and command", func(t *testing.T) {
		dir := t.TempDir()
		compose := `
services:
  api:
    build: .
    ports:
      - "8080:8080"
  worker:
    build: .
    command: ["./server", "-worker"]
`
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644)
		d := NewComposeDiscoverer()
		result, err := d.Discover(dir)
		if err != nil {
			t.Fatal(err)
		}

		api := result.Services["api"]
		if api.Build != "." {
			t.Errorf("expected build '.', got '%s'", api.Build)
		}
		if !api.IsContainer() {
			t.Error("api with build should be container")
		}

		worker := result.Services["worker"]
		if worker.Build != "." {
			t.Errorf("expected build '.', got '%s'", worker.Build)
		}
		if worker.Command != "./server -worker" {
			t.Errorf("expected command './server -worker', got '%s'", worker.Command)
		}
	})

	t.Run("map format", func(t *testing.T) {
		content := `
services:
  web:
    image: nginx
    depends_on:
      db:
        condition: service_healthy
  db:
    image: postgres
`
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(content), 0644)

		result, err := d.Discover(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		web := result.Services["web"]
		if len(web.DependsOn) != 1 || web.DependsOn[0] != "db" {
			t.Errorf("expected depends_on [db], got %v", web.DependsOn)
		}
	})
}
