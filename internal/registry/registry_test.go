package registry_test

import (
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/registry"
)

func TestRegisterAndList(t *testing.T) {
	dir := t.TempDir()
	r, err := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = r.Register("skeetr", "/home/user/dev/skeetr")
	if err != nil {
		t.Fatal(err)
	}
	list := r.List()
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].Name != "skeetr" {
		t.Errorf("expected skeetr, got %s", list[0].Name)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	r.Register("skeetr", "/path/a")
	err := r.Register("skeetr", "/path/b")
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestGetByName(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	r.Register("skeetr", "/some/path")
	entry, err := r.Get("skeetr")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Path != "/some/path" {
		t.Errorf("expected /some/path, got %s", entry.Path)
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	_, err := r.Get("nope")
	if err == nil {
		t.Fatal("expected error for missing")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	r, _ := registry.NewFileRegistry(filepath.Join(dir, "workspaces.json"))
	r.Register("skeetr", "/path")
	r.Remove("skeetr")
	if len(r.List()) != 0 {
		t.Fatal("expected 0 after remove")
	}
}

func TestPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspaces.json")
	r1, _ := registry.NewFileRegistry(path)
	r1.Register("skeetr", "/path")
	r2, _ := registry.NewFileRegistry(path)
	if len(r2.List()) != 1 {
		t.Fatal("expected 1 after reload")
	}
}
