package api

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/andybarilla/rook/internal/buildcache"
	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/settings"
	"github.com/andybarilla/rook/internal/workspace"
)

// WorkspaceAPI is the service layer that the GUI (and CLI) use to manage workspaces.
type WorkspaceAPI struct {
	registry       registry.Registry
	portAlloc      ports.PortAllocator
	orch           *orchestrator.Orchestrator
	discoverers    []discovery.Discoverer
	logBuffer      *LogBuffer
	emitter        EventEmitter
	activeProfiles map[string]string
	settingsPath   string
	portsPath      string // path to ports.json
	logMu          sync.Mutex
	logCancels     map[string]context.CancelFunc
}

// NewWorkspaceAPI creates a new WorkspaceAPI with the given dependencies.
func NewWorkspaceAPI(reg registry.Registry, alloc ports.PortAllocator, orch *orchestrator.Orchestrator, discoverers []discovery.Discoverer) *WorkspaceAPI {
	return &WorkspaceAPI{
		registry:       reg,
		portAlloc:      alloc,
		orch:           orch,
		discoverers:    discoverers,
		logBuffer:      NewLogBuffer(10000),
		emitter:        NoopEmitter{},
		activeProfiles: make(map[string]string),
		logCancels:     make(map[string]context.CancelFunc),
	}
}

// NewWorkspaceAPIWithSettings creates a new WorkspaceAPI with a settings file path.
func NewWorkspaceAPIWithSettings(reg registry.Registry, alloc ports.PortAllocator, orch *orchestrator.Orchestrator, discoverers []discovery.Discoverer, settingsPath string) *WorkspaceAPI {
	return &WorkspaceAPI{
		registry:       reg,
		portAlloc:      alloc,
		orch:           orch,
		discoverers:    discoverers,
		logBuffer:      NewLogBuffer(10000),
		emitter:        NoopEmitter{},
		activeProfiles: make(map[string]string),
		settingsPath:   settingsPath,
		logCancels:     make(map[string]context.CancelFunc),
	}
}

// NewWorkspaceAPIFull creates a new WorkspaceAPI with both settings and ports file paths.
func NewWorkspaceAPIFull(reg registry.Registry, alloc ports.PortAllocator, orch *orchestrator.Orchestrator, discoverers []discovery.Discoverer, settingsPath, portsPath string) *WorkspaceAPI {
	return &WorkspaceAPI{
		registry:       reg,
		portAlloc:      alloc,
		orch:           orch,
		discoverers:    discoverers,
		logBuffer:      NewLogBuffer(10000),
		emitter:        NoopEmitter{},
		activeProfiles: make(map[string]string),
		settingsPath:   settingsPath,
		portsPath:      portsPath,
		logCancels:     make(map[string]context.CancelFunc),
	}
}

// SetEmitter sets the event emitter (typically set after Wails startup).
func (w *WorkspaceAPI) SetEmitter(e EventEmitter) {
	w.emitter = e
}

// BufferLog adds a log line to the buffer and emits a service:log event.
func (w *WorkspaceAPI) BufferLog(ws, service, line string) {
	w.logBuffer.Add(ws, service, line)
	w.emitter.Emit("service:log", LogEvent{
		Workspace: ws,
		Service:   service,
		Line:      line,
	})
}

func logKey(ws, svc string) string {
	return ws + "/" + svc
}

// StreamFromReader starts a goroutine that reads lines from r and buffers them.
func (w *WorkspaceAPI) StreamFromReader(ws, svc string, r io.ReadCloser) {
	ctx, cancel := context.WithCancel(context.Background())

	w.logMu.Lock()
	if existing, ok := w.logCancels[logKey(ws, svc)]; ok {
		existing()
	}
	w.logCancels[logKey(ws, svc)] = cancel
	w.logMu.Unlock()

	go func() {
		defer r.Close()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				w.BufferLog(ws, svc, scanner.Text())
			}
		}
		w.logMu.Lock()
		delete(w.logCancels, logKey(ws, svc))
		w.logMu.Unlock()
	}()
}

