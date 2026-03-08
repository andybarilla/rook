# Mise Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate mise as an optional runtime backend — using it when available, falling back to system PATH when not — with auto-detection, install warnings, and a Settings status indicator.

**Architecture:** A shared `RuntimeResolver` in `internal/mise/` shells out to the `mise` CLI for binary resolution, version detection, and installation. Plugins call the resolver instead of assuming system binaries. The frontend gains auto-detection in the Add Site modal, warning badges for missing runtimes, and a mise status card in Settings.

**Tech Stack:** Go (backend), Svelte + DaisyUI (frontend), Wails v2 (bindings)

**Prerequisite:** PR #26 (UI redesign) must be merged first. This plan assumes the modal-based AddSiteForm, card-based SiteCard/SiteList, and SettingsTab from that PR are on `main`.

---

### Task 1: RuntimeResolver — Available() and Which()

Create the `internal/mise/` package with the core resolver struct, availability check, and basic tool resolution with fallback.

**Files:**
- Create: `internal/mise/resolver.go`
- Create: `internal/mise/resolver_test.go`

**Step 1: Write the failing tests**

```go
// internal/mise/resolver_test.go
package mise_test

import (
	"os/exec"
	"testing"

	"github.com/andybarilla/flock/internal/mise"
)

func TestAvailable_ReturnsTrueWhenMiseInPath(t *testing.T) {
	// This test checks real system state — skip in CI if mise isn't installed
	_, err := exec.LookPath("mise")
	if err != nil {
		t.Skip("mise not installed, skipping")
	}
	r := mise.New()
	if !r.Available() {
		t.Error("expected Available() = true when mise is in PATH")
	}
}

func TestAvailable_CachesResult(t *testing.T) {
	r := mise.New()
	first := r.Available()
	second := r.Available()
	if first != second {
		t.Error("Available() should return same result on repeated calls")
	}
}

func TestWhich_FallsBackToLookPath(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	// "go" should be findable via exec.LookPath on any dev machine
	path, err := r.Which("go")
	if err != nil {
		t.Fatalf("Which(go) error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for 'go'")
	}
}

func TestWhich_UsesMiseWhenAvailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{
		available:  true,
		whichOut:   "/home/user/.local/share/mise/installs/node/20.0.0/bin/node",
		whichErr:   nil,
	})
	path, err := r.Which("node")
	if err != nil {
		t.Fatalf("Which(node) error: %v", err)
	}
	if path != "/home/user/.local/share/mise/installs/node/20.0.0/bin/node" {
		t.Errorf("path = %q, want mise-resolved path", path)
	}
}

func TestWhich_ErrorWhenToolNotFound(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	_, err := r.Which("nonexistent-tool-xyz")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

// --- Stub executor for testing ---

type stubExecutor struct {
	available  bool
	version    string
	whichOut   string
	whichErr   error
}

func (s *stubExecutor) Available() (bool, string) {
	return s.available, s.version
}

func (s *stubExecutor) Which(tool string) (string, error) {
	return s.whichOut, s.whichErr
}

func (s *stubExecutor) WhichVersion(tool, version string) (string, error) {
	return s.whichOut, s.whichErr
}

func (s *stubExecutor) Detect(dir string) (map[string]string, error) {
	return nil, nil
}

func (s *stubExecutor) Install(tool, version string) error {
	return nil
}

func (s *stubExecutor) IsInstalled(tool, version string) (bool, error) {
	return false, nil
}

func (s *stubExecutor) ListInstalled(tool string) ([]string, error) {
	return nil, nil
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mise/ -v`
Expected: FAIL — package does not exist

**Step 3: Write the implementation**

