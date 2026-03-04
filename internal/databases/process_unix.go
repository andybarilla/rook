//go:build !windows

package databases

import (
	"os"
	"os/exec"
	"syscall"
)

// setSysProcAttr configures the command to run in its own process group.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// isProcessAlive checks whether a process is still running.
func isProcessAlive(p *os.Process) bool {
	return p.Signal(syscall.Signal(0)) == nil
}

// stopProcess sends SIGTERM and waits for the process to exit.
func stopProcess(p *os.Process) error {
	if err := p.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	_, _ = p.Wait()
	return nil
}
