package ports

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

// PortEntry represents a single port allocation for a workspace service.
type PortEntry struct {
	Workspace string `json:"workspace"`
	Service   string `json:"service"`
	Port      int    `json:"port"`
	Pinned    bool   `json:"pinned,omitempty"`
}

// LookupResult is the result of a Get operation.
type LookupResult struct {
	Port int
	OK   bool
}

// PortAllocator defines the interface for port allocation.
type PortAllocator interface {
	Allocate(workspace, service string, preferred int) (int, error)
	AllocatePinned(workspace, service string, port int) (int, error)
	Release(workspace, service string) error
	Get(workspace, service string) LookupResult
	All() []PortEntry
}

// FileAllocator is a file-backed port allocator that persists allocations to a JSON file.
type FileAllocator struct {
	mu      sync.Mutex
	path    string
	minPort int
	maxPort int
	entries []PortEntry
	used    map[int]bool
}

// NewFileAllocator creates a new FileAllocator backed by the given file path.
// It loads existing allocations from the file if it exists.
func NewFileAllocator(path string, minPort, maxPort int) (*FileAllocator, error) {
	a := &FileAllocator{
		path:    path,
		minPort: minPort,
		maxPort: maxPort,
		used:    make(map[int]bool),
	}
	if err := a.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return a, nil
}

func (a *FileAllocator) load() error {
	data, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &a.entries); err != nil {
		return err
	}
	for _, e := range a.entries {
		a.used[e.Port] = true
	}
	return nil
}

func (a *FileAllocator) save() error {
	data, err := json.MarshalIndent(a.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.path, data, 0644)
}

func (a *FileAllocator) findIndex(workspace, service string) int {
	for i, e := range a.entries {
		if e.Workspace == workspace && e.Service == service {
			return i
		}
	}
	return -1
}

// portAvailable checks if a port is free on the system by attempting to listen on it.
func portAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Allocate assigns a port for the given workspace and service.
// If preferred is non-zero, available in the allocator, and free on the system, it will be used.
// If the preferred port is taken on the system, a port from the allocator range is assigned instead.
// If the workspace/service already has an allocation, the existing port is returned.
func (a *FileAllocator) Allocate(workspace, service string, preferred int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if idx := a.findIndex(workspace, service); idx >= 0 {
		return a.entries[idx].Port, nil
	}

	if preferred > 0 && !a.used[preferred] && portAvailable(preferred) {
		return a.assign(workspace, service, preferred, false)
	}

	for p := a.minPort; p <= a.maxPort; p++ {
		if !a.used[p] && portAvailable(p) {
			return a.assign(workspace, service, p, false)
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", a.minPort, a.maxPort)
}

// AllocatePinned assigns a specific port for the given workspace and service.
// It returns an error if the port is already in use by another allocation.
func (a *FileAllocator) AllocatePinned(workspace, service string, port int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if idx := a.findIndex(workspace, service); idx >= 0 {
		return a.entries[idx].Port, nil
	}

	if a.used[port] {
		for _, e := range a.entries {
			if e.Port == port {
				return 0, fmt.Errorf("port %d already pinned by %s.%s", port, e.Workspace, e.Service)
			}
		}
	}

	return a.assign(workspace, service, port, true)
}

func (a *FileAllocator) assign(workspace, service string, port int, pinned bool) (int, error) {
	a.entries = append(a.entries, PortEntry{
		Workspace: workspace,
		Service:   service,
		Port:      port,
		Pinned:    pinned,
	})
	a.used[port] = true
	if err := a.save(); err != nil {
		return 0, err
	}
	return port, nil
}

// Release removes the port allocation for the given workspace and service.
func (a *FileAllocator) Release(workspace, service string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	idx := a.findIndex(workspace, service)
	if idx < 0 {
		return nil
	}

	port := a.entries[idx].Port
	a.entries = append(a.entries[:idx], a.entries[idx+1:]...)
	delete(a.used, port)
	return a.save()
}

// Get looks up the port for a given workspace and service without allocating.
func (a *FileAllocator) Get(workspace, service string) LookupResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	idx := a.findIndex(workspace, service)
	if idx < 0 {
		return LookupResult{}
	}
	return LookupResult{Port: a.entries[idx].Port, OK: true}
}

// All returns a copy of all current port allocations.
func (a *FileAllocator) All() []PortEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]PortEntry, len(a.entries))
	copy(out, a.entries)
	return out
}

// Clear removes all port allocations from memory.
func (a *FileAllocator) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = nil
	a.used = make(map[int]bool)
}