```go
// internal/mise/resolver.go
package mise

import (
	"os/exec"
	"strings"
	"sync"
)

// Executor abstracts mise CLI calls for testing.
type Executor interface {
	Available() (bool, string)
	Which(tool string) (string, error)
	WhichVersion(tool, version string) (string, error)
	Detect(dir string) (map[string]string, error)
	Install(tool, version string) error
	IsInstalled(tool, version string) (bool, error)
	ListInstalled(tool string) ([]string, error)
}

// RuntimeResolver resolves binary paths using mise (if available) or system PATH.
type RuntimeResolver struct {
	exec      Executor
	available *bool
	version   string
	mu        sync.Mutex
}

// New creates a RuntimeResolver that shells out to the real mise CLI.
func New() *RuntimeResolver {
	return &RuntimeResolver{exec: &cliExecutor{}}
}

// NewWithExecutor creates a RuntimeResolver with a custom executor (for testing).
func NewWithExecutor(exec Executor) *RuntimeResolver {
	return &RuntimeResolver{exec: exec}
}

// Available returns true if mise is installed and reachable.
func (r *RuntimeResolver) Available() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.available != nil {
		return *r.available
	}
	avail, ver := r.exec.Available()
	r.available = &avail
	r.version = ver
	return avail
}

// Version returns the mise version string (empty if unavailable).
func (r *RuntimeResolver) Version() string {
	r.Available() // ensure cached
	return r.version
}

// Which resolves the path of a tool binary. Uses mise if available, falls back to exec.LookPath.
func (r *RuntimeResolver) Which(tool string) (string, error) {
	if r.Available() {
		path, err := r.exec.Which(tool)
		if err == nil && path != "" {
			return path, nil
		}
	}
	return exec.LookPath(tool)
}

// --- CLI executor (real mise calls) ---

type cliExecutor struct{}

func (c *cliExecutor) Available() (bool, string) {
	out, err := exec.Command("mise", "--version").Output()
	if err != nil {
		return false, ""
	}
	return true, strings.TrimSpace(string(out))
}

func (c *cliExecutor) Which(tool string) (string, error) {
	out, err := exec.Command("mise", "which", tool).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *cliExecutor) WhichVersion(tool, version string) (string, error) {
	out, err := exec.Command("mise", "which", tool+"@"+version).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *cliExecutor) Detect(dir string) (map[string]string, error) {
	// Implemented in Task 2
	return nil, nil
}

func (c *cliExecutor) Install(tool, version string) error {
	return exec.Command("mise", "install", tool+"@"+version).Run()
}

func (c *cliExecutor) IsInstalled(tool, version string) (bool, error) {
	out, err := exec.Command("mise", "ls", "--installed", tool).Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), version), nil
}

func (c *cliExecutor) ListInstalled(tool string) ([]string, error) {
	out, err := exec.Command("mise", "ls", "--installed", tool, "--json").Output()
	if err != nil {
		return nil, err
	}
	_ = out // parsed in Task 3
	return nil, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mise/ -v`
Expected: PASS (skipping real mise tests if not installed)

**Step 5: Commit**

```bash
git add internal/mise/resolver.go internal/mise/resolver_test.go
git commit -m "feat(mise): add RuntimeResolver with Available() and Which()"
```

---

### Task 2: RuntimeResolver — WhichVersion() and Detect()

Add version-specific resolution and project config file detection.

**Files:**
- Modify: `internal/mise/resolver.go`
- Modify: `internal/mise/resolver_test.go`

**Step 1: Write the failing tests**

Add to `internal/mise/resolver_test.go`:

```go
func TestWhichVersion_UsesMiseWhenAvailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{
		available: true,
		whichOut:  "/home/user/.local/share/mise/installs/php/8.3.0/bin/php",
	})
	path, err := r.WhichVersion("php", "8.3")
	if err != nil {
		t.Fatalf("WhichVersion error: %v", err)
	}
	if path != "/home/user/.local/share/mise/installs/php/8.3.0/bin/php" {
		t.Errorf("path = %q, want mise-resolved path", path)
	}
}

func TestWhichVersion_FallsBackWhenMiseUnavailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	// Falls back to exec.LookPath("php") — may or may not exist
	_, _ = r.WhichVersion("php", "8.3")
	// Just verifying it doesn't panic
}

func TestDetect_ReturnsParsedVersions(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{
		available: true,
		detectOut: map[string]string{"php": "8.3", "node": "20"},
	})
	versions, err := r.Detect("/tmp/project")
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if versions["php"] != "8.3" {
		t.Errorf("php = %q, want 8.3", versions["php"])
	}
	if versions["node"] != "20" {
		t.Errorf("node = %q, want 20", versions["node"])
	}
}

func TestDetect_ReturnsEmptyWhenMiseUnavailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	versions, err := r.Detect("/tmp/project")
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected empty map, got %v", versions)
	}
}
```

Update the stub to add `detectOut`:

```go
type stubExecutor struct {
	available  bool
	version    string
	whichOut   string
	whichErr   error
	detectOut  map[string]string
	detectErr  error
}

func (s *stubExecutor) Detect(dir string) (map[string]string, error) {
	if s.detectOut != nil {
		return s.detectOut, s.detectErr
	}
	return map[string]string{}, s.detectErr
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mise/ -v -run "WhichVersion|Detect"`
Expected: FAIL — WhichVersion and Detect methods don't exist on RuntimeResolver

