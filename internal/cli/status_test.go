package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
)

func TestProcessStatus_NoPIDFile(t *testing.T) {
	wsRoot := t.TempDir()
	got := processStatus(wsRoot, "nonexistent")
	if got != runner.StatusStopped {
		t.Errorf("expected stopped for missing PID file, got %s", got)
	}
}

func TestProcessStatus_ProcessAlive(t *testing.T) {
	wsRoot := t.TempDir()
	pidDir := runner.PIDDirPath(wsRoot)
	os.MkdirAll(pidDir, 0755)

	// Use current process PID (guaranteed to be alive)
	info := runner.PIDInfo{
		PID:       os.Getpid(),
		Command:   "test",
		StartedAt: time.Now(),
	}
	if err := runner.WritePIDFile(pidDir, "myservice", info); err != nil {
		t.Fatal(err)
	}

	got := processStatus(wsRoot, "myservice")
	if got != runner.StatusRunning {
		t.Errorf("expected running for live PID, got %s", got)
	}
}

func TestProcessStatus_ProcessDead(t *testing.T) {
	wsRoot := t.TempDir()
	pidDir := runner.PIDDirPath(wsRoot)
	os.MkdirAll(pidDir, 0755)

	// Use PID 999999 (very unlikely to exist)
	info := runner.PIDInfo{
		PID:       999999,
		Command:   "test",
		StartedAt: time.Now(),
	}
	if err := runner.WritePIDFile(pidDir, "deadservice", info); err != nil {
		t.Fatal(err)
	}

	got := processStatus(wsRoot, "deadservice")
	if got != runner.StatusStopped {
		t.Errorf("expected stopped for dead PID, got %s", got)
	}

	// Verify stale PID file was cleaned up
	pidPath := filepath.Join(pidDir, "deadservice.pid")
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("expected stale PID file to be removed")
	}
}
