package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/discovery"
)

func TestScanComposeFiles(t *testing.T) {
	t.Run("finds_multiple_compose_files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services:\n  postgres:\n    image: postgres:16\n")
		writeFile(t, filepath.Join(dir, "docker-compose.dev.yml"), "services:\n  app:\n    build: .\n")
		os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0755)
		writeFile(t, filepath.Join(dir, ".devcontainer", "docker-compose.yml"), "services:\n  app:\n    build:\n      context: ..\n")

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 3 {
			t.Fatalf("expected 3 compose files, got %d", len(results))
		}
		// .devcontainer should be first
		if results[0].RelPath != ".devcontainer/docker-compose.yml" {
			t.Errorf("expected .devcontainer first, got %s", results[0].RelPath)
		}
	})

	t.Run("returns_service_names_per_file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services:\n  postgres:\n    image: postgres:16\n  redis:\n    image: redis:7\n")

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 1 {
			t.Fatalf("expected 1 compose file, got %d", len(results))
		}
		if len(results[0].ServiceNames) != 2 {
			t.Fatalf("expected 2 service names, got %d", len(results[0].ServiceNames))
		}
		// Names should be sorted
		if results[0].ServiceNames[0] != "postgres" || results[0].ServiceNames[1] != "redis" {
			t.Errorf("unexpected service names: %v", results[0].ServiceNames)
		}
	})

	t.Run("returns_empty_for_no_compose_files", func(t *testing.T) {
		dir := t.TempDir()
		results := discovery.ScanComposeFiles(dir)
		if len(results) != 0 {
			t.Fatalf("expected 0 compose files, got %d", len(results))
		}
	})

	t.Run("finds_compose_yml_variant", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "compose.yml"), "services:\n  web:\n    image: nginx\n")

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 1 {
			t.Fatalf("expected 1 compose file, got %d", len(results))
		}
		if results[0].RelPath != "compose.yml" {
			t.Errorf("expected compose.yml, got %s", results[0].RelPath)
		}
	})

	t.Run("finds_glob_variants", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services:\n  db:\n    image: postgres\n")
		writeFile(t, filepath.Join(dir, "docker-compose.override.yml"), "services:\n  db:\n    ports:\n      - 5432:5432\n")
		writeFile(t, filepath.Join(dir, "docker-compose.test.yaml"), "services:\n  testdb:\n    image: postgres\n")

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 3 {
			t.Fatalf("expected 3 compose files, got %d", len(results))
		}
	})

	t.Run("skips_unparseable_files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "not: valid: yaml: [")
		writeFile(t, filepath.Join(dir, "compose.yml"), "services:\n  web:\n    image: nginx\n")

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 1 {
			t.Fatalf("expected 1 compose file (skipping invalid), got %d", len(results))
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
