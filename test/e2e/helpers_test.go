package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildRook builds the rook binary and returns its path.
func buildRook(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "rook")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/rook")
	build.Dir = findProjectRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return binPath
}

// createTestWorkspace creates a workspace directory with a manifest and returns both paths.
func createTestWorkspace(t *testing.T, manifest string) (wsDir string, configDir string) {
	t.Helper()
	wsDir = t.TempDir()
	configDir = t.TempDir()

	if manifest != "" {
		os.WriteFile(filepath.Join(wsDir, "rook.yaml"), []byte(manifest), 0644)
	}
	return wsDir, configDir
}

// runRook executes the rook binary with the given args and config dir.
func runRook(t *testing.T, binPath, configDir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