// startLogStream starts streaming logs for a service via the orchestrator.
func (w *WorkspaceAPI) startLogStream(ws, svc string) {
	reader, err := w.orch.StreamServiceLogs(ws, svc)
	if err != nil {
		return
	}
	w.StreamFromReader(ws, svc, reader)
}

// StopLogStream cancels an active log stream for a service.
func (w *WorkspaceAPI) StopLogStream(ws, svc string) {
	w.logMu.Lock()
	if cancel, ok := w.logCancels[logKey(ws, svc)]; ok {
		cancel()
		delete(w.logCancels, logKey(ws, svc))
	}
	w.logMu.Unlock()
}

// stopAllLogStreams cancels all active log streams for a workspace.
func (w *WorkspaceAPI) stopAllLogStreams(ws string) {
	w.logMu.Lock()
	prefix := ws + "/"
	for key, cancel := range w.logCancels {
		if strings.HasPrefix(key, prefix) {
			cancel()
			delete(w.logCancels, key)
		}
	}
	w.logMu.Unlock()
}

// ListWorkspaces returns a summary of all registered workspaces.
func (w *WorkspaceAPI) ListWorkspaces() []WorkspaceInfo {
	entries := w.registry.List()
	result := make([]WorkspaceInfo, 0, len(entries))

	for _, e := range entries {
		info := WorkspaceInfo{
			Name: e.Name,
			Path: e.Path,
		}

		ws, err := w.loadWorkspace(e.Name)
		if err == nil {
			info.ServiceCount = len(ws.Services)

			statuses, err := w.orch.Status(*ws)
			if err == nil {
				for _, s := range statuses {
					if s == runner.StatusRunning || s == runner.StatusStarting {
						info.RunningCount++
					}
				}
			}
		}

		info.ActiveProfile = w.activeProfiles[e.Name]
		result = append(result, info)
	}

	return result
}

// GetWorkspace returns full detail for a workspace including topo-sorted services.
func (w *WorkspaceAPI) GetWorkspace(name string) (*WorkspaceDetail, error) {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return nil, err
	}

	entry, err := w.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// Get service order via topo sort of all services
	allNames := ws.ServiceNames()
	order, err := orchestrator.TopoSort(ws.Services, allNames)
	if err != nil {
		// Fall back to alphabetical if topo sort fails
		order = allNames
	}

	statuses, _ := w.orch.Status(*ws)

	services := make([]ServiceInfo, 0, len(order))
	for _, svcName := range order {
		svc := ws.Services[svcName]
		si := ServiceInfo{
			Name:      svcName,
			Image:     svc.Image,
			Command:   svc.Command,
			DependsOn: svc.DependsOn,
			Status:    runner.StatusStopped,
			HasBuild:  svc.Build != "",
		}
		if s, ok := statuses[svcName]; ok {
			si.Status = s
		}
		if result := w.portAlloc.Get(name, svcName); result.OK {
			si.Port = result.Port
		}
		services = append(services, si)
	}

	return &WorkspaceDetail{
		Name:          name,
		Path:          entry.Path,
		Services:      services,
		Profiles:      ws.Profiles,
		Groups:        ws.Groups,
		ActiveProfile: w.activeProfiles[name],
	}, nil
}

// AddWorkspace discovers services at the given path, writes a manifest, registers
// the workspace, and allocates ports.
func (w *WorkspaceAPI) AddWorkspace(path string) (*DiscoverResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	dr, err := discovery.RunAll(absPath, w.discoverers)
	if err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}

	name := filepath.Base(absPath)

	manifest := &workspace.Manifest{
		Name:     name,
		Type:     workspace.TypeMulti,
		Services: dr.Services,
		Groups:   dr.Groups,
	}

	manifestPath := filepath.Join(absPath, "rook.yaml")
	if err := workspace.WriteManifest(manifestPath, manifest); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	if err := w.registry.Register(name, absPath); err != nil {
		return nil, fmt.Errorf("registering: %w", err)
	}

	// Allocate ports for each service
	for svcName, svc := range manifest.Services {
		if len(svc.Ports) > 0 {
			if _, err := w.portAlloc.Allocate(name, svcName); err != nil {
				return nil, fmt.Errorf("allocating port for %s: %w", svcName, err)
			}
		}
	}

	return &DiscoverResult{
		Source:   dr.Source,
		Services: dr.Services,
		Groups:   dr.Groups,
	}, nil
}

