# Site Registry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the site registry — an in-memory + JSON-persisted store of local dev sites with CRUD, domain lookup, path validation, and change notifications.

**Architecture:** Single `internal/registry` package. `Site` struct + `ChangeEvent` types in `site.go`, `Registry` struct with all methods in `registry.go`. JSON file persistence via `os.WriteFile`. Synchronous callback-based change notifications.

**Tech Stack:** Go 1.23, standard library only (encoding/json, os, path/filepath)

---

## Task 1: Site Registry

**Files:**
- Create: `internal/registry/site.go`
- Create: `internal/registry/registry.go`
- Create: `internal/registry/registry_test.go`

**Step 1: Write failing tests**

Create `internal/registry/registry_test.go`:

```go
package registry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/registry"
)

func tempFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "sites.json")
}

func realDir(t *testing.T) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), "myapp")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAddAndList(t *testing.T) {
	dir := realDir(t)
	r := registry.New(tempFile(t))

	site := registry.Site{
		Path:   dir,
		Domain: "myapp.test",
		TLS:    true,
	}

	if err := r.Add(site); err != nil {
		t.Fatalf("Add: %v", err)
	}

	sites := r.List()
	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}
	if sites[0].Domain != "myapp.test" {
		t.Errorf("domain = %q, want %q", sites[0].Domain, "myapp.test")
	}
}

func TestAddDuplicateDomainErrors(t *testing.T) {
	dir := realDir(t)
	r := registry.New(tempFile(t))
	site := registry.Site{Path: dir, Domain: "myapp.test", TLS: true}

	if err := r.Add(site); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if err := r.Add(site); err == nil {
		t.Fatal("expected error adding duplicate domain, got nil")
	}
}

func TestAddNonexistentPathErrors(t *testing.T) {
	r := registry.New(tempFile(t))
	site := registry.Site{Path: "/no/such/directory", Domain: "nope.test", TLS: true}

	if err := r.Add(site); err == nil {
		t.Fatal("expected error adding nonexistent path, got nil")
	}
}

func TestGetFound(t *testing.T) {
	dir := realDir(t)
	r := registry.New(tempFile(t))
	_ = r.Add(registry.Site{Path: dir, Domain: "myapp.test", TLS: true})

	site, ok := r.Get("myapp.test")
	if !ok {
		t.Fatal("expected to find site")
	}
	if site.Domain != "myapp.test" {
		t.Errorf("domain = %q, want %q", site.Domain, "myapp.test")
	}
}

func TestGetNotFound(t *testing.T) {
	r := registry.New(tempFile(t))
	_, ok := r.Get("nope.test")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestUpdate(t *testing.T) {
	dir := realDir(t)
	r := registry.New(tempFile(t))
	_ = r.Add(registry.Site{Path: dir, Domain: "myapp.test", TLS: false})

	err := r.Update("myapp.test", func(s *registry.Site) {
		s.TLS = true
		s.PHPVersion = "8.3"
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	site, _ := r.Get("myapp.test")
	if !site.TLS {
		t.Error("expected TLS to be true after update")
	}
	if site.PHPVersion != "8.3" {
		t.Errorf("PHPVersion = %q, want %q", site.PHPVersion, "8.3")
	}
}

func TestUpdateNotFoundErrors(t *testing.T) {
	r := registry.New(tempFile(t))
	err := r.Update("nope.test", func(s *registry.Site) {})
	if err == nil {
		t.Fatal("expected error updating nonexistent domain")
	}
}

func TestRemove(t *testing.T) {
	dir := realDir(t)
	r := registry.New(tempFile(t))
	_ = r.Add(registry.Site{Path: dir, Domain: "myapp.test", TLS: true})

	if err := r.Remove("myapp.test"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(r.List()) != 0 {
		t.Fatal("expected 0 sites after remove")
	}
}

func TestRemoveNotFoundErrors(t *testing.T) {
	r := registry.New(tempFile(t))
	if err := r.Remove("nope.test"); err == nil {
		t.Fatal("expected error removing nonexistent domain")
	}
}

func TestPersistence(t *testing.T) {
	dir := realDir(t)
	path := tempFile(t)

	r1 := registry.New(path)
	_ = r1.Add(registry.Site{Path: dir, Domain: "myapp.test", TLS: true})

	r2 := registry.New(path)
	if err := r2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	sites := r2.List()
	if len(sites) != 1 {
		t.Fatalf("expected 1 site after reload, got %d", len(sites))
	}
	if sites[0].Domain != "myapp.test" {
		t.Errorf("domain = %q, want %q", sites[0].Domain, "myapp.test")
	}
}

func TestOnChange(t *testing.T) {
	dir := realDir(t)
	r := registry.New(tempFile(t))

	var events []registry.ChangeEvent
	r.OnChange(func(e registry.ChangeEvent) {
		events = append(events, e)
	})

	site := registry.Site{Path: dir, Domain: "myapp.test", TLS: true}
	_ = r.Add(site)
	_ = r.Update("myapp.test", func(s *registry.Site) { s.TLS = false })
	_ = r.Remove("myapp.test")

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Type != registry.SiteAdded {
		t.Errorf("event[0].Type = %v, want SiteAdded", events[0].Type)
	}
	if events[1].Type != registry.SiteUpdated {
		t.Errorf("event[1].Type = %v, want SiteUpdated", events[1].Type)
	}
	if events[1].OldSite == nil {
		t.Fatal("event[1].OldSite should not be nil for update")
	}
	if !events[1].OldSite.TLS {
		t.Error("OldSite.TLS should be true (before update)")
	}
	if events[1].Site.TLS {
		t.Error("Site.TLS should be false (after update)")
	}
	if events[2].Type != registry.SiteRemoved {
		t.Errorf("event[2].Type = %v, want SiteRemoved", events[2].Type)
	}
}

func TestInferDomain(t *testing.T) {
	cases := []struct {
		path   string
		domain string
	}{
		{"/home/user/myapp", "myapp.test"},
		{"/home/user/my-cool-app", "my-cool-app.test"},
		{"/home/user/myapp/", "myapp.test"},
	}
	for _, c := range cases {
		got := registry.InferDomain(c.path)
		if got != c.domain {
			t.Errorf("InferDomain(%q) = %q, want %q", c.path, got, c.domain)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/registry/... -v`
