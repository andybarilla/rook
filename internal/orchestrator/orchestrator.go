package orchestrator

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/andybarilla/rook/internal/health"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/profile"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

// Orchestrator manages the lifecycle of workspace services, handling
// dependency ordering, port allocation, and incremental profile switching.
type Orchestrator struct {
	mu              sync.Mutex
	containerRunner runner.Runner
	processRunner   runner.Runner
	portAllocator   ports.PortAllocator
	handles         map[string]map[string]runner.RunHandle
}

// New creates a new Orchestrator with the given runners and port allocator.
func New(containerRunner, processRunner runner.Runner, portAllocator ports.PortAllocator) *Orchestrator {
	return &Orchestrator{
		containerRunner: containerRunner,
		processRunner:   processRunner,
		portAllocator:   portAllocator,
		handles:         make(map[string]map[string]runner.RunHandle),
	}
}

// Up starts the services for the given profile, performing incremental
// profile switching by only starting new services and stopping removed ones.
func (o *Orchestrator) Up(ctx context.Context, ws workspace.Workspace, profileName string) error {
	services, err := profile.Resolve(ws, profileName)
	if err != nil {
		return fmt.Errorf("resolving profile: %w", err)
	}

	order, err := TopoSort(ws.Services, services)
	if err != nil {
		return fmt.Errorf("sorting dependencies: %w", err)
	}

	portMap := make(runner.PortMap)
	if o.portAllocator != nil {
		for _, name := range order {
			svc := ws.Services[name]
			if len(svc.Ports) > 0 {
				if svc.PinPort > 0 {
					port, err := o.portAllocator.AllocatePinned(ws.Name, name, svc.PinPort)
					if err != nil {
						return fmt.Errorf("pinning port for %s: %w", name, err)
					}
					portMap[name] = port
				} else {
					port, err := o.portAllocator.Allocate(ws.Name, name, svc.Ports[0])
					if err != nil {
						return fmt.Errorf("allocating port for %s: %w", name, err)
					}
					portMap[name] = port
				}
			}
		}
	}

	// Incremental profile switching
	o.mu.Lock()
	if o.handles[ws.Name] == nil {
		o.handles[ws.Name] = make(map[string]runner.RunHandle)
	}
	currentHandles := make(map[string]runner.RunHandle, len(o.handles[ws.Name]))
	for k, v := range o.handles[ws.Name] {
		currentHandles[k] = v
	}
	o.mu.Unlock()

	desiredSet := make(map[string]bool, len(order))
	for _, name := range order {
		desiredSet[name] = true
	}

	// Stop services not in new profile
	for name, handle := range currentHandles {
		if !desiredSet[name] {
			var r runner.Runner
			if handle.Type == "process" {
				r = o.processRunner
			} else {
				r = o.containerRunner
			}
			r.Stop(handle)
			o.mu.Lock()
			delete(o.handles[ws.Name], name)
			o.mu.Unlock()
		}
	}

	// Start new services in dependency order
	for _, name := range order {
		o.mu.Lock()
		_, alreadyRunning := o.handles[ws.Name][name]
		o.mu.Unlock()
		if alreadyRunning {
			continue
		}

		svc := ws.Services[name]
		var r runner.Runner
		if svc.IsContainer() {
			r = o.containerRunner
		} else {
			r = o.processRunner
		}

		handle, err := r.Start(ctx, name, svc, portMap, ws.Root)
		if err != nil {
			return fmt.Errorf("starting %s: %w", name, err)
		}

		o.mu.Lock()
		o.handles[ws.Name][name] = handle
		o.mu.Unlock()

		// Brief pause to catch immediate crashes (e.g., missing env vars)
		if svc.IsContainer() {
			time.Sleep(1 * time.Second)
			status, _ := r.Status(handle)
			if status == runner.StatusCrashed || status == runner.StatusStopped {
				// Fetch last logs for the error message
				var lastLogs string
				if logReader, err := r.Logs(handle); err == nil {
					if data, err := io.ReadAll(logReader); err == nil {
						lines := strings.Split(strings.TrimSpace(string(data)), "\n")
						// Show last 20 lines
						if len(lines) > 20 {
							lines = lines[len(lines)-20:]
						}
						lastLogs = "\n  " + strings.Join(lines, "\n  ")
					}
					logReader.Close()
				}
				return fmt.Errorf("service %s crashed immediately after starting%s", name, lastLogs)
			}
		}

		// Wait for health check if defined
		if svc.Healthcheck != nil {
			check, cfg, err := health.ParseFromService(svc.Healthcheck)
			if err == nil {
				hctx, hcancel := context.WithTimeout(ctx, cfg.Timeout)
				if waitErr := health.WaitUntilHealthy(hctx, check, cfg.Interval); waitErr != nil {
					hcancel()
					return fmt.Errorf("health check failed for %s: %w", name, waitErr)
				}
				hcancel()
			}
		}
	}

	return nil
}

// Down stops all services in the given workspace.
func (o *Orchestrator) Down(ctx context.Context, ws workspace.Workspace) error {
	o.mu.Lock()
	handles, ok := o.handles[ws.Name]
	o.mu.Unlock()
	if !ok {
		return nil
	}

	var errs []error
	for name, handle := range handles {
		var r runner.Runner
		if handle.Type == "process" {
			r = o.processRunner
		} else {
			r = o.containerRunner
		}
		if err := r.Stop(handle); err != nil {
			errs = append(errs, fmt.Errorf("stopping %s: %w", name, err))
		}
	}

	o.mu.Lock()
	delete(o.handles, ws.Name)
	o.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("errors: %v", errs)
	}
	return nil
}

