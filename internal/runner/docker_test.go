package runner_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

func dockerAvailable() bool { return exec.Command("docker", "info").Run() == nil }

func TestDockerRunner_StartAndStop(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	r := runner.NewDockerRunner("rook-test")
	svc := workspace.Service{Image: "alpine:latest", Ports: []int{8080}}
	ports := runner.PortMap{"test-container": 18080}
	handle, err := r.Start(context.Background(), "test-container", svc, ports, "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	r.Stop(handle)
}
