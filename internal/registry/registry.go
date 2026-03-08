package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type Registry struct {
	path      string
	sites     []Site
	listeners []func(ChangeEvent)
}

func New(path string) *Registry {
	return &Registry{path: path}
}

func (r *Registry) withFileLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	lockPath := r.path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	return fn()
}

func (r *Registry) loadFromDisk() error {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	return json.Unmarshal(data, &r.sites)
}

func (r *Registry) Load() error {
	return r.withFileLock(func() error {
		return r.loadFromDisk()
	})
}

func (r *Registry) List() []Site {
	out := make([]Site, len(r.sites))
	copy(out, r.sites)
	return out
}

func (r *Registry) Get(domain string) (Site, bool) {
	for _, s := range r.sites {
		if s.Domain == domain {
			return s, true
		}
	}
	return Site{}, false
}

func (r *Registry) Add(s Site) error {
	return r.withFileLock(func() error {
		if err := r.loadFromDisk(); err != nil {
			return err
		}

		info, err := os.Stat(s.Path)
		if err != nil {
			return fmt.Errorf("path %q: %w", s.Path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %q is not a directory", s.Path)
		}
		for _, existing := range r.sites {
			if existing.Domain == s.Domain {
				return fmt.Errorf("domain %q is already registered", s.Domain)
			}
		}
		r.sites = append(r.sites, s)
		if err := r.saveToDisk(); err != nil {
			r.sites = r.sites[:len(r.sites)-1]
			return err
		}
		r.notify(ChangeEvent{Type: SiteAdded, Site: s})
		return nil
	})
}

func (r *Registry) Update(domain string, fn func(*Site)) error {
	return r.withFileLock(func() error {
		if err := r.loadFromDisk(); err != nil {
			return err
		}

		for i, s := range r.sites {
			if s.Domain == domain {
				old := s
				fn(&r.sites[i])
				if err := r.saveToDisk(); err != nil {
					r.sites[i] = old
					return err
				}
				r.notify(ChangeEvent{Type: SiteUpdated, Site: r.sites[i], OldSite: &old})
				return nil
			}
		}
		return fmt.Errorf("domain %q not found", domain)
	})
}

func (r *Registry) Remove(domain string) error {
	return r.withFileLock(func() error {
		if err := r.loadFromDisk(); err != nil {
			return err
		}

		for i, s := range r.sites {
			if s.Domain == domain {
				r.sites = append(r.sites[:i], r.sites[i+1:]...)
				if err := r.saveToDisk(); err != nil {
					return err
				}
				r.notify(ChangeEvent{Type: SiteRemoved, Site: s})
				return nil
			}
		}
		return fmt.Errorf("domain %q not found", domain)
	})
}

func (r *Registry) OnChange(fn func(ChangeEvent)) {
	r.listeners = append(r.listeners, fn)
}

func (r *Registry) notify(e ChangeEvent) {
	for _, fn := range r.listeners {
		fn(e)
	}
}

func (r *Registry) saveToDisk() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(r.sites, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	return os.WriteFile(r.path, data, 0o644)
}

func InferDomain(path string) string {
	name := filepath.Base(strings.TrimRight(path, "/\\"))
	return name + ".test"
}
