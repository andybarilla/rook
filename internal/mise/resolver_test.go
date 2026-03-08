package mise

import (
	"fmt"
	"os/exec"
	"testing"
)

// stubExecutor implements Executor for testing.
type stubExecutor struct {
	available           bool
	version             string
	whichResults        map[string]string // tool -> path
	whichErrors         map[string]error
	whichVersionResults map[string]string // "tool@version" -> path
	whichVersionErrors  map[string]error
	detectOut           map[string]string
	detectErr           error

	// Install tracking
	installCalled  bool
	installTool    string
	installVersion string
	installErr     error

	// IsInstalled tracking
	isInstalled    bool
	isInstalledErr error

	// ListInstalled tracking
	listInstalled    []string
	listInstalledErr error
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
	key := tool + "@" + version
	if err, ok := s.whichVersionErrors[key]; ok {
		return "", err
	}
	if path, ok := s.whichVersionResults[key]; ok {
		return path, nil
	}
	return "", fmt.Errorf("tool %q version %q not found", tool, version)
}

func (s *stubExecutor) Detect(dir string) (map[string]string, error) {
	if s.detectErr != nil {
		return nil, s.detectErr
	}
	if s.detectOut != nil {
		return s.detectOut, nil
	}
	return map[string]string{}, nil
}

func (s *stubExecutor) Install(tool, version string) error {
	s.installCalled = true
	s.installTool = tool
	s.installVersion = version
	return s.installErr
}

func (s *stubExecutor) IsInstalled(tool, version string) (bool, error) {
	return s.isInstalled, s.isInstalledErr
}

