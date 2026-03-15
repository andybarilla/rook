package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWithManifest(t *testing.T) {
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "rook")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/rook")
	build.Dir = findProjectRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	projectDir := t.TempDir()
	manifest := `
name: test-project
type: single
services:
  postgres:
    image: postgres:16-alpine
    ports: [5432]
  app:
    command: echo hello
    ports: [8080]
    depends_on: [postgres]
`
	os.WriteFile(filepath.Join(projectDir, "rook.yaml"), []byte(manifest), 0644)

	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	cmd := exec.Command(binPath, "init", projectDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %s\n%s", err, output)
	}
	if !strings.Contains(string(output), "test-project") {
		t.Errorf("expected workspace name in output:\n%s", string(output))
	}

	cmd = exec.Command(binPath, "list")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %s\n%s", err, output)
	}
	if !strings.Contains(string(output), "test-project") {
		t.Errorf("expected test-project in list:\n%s", string(output))
	}

	cmd = exec.Command(binPath, "ports")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ports failed: %s\n%s", err, output)
	}
	if !strings.Contains(string(output), "postgres") {
		t.Errorf("expected postgres in ports:\n%s", string(output))
	}
}

func TestInitWithDiscovery(t *testing.T) {
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "rook")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/rook")
	build.Dir = findProjectRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}

	projectDir := t.TempDir()
	compose := `
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
  redis:
    image: redis:7
    ports:
      - "6379:6379"
`
	os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(compose), 0644)

	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	cmd := exec.Command(binPath, "init", projectDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %s\n%s", err, output)
	}
	if !strings.Contains(string(output), "Discovered") {
		t.Errorf("expected discovery output:\n%s", string(output))
	}

	// Verify rook.yaml was generated
	if _, err := os.Stat(filepath.Join(projectDir, "rook.yaml")); os.IsNotExist(err) {
		t.Error("expected rook.yaml to be generated")
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}
