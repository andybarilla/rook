package runner_test

import (
	"context"
	"io"
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
