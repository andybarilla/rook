package buildcache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/buildcache"
)

func TestCache_LoadMissingReturnsEmpty(t *testing.T) {
	cache, err := buildcache.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if len(cache.Services) != 0 {
		t.Errorf("expected empty services, got %d", len(cache.Services))
	}
}

func TestCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "build-cache.json")

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:abc123",
				DockerfileHash: "hash1",
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: 12345, Hash: "hash2"},
				},
			},
		},
	}

	if err := cache.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := buildcache.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 1 {
		t.Errorf("version: got %d", loaded.Version)
	}
	if loaded.Services["api"].ImageID != "sha256:abc123" {
		t.Errorf("image ID: got %s", loaded.Services["api"].ImageID)
	}
	if loaded.Services["api"].ContextFiles["main.go"].Mtime != 12345 {
		t.Errorf("mtime: got %d", loaded.Services["api"].ContextFiles["main.go"].Mtime)
	}
}

func TestCache_CreateDirIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "build-cache.json")

	cache := &buildcache.Cache{Version: 1, Services: map[string]buildcache.ServiceCache{}}
	if err := cache.Save(path); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatal("expected file to exist:", err)
	}
}

func TestHashFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := buildcache.HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// SHA256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("hash: got %s, want %s", hash, expected)
	}
}

func TestHashFile_Missing(t *testing.T) {
	_, err := buildcache.HashFile(filepath.Join(t.TempDir(), "missing.txt"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestCache_UpdateAfterBuild(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "Dockerfile", "FROM alpine")
	createTestFile(t, dir, "main.go", "package main")

	cache := &buildcache.Cache{Version: 1, Services: map[string]buildcache.ServiceCache{}}

	err := cache.UpdateAfterBuild("api", dir, dir, "Dockerfile", "sha256:newimage")
	if err != nil {
		t.Fatal(err)
	}

	if cache.Services["api"].ImageID != "sha256:newimage" {
		t.Errorf("image ID: got %s", cache.Services["api"].ImageID)
	}
	if cache.Services["api"].DockerfileHash == "" {
		t.Error("expected Dockerfile hash to be set")
	}
	if len(cache.Services["api"].ContextFiles) == 0 {
		t.Error("expected context files to be populated")
	}
	if _, exists := cache.Services["api"].ContextFiles["main.go"]; !exists {
		t.Error("expected main.go in context files")
	}
}

func TestCache_UpdateAfterBuild_RelativeBuildContext(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "docker/Dockerfile.dev", "FROM alpine")
	createTestFile(t, dir, "app/main.go", "package main")

	cache := &buildcache.Cache{Version: 1, Services: map[string]buildcache.ServiceCache{}}

	err := cache.UpdateAfterBuild("api", dir, filepath.Join(dir, "app"), "docker/Dockerfile.dev", "sha256:newimage")
	if err != nil {
		t.Fatal(err)
	}

	if cache.Services["api"].ImageID != "sha256:newimage" {
		t.Errorf("image ID: got %s", cache.Services["api"].ImageID)
	}
	// Should have files from the build context (app/), not workspace root
	if _, exists := cache.Services["api"].ContextFiles["main.go"]; !exists {
		t.Error("expected main.go in context files (relative to build context)")
	}
}

func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
