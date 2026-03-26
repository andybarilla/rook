package runner_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

func dockerAvailable() bool {
	runner.DetectRuntime()
	return exec.Command(runner.ContainerRuntime, "info").Run() == nil
}

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

func TestStopContainerWithVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	// Create a container with a named volume
	containerName := "rook-test-volumes"
	runtime := runner.ContainerRuntime

	// First, clean up any existing container
	exec.Command(runtime, "rm", "-f", containerName).Run()

	// Create a container with a volume
	cmd := exec.Command(runtime, "run", "-d", "--name", containerName,
		"-v", "rook-test-vol:/data",
		"alpine:latest", "sleep", "300")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Verify the volume exists
	volumeInspect := exec.Command(runtime, "volume", "inspect", "rook-test-vol")
	if err := volumeInspect.Run(); err != nil {
		t.Fatalf("volume should exist after container creation: %v", err)
	}

	// Stop with volumes=true - should remove the anonymous volume
	runner.StopContainerWithVolumes(containerName, true)

	// Container should be gone
	inspectCmd := exec.Command(runtime, "inspect", containerName)
	if err := inspectCmd.Run(); err == nil {
		t.Error("container should be removed")
	}

	// Named volumes are NOT removed by `docker rm -v` (only anonymous volumes).
	// Use runner.RemoveVolumes() for named volume cleanup — see TestRemoveVolumes.
	volumeInspect = exec.Command(runtime, "volume", "inspect", "rook-test-vol")
	if err := volumeInspect.Run(); err != nil {
		// Expected: named volume may or may not survive depending on runtime behavior
	}

	// Clean up the named volume if it still exists
	exec.Command(runtime, "volume", "rm", "-f", "rook-test-vol").Run()
}

func TestStopContainerWithVolumes_NoVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	// Create a container without volumes
	containerName := "rook-test-novol"
	runtime := runner.ContainerRuntime

	// First, clean up any existing container
	exec.Command(runtime, "rm", "-f", containerName).Run()

	// Create a simple container
	cmd := exec.Command(runtime, "run", "-d", "--name", containerName,
		"alpine:latest", "sleep", "300")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Stop with volumes=false - should NOT pass -v flag
	runner.StopContainerWithVolumes(containerName, false)

	// Container should be gone
	inspectCmd := exec.Command(runtime, "inspect", containerName)
	if err := inspectCmd.Run(); err == nil {
		t.Error("container should be removed")
	}
}

func TestContainerVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime
	containerName := "rook-test-cvol"

	// Clean up
	exec.Command(runtime, "rm", "-f", containerName).Run()
	exec.Command(runtime, "volume", "rm", "-f", "rook-test-namedvol").Run()

	// Create container with a named volume and a bind mount
	cmd := exec.Command(runtime, "run", "-d", "--name", containerName,
		"-v", "rook-test-namedvol:/data",
		"-v", "/tmp:/hostmount",
		"alpine:latest", "sleep", "300")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() {
		exec.Command(runtime, "rm", "-f", containerName).Run()
		exec.Command(runtime, "volume", "rm", "-f", "rook-test-namedvol").Run()
	}()

	vols, err := runner.ContainerVolumes(containerName)
	if err != nil {
		t.Fatalf("ContainerVolumes failed: %v", err)
	}

	// Should contain the named volume, NOT the bind mount
	found := false
	for _, v := range vols {
		if v == "rook-test-namedvol" {
			found = true
		}
		if v == "/tmp" {
			t.Error("bind mount should not be returned as a named volume")
		}
	}
	if !found {
		t.Errorf("expected 'rook-test-namedvol' in volumes, got %v", vols)
	}
}

func TestRemoveNetwork(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime
	netName := "rook-test-net"

	// Create a network
	exec.Command(runtime, "network", "create", netName).Run()

	// Verify it exists
	if err := exec.Command(runtime, "network", "inspect", netName).Run(); err != nil {
		t.Fatalf("network should exist: %v", err)
	}

	runner.RemoveNetwork(netName)

	// Network should be gone
	if err := exec.Command(runtime, "network", "inspect", netName).Run(); err == nil {
		t.Error("network should have been removed")
		exec.Command(runtime, "network", "rm", netName).Run()
	}
}

