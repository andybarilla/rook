package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/discovery"
)

func TestScanLocalSignals(t *testing.T) {
	t.Run("go_with_cmd_dirs", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/myapp\n"), 0644)
		os.MkdirAll(filepath.Join(dir, "cmd", "api"), 0755)
		os.MkdirAll(filepath.Join(dir, "cmd", "worker"), 0755)

		signals := discovery.ScanLocalSignals(dir)

		goSignals := filterType(signals, "go")
		if len(goSignals) != 2 {
			t.Fatalf("expected 2 go signals, got %d", len(goSignals))
		}
		names := map[string]bool{}
		for _, s := range goSignals {
			names[s.Name] = true
			if s.Name == "api" && s.Command != "go run ./cmd/api" {
				t.Errorf("expected 'go run ./cmd/api', got %q", s.Command)
			}
		}
		if !names["api"] || !names["worker"] {
			t.Errorf("expected api and worker signals, got %v", names)
		}
	})

	t.Run("go_without_cmd_dirs", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/myapp\n"), 0644)

		signals := discovery.ScanLocalSignals(dir)

		goSignals := filterType(signals, "go")
		if len(goSignals) != 1 {
			t.Fatalf("expected 1 go signal, got %d", len(goSignals))
		}
		if goSignals[0].Name != "myapp" {
			t.Errorf("expected name 'myapp', got %q", goSignals[0].Name)
		}
		if goSignals[0].Command != "go run ." {
			t.Errorf("expected 'go run .', got %q", goSignals[0].Command)
		}
	})

	t.Run("node_with_dev_script", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"myapp","scripts":{"dev":"next dev","build":"next build"}}`), 0644)

		signals := discovery.ScanLocalSignals(dir)

		nodeSignals := filterType(signals, "node")
		if len(nodeSignals) != 1 {
			t.Fatalf("expected 1 node signal, got %d", len(nodeSignals))
		}
		if nodeSignals[0].Command != "npm run dev" {
			t.Errorf("expected 'npm run dev', got %q", nodeSignals[0].Command)
		}
		if nodeSignals[0].Name != "myapp" {
			t.Errorf("expected name 'myapp', got %q", nodeSignals[0].Name)
		}
	})

	t.Run("node_with_start_no_dev", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"myapp","scripts":{"start":"node server.js"}}`), 0644)

		signals := discovery.ScanLocalSignals(dir)

		nodeSignals := filterType(signals, "node")
		if len(nodeSignals) != 1 {
			t.Fatalf("expected 1 node signal, got %d", len(nodeSignals))
		}
		if nodeSignals[0].Command != "npm start" {
			t.Errorf("expected 'npm start', got %q", nodeSignals[0].Command)
		}
	})

	t.Run("makefile_with_dev_target", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\tgo build\ndev:\n\tgo run .\n"), 0644)

		signals := discovery.ScanLocalSignals(dir)

		makeSignals := filterType(signals, "makefile")
		if len(makeSignals) != 1 {
			t.Fatalf("expected 1 makefile signal, got %d", len(makeSignals))
		}
		if makeSignals[0].Command != "make dev" {
			t.Errorf("expected 'make dev', got %q", makeSignals[0].Command)
		}
	})

	t.Run("procfile", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "Procfile"), []byte("web: node server.js\nworker: node worker.js\n"), 0644)

		signals := discovery.ScanLocalSignals(dir)

		procSignals := filterType(signals, "procfile")
		if len(procSignals) != 2 {
			t.Fatalf("expected 2 procfile signals, got %d", len(procSignals))
		}
	})

	t.Run("python", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"myapp\"\n"), 0644)

		signals := discovery.ScanLocalSignals(dir)

		pySignals := filterType(signals, "python")
		if len(pySignals) != 1 {
			t.Fatalf("expected 1 python signal, got %d", len(pySignals))
		}
		if pySignals[0].Command != "" {
			t.Errorf("expected empty command for python, got %q", pySignals[0].Command)
		}
	})

	t.Run("rust", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"myapp\"\n"), 0644)

		signals := discovery.ScanLocalSignals(dir)

		rustSignals := filterType(signals, "rust")
		if len(rustSignals) != 1 {
			t.Fatalf("expected 1 rust signal, got %d", len(rustSignals))
		}
		if rustSignals[0].Command != "cargo run" {
			t.Errorf("expected 'cargo run', got %q", rustSignals[0].Command)
		}
	})

	t.Run("empty_directory", func(t *testing.T) {
		dir := t.TempDir()
		signals := discovery.ScanLocalSignals(dir)
		if len(signals) != 0 {
			t.Errorf("expected 0 signals, got %d", len(signals))
		}
	})
}

func filterType(signals []discovery.LocalSignal, typ string) []discovery.LocalSignal {
	var result []discovery.LocalSignal
	for _, s := range signals {
		if s.Type == typ {
			result = append(result, s)
		}
	}
	return result
}
