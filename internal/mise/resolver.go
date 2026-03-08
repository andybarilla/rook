package mise

import (
	"fmt"
	"os/exec"
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

// Version returns the cached mise version string, or empty if not yet checked.
func (r *RuntimeResolver) Version() string {
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
	return "", fmt.Errorf("not implemented")
}

func (c *cliExecutor) Detect(dir string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *cliExecutor) Install(tool, version string) error {
	return fmt.Errorf("not implemented")
}

func (c *cliExecutor) IsInstalled(tool, version string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (c *cliExecutor) ListInstalled(tool string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}
