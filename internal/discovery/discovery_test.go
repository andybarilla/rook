package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
}

func TestScanNonexistentDir(t *testing.T) {
	manifests, errs := Scan("/nonexistent/path")
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
}

func TestScanValidPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "flock-node")
	os.MkdirAll(pluginDir, 0o755)

	manifest := `{
		"id": "flock-node",
		"name": "Flock Node",
		"version": "0.1.0",
		"executable": "flock-node",
		"capabilities": ["runtime"]
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)

	// Create a fake executable
	exePath := filepath.Join(pluginDir, "flock-node")
	os.WriteFile(exePath, []byte("#!/bin/sh\n"), 0o755)

	manifests, errs := Scan(dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
	m := manifests[0]
	if m.ID != "flock-node" {
		t.Errorf("ID = %q, want flock-node", m.ID)
	}
	if m.Name != "Flock Node" {
		t.Errorf("Name = %q, want Flock Node", m.Name)
	}
	if m.ExePath != exePath {
		t.Errorf("ExePath = %q, want %q", m.ExePath, exePath)
	}
	if len(m.Capabilities) != 1 || m.Capabilities[0] != "runtime" {
		t.Errorf("Capabilities = %v, want [runtime]", m.Capabilities)
	}
}

func TestScanSkipsMissingManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "no-manifest"), 0o755)

	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestScanSkipsMissingExecutable(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "bad-exe")
	os.MkdirAll(pluginDir, 0o755)

	manifest := `{
		"id": "bad-exe",
		"name": "Bad Exe",
		"version": "0.1.0",
		"executable": "nonexistent",
		"capabilities": ["runtime"]
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)

	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestScanSkipsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "no-id")
	os.MkdirAll(pluginDir, 0o755)

	manifest := `{"name": "No ID", "version": "0.1.0", "executable": "x", "capabilities": ["runtime"]}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)
	os.WriteFile(filepath.Join(pluginDir, "x"), []byte("#!/bin/sh\n"), 0o755)

	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestScanMultiplePlugins(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"plugin-a", "plugin-b"} {
		pluginDir := filepath.Join(dir, name)
		os.MkdirAll(pluginDir, 0o755)
		manifest := `{"id":"` + name + `","name":"` + name + `","version":"0.1.0","executable":"` + name + `","capabilities":["runtime"]}`
		os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)
		os.WriteFile(filepath.Join(pluginDir, name), []byte("#!/bin/sh\n"), 0o755)
	}

	manifests, errs := Scan(dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(manifests) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(manifests))
	}
}