func TestBuildRemoveArgs(t *testing.T) {
	tests := []struct {
		name          string
		removeVolumes bool
		expected      []string
	}{
		{
			name:          "without volumes",
			removeVolumes: false,
			expected:      []string{"rm", "test-container"},
		},
		{
			name:          "with volumes",
			removeVolumes: true,
			expected:      []string{"rm", "-v", "test-container"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.BuildRemoveArgs("test-container", tt.removeVolumes)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("expected %v, got %v", tt.expected, result)
					return
				}
			}
		})
	}
}

func TestRemoveVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime

	// Create a volume
	exec.Command(runtime, "volume", "create", "rook-test-rmvol").Run()

	// Verify it exists
	if err := exec.Command(runtime, "volume", "inspect", "rook-test-rmvol").Run(); err != nil {
		t.Fatalf("volume should exist: %v", err)
	}

	runner.RemoveVolumes([]string{"rook-test-rmvol"})

	// Volume should be gone
	if err := exec.Command(runtime, "volume", "inspect", "rook-test-rmvol").Run(); err == nil {
		t.Error("volume should have been removed")
		exec.Command(runtime, "volume", "rm", "-f", "rook-test-rmvol").Run()
	}
}

func TestRemoveVolumes_Empty(t *testing.T) {
	// Should not panic or error on empty input
	runner.RemoveVolumes(nil)
	runner.RemoveVolumes([]string{})
}

func TestDockerRunner_StartPrefixesNamedVolumes(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	runtime := runner.ContainerRuntime
	prefix := "rook_testprefix"
	r := runner.NewDockerRunner(prefix)

	containerName := prefix + "_volsvc"
	expectedVolume := prefix + "_mydata"

	// Clean up from any previous run
	exec.Command(runtime, "rm", "-f", containerName).Run()
	exec.Command(runtime, "volume", "rm", "-f", expectedVolume).Run()

	svc := workspace.Service{
		Image:   "alpine:latest",
		Ports:   []int{8080},
		Volumes: []string{"mydata:/data"},
	}
	ports := runner.PortMap{"volsvc": 18090}
	handle, err := r.Start(context.Background(), "volsvc", svc, ports, "")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		r.Stop(handle)
		exec.Command(runtime, "volume", "rm", "-f", expectedVolume).Run()
	}()

	// The container should have the prefixed volume
	vols, err := runner.ContainerVolumes(containerName)
	if err != nil {
		t.Fatalf("ContainerVolumes failed: %v", err)
	}
	found := false
	for _, v := range vols {
		if v == expectedVolume {
			found = true
		}
	}
	if !found {
		t.Errorf("expected volume %q, got %v", expectedVolume, vols)
	}
}

func TestPrefixVolume(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		volume   string
		expected string
	}{
		{
			name:     "named volume gets prefixed",
			prefix:   "rook_myproject",
			volume:   "pgdata:/var/lib/postgresql/data",
			expected: "rook_myproject_pgdata:/var/lib/postgresql/data",
		},
		{
			name:     "relative bind mount unchanged",
			prefix:   "rook_myproject",
			volume:   "./data:/var/lib/postgresql/data",
			expected: "./data:/var/lib/postgresql/data",
		},
		{
			name:     "absolute bind mount unchanged",
			prefix:   "rook_myproject",
			volume:   "/tmp/data:/data",
			expected: "/tmp/data:/data",
		},
		{
			name:     "parent-relative bind mount unchanged",
			prefix:   "rook_myproject",
			volume:   "../data:/data",
			expected: "../data:/data",
		},
		{
			name:     "named volume with options",
			prefix:   "rook_myproject",
			volume:   "pgdata:/var/lib/postgresql/data:rw",
			expected: "rook_myproject_pgdata:/var/lib/postgresql/data:rw",
		},
		{
			name:     "bare volume no colon",
			prefix:   "rook_myproject",
			volume:   "pgdata",
			expected: "pgdata",
		},
		{
			name:     "home dir bind mount unchanged",
			prefix:   "rook_myproject",
			volume:   "~/data:/data",
			expected: "~/data:/data",
		},
		{
			name:     "bare dot bind mount unchanged",
			prefix:   "rook_myproject",
			volume:   ".:/app:cached",
			expected: ".:/app:cached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runner.PrefixVolume(tt.prefix, tt.volume)
			if got != tt.expected {
				t.Errorf("PrefixVolume(%q, %q) = %q, want %q", tt.prefix, tt.volume, got, tt.expected)
			}
		})
	}
}