func (s *stubExecutor) ListInstalled(tool string) ([]string, error) {
	if s.listInstalledErr != nil {
		return nil, s.listInstalledErr
	}
	return s.listInstalled, nil
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

func TestVersion_BeforeAvailable(t *testing.T) {
	stub := &stubExecutor{
		available: true,
		version:   "2024.5.0",
	}

	r := NewWithExecutor(stub)

	// Call Version() as the FIRST method, without calling Available() first.
	gotVer := r.Version()
	if gotVer != "2024.5.0" {
		t.Fatalf("Version() before Available(): expected 2024.5.0, got %s", gotVer)
	}
}

func TestWhich_FallsBackToLookPathWhenMiseWhichErrors(t *testing.T) {
	// mise IS available, but Which() returns an error for the tool.
	// Should fall back to exec.LookPath.
	stub := &stubExecutor{
		available:    true,
		version:      "1.0.0",
		whichResults: map[string]string{},
		whichErrors: map[string]error{
			"go": fmt.Errorf("mise which go: tool not configured"),
		},
	}

	r := NewWithExecutor(stub)

	// "go" should be findable via LookPath fallback
	path, err := r.Which("go")
	if err != nil {
		t.Skipf("go not in PATH, skipping: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path for 'go' via LookPath fallback")
	}
	t.Logf("fallback found go at: %s (mise was available but Which errored)", path)
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

func TestWhichVersion_UsesMiseWhenAvailable(t *testing.T) {
	stub := &stubExecutor{
		available: true,
		version:   "1.0.0",
		whichVersionResults: map[string]string{
			"node@20.0.0": "/home/user/.local/share/mise/installs/node/20.0.0/bin/node",
		},
	}

	r := NewWithExecutor(stub)

	path, err := r.WhichVersion("node", "20.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/home/user/.local/share/mise/installs/node/20.0.0/bin/node"
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestWhichVersion_FallsBackWhenMiseUnavailable(t *testing.T) {
	stub := &stubExecutor{
		available: false,
		version:   "",
	}

	r := NewWithExecutor(stub)

	// "go" should be findable via LookPath on any dev machine
	path, err := r.WhichVersion("go", "1.21.0")
	if err != nil {
		t.Skipf("go not in PATH, skipping: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path for 'go' via LookPath fallback")
	}
	t.Logf("fallback found go at: %s", path)
}

func TestDetect_ReturnsParsedVersions(t *testing.T) {
	stub := &stubExecutor{
		available: true,
		version:   "1.0.0",
		detectOut: map[string]string{
			"php":  "8.3.0",
			"node": "20.0.0",
		},
	}

	r := NewWithExecutor(stub)

	result, err := r.Detect("/some/site/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["php"] != "8.3.0" {
		t.Fatalf("expected php 8.3.0, got %s", result["php"])
	}
	if result["node"] != "20.0.0" {
		t.Fatalf("expected node 20.0.0, got %s", result["node"])
	}
}

func TestDetect_ReturnsEmptyWhenMiseUnavailable(t *testing.T) {
	stub := &stubExecutor{
		available: false,
		version:   "",
	}

	r := NewWithExecutor(stub)

	result, err := r.Detect("/some/site/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

func TestParseCurrentVersions(t *testing.T) {
	input := []byte(`{
		"php": [{"version": "8.3.0", "source": {}}],
		"node": [{"version": "20.0.0", "source": {}}],
		"empty": []
	}`)

	result, err := parseCurrentVersions(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(result), result)
	}
	if result["php"] != "8.3.0" {
		t.Fatalf("expected php 8.3.0, got %s", result["php"])
	}
	if result["node"] != "20.0.0" {
		t.Fatalf("expected node 20.0.0, got %s", result["node"])
	}
}

func TestParseCurrentVersions_InvalidJSON(t *testing.T) {
	result, err := parseCurrentVersions([]byte(`not json`))
	if err != nil {
		t.Fatalf("expected no error on invalid JSON, got: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

func TestInstall_ErrorsWhenMiseUnavailable(t *testing.T) {
	stub := &stubExecutor{available: false}
	r := NewWithExecutor(stub)

	err := r.Install("php", "8.3.0")
	if err == nil {
		t.Fatal("expected error when mise is unavailable")
	}
	if err.Error() != "mise is not available" {
		t.Fatalf("expected 'mise is not available', got: %v", err)
	}
}

func TestInstall_DelegatesToExecutor(t *testing.T) {
	stub := &stubExecutor{available: true, version: "1.0.0"}
	r := NewWithExecutor(stub)

	err := r.Install("php", "8.3.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.installCalled {
		t.Fatal("expected Install to be called on executor")
	}
	if stub.installTool != "php" {
		t.Fatalf("expected tool 'php', got %q", stub.installTool)
	}
	if stub.installVersion != "8.3.0" {
		t.Fatalf("expected version '8.3.0', got %q", stub.installVersion)
	}
}

func TestIsInstalled_ReturnsFalseWhenMiseUnavailable(t *testing.T) {
	stub := &stubExecutor{available: false}
	r := NewWithExecutor(stub)

	result := r.IsInstalled("php", "8.3.0")
	if result {
		t.Fatal("expected false when mise is unavailable")
	}
}

func TestIsInstalled_DelegatesToExecutor(t *testing.T) {
	stub := &stubExecutor{
		available:   true,
		version:     "1.0.0",
		isInstalled: true,
	}
	r := NewWithExecutor(stub)

	result := r.IsInstalled("php", "8.3.0")
	if !result {
		t.Fatal("expected true when executor returns true")
	}
}

func TestListInstalled_ErrorsWhenMiseUnavailable(t *testing.T) {
	stub := &stubExecutor{available: false}
	r := NewWithExecutor(stub)

	_, err := r.ListInstalled("php")
	if err == nil {
		t.Fatal("expected error when mise is unavailable")
	}
	if err.Error() != "mise is not available" {
		t.Fatalf("expected 'mise is not available', got: %v", err)
	}
}

func TestListInstalled_ReturnsVersions(t *testing.T) {
	stub := &stubExecutor{
		available:     true,
		version:       "1.0.0",
		listInstalled: []string{"8.2.0", "8.3.0"},
	}
	r := NewWithExecutor(stub)

	versions, err := r.ListInstalled("php")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0] != "8.2.0" || versions[1] != "8.3.0" {
		t.Fatalf("expected [8.2.0 8.3.0], got %v", versions)
	}
}
