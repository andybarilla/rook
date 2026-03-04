package node

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type appEntry struct {
	cmd  *exec.Cmd
	port int
	done chan struct{}
}

type ProcessRunner struct {
	mu   sync.Mutex
	apps map[string]*appEntry // siteDir -> entry
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{
		apps: map[string]*appEntry{},
	}
}

func (r *ProcessRunner) StartApp(siteDir string, port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.Command("npm", "start")
	cmd.Dir = siteDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start npm in %s: %w", siteDir, err)
	}

	done := make(chan struct{})
	r.apps[siteDir] = &appEntry{cmd: cmd, port: port, done: done}

	// Goroutine to clean up when process exits
	go func() {
		cmd.Wait()
		close(done)
		r.mu.Lock()
		defer r.mu.Unlock()
		if entry, ok := r.apps[siteDir]; ok && entry.cmd == cmd {
			delete(r.apps, siteDir)
		}
	}()

	return nil
}

func (r *ProcessRunner) StopApp(siteDir string) error {
	r.mu.Lock()
	entry, ok := r.apps[siteDir]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("no running app at %s", siteDir)
	}
	delete(r.apps, siteDir)
	r.mu.Unlock()

	// Kill the process. Use Kill directly for cross-platform support
	// (os.Interrupt is not supported on Windows).
	entry.cmd.Process.Kill()

	// Wait for the process to fully exit so all file handles are released.
	<-entry.done
	return nil
}

func (r *ProcessRunner) IsRunning(siteDir string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.apps[siteDir]
	return ok
}

func (r *ProcessRunner) AppPort(siteDir string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.apps[siteDir]
	if !ok {
		return 0
	}
	return entry.port
}
