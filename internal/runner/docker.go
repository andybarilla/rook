package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/andybarilla/rook/internal/workspace"
)

type DockerRunner struct {
	mu         sync.Mutex
	prefix     string
	containers map[string]string
}

func NewDockerRunner(prefix string) *DockerRunner {
	return &DockerRunner{prefix: prefix, containers: make(map[string]string)}
}

func (r *DockerRunner) containerName(name string) string {
	return fmt.Sprintf("%s_%s", r.prefix, name)
}

func (r *DockerRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	containerName := r.containerName(name)

	// Remove any existing container with the same name
	exec.CommandContext(ctx, "docker", "rm", "-f", containerName).Run()

	args := []string{"run", "-d", "--name", containerName}

	if port, ok := ports[name]; ok && len(svc.Ports) > 0 {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, svc.Ports[0]))
	}

	for k, v := range svc.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, vol := range svc.Volumes {
		args = append(args, "-v", vol)
	}

	args = append(args, svc.Image)

	if svc.Command != "" {
		args = append(args, "sh", "-c", svc.Command)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if _, err := cmd.Output(); err != nil {
		return RunHandle{}, fmt.Errorf("docker run %s: %s: %w", containerName, stderr.String(), err)
	}

	r.containers[name] = containerName
	return RunHandle{ID: name, Type: "docker"}, nil
}

func (r *DockerRunner) Stop(handle RunHandle) error {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	exec.Command("docker", "stop", containerName).Run()
	exec.Command("docker", "rm", containerName).Run()
	return nil
}

func (r *DockerRunner) Status(handle RunHandle) (ServiceStatus, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()
	if !ok {
		return StatusStopped, nil
	}

	output, err := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err != nil {
		return StatusStopped, nil
	}

	switch strings.TrimSpace(string(output)) {
	case "running":
		return StatusRunning, nil
	case "exited":
		out, _ := exec.Command("docker", "inspect", "-f", "{{.State.ExitCode}}", containerName).Output()
		if strings.TrimSpace(string(out)) != "0" {
			return StatusCrashed, nil
		}
		return StatusStopped, nil
	default:
		return StatusStopped, nil
	}
}

func (r *DockerRunner) Logs(handle RunHandle) (io.ReadCloser, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("no container for %s", handle.ID)
	}

	output, err := exec.Command("docker", "logs", containerName).Output()
	if err != nil {
		return nil, fmt.Errorf("logs for %s: %w", containerName, err)
	}
	return io.NopCloser(bytes.NewReader(output)), nil
}