// StartService starts a single service by name. It builds a workspace-wide
// portMap so that env templates can resolve all service ports, not just the
// target service's port.
func (o *Orchestrator) StartService(ctx context.Context, ws workspace.Workspace, serviceName string) error {
	svc, ok := ws.Services[serviceName]
	if !ok {
		return fmt.Errorf("unknown service: %q", serviceName)
	}

	o.mu.Lock()
	if o.handles[ws.Name] == nil {
		o.handles[ws.Name] = make(map[string]runner.RunHandle)
	}
	if _, running := o.handles[ws.Name][serviceName]; running {
		o.mu.Unlock()
		return nil
	}
	o.mu.Unlock()

	portMap := make(runner.PortMap)
	if o.portAllocator != nil {
		if len(svc.Ports) > 0 {
			if svc.PinPort > 0 {
				port, err := o.portAllocator.AllocatePinned(ws.Name, serviceName, svc.PinPort)
				if err != nil {
					return fmt.Errorf("pinning port for %s: %w", serviceName, err)
				}
				portMap[serviceName] = port
			} else {
				port, err := o.portAllocator.Allocate(ws.Name, serviceName, svc.Ports[0])
				if err != nil {
					return fmt.Errorf("allocating port for %s: %w", serviceName, err)
				}
				portMap[serviceName] = port
			}
		}
		for name := range ws.Services {
			if name == serviceName {
				continue
			}
			if result := o.portAllocator.Get(ws.Name, name); result.OK {
				portMap[name] = result.Port
			}
		}
	}

	var r runner.Runner
	if svc.IsContainer() {
		r = o.containerRunner
	} else {
		r = o.processRunner
	}
	handle, err := r.Start(ctx, serviceName, svc, portMap, ws.Root)
	if err != nil {
		return fmt.Errorf("starting %s: %w", serviceName, err)
	}
	o.mu.Lock()
	o.handles[ws.Name][serviceName] = handle
	o.mu.Unlock()
	return nil
}

// StopService stops a single service by name.
func (o *Orchestrator) StopService(ctx context.Context, ws workspace.Workspace, serviceName string) error {
	o.mu.Lock()
	handles, ok := o.handles[ws.Name]
	if !ok {
		o.mu.Unlock()
		return nil
	}
	handle, running := handles[serviceName]
	o.mu.Unlock()
	if !running {
		return nil
	}
	var r runner.Runner
	if handle.Type == "process" {
		r = o.processRunner
	} else {
		r = o.containerRunner
	}
	if err := r.Stop(handle); err != nil {
		return fmt.Errorf("stopping %s: %w", serviceName, err)
	}
	o.mu.Lock()
	delete(o.handles[ws.Name], serviceName)
	o.mu.Unlock()
	return nil
}

// RestartService stops and then starts a single service by name.
func (o *Orchestrator) RestartService(ctx context.Context, ws workspace.Workspace, serviceName string) error {
	if err := o.StopService(ctx, ws, serviceName); err != nil {
		return err
	}
	return o.StartService(ctx, ws, serviceName)
}

// Status returns the status of all services in the workspace.
func (o *Orchestrator) Status(ws workspace.Workspace) (map[string]runner.ServiceStatus, error) {
	o.mu.Lock()
	handles, ok := o.handles[ws.Name]
	o.mu.Unlock()

	result := make(map[string]runner.ServiceStatus)
	if !ok {
		for name := range ws.Services {
			result[name] = runner.StatusStopped
		}
		return result, nil
	}

	for name, handle := range handles {
		var r runner.Runner
		if handle.Type == "process" {
			r = o.processRunner
		} else {
			r = o.containerRunner
		}
		status, _ := r.Status(handle)
		result[name] = status
	}

	for name := range ws.Services {
		if _, ok := result[name]; !ok {
			result[name] = runner.StatusStopped
		}
	}

	return result, nil
}

// Reconnect discovers already-running Docker containers for a workspace
// and populates the orchestrator's handle map so subsequent operations
// (Up, Down, Status) are aware of them.
func (o *Orchestrator) Reconnect(ws workspace.Workspace) error {
	rc, ok := o.containerRunner.(runner.Reconnectable)
	if !ok {
		return nil // runner doesn't support reconnection
	}

	prefix := rc.Prefix() + "_"
	containers, err := runner.FindContainers(prefix)
	if err != nil {
		return fmt.Errorf("finding containers for %s: %w", ws.Name, err)
	}

	o.mu.Lock()
	if o.handles[ws.Name] == nil {
		o.handles[ws.Name] = make(map[string]runner.RunHandle)
	}
	o.mu.Unlock()

	for _, containerName := range containers {
		if !strings.HasPrefix(containerName, prefix) {
			continue
		}
		if runner.ContainerStatus(containerName) != runner.StatusRunning {
			continue
		}
		serviceName := strings.TrimPrefix(containerName, prefix)
		if _, exists := ws.Services[serviceName]; !exists {
			continue
		}
		handle := rc.Adopt(serviceName)
		o.mu.Lock()
		o.handles[ws.Name][serviceName] = handle
		o.mu.Unlock()
	}

	return nil
}
