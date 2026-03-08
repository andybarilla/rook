package mise

import (
	"fmt"
	"os/exec"
	"testing"
)

// stubExecutor implements Executor for testing.
type stubExecutor struct {
	available    bool
	version      string
	whichResults map[string]string // tool -> path
	whichErrors  map[string]error
}

func (s *stubExecutor) Available() (bool, string) {
	return s.available, s.version
}

func (s *stubExecutor) Which(tool string) (string, error) {
	if err, ok := s.whichErrors[tool]; ok {
		return "", err
	}
	if path, ok := s.whichResults[tool]; ok {
		return path, nil
	}
	return "", fmt.Errorf("tool %q not found", tool)
}

func (s *stubExecutor) WhichVersion(tool, version string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (s *stubExecutor) Detect(dir string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *stubExecutor) Install(tool, version string) error {
	return fmt.Errorf("not implemented")
}

func (s *stubExecutor) IsInstalled(tool, version string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (s *stubExecutor) ListInstalled(tool string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestAvailable_ReturnsTrueWhenMiseInPath(t *testing.T) {
	_, err := exec.LookPath("mise")
	if err != nil {
		t.Skip("mise not installed, skipping integration test")
	}

	r := New()
	ok, ver := r.Available()
	if !ok {
		t.Fatal("expected Available() to return true when mise is installed")
	}
	if ver == "" {
		t.Fatal("expected non-empty version string")
	}
	t.Logf("mise version: %s", ver)
}

func TestAvailable_CachesResult(t *testing.T) {
	callCount := 0
	stub := &countingExecutor{
		available: true,
		version:   "1.0.0",
		callCount: &callCount,
	}

	r := NewWithExecutor(stub)

	// Call Available() multiple times
	for i := 0; i < 5; i++ {
		ok, ver := r.Available()
		if !ok {
			t.Fatal("expected Available() to return true")
		}
		if ver != "1.0.0" {
			t.Fatalf("expected version 1.0.0, got %s", ver)
		}
	}

	if callCount != 1 {
		t.Fatalf("expected executor.Available() to be called once (cached), but was called %d times", callCount)
	}
}

func TestWhich_FallsBackToLookPath(t *testing.T) {
	stub := &stubExecutor{
		available:    false,
		version:      "",
		whichResults: map[string]string{},
		whichErrors:  map[string]error{},
	}

	r := NewWithExecutor(stub)

	// "go" should be findable via LookPath on any dev machine
	path, err := r.Which("go")
	if err != nil {
		t.Skipf("go not in PATH, skipping: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path for 'go'")
	}
	t.Logf("fallback found go at: %s", path)
}

func TestWhich_UsesMiseWhenAvailable(t *testing.T) {
	stub := &stubExecutor{
		available: true,
		version:   "1.0.0",
		whichResults: map[string]string{
			"node": "/home/user/.local/share/mise/installs/node/20.0.0/bin/node",
		},
		whichErrors: map[string]error{},
	}

	r := NewWithExecutor(stub)

	path, err := r.Which("node")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/home/user/.local/share/mise/installs/node/20.0.0/bin/node"
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestWhich_ErrorWhenToolNotFound(t *testing.T) {
	stub := &stubExecutor{
		available:    true,
		version:      "1.0.0",
		whichResults: map[string]string{},
		whichErrors: map[string]error{
			"nonexistent-tool-xyz": fmt.Errorf("tool not found"),
		},
	}

	r := NewWithExecutor(stub)

	_, err := r.Which("nonexistent-tool-xyz")
	if err == nil {
		t.Fatal("expected error when tool not found by mise and not in PATH")
	}
	t.Logf("got expected error: %v", err)
}

func TestVersion(t *testing.T) {
	stub := &stubExecutor{
		available: true,
		version:   "2024.1.0",
	}

	r := NewWithExecutor(stub)
	_, ver := r.Available() // trigger cache
	if ver != "2024.1.0" {
		t.Fatalf("expected version 2024.1.0, got %s", ver)
	}

	gotVer := r.Version()
	if gotVer != "2024.1.0" {
		t.Fatalf("Version() expected 2024.1.0, got %s", gotVer)
	}
}

// countingExecutor wraps stubExecutor but counts Available() calls.
type countingExecutor struct {
	available bool
	version   string
	callCount *int
}

func (c *countingExecutor) Available() (bool, string) {
	*c.callCount++
	return c.available, c.version
}

func (c *countingExecutor) Which(tool string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *countingExecutor) WhichVersion(tool, version string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (c *countingExecutor) Detect(dir string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *countingExecutor) Install(tool, version string) error {
	return fmt.Errorf("not implemented")
}

func (c *countingExecutor) IsInstalled(tool, version string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (c *countingExecutor) ListInstalled(tool string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