**Step 3: Write the implementation**

Add to `internal/mise/resolver.go`:

```go
// WhichVersion resolves the path of a specific version of a tool.
// Falls back to exec.LookPath(tool) when mise is unavailable.
func (r *RuntimeResolver) WhichVersion(tool, version string) (string, error) {
	if r.Available() {
		path, err := r.exec.WhichVersion(tool, version)
		if err == nil && path != "" {
			return path, nil
		}
	}
	return exec.LookPath(tool)
}

// Detect scans a project directory for mise config files (.mise.toml, .tool-versions)
// and returns detected tool versions. Returns empty map if mise is unavailable.
func (r *RuntimeResolver) Detect(siteDir string) (map[string]string, error) {
	if !r.Available() {
		return map[string]string{}, nil
	}
	return r.exec.Detect(siteDir)
}
```

Update `cliExecutor.Detect()`:

```go
func (c *cliExecutor) Detect(dir string) (map[string]string, error) {
	out, err := exec.Command("mise", "ls", "--current", "--json", "-C", dir).Output()
	if err != nil {
		// No config file found or other error — return empty
		return map[string]string{}, nil
	}
	return parseCurrentVersions(out)
}
```

Add a parser function:

```go
import "encoding/json"

// parseCurrentVersions extracts tool→version from `mise ls --current --json` output.
// The JSON structure is: {"tool": [{"version": "x.y.z", ...}], ...}
func parseCurrentVersions(data []byte) (map[string]string, error) {
	var raw map[string][]struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]string{}, nil
	}
	result := map[string]string{}
	for tool, entries := range raw {
		if len(entries) > 0 && entries[0].Version != "" {
			result[tool] = entries[0].Version
		}
	}
	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mise/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mise/resolver.go internal/mise/resolver_test.go
git commit -m "feat(mise): add WhichVersion() and Detect() to RuntimeResolver"
```

---

### Task 3: RuntimeResolver — Install, IsInstalled, ListInstalled

Add installation and version listing methods.

**Files:**
- Modify: `internal/mise/resolver.go`
- Modify: `internal/mise/resolver_test.go`

**Step 1: Write the failing tests**

Add to `internal/mise/resolver_test.go`:

```go
func TestInstall_ErrorsWhenMiseUnavailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	err := r.Install("php", "8.3")
	if err == nil {
		t.Error("expected error when mise unavailable")
	}
}

func TestInstall_DelegatesToExecutor(t *testing.T) {
	stub := &stubExecutor{available: true}
	r := mise.NewWithExecutor(stub)
	err := r.Install("php", "8.3")
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if !stub.installCalled {
		t.Error("expected Install to delegate to executor")
	}
	if stub.installTool != "php" || stub.installVersion != "8.3" {
		t.Errorf("Install called with %q@%q, want php@8.3", stub.installTool, stub.installVersion)
	}
}

func TestIsInstalled_ReturnsFalseWhenMiseUnavailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	installed := r.IsInstalled("php", "8.3")
	if installed {
		t.Error("expected false when mise unavailable")
	}
}

func TestIsInstalled_DelegatesToExecutor(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{
		available:    true,
		isInstalled:  true,
	})
	if !r.IsInstalled("php", "8.3") {
		t.Error("expected true")
	}
}

func TestListInstalled_ErrorsWhenMiseUnavailable(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{available: false})
	_, err := r.ListInstalled("php")
	if err == nil {
		t.Error("expected error when mise unavailable")
	}
}

func TestListInstalled_ReturnsVersions(t *testing.T) {
	r := mise.NewWithExecutor(&stubExecutor{
		available:     true,
		listInstalled: []string{"8.2.0", "8.3.0"},
	})
	versions, err := r.ListInstalled("php")
	if err != nil {
		t.Fatalf("ListInstalled error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}
```

Update the stub to track install calls:

```go
type stubExecutor struct {
	available      bool
	version        string
	whichOut       string
	whichErr       error
	detectOut      map[string]string
	detectErr      error
	installCalled  bool
	installTool    string
	installVersion string
	installErr     error
	isInstalled    bool
	listInstalled  []string
}

func (s *stubExecutor) Install(tool, version string) error {
	s.installCalled = true
	s.installTool = tool
	s.installVersion = version
	return s.installErr
}

func (s *stubExecutor) IsInstalled(tool, version string) (bool, error) {
	return s.isInstalled, nil
}

func (s *stubExecutor) ListInstalled(tool string) ([]string, error) {
	return s.listInstalled, nil
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mise/ -v -run "Install|IsInstalled|ListInstalled"`
Expected: FAIL — methods don't exist

