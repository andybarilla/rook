package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/settings"
)

func TestLoad_ReturnsDefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	s, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !s.AutoRebuild {
		t.Error("expected AutoRebuild to be true by default")
	}
}

func TestSave_AndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	s := &settings.Settings{AutoRebuild: false}
	if err := s.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := settings.Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.AutoRebuild {
		t.Error("expected AutoRebuild to be false")
	}
}

func TestSave_CreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "another", "settings.json")

	s := &settings.Settings{AutoRebuild: true}
	if err := s.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}
