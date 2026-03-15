package ports_test

import (
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/ports"
)

func TestAllocate_AssignsFromRange(t *testing.T) {
	dir := t.TempDir()
	a, err := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	if err != nil {
		t.Fatal(err)
	}
	port, err := a.Allocate("ws1", "postgres", 0)
	if err != nil {
		t.Fatal(err)
	}
	if port < 10000 || port > 10010 {
		t.Errorf("port %d outside range", port)
	}
}

func TestAllocate_PreferredPort(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	port, _ := a.Allocate("ws1", "app", 10005)
	if port != 10005 {
		t.Errorf("expected 10005, got %d", port)
	}
}

func TestAllocate_StablePorts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ports.json")
	a1, _ := ports.NewFileAllocator(path, 10000, 10010)
	port1, _ := a1.Allocate("ws1", "postgres", 0)
	a2, _ := ports.NewFileAllocator(path, 10000, 10010)
	port2 := a2.Get("ws1", "postgres")
	if !port2.OK {
		t.Fatal("expected port to persist")
	}
	if port1 != port2.Port {
		t.Errorf("port changed: %d -> %d", port1, port2.Port)
	}
}

func TestAllocate_NoConflicts(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10002)
	a.Allocate("ws1", "a", 0)
	a.Allocate("ws1", "b", 0)
	a.Allocate("ws1", "c", 0)
	_, err := a.Allocate("ws1", "d", 0)
	if err == nil {
		t.Fatal("expected error when exhausted")
	}
}

func TestAllocate_PinnedConflict(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	a.AllocatePinned("ws1", "app", 8080)
	_, err := a.AllocatePinned("ws2", "app", 8080)
	if err == nil {
		t.Fatal("expected error for pinned conflict")
	}
}

func TestRelease(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10001)
	a.Allocate("ws1", "a", 0)
	a.Allocate("ws1", "b", 0)
	a.Release("ws1", "a")
	_, err := a.Allocate("ws1", "c", 0)
	if err != nil {
		t.Fatal("should allocate after release")
	}
}

func TestAll(t *testing.T) {
	dir := t.TempDir()
	a, _ := ports.NewFileAllocator(filepath.Join(dir, "ports.json"), 10000, 10010)
	a.Allocate("ws1", "postgres", 0)
	a.Allocate("ws2", "redis", 0)
	all := a.All()
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}
