//go:build windows

package databases

import (
	"os"
	"os/exec"
)

// setSysProcAttr is a no-op on Windows (Setpgid not available).
func setSysProcAttr(cmd *exec.Cmd) {}

// isProcessAlive checks whether a process is still running.
// On Windows, Signal(0) is not supported, so we attempt to find
// the process — FindProcess always succeeds on Windows, so we
// try sending a nil signal via os.Process.Signal which returns
// an error if the process has exited.
func isProcessAlive(p *os.Process) bool {
	err := p.Signal(os.Signal(nil))
	return err == nil
}

// stopProcess kills the process on Windows (no SIGTERM support).
func stopProcess(p *os.Process) error {
	if err := p.Kill(); err != nil {
		return err
	}
	_, _ = p.Wait()
	return nil
}
