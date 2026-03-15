package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Entry represents a registered workspace.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Registry defines the interface for workspace registration.
type Registry interface {
	Register(name, path string) error
	Remove(name string)
	Get(name string) (Entry, error)
	List() []Entry
}

// FileRegistry is a file-backed implementation of Registry.
type FileRegistry struct {
	mu      sync.Mutex
	path    string
	entries []Entry
}

// NewFileRegistry creates a new FileRegistry backed by the given JSON file.
// If the file exists, its contents are loaded. If it does not exist, an empty
// registry is created.
func NewFileRegistry(path string) (*FileRegistry, error) {
	r := &FileRegistry{path: path}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return r, nil
}

func (r *FileRegistry) load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &r.entries)
}

func (r *FileRegistry) save() error {
	data, err := json.MarshalIndent(r.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0644)
}

// Register adds a new workspace entry. Returns an error if a workspace with
// the same name is already registered.
func (r *FileRegistry) Register(name, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.Name == name {
			return fmt.Errorf("workspace %q already registered", name)
		}
	}
	r.entries = append(r.entries, Entry{Name: name, Path: path})
	return r.save()
}

// Remove deletes a workspace entry by name. No-op if not found.
func (r *FileRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.entries {
		if e.Name == name {
			r.entries = append(r.entries[:i], r.entries[i+1:]...)
			r.save()
			return
		}
	}
}

// Get retrieves a workspace entry by name. Returns an error if not found.
func (r *FileRegistry) Get(name string) (Entry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.entries {
		if e.Name == name {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("workspace %q not found", name)
}

// List returns a copy of all registered workspace entries.
func (r *FileRegistry) List() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}