**Step 3: Write the implementation**

Add to `internal/mise/resolver.go`:

```go
import "fmt"

// Install installs a specific version of a tool via mise. Requires mise to be available.
func (r *RuntimeResolver) Install(tool, version string) error {
	if !r.Available() {
		return fmt.Errorf("mise is not available")
	}
	return r.exec.Install(tool, version)
}

// IsInstalled checks if a specific tool version is installed via mise.
// Returns false if mise is unavailable.
func (r *RuntimeResolver) IsInstalled(tool, version string) bool {
	if !r.Available() {
		return false
	}
	installed, err := r.exec.IsInstalled(tool, version)
	return err == nil && installed
}

// ListInstalled returns installed versions for a tool. Requires mise.
func (r *RuntimeResolver) ListInstalled(tool string) ([]string, error) {
	if !r.Available() {
		return nil, fmt.Errorf("mise is not available")
	}
	return r.exec.ListInstalled(tool)
}
```

Update `cliExecutor.ListInstalled()` to parse JSON:

```go
func (c *cliExecutor) ListInstalled(tool string) ([]string, error) {
	out, err := exec.Command("mise", "ls", "--installed", tool, "--json").Output()
	if err != nil {
		return nil, err
	}
	var raw map[string][]struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	var versions []string
	for _, entries := range raw {
		for _, e := range entries {
			if e.Version != "" {
				versions = append(versions, e.Version)
			}
		}
	}
	return versions, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mise/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mise/resolver.go internal/mise/resolver_test.go
git commit -m "feat(mise): add Install, IsInstalled, ListInstalled to RuntimeResolver"
```

---

### Task 4: Wire RuntimeResolver into Core

Add the resolver to `core.Config`, create it during startup, and expose new methods on the Core struct.

**Files:**
- Modify: `internal/core/core.go`
- Modify: `internal/core/core_test.go`

**Step 1: Write the failing tests**

Add to `internal/core/core_test.go`:

```go
func TestDetectSiteVersions_ReturnsDetectedVersions(t *testing.T) {
	cfg, _, _, _, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	// DetectSiteVersions returns whatever the resolver finds.
	// With no real mise installed, expect empty map.
	versions, err := c.DetectSiteVersions(t.TempDir())
	if err != nil {
		t.Fatalf("DetectSiteVersions: %v", err)
	}
	if versions == nil {
		t.Error("expected non-nil map")
	}
}

func TestMiseStatus_ReturnsInfo(t *testing.T) {
	cfg, _, _, _, _, _ := testConfig(t)
	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	info := c.MiseStatus()
	// Just check it doesn't panic; Available may be true or false
	_ = info.Available
	_ = info.Version
}

func TestCheckRuntimes_ReturnsStatusForSites(t *testing.T) {
	cfg, _, _, _, _, _ := testConfig(t)

	dir := t.TempDir()
	sitesJSON := fmt.Sprintf(`[{"path":%q,"domain":"app.test","php_version":"8.3"}]`, dir)
	os.MkdirAll(filepath.Dir(cfg.SitesFile), 0o755)
	os.WriteFile(cfg.SitesFile, []byte(sitesJSON), 0o644)

	c := core.NewCore(cfg)
	_ = c.Start()
	defer c.Stop()

	statuses := c.CheckRuntimes()
	if len(statuses) == 0 {
		t.Error("expected at least one runtime status entry")
	}
	found := false
	for _, s := range statuses {
		if s.Tool == "php" && s.Version == "8.3" && s.Domain == "app.test" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected status for php@8.3 / app.test, got: %v", statuses)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/core/ -v -run "DetectSiteVersions|MiseStatus|CheckRuntimes"`
Expected: FAIL — methods don't exist

**Step 3: Write the implementation**

Add to `internal/core/core.go`:

```go
import "github.com/andybarilla/flock/internal/mise"
```

Add `Resolver` to `Config`:

```go
type Config struct {
	// ... existing fields ...
	Resolver *mise.RuntimeResolver // nil = create default
}
```

Add `resolver` field to `Core`:

```go
type Core struct {
	// ... existing fields ...
	resolver *mise.RuntimeResolver
}
```

In `NewCore`, initialize the resolver:

```go
resolver := cfg.Resolver
if resolver == nil {
	resolver = mise.New()
}
// ... after creating all plugins ...
c := &Core{
	// ... existing fields ...
	resolver: resolver,
}
```

Add new methods:

```go
// MiseInfo holds mise availability status.
type MiseInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
}

// RuntimeStatus holds the install status of a runtime for a site.
type RuntimeStatus struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	Domain    string `json:"domain"`
}

func (c *Core) DetectSiteVersions(path string) (map[string]string, error) {
	return c.resolver.Detect(path)
}

func (c *Core) MiseStatus() MiseInfo {
	return MiseInfo{
		Available: c.resolver.Available(),
		Version:   c.resolver.Version(),
	}
}

func (c *Core) CheckRuntimes() []RuntimeStatus {
	var statuses []RuntimeStatus
	for _, site := range c.registry.List() {
		if site.PHPVersion != "" {
			statuses = append(statuses, RuntimeStatus{
				Tool:      "php",
				Version:   site.PHPVersion,
				Installed: c.resolver.IsInstalled("php", site.PHPVersion),
				Domain:    site.Domain,
			})
		}
		if site.NodeVersion != "" {
			statuses = append(statuses, RuntimeStatus{
				Tool:      "node",
				Version:   site.NodeVersion,
				Installed: c.resolver.IsInstalled("node", site.NodeVersion),
				Domain:    site.Domain,
			})
		}
	}
	return statuses
}

func (c *Core) InstallRuntime(tool, version string) error {
	return c.resolver.Install(tool, version)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/core/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/core.go internal/core/core_test.go
git commit -m "feat(core): wire RuntimeResolver and expose mise APIs"
```

---

### Task 5: Wails Bindings — Expose Mise APIs to Frontend

Add `DetectSiteVersions`, `MiseStatus`, `CheckRuntimes`, and `InstallRuntime` methods to `app.go`.

**Files:**
- Modify: `app.go`

**Step 1: Write the failing test**