// RemoveWorkspace stops all services and unregisters the workspace.
func (w *WorkspaceAPI) RemoveWorkspace(name string) error {
	ws, err := w.loadWorkspace(name)
	if err == nil {
		_ = w.orch.Down(context.Background(), *ws)
	}
	w.registry.Remove(name)
	delete(w.activeProfiles, name)
	return nil
}

// SaveManifest writes a manifest to the workspace's rook.yaml file.
func (w *WorkspaceAPI) SaveManifest(name string, manifest *Manifest) error {
	entry, err := w.registry.Get(name)
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	return workspace.WriteManifest(manifestPath, manifest)
}

// StartWorkspace starts all services for the given profile.
// forceBuild forces rebuild of services with build contexts.
func (w *WorkspaceAPI) StartWorkspace(name, profile string, forceBuild bool) error {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return err
	}

	// Mark services for forced rebuild
	if forceBuild {
		for svcName, svc := range ws.Services {
			if svc.Build != "" {
				svc.ForceBuild = true
				ws.Services[svcName] = svc
			}
		}
	}

	if err := w.orch.Up(context.Background(), *ws, profile); err != nil {
		return err
	}
	w.activeProfiles[name] = profile

	statuses, _ := w.orch.Status(*ws)
	for svcName, status := range statuses {
		if status == runner.StatusRunning {
			w.startLogStream(name, svcName)
		}
	}
	return nil
}

// StopWorkspace stops all services in the workspace.
func (w *WorkspaceAPI) StopWorkspace(name string) error {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return err
	}
	w.stopAllLogStreams(name)
	if err := w.orch.Down(context.Background(), *ws); err != nil {
		return err
	}
	delete(w.activeProfiles, name)
	return nil
}

// StartService starts a single service, emitting status events.
func (w *WorkspaceAPI) StartService(ws, svc string) error {
	wks, err := w.loadWorkspace(ws)
	if err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusStarting})
	if err := w.orch.StartService(context.Background(), *wks, svc); err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusRunning})
	w.startLogStream(ws, svc)
	return nil
}

// StopService stops a single service, emitting status events.
func (w *WorkspaceAPI) StopService(ws, svc string) error {
	wks, err := w.loadWorkspace(ws)
	if err != nil {
		return err
	}
	w.StopLogStream(ws, svc)
	if err := w.orch.StopService(context.Background(), *wks, svc); err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusStopped})
	return nil
}

// RestartService restarts a single service, emitting status events.
func (w *WorkspaceAPI) RestartService(ws, svc string) error {
	wks, err := w.loadWorkspace(ws)
	if err != nil {
		return err
	}
	w.StopLogStream(ws, svc)
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusStarting})
	if err := w.orch.RestartService(context.Background(), *wks, svc); err != nil {
		return err
	}
	w.emitter.Emit("service:status", StatusEvent{Workspace: ws, Service: svc, Status: runner.StatusRunning})
	w.startLogStream(ws, svc)
	return nil
}

// GetPorts returns all port allocations.
func (w *WorkspaceAPI) GetPorts() []PortEntry {
	return w.portAlloc.All()
}

