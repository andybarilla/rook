package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/andybarilla/rook/internal/workspace"
)

// ContainerRuntime is the detected container runtime binary ("podman" or "docker").
// Set once at startup via DetectRuntime().
var ContainerRuntime = "docker"

// DetectRuntime checks for podman first, then docker, and sets ContainerRuntime.
func DetectRuntime() string {
	if _, err := exec.LookPath("podman"); err == nil {
		ContainerRuntime = "podman"
		return ContainerRuntime
	}
	if _, err := exec.LookPath("docker"); err == nil {
		ContainerRuntime = "docker"
		return ContainerRuntime
	}
	return ContainerRuntime
}

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

func (r *DockerRunner) Prefix() string { return r.prefix }

func (r *DockerRunner) Adopt(serviceName string) RunHandle {
	containerName := r.containerName(serviceName)
	r.mu.Lock()
	r.containers[serviceName] = containerName
	r.mu.Unlock()
	return RunHandle{ID: serviceName, Type: "docker"}
}

func (r *DockerRunner) Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	containerName := r.containerName(name)

	// Check if container already exists
	output, err := exec.Command(ContainerRuntime, "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err == nil {
		state := strings.TrimSpace(string(output))
		if state == "running" {
			r.containers[name] = containerName
			return RunHandle{ID: name, Type: "docker"}, nil
		}
		exec.Command(ContainerRuntime, "rm", "-f", containerName).Run()
	}

	// Determine the image to use
	imageTag := svc.Image
	if svc.Build != "" {
		wsName := strings.TrimPrefix(r.prefix, "rook_")
		imageTag = fmt.Sprintf("rook-%s-%s:latest", wsName, name)

		needsBuild := svc.ForceBuild
		if !needsBuild {
			if err := exec.Command(ContainerRuntime, "image", "inspect", imageTag).Run(); err != nil {
				needsBuild = true
			}
		}

		if needsBuild {
			if workDir == "" {
				return RunHandle{}, fmt.Errorf("cannot build %s: workspace root is empty", name)
			}
			buildCtx := filepath.Join(workDir, svc.Build)
			fmt.Fprintf(os.Stderr, "Building %s from %s...\n", name, buildCtx)
			buildCmd := exec.CommandContext(ctx, ContainerRuntime, "build", "-t", imageTag, buildCtx)
			buildCmd.Stdout = os.Stderr
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				return RunHandle{}, fmt.Errorf("building %s: %w", name, err)
			}
		}
	}

	// Ensure workspace network exists
	networkName := r.prefix // e.g., "rook_kern-app"
	exec.Command(ContainerRuntime, "network", "create", networkName).Run() // ignore error if exists

	// Create new container
	args := []string{"run", "-d", "--name", containerName, "--network", networkName}

	if port, ok := ports[name]; ok && len(svc.Ports) > 0 {
		args = append(args, "-p", fmt.Sprintf("%d:%d", port, svc.Ports[0]))
	}

	if svc.EnvFile != "" {
		envFilePath := filepath.Join(workDir, svc.EnvFile)
		args = append(args, "--env-file", envFilePath)
	}

	for k, v := range svc.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, vol := range svc.Volumes {
		args = append(args, "-v", vol)
	}

	args = append(args, imageTag)

	if svc.Command != "" {
		args = append(args, "sh", "-c", svc.Command)
	}

	cmd := exec.CommandContext(ctx, ContainerRuntime, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if _, err := cmd.Output(); err != nil {
		return RunHandle{}, fmt.Errorf("%s run %s: %s: %w", ContainerRuntime, containerName, stderr.String(), err)
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
	exec.Command(ContainerRuntime, "stop", containerName).Run()
	exec.Command(ContainerRuntime, "rm", containerName).Run()
	return nil
}

func (r *DockerRunner) Status(handle RunHandle) (ServiceStatus, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()
	if !ok {
		return StatusStopped, nil
	}

	output, err := exec.Command(ContainerRuntime, "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err != nil {
		return StatusStopped, nil
	}

	switch strings.TrimSpace(string(output)) {
	case "running":
		return StatusRunning, nil
	case "exited":
		out, _ := exec.Command(ContainerRuntime, "inspect", "-f", "{{.State.ExitCode}}", containerName).Output()
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

	output, err := exec.Command(ContainerRuntime, "logs", containerName).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("logs for %s: %w", containerName, err)
	}
	return io.NopCloser(bytes.NewReader(output)), nil
}

// FindContainers returns container names matching the given prefix.
func FindContainers(prefix string) ([]string, error) {
	cmd := exec.Command(ContainerRuntime, "ps", "-a",
		"--filter", fmt.Sprintf("name=%s", prefix),
		"--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// ContainerStatus checks the status of a container by name.
func ContainerStatus(containerName string) ServiceStatus {
	output, err := exec.Command(ContainerRuntime, "inspect", "-f", "{{.State.Status}}", containerName).Output()
	if err != nil {
		return StatusStopped
	}
	switch strings.TrimSpace(string(output)) {
	case "running":
		return StatusRunning
	case "exited":
		out, _ := exec.Command(ContainerRuntime, "inspect", "-f", "{{.State.ExitCode}}", containerName).Output()
		if strings.TrimSpace(string(out)) != "0" {
			return StatusCrashed
		}
		return StatusStopped
	default:
		return StatusStopped
	}
}

// StopContainer stops and removes a container by name.
func StopContainer(name string) {
	exec.Command(ContainerRuntime, "stop", name).Run()
	exec.Command(ContainerRuntime, "rm", name).Run()
}

// StreamLogs returns a streaming reader for a container's logs.
func (r *DockerRunner) StreamLogs(handle RunHandle) (io.ReadCloser, *exec.Cmd, error) {
	r.mu.Lock()
	containerName, ok := r.containers[handle.ID]
	r.mu.Unlock()
	if !ok {
		containerName = r.containerName(handle.ID)
	}
	cmd := exec.Command(ContainerRuntime, "logs", "-f", "--follow", containerName)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("streaming logs for %s: %w", containerName, err)
	}
	return stdout, cmd, nil
}
