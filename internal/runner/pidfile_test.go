package runner_test

import (
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
)

func TestPIDDirPath(t *testing.T) {
	got := runner.PIDDirPath("/home/user/myproject")
	want := filepath.Join("/home/user/myproject", ".rook", ".cache", "pids")
	if got != want {
		t.Errorf("PIDDirPath = %q, want %q", got, want)
	}
}

func TestWriteReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	info := runner.PIDInfo{
		PID:       12345,
		Command:   "make run",
		StartedAt: time.Now().Truncate(time.Second),
	}
	if err := runner.WritePIDFile(dir, "api", info); err != nil {
		t.Fatal(err)
	}
	got, err := runner.ReadPIDFile(dir, "api")
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != info.PID {
		t.Errorf("PID = %d, want %d", got.PID, info.PID)
	}
	if got.Command != info.Command {
		t.Errorf("Command = %q, want %q", got.Command, info.Command)
	}
	if !got.StartedAt.Equal(info.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, info.StartedAt)
	}
}

func TestReadPIDFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := runner.ReadPIDFile(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent PID file")
	}
}

func TestRemovePIDFile(t *testing.T) {
	dir := t.TempDir()
	info := runner.PIDInfo{PID: 1, Command: "echo", StartedAt: time.Now()}
	runner.WritePIDFile(dir, "svc", info)
	if err := runner.RemovePIDFile(dir, "svc"); err != nil {
		t.Fatal(err)
	}
	_, err := runner.ReadPIDFile(dir, "svc")
	if err == nil {
		t.Error("expected error after removal")
	}
}

func TestRemovePIDFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	if err := runner.RemovePIDFile(dir, "nonexistent"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListPIDFiles(t *testing.T) {
	dir := t.TempDir()
	runner.WritePIDFile(dir, "api", runner.PIDInfo{PID: 1, Command: "a", StartedAt: time.Now()})
	runner.WritePIDFile(dir, "worker", runner.PIDInfo{PID: 2, Command: "b", StartedAt: time.Now()})

	names, err := runner.ListPIDFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 PID files, got %d", len(names))
	}
	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	if !got["api"] || !got["worker"] {
		t.Errorf("expected api and worker, got %v", names)
	}
}

func TestListPIDFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names, err := runner.ListPIDFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0, got %d", len(names))
	}
}

func TestListPIDFiles_NonexistentDir(t *testing.T) {
	names, err := runner.ListPIDFiles("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0, got %d", len(names))
	}
}

func TestIsProcessAlive_RunningProcess(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	if !runner.IsProcessAlive(cmd.Process.Pid) {
		t.Error("expected alive for running process")
	}
}

func TestIsProcessAlive_DeadProcess(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Kill()
	cmd.Wait()

	if runner.IsProcessAlive(pid) {
		t.Error("expected dead for killed process")
	}
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	if runner.IsProcessAlive(0) {
		t.Error("expected false for PID 0")
	}
	if runner.IsProcessAlive(-1) {
		t.Error("expected false for negative PID")
	}
}