// ResetPorts stops all rook containers and clears port allocations.
func (w *WorkspaceAPI) ResetPorts() error {
	// Stop all rook containers
	for _, e := range w.registry.List() {
		prefix := fmt.Sprintf("rook_%s_", e.Name)
		containers, _ := runner.FindContainers(prefix)
		for _, c := range containers {
			runner.StopContainer(c)
		}
	}

	// Clear in-memory port allocations
	if fa, ok := w.portAlloc.(*ports.FileAllocator); ok {
		fa.Clear()
	}

	// Delete the ports file
	if w.portsPath != "" {
		if err := os.Remove(w.portsPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing ports file: %w", err)
		}
	}

	return nil
}

// GetEnv resolves environment templates for a workspace and returns them grouped by service.
func (w *WorkspaceAPI) GetEnv(name string) (map[string][]EnvVar, error) {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return nil, err
	}

	// Build portMap from allocator
	portMap := make(map[string]int)
	for svcName := range ws.Services {
		if result := w.portAlloc.Get(name, svcName); result.OK {
			portMap[svcName] = result.Port
		}
	}

	result := make(map[string][]EnvVar)
	for svcName, svc := range ws.Services {
		if len(svc.Environment) == 0 {
			continue
		}

		resolved, err := envgen.ResolveTemplates(svc.Environment, portMap)
		if err != nil {
			return nil, fmt.Errorf("resolving env for %s: %w", svcName, err)
		}

		keys := make([]string, 0, len(svc.Environment))
		for k := range svc.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		vars := make([]EnvVar, 0, len(keys))
		for _, k := range keys {
			vars = append(vars, EnvVar{
				Key:      k,
				Template: svc.Environment[k],
				Resolved: resolved[k],
			})
		}
		result[svcName] = vars
	}

	return result, nil
}

// GetLogs returns log lines from the buffer.
func (w *WorkspaceAPI) GetLogs(ws, svc string, lines int) ([]LogLine, error) {
	return w.logBuffer.Get(ws, svc, lines), nil
}

// PreviewManifest marshals a manifest to YAML for preview.
func (w *WorkspaceAPI) PreviewManifest(manifest *Manifest) (string, error) {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("marshaling manifest: %w", err)
	}
	return string(data), nil
}

// GetSettings returns current settings with defaults applied.
func (w *WorkspaceAPI) GetSettings() *Settings {
	if w.settingsPath == "" {
		return &Settings{AutoRebuild: true}
	}
	s, err := settings.Load(w.settingsPath)
	if err != nil {
		return &Settings{AutoRebuild: true}
	}
	return &Settings{AutoRebuild: s.GetAutoRebuild()}
}

// SaveSettings persists settings to the settings file.
func (w *WorkspaceAPI) SaveSettings(s *Settings) error {
	if w.settingsPath == "" {
		return fmt.Errorf("settings path not configured")
	}
	internal := &settings.Settings{}
	internal.SetAutoRebuild(s.AutoRebuild)
	return internal.Save(w.settingsPath)
}

// CheckBuilds returns build status for all services in a workspace.
func (w *WorkspaceAPI) CheckBuilds(name string) (*BuildCheckResult, error) {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(ws.Root, ".rook", ".cache", "build-cache.json")
	cache, err := buildcache.Load(cachePath)
	if err != nil {
		return nil, fmt.Errorf("loading build cache: %w", err)
	}

	docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", name))

	services := make([]BuildStatus, 0, len(ws.Services))
	hasStale := false

	for svcName, svc := range ws.Services {
		bs := BuildStatus{
			Name:     svcName,
			HasBuild: svc.Build != "",
			Status:   "no_build_context",
		}

		if svc.Build != "" {
			currentImageID, _ := docker.GetImageID(svcName)
			result, err := buildcache.DetectStale(cache, svcName, svc, ws.Root, currentImageID)
			if err != nil {
				return nil, fmt.Errorf("checking %s: %w", svcName, err)
			}

			if result.NeedsRebuild {
				bs.Status = "needs_rebuild"
				bs.Reasons = result.Reasons
				hasStale = true
			} else {
				bs.Status = "up_to_date"
			}
		}

		services = append(services, bs)
	}

	// Sort: needs_rebuild first, then up_to_date, then no_build_context
	sort.Slice(services, func(i, j int) bool {
		order := map[string]int{"needs_rebuild": 0, "up_to_date": 1, "no_build_context": 2}
		return order[services[i].Status] < order[services[j].Status]
	})

	return &BuildCheckResult{
		Services: services,
		HasStale: hasStale,
	}, nil
}

