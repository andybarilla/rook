package registry_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/registry"
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

func TestNodeVersionPersistence(t *testing.T) {
	dir := realDir(t)
	path := tempFile(t)

	r1 := registry.New(path)
	_ = r1.Add(registry.Site{Path: dir, Domain: "nodeapp.test", NodeVersion: "system"})

	r2 := registry.New(path)
	if err := r2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	sites := r2.List()
	if len(sites) != 1 {
		t.Fatalf("expected 1 site, got %d", len(sites))
	}
	if sites[0].NodeVersion != "system" {
		t.Errorf("NodeVersion = %q, want %q", sites[0].NodeVersion, "system")
	}
}

func TestConcurrentAddDoesNotCorrupt(t *testing.T) {
	path := tempFile(t)

	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			r := registry.New(path)
			_ = r.Load()
			dir := realDir(t)
			errs <- r.Add(registry.Site{
				Path:   dir,
				Domain: fmt.Sprintf("site%d.test", n),
			})
		}(i)
	}

	errCount := 0
	for i := 0; i < 10; i++ {
		if err := <-errs; err != nil {
			errCount++
		}
	}

	// Load final state and verify no corruption
	r := registry.New(path)
	if err := r.Load(); err != nil {
		t.Fatalf("Load after concurrent writes: %v", err)
	}

	sites := r.List()
	if len(sites) == 0 {
		t.Error("expected at least 1 site after concurrent adds")
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
