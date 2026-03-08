package mise

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Executor abstracts mise CLI interactions for testability.
type Executor interface {
	Available() (bool, string)
	Which(tool string) (string, error)
	WhichVersion(tool, version string) (string, error)
	Detect(dir string) (map[string]string, error)
	Install(tool, version string) error
	IsInstalled(tool, version string) (bool, error)
	ListInstalled(tool string) ([]string, error)
}

// RuntimeResolver provides runtime path resolution via mise with fallback.
type RuntimeResolver struct {
	executor Executor

	mu        sync.Mutex
	cached    *bool
	cachedVer string
}

// New creates a RuntimeResolver using the real mise CLI.
func New() *RuntimeResolver {
	return &RuntimeResolver{executor: &cliExecutor{}}
}

// NewWithExecutor creates a RuntimeResolver with a custom Executor (for testing).
func NewWithExecutor(executor Executor) *RuntimeResolver {
	return &RuntimeResolver{executor: executor}
}

// Available returns whether mise is available and its version string.
// The result is cached after the first call.
func (r *RuntimeResolver) Available() (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cached != nil {
		return *r.cached, r.cachedVer
	}

	ok, ver := r.executor.Available()
	r.cached = &ok
	r.cachedVer = ver
	return ok, ver
}

// Version returns the mise version string, calling Available() first to ensure
// the cache is populated.
func (r *RuntimeResolver) Version() string {
	r.Available() // ensure cache is populated
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cachedVer
}

// Which resolves the path to a tool binary. It tries mise first (if available),
// then falls back to exec.LookPath.
func (r *RuntimeResolver) Which(tool string) (string, error) {
	ok, _ := r.Available()
	if ok {
		path, err := r.executor.Which(tool)
		if err == nil {
			return path, nil
		}
		// mise failed, fall through to LookPath
	}

	path, err := exec.LookPath(tool)
	if err != nil {
		return "", fmt.Errorf("tool %q not found via mise or PATH: %w", tool, err)
	}
	return path, nil
}

// WhichVersion resolves the path to a specific version of a tool binary.
// It tries mise first (if available), then falls back to exec.LookPath.
func (r *RuntimeResolver) WhichVersion(tool, version string) (string, error) {
	ok, _ := r.Available()
	if ok {
		path, err := r.executor.WhichVersion(tool, version)
		if err == nil {
			return path, nil
		}
		// mise failed, fall through to LookPath
	}

	path, err := exec.LookPath(tool)
	if err != nil {
		return "", fmt.Errorf("tool %q version %q not found via mise or PATH: %w", tool, version, err)
	}
	return path, nil
}

// Detect returns tool versions configured for a site directory.
// It checks mise first (if available), then falls back to project config
// files (composer.json, package.json) to fill any gaps.
// Errors from mise are swallowed intentionally — this is best-effort detection.
func (r *RuntimeResolver) Detect(siteDir string) (map[string]string, error) {
	var result map[string]string

	ok, _ := r.Available()
	if ok {
		var err error
		result, err = r.executor.Detect(siteDir)
		if err != nil {
			result = map[string]string{}
		}
	} else {
		result = map[string]string{}
	}

	// Fill gaps from project config files
	for tool, version := range detectFromProjectFiles(siteDir) {
		if _, exists := result[tool]; !exists {
			result[tool] = version
		}
	}

	return result, nil
}

// Install installs a specific version of a tool via mise.
// Returns an error if mise is not available.
func (r *RuntimeResolver) Install(tool, version string) error {
	ok, _ := r.Available()
	if !ok {
		return fmt.Errorf("mise is not available")
	}
	return r.executor.Install(tool, version)
}

// IsInstalled checks whether a specific version of a tool is installed via mise.
// Returns false if mise is not available or if the check fails.
func (r *RuntimeResolver) IsInstalled(tool, version string) bool {
	ok, _ := r.Available()
	if !ok {
		return false
	}
	installed, err := r.executor.IsInstalled(tool, version)
	if err != nil {
		return false
	}
	return installed
}

// ListInstalled returns all installed versions of a tool via mise.
// Returns an error if mise is not available.
func (r *RuntimeResolver) ListInstalled(tool string) ([]string, error) {
	ok, _ := r.Available()
	if !ok {
		return nil, fmt.Errorf("mise is not available")
	}
	return r.executor.ListInstalled(tool)
}

// cliExecutor implements Executor by calling the real mise CLI.
type cliExecutor struct{}

func (c *cliExecutor) Available() (bool, string) {
	out, err := exec.Command("mise", "--version").Output()
	if err != nil {
		return false, ""
	}
	ver := strings.TrimSpace(strings.SplitN(string(out), " ", 2)[0])
	return true, ver
}

func (c *cliExecutor) Which(tool string) (string, error) {
	out, err := exec.Command("mise", "which", tool).Output()
	if err != nil {
		return "", fmt.Errorf("mise which %s: %w", tool, err)
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
	out, err := exec.Command("mise", "ls", "--current", "--json", "-C", dir).Output()
	if err != nil {
		return map[string]string{}, nil
	}
	return parseCurrentVersions(out)
}

// parseCurrentVersions parses the JSON output of `mise ls --current --json`.
// The format is: {"tool": [{"version": "x.y.z", ...}], ...}
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

func (c *cliExecutor) Install(tool, version string) error {
	return exec.Command("mise", "install", tool+"@"+version).Run()
}

func (c *cliExecutor) IsInstalled(tool, version string) (bool, error) {
	versions, err := c.ListInstalled(tool)
	if err != nil {
		return false, err
	}
	for _, v := range versions {
		if v == version || strings.HasPrefix(v, version+".") {
			return true, nil
		}
	}
	return false, nil
}

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

// Package-level compiled regexes for parseVersionConstraint.
var (
	versionPrefixRe = regexp.MustCompile(`^[~^>=<v]+`)
	versionNumberRe = regexp.MustCompile(`^\d+(\.\d+)*$`)
)

// parseVersionConstraint extracts a base version number from a version
// constraint string (e.g., "^8.2" -> "8.2", ">=18" -> "18").
// For compound constraints (e.g., "^8.2 || ^8.3"), returns the first version.
// Returns empty string if no version can be extracted.
func parseVersionConstraint(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// For compound constraints, take the first part
	if idx := strings.Index(s, "||"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	// For space-separated constraints (e.g., ">=8.2 <9.0"), take the first part
	if idx := strings.Index(s, " "); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}
	// Strip leading operators and 'v' prefix
	s = versionPrefixRe.ReplaceAllString(s, "")
	// Validate it looks like a version number
	if !versionNumberRe.MatchString(s) {
		return ""
	}
	return s
}

// detectFromProjectFiles checks for composer.json and package.json in the
// given directory and extracts runtime version requirements.
// Returns a map of tool name -> version (e.g., {"php": "8.2", "node": "18"}).
func detectFromProjectFiles(dir string) map[string]string {
	result := map[string]string{}

	// Check composer.json for PHP version
	if data, err := os.ReadFile(filepath.Join(dir, "composer.json")); err == nil {
		var composer struct {
			Require map[string]string `json:"require"`
		}
		if json.Unmarshal(data, &composer) == nil {
			if constraint, ok := composer.Require["php"]; ok {
				if v := parseVersionConstraint(constraint); v != "" {
					result["php"] = v
				}
			}
		}
	}

	// Check package.json for Node version
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg struct {
			Engines map[string]string `json:"engines"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			if constraint, ok := pkg.Engines["node"]; ok {
				if v := parseVersionConstraint(constraint); v != "" {
					result["node"] = v
				}
			}
		}
	}

	return result
}
