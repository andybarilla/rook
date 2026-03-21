package runner_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestProcessRunner_StartAndStop(t *testing.T) {
	r := runner.NewProcessRunner()
	svc := workspace.Service{Command: "sleep 60"}
	handle, err := r.Start(context.Background(), "test-svc", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, _ := r.Status(handle)
	if status != runner.StatusRunning {
		t.Errorf("expected running, got %s", status)
	}
	r.Stop(handle)
	time.Sleep(100 * time.Millisecond)
	status, _ = r.Status(handle)
	if status == runner.StatusRunning {
		t.Error("expected not running after stop")
	}
}

func TestProcessRunner_Logs(t *testing.T) {
	r := runner.NewProcessRunner()
	svc := workspace.Service{Command: "echo hello-from-process"}
	handle, _ := r.Start(context.Background(), "echo-svc", svc, nil, t.TempDir())
	time.Sleep(200 * time.Millisecond)
	reader, _ := r.Logs(handle)
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	if len(data) == 0 {
		t.Error("expected log output")
	}
}

func TestProcessRunner_WorkingDir(t *testing.T) {
	r := runner.NewProcessRunner()
	dir := t.TempDir()
	svc := workspace.Service{Command: "pwd"}
	handle, _ := r.Start(context.Background(), "pwd-svc", svc, nil, dir)
	time.Sleep(200 * time.Millisecond)
	reader, _ := r.Logs(handle)
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	if len(data) == 0 {
		t.Error("expected pwd output")
	}
	r.Stop(handle)
}

func TestProcessRunner_FileLogging(t *testing.T) {
	logDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetLogDir(logDir)

	svc := workspace.Service{Command: "echo hello-from-log"}
	handle, err := r.Start(context.Background(), "log-svc", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	r.Stop(handle)

	logPath := filepath.Join(logDir, "log-svc.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "--- rook up") {
		t.Error("expected session separator in log file")
	}
	if !strings.Contains(content, "hello-from-log") {
		t.Errorf("expected process output in log file, got: %s", content)
	}
}

func TestProcessRunner_Start_CreatesPIDFile(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	svc := workspace.Service{Command: "sleep 60"}
	handle, err := r.Start(context.Background(), "api", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop(handle)

	info, err := runner.ReadPIDFile(pidDir, "api")
	if err != nil {
		t.Fatalf("PID file not created: %v", err)
	}
	if info.PID <= 0 {
		t.Errorf("expected positive PID, got %d", info.PID)
	}
	if info.Command != "sleep 60" {
		t.Errorf("expected command 'sleep 60', got %q", info.Command)
	}
}

func TestProcessRunner_Stop_RemovesPIDFile(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	svc := workspace.Service{Command: "sleep 60"}
	handle, err := r.Start(context.Background(), "api", svc, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := runner.ReadPIDFile(pidDir, "api"); err != nil {
		t.Fatalf("PID file not created: %v", err)
	}

	r.Stop(handle)

	if _, err := runner.ReadPIDFile(pidDir, "api"); err == nil {
		t.Error("PID file should have been removed after Stop")
	}
}

func TestProcessRunner_FileLogging_AppendsSessions(t *testing.T) {
	logDir := t.TempDir()

	// First session
	r1 := runner.NewProcessRunner()
	r1.SetLogDir(logDir)
	svc := workspace.Service{Command: "echo session-one"}
	h1, _ := r1.Start(context.Background(), "app", svc, nil, t.TempDir())
	time.Sleep(200 * time.Millisecond)
	r1.Stop(h1)

	// Second session
	r2 := runner.NewProcessRunner()
	r2.SetLogDir(logDir)
	svc2 := workspace.Service{Command: "echo session-two"}
	h2, _ := r2.Start(context.Background(), "app", svc2, nil, t.TempDir())
	time.Sleep(200 * time.Millisecond)
	r2.Stop(h2)

	data, _ := os.ReadFile(filepath.Join(logDir, "app.log"))
	content := string(data)
	if !strings.Contains(content, "session-one") {
		t.Error("expected first session output")
	}
	if !strings.Contains(content, "session-two") {
		t.Error("expected second session output")
	}
	if strings.Count(content, "--- rook up") != 2 {
		t.Errorf("expected 2 session separators, got %d", strings.Count(content, "--- rook up"))
	}
}

func TestProcessRunner_Reconnect_AliveProcess(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	handle, err := r.Reconnect("worker")
	if err != nil {
		t.Fatal(err)
	}
	if handle.Type != "process" {
		t.Errorf("expected type process, got %s", handle.Type)
	}

	status, _ := r.Status(handle)
	if status != runner.StatusRunning {
		t.Errorf("expected running, got %s", status)
	}
}

func TestProcessRunner_Reconnect_DeadProcess(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	cmd.Process.Kill()
	cmd.Wait()

	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	_, err := r.Reconnect("worker")
	if err == nil {
		t.Error("expected error for dead process")
	}

	if _, readErr := runner.ReadPIDFile(pidDir, "worker"); readErr == nil {
		t.Error("stale PID file should have been removed")
	}
}

func TestProcessRunner_Reconnect_NoPIDFile(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	_, err := r.Reconnect("nonexistent")
	if err == nil {
		t.Error("expected error when no PID file exists")
	}
}

func TestProcessRunner_Status_ReconnectedDies(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       cmd.Process.Pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	handle, err := r.Reconnect("worker")
	if err != nil {
		t.Fatal(err)
	}

	cmd.Process.Kill()
	cmd.Wait()
	time.Sleep(100 * time.Millisecond)

	status, _ := r.Status(handle)
	if status == runner.StatusRunning {
		t.Error("expected non-running status after process death")
	}
}

func TestProcessRunner_Stop_ReconnectedEntry(t *testing.T) {
	pidDir := t.TempDir()
	r := runner.NewProcessRunner()
	r.SetPIDDir(pidDir)

	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	runner.WritePIDFile(pidDir, "worker", runner.PIDInfo{
		PID:       pid,
		Command:   "sleep 60",
		StartedAt: time.Now(),
	})

	handle, err := r.Reconnect("worker")
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Stop(handle); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	if runner.IsProcessAlive(pid) {
		t.Error("process should be dead after Stop")
	}

	if _, readErr := runner.ReadPIDFile(pidDir, "worker"); readErr == nil {
		t.Error("PID file should have been removed after Stop")
	}

	status, _ := r.Status(handle)
	if status != runner.StatusStopped {
		t.Errorf("expected stopped, got %s", status)
	}
}