There are no Go tests for `app.go` (it's a thin Wails binding layer). We verify by building:

Run: `go build ./...`
Expected: PASS (current state compiles)

**Step 2: Add the binding methods**

Add to `app.go`:

```go
import "github.com/andybarilla/flock/internal/core"

// DetectSiteVersions scans a project directory for mise config files
// and returns detected runtime versions (e.g., {"php": "8.3", "node": "20"}).
func (a *App) DetectSiteVersions(path string) (map[string]string, error) {
	return a.core.DetectSiteVersions(path)
}

// MiseStatus returns whether mise is available and its version.
func (a *App) MiseStatus() core.MiseInfo {
	return a.core.MiseStatus()
}

// CheckRuntimes returns installation status for all site runtimes.
func (a *App) CheckRuntimes() []core.RuntimeStatus {
	return a.core.CheckRuntimes()
}

// InstallRuntime installs a specific tool version via mise.
func (a *App) InstallRuntime(tool, version string) error {
	return a.core.InstallRuntime(tool, version)
}
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: PASS

**Step 4: Run all Go tests**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add app.go
git commit -m "feat(app): expose DetectSiteVersions, MiseStatus, CheckRuntimes, InstallRuntime bindings"
```

---

### Task 6: Frontend — Auto-Detection in Add Site Modal

Add a debounced call to `DetectSiteVersions` when the path input changes. Pre-fill PHP and Node version fields with detected values and show a detection hint.

**Files:**
- Modify: `frontend/src/AddSiteForm.svelte`
- Modify: `frontend/src/AddSiteForm.test.js`

**Step 1: Write the failing tests**

Add to `frontend/src/AddSiteForm.test.js`:

```javascript
import { vi, describe, it, expect } from 'vitest';

// Update the existing mock at the top of the file to include DetectSiteVersions:
vi.mock('../../wailsjs/go/main/App.js', () => ({
  DetectSiteVersions: vi.fn().mockResolvedValue({}),
}));

// Add these test cases:
describe('auto-detection', () => {
  it('calls DetectSiteVersions when path changes', async () => {
    const { DetectSiteVersions } = await import('../../wailsjs/go/main/App.js');
    DetectSiteVersions.mockResolvedValue({ php: '8.3', node: '20' });
    vi.useFakeTimers();

    const { container } = render(AddSiteForm, { props: { open: true, onAdd: vi.fn() } });
    const pathInput = container.querySelector('input[placeholder="/home/user/projects/myapp"]');
    await fireEvent.input(pathInput, { target: { value: '/tmp/myapp' } });

    vi.advanceTimersByTime(500); // debounce delay
    await vi.waitFor(() => {
      expect(DetectSiteVersions).toHaveBeenCalledWith('/tmp/myapp');
    });

    vi.useRealTimers();
  });

  it('shows detection hint when versions are found', async () => {
    const { DetectSiteVersions } = await import('../../wailsjs/go/main/App.js');
    DetectSiteVersions.mockResolvedValue({ php: '8.3', node: '20' });
    vi.useFakeTimers();

    const { container } = render(AddSiteForm, { props: { open: true, onAdd: vi.fn() } });
    const pathInput = container.querySelector('input[placeholder="/home/user/projects/myapp"]');
    await fireEvent.input(pathInput, { target: { value: '/tmp/myapp' } });

    vi.advanceTimersByTime(500);
    await vi.waitFor(() => {
      expect(container.textContent).toContain('detected');
    });

    vi.useRealTimers();
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npm test -- --run`
Expected: FAIL — DetectSiteVersions not called, no detection hint

**Step 3: Write the implementation**

Update `frontend/src/AddSiteForm.svelte`:

Add to `<script>`:

```javascript
import { DetectSiteVersions } from '../wailsjs/go/main/App.js';

let detectedSource = ''; // e.g., ".mise.toml"
let detectTimer;

function handlePathInput() {
  if (!domain || domain === inferDomain(path.slice(0, path.length - 1))) {
    domain = inferDomain(path);
  }
  // Debounced auto-detection
  clearTimeout(detectTimer);
  if (path) {
    detectTimer = setTimeout(async () => {
      try {
        const versions = await DetectSiteVersions(path);
        if (versions && Object.keys(versions).length > 0) {
          if (versions.php && !phpVersion) phpVersion = versions.php;
          if (versions.node && !nodeVersion) nodeVersion = versions.node;
          detectedSource = 'detected from project config';
        } else {
          detectedSource = '';
        }
      } catch {
        detectedSource = '';
      }
    }, 300);
  }
}
```

Add detection hint below the version fields:

```svelte
{#if detectedSource}
  <p class="text-xs text-success mt-1">{detectedSource}</p>
{/if}
```

Clear `detectedSource` when the form is reset (in `handleSubmit` success block).

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npm test -- --run`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/AddSiteForm.svelte frontend/src/AddSiteForm.test.js
git commit -m "feat(frontend): auto-detect runtime versions from project config in Add Site modal"
```

---

### Task 7: Frontend — Mise Status in Settings

Add a "Runtime Manager" card to the Settings tab showing mise availability.

**Files:**
- Modify: `frontend/src/SettingsTab.svelte`
- Modify: `frontend/src/SettingsTab.test.js`

**Step 1: Write the failing tests**

Add to `frontend/src/SettingsTab.test.js`:

```javascript
vi.mock('../../wailsjs/go/main/App.js', () => ({
  MiseStatus: vi.fn().mockResolvedValue({ available: false, version: '' }),
}));

describe('mise status', () => {
  it('shows detected badge when mise is available', async () => {
    const { MiseStatus } = await import('../../wailsjs/go/main/App.js');
    MiseStatus.mockResolvedValue({ available: true, version: 'mise 2024.12.0' });

    const { getByText } = render(SettingsTab);
    await vi.waitFor(() => {
      expect(getByText(/mise 2024\.12\.0/)).toBeTruthy();
    });
  });

  it('shows not-found message when mise is unavailable', async () => {
    const { MiseStatus } = await import('../../wailsjs/go/main/App.js');
    MiseStatus.mockResolvedValue({ available: false, version: '' });

    const { getByText } = render(SettingsTab);
    await vi.waitFor(() => {
      expect(getByText(/Install mise/i)).toBeTruthy();
    });
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npm test -- --run`
Expected: FAIL — no mise status card

**Step 3: Write the implementation**

Update `frontend/src/SettingsTab.svelte`:

```svelte
<script>
  import { onMount } from 'svelte';
  import { theme, toggleTheme } from './lib/theme.js';
  import { MiseStatus } from '../wailsjs/go/main/App.js';

  let miseInfo = { available: false, version: '' };

  onMount(async () => {
    try {
      miseInfo = await MiseStatus();
    } catch {
      // ignore — not critical
    }
  });
</script>

<div class="space-y-6">
  <!-- Appearance card (existing) -->
  <div class="card bg-base-200 p-6">
    <h3 class="font-semibold text-base-content mb-4">Appearance</h3>
    <label class="flex items-center justify-between cursor-pointer">
      <div>
        <span class="font-medium text-base-content">Dark mode</span>
        <p class="text-sm text-base-content/60">Switch between light and dark themes</p>
      </div>
      <input
        type="checkbox"
        class="toggle toggle-primary"
        checked={$theme === 'dark'}
        on:change={toggleTheme}
        aria-label="Dark mode"
      />
    </label>
  </div>

  <!-- Runtime Manager card (new) -->
  <div class="card bg-base-200 p-6">
    <h3 class="font-semibold text-base-content mb-4">Runtime Manager</h3>
    {#if miseInfo.available}
      <div class="flex items-center gap-2">
        <span class="badge badge-success badge-sm">Detected</span>
        <span class="text-sm text-base-content">{miseInfo.version}</span>
      </div>
    {:else}
      <div class="flex items-center gap-2">
        <span class="badge badge-ghost badge-sm">Not found</span>
        <span class="text-sm text-base-content/60">
          Install mise for automatic runtime version management
        </span>
      </div>
      <a href="https://mise.jdx.dev" target="_blank" rel="noopener noreferrer" class="link link-primary text-sm mt-2 inline-block">
        mise.jdx.dev
      </a>
    {/if}
  </div>

  <!-- About card (existing) -->
  <div class="card bg-base-200 p-6">
    <h3 class="font-semibold text-base-content mb-4">About</h3>
    <div class="space-y-2 text-sm">
      <div class="flex justify-between">
        <span class="text-base-content/70">Version</span>
        <span class="font-medium text-base-content">1.0.0</span>
      </div>
    </div>
  </div>
</div>
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npm test -- --run`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/SettingsTab.svelte frontend/src/SettingsTab.test.js
git commit -m "feat(frontend): add mise status card to Settings tab"
```

---

### Task 8: Frontend — Runtime Warning Badges and Install Button

Add warning badges to SiteCard and SiteList table for missing runtimes, with an Install button when mise is available.

**Files:**
- Modify: `frontend/src/SiteCard.svelte`
- Create: `frontend/src/SiteCard.test.js` (update existing)
- Modify: `frontend/src/SiteList.svelte`
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/App.test.js`

**Step 1: Write the failing tests**

Add to `frontend/src/SiteCard.test.js`:

```javascript
describe('runtime warnings', () => {
  it('shows warning badge when runtime is not installed', () => {
    const { getByText } = render(SiteCard, {
      props: {
        site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
        runtimeStatuses: [
          { tool: 'php', version: '8.3', installed: false, domain: 'app.test' },
        ],
        miseAvailable: true,
      },
    });
    expect(getByText(/PHP 8\.3 not found/)).toBeTruthy();
  });

  it('shows Install button when mise is available', () => {
    const { getByRole } = render(SiteCard, {
      props: {
        site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
        runtimeStatuses: [
          { tool: 'php', version: '8.3', installed: false, domain: 'app.test' },
        ],
        miseAvailable: true,
      },
    });
    expect(getByRole('button', { name: /install/i })).toBeTruthy();
  });

  it('hides Install button when mise is unavailable', () => {
    const { queryByRole } = render(SiteCard, {
      props: {
        site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
        runtimeStatuses: [
          { tool: 'php', version: '8.3', installed: false, domain: 'app.test' },
        ],
        miseAvailable: false,
      },
    });
    expect(queryByRole('button', { name: /install/i })).toBeNull();
  });

  it('shows no warning when runtime is installed', () => {
    const { queryByText } = render(SiteCard, {
      props: {
        site: { domain: 'app.test', path: '/tmp', php_version: '8.3', tls: false },
        runtimeStatuses: [
          { tool: 'php', version: '8.3', installed: true, domain: 'app.test' },
        ],
        miseAvailable: true,
      },
    });
    expect(queryByText(/not found/)).toBeNull();
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npm test -- --run`
Expected: FAIL — SiteCard doesn't accept runtimeStatuses prop

**Step 3: Write the implementation**

Update `frontend/src/SiteCard.svelte`:

```svelte
<script>
  import { createEventDispatcher } from 'svelte';

  export let site;
  export let onRemove = () => {};
  export let runtimeStatuses = [];
  export let miseAvailable = false;
  export let onInstall = () => {};

  const dispatch = createEventDispatcher();

  $: phpBadge = site.php_version ? `PHP ${site.php_version}` : '';
  $: nodeBadge = site.node_version ? `Node ${site.node_version}` : '';

  $: missingRuntimes = runtimeStatuses.filter(
    s => s.domain === site.domain && !s.installed
  );
</script>
```

Add warning section after the badges:

```svelte
{#each missingRuntimes as missing}
  <div class="mt-2 flex items-center gap-2">
    <span class="badge badge-warning badge-sm">{missing.tool === 'php' ? 'PHP' : 'Node'} {missing.version} not found</span>
    {#if miseAvailable}
      <button class="btn btn-xs btn-outline btn-warning" on:click={() => onInstall(missing.tool, missing.version)}>
        Install
      </button>
    {/if}
  </div>
{/each}
```

Update `SiteList.svelte` to pass the new props through:

```svelte
export let runtimeStatuses = [];
export let miseAvailable = false;
export let onInstall = () => {};
```

And pass them to `SiteCard`:

```svelte
<SiteCard {site} onRemove={requestRemove} {runtimeStatuses} {miseAvailable} {onInstall} />
```

Also add warning badges in the table view rows for missing runtimes.

Update `App.svelte` to fetch runtime statuses:

```javascript
import { CheckRuntimes, InstallRuntime, MiseStatus } from '../wailsjs/go/main/App.js';

let runtimeStatuses = [];
let miseAvailable = false;

async function refreshRuntimes() {
  try {
    runtimeStatuses = await CheckRuntimes() || [];
    const status = await MiseStatus();
    miseAvailable = status.available;
  } catch {
    // non-critical
  }
}

async function handleInstall(tool, version) {
  try {
    await InstallRuntime(tool, version);
    notifySuccess(`${tool}@${version} installed.`);
    await refreshRuntimes();
  } catch (e) {
    notifyError(friendlyError(e.message || String(e)));
  }
}

onMount(() => {
  initTheme();
  refreshSites();
  refreshServices();
  refreshRuntimes();
});
```

Pass to SiteList:

```svelte
<SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} {runtimeStatuses} {miseAvailable} onInstall={handleInstall} on:addsite={() => { addFormOpen = true; }} />
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npm test -- --run`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/SiteCard.svelte frontend/src/SiteCard.test.js frontend/src/SiteList.svelte frontend/src/App.svelte frontend/src/App.test.js
git commit -m "feat(frontend): add runtime warning badges and Install button to site views"
```

---

### Task 9: Update Wails Bindings and Integration Test

Regenerate Wails TypeScript/JavaScript bindings and run the full test suite.

**Files:**
- Auto-generated: `frontend/wailsjs/go/main/App.js`
- Auto-generated: `frontend/wailsjs/go/main/App.d.ts`

**Step 1: Regenerate Wails bindings**

Run: `wails generate module`

If `wails generate` is not available, manually add the new exports to `frontend/wailsjs/go/main/App.js`:

```javascript
export function DetectSiteVersions(arg1) {
  return window['go']['main']['App']['DetectSiteVersions'](arg1);
}

export function MiseStatus() {
  return window['go']['main']['App']['MiseStatus']();
}

export function CheckRuntimes() {
  return window['go']['main']['App']['CheckRuntimes']();
}

export function InstallRuntime(arg1, arg2) {
  return window['go']['main']['App']['InstallRuntime'](arg1, arg2);
}
```

**Step 2: Run the full Go test suite**

Run: `go test ./...`
Expected: PASS

**Step 3: Run the full frontend test suite**

Run: `cd frontend && npm test -- --run`
Expected: PASS

**Step 4: Commit**

```bash
git add frontend/wailsjs/go/main/App.js frontend/wailsjs/go/main/App.d.ts
git commit -m "chore: regenerate Wails bindings for mise integration APIs"
```

---

### Task 10: Update Roadmap

Mark the mise integration as complete in the roadmap.

**Files:**
- Modify: `docs/ROADMAP.md`

**Step 1: Add mise integration to the roadmap**

Add under Phase 4 or a new section:

```markdown
### Phase 5 — Runtime Management

- [x] Mise integration (auto-detect, install, runtime resolution) — See: docs/plans/2026-03-08-mise-integration-design.md
```

**Step 2: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs: add mise integration to roadmap"
```