Expected: FAIL — package does not exist.

**Step 3: Implement Site type and event types**

Create `internal/registry/site.go`:

```go
package registry

type Site struct {
	Path       string `json:"path"`
	Domain     string `json:"domain"`
	PHPVersion string `json:"php_version,omitempty"`
	TLS        bool   `json:"tls"`
}

type EventType int

const (
	SiteAdded EventType = iota
	SiteRemoved
	SiteUpdated
)

type ChangeEvent struct {
	Type    EventType
	Site    Site
	OldSite *Site
}
```

**Step 4: Implement Registry**

Create `internal/registry/registry.go`:

```go
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Registry struct {
	path      string
	sites     []Site
	listeners []func(ChangeEvent)
}

func New(path string) *Registry {
	return &Registry{path: path}
}

func (r *Registry) Load() error {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	return json.Unmarshal(data, &r.sites)
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
	if err := r.save(); err != nil {
		r.sites = r.sites[:len(r.sites)-1]
		return err
	}
	r.notify(ChangeEvent{Type: SiteAdded, Site: s})
	return nil
}

func (r *Registry) Update(domain string, fn func(*Site)) error {
	for i, s := range r.sites {
		if s.Domain == domain {
			old := s
			fn(&r.sites[i])
			if err := r.save(); err != nil {
				r.sites[i] = old
				return err
			}
			r.notify(ChangeEvent{Type: SiteUpdated, Site: r.sites[i], OldSite: &old})
			return nil
		}
	}
	return fmt.Errorf("domain %q not found", domain)
}

func (r *Registry) Remove(domain string) error {
	for i, s := range r.sites {
		if s.Domain == domain {
			r.sites = append(r.sites[:i], r.sites[i+1:]...)
			if err := r.save(); err != nil {
				return err
			}
			r.notify(ChangeEvent{Type: SiteRemoved, Site: s})
			return nil
		}
	}
	return fmt.Errorf("domain %q not found", domain)
}

func (r *Registry) OnChange(fn func(ChangeEvent)) {
	r.listeners = append(r.listeners, fn)
}

func (r *Registry) notify(e ChangeEvent) {
	for _, fn := range r.listeners {
		fn(e)
	}
}

func (r *Registry) save() error {
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
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/registry/... -v`
Expected: PASS — all 12 tests pass.

**Step 6: Run full test suite**

Run: `go test ./... -v`
Expected: PASS — all tests (config + registry) pass.

**Step 7: Commit**

```bash
git add internal/registry/
git commit -m "feat: add site registry with CRUD, persistence, and change notifications"
```
