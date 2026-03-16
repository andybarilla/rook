package runner

import (
	"context"
	"io"

	"github.com/andybarilla/rook/internal/workspace"
)

type ServiceStatus string

const (
	StatusStarting ServiceStatus = "starting"
	StatusRunning  ServiceStatus = "running"
	StatusStopped  ServiceStatus = "stopped"
	StatusCrashed  ServiceStatus = "crashed"
)

type RunHandle struct {
	ID   string
	Type string // "process" or "docker"
}

type PortMap map[string]int

type Runner interface {
	Start(ctx context.Context, name string, svc workspace.Service, ports PortMap, workDir string) (RunHandle, error)
	Stop(handle RunHandle) error
	Status(handle RunHandle) (ServiceStatus, error)
	Logs(handle RunHandle) (io.ReadCloser, error)
}