// DiscoverWorkspace re-runs discovery and returns the diff against the current manifest.
func (w *WorkspaceAPI) DiscoverWorkspace(name string) (*DiscoverDiff, error) {
	entry, err := w.registry.Get(name)
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	manifest, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	result, err := discovery.RunAll(entry.Path, w.discoverers)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	diff := &DiscoverDiff{
		Source:          result.Source,
		NewServices:     []ServiceDiff{},
		RemovedServices: []ServiceDiff{},
	}

	for svcName, svc := range result.Services {
		if _, exists := manifest.Services[svcName]; !exists {
			sd := ServiceDiff{Name: svcName}
			if svc.Image != "" {
				sd.Image = svc.Image
			}
			if svc.Build != "" {
				sd.Build = svc.Build
			}
			diff.NewServices = append(diff.NewServices, sd)
		}
	}

	for svcName := range manifest.Services {
		if _, exists := result.Services[svcName]; !exists {
			diff.RemovedServices = append(diff.RemovedServices, ServiceDiff{
				Name:   svcName,
				Reason: "No longer in discovery source",
			})
		}
	}

	sort.Slice(diff.NewServices, func(i, j int) bool { return diff.NewServices[i].Name < diff.NewServices[j].Name })
	sort.Slice(diff.RemovedServices, func(i, j int) bool { return diff.RemovedServices[i].Name < diff.RemovedServices[j].Name })

	diff.HasChanges = len(diff.NewServices) > 0 || len(diff.RemovedServices) > 0

	return diff, nil
}

// ApplyDiscovery applies selected changes to the manifest.
func (w *WorkspaceAPI) ApplyDiscovery(name string, newServices []string, removedServices []string) error {
	entry, err := w.registry.Get(name)
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	manifest, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	result, err := discovery.RunAll(entry.Path, w.discoverers)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Validate and add new services
	for _, svcName := range newServices {
		svc, exists := result.Services[svcName]
		if !exists {
			return fmt.Errorf("service %q not found in discovery result", svcName)
		}
		manifest.Services[svcName] = svc
	}

	// Validate and remove services
	for _, svcName := range removedServices {
		if _, exists := manifest.Services[svcName]; !exists {
			return fmt.Errorf("service %q not found in manifest", svcName)
		}
		delete(manifest.Services, svcName)
	}

	// Allocate ports for new services
	for _, svcName := range newServices {
		svc := result.Services[svcName]
		if len(svc.Ports) > 0 {
			if _, err := w.portAlloc.Allocate(name, svcName); err != nil {
				return fmt.Errorf("allocating port for %s: %w", svcName, err)
			}
		}
	}

	if err := workspace.WriteManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	w.emitter.Emit("workspace:changed", WorkspaceChangedEvent{Workspace: name})

	return nil
}

// ReconnectWorkspace discovers already-running services and starts log streaming for them.
func (w *WorkspaceAPI) ReconnectWorkspace(name string) error {
	ws, err := w.loadWorkspace(name)
	if err != nil {
		return err
	}
	if err := w.orch.Reconnect(*ws); err != nil {
		return err
	}
	statuses, _ := w.orch.Status(*ws)
	for svcName, status := range statuses {
		if status == runner.StatusRunning {
			w.startLogStream(name, svcName)
		}
	}
	return nil
}

// loadWorkspace reads the manifest from the registry path and converts to a Workspace.
func (w *WorkspaceAPI) loadWorkspace(name string) (*workspace.Workspace, error) {
	entry, err := w.registry.Get(name)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	m, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("loading manifest for %q: %w", name, err)
	}
	return m.ToWorkspace(entry.Path)
}
