package buildcache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/buildcache"
	"github.com/andybarilla/rook/internal/workspace"
)

func TestDetectStale_NoCacheNeedsRebuild(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "Dockerfile", "FROM alpine")
	createFile(t, dir, "main.go", "package main")

	svc := workspace.Service{Build: "."}
	cache := &buildcache.Cache{Version: 1, Services: map[string]buildcache.ServiceCache{}}

	result, err := buildcache.DetectStale(cache, "api", svc, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (no cache)")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected reason for rebuild")
	}
}

func TestDetectStale_ImageIDChanged(t *testing.T) {
	dir := t.TempDir()
	dfPath := createFile(t, dir, "Dockerfile", "FROM alpine")
	_ = createFile(t, dir, "main.go", "package main")

	dfHash, _ := buildcache.HashFile(dfPath)
	goStat, _ := os.Stat(filepath.Join(dir, "main.go"))
	goHash, _ := buildcache.HashFile(filepath.Join(dir, "main.go"))

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:oldimage",
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix(), Hash: goHash},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "sha256:newimage")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (image ID changed)")
	}
	if len(result.Reasons) == 0 || result.Reasons[0] != "image rebuilt externally" {
		t.Errorf("expected 'image rebuilt externally' reason, got %v", result.Reasons)
	}
}

func TestDetectStale_ImageDeleted(t *testing.T) {
	dir := t.TempDir()
	dfPath := createFile(t, dir, "Dockerfile", "FROM alpine")
	_ = createFile(t, dir, "main.go", "package main")

	dfHash, _ := buildcache.HashFile(dfPath)
	goStat, _ := os.Stat(filepath.Join(dir, "main.go"))
	goHash, _ := buildcache.HashFile(filepath.Join(dir, "main.go"))

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:oldimage",
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix(), Hash: goHash},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (image missing)")
	}
	if len(result.Reasons) == 0 || result.Reasons[0] != "image missing" {
		t.Errorf("expected 'image missing' reason, got %v", result.Reasons)
	}
}

func TestDetectStale_DockerfileChanged(t *testing.T) {
	dir := t.TempDir()
	_ = createFile(t, dir, "Dockerfile", "FROM alpine\nRUN echo changed")
	_ = createFile(t, dir, "main.go", "package main")

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				DockerfileHash: "oldhash",
				ContextFiles:   map[string]buildcache.FileEntry{},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (Dockerfile changed)")
	}
	if len(result.Reasons) == 0 || result.Reasons[0] != "Dockerfile modified" {
		t.Errorf("expected 'Dockerfile modified' reason, got %v", result.Reasons)
	}
}

func TestDetectStale_FileAdded(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "Dockerfile", "FROM alpine")
	createFile(t, dir, "main.go", "package main")

	dfHash, _ := buildcache.HashFile(filepath.Join(dir, "Dockerfile"))

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				DockerfileHash: dfHash,
				ContextFiles:   map[string]buildcache.FileEntry{},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (new file)")
	}
}

func TestDetectStale_FileModified(t *testing.T) {
	dir := t.TempDir()
	dfPath := createFile(t, dir, "Dockerfile", "FROM alpine")
	goPath := createFile(t, dir, "main.go", "package main")

	dfHash, _ := buildcache.HashFile(dfPath)
	goStat, _ := os.Stat(goPath)

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix() - 1000, Hash: "oldhash"},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (file modified)")
	}
}

func TestDetectStale_FileUnchanged(t *testing.T) {
	dir := t.TempDir()
	dfPath := createFile(t, dir, "Dockerfile", "FROM alpine")
	goPath := createFile(t, dir, "main.go", "package main")

	dfHash, _ := buildcache.HashFile(dfPath)
	goStat, _ := os.Stat(goPath)
	goHash, _ := buildcache.HashFile(goPath)

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:abc123",
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix(), Hash: goHash},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "sha256:abc123")
	if err != nil {
		t.Fatal(err)
	}
	if result.NeedsRebuild {
		t.Errorf("expected no rebuild, got reasons: %v", result.Reasons)
	}
}

func TestDetectStale_RespectsDockerignore(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "Dockerfile", "FROM alpine")
	createFile(t, dir, "main.go", "package main")
	createFile(t, dir, "test.log", "log content")
	createFile(t, dir, ".dockerignore", "*.log\n")

	dfHash, _ := buildcache.HashFile(filepath.Join(dir, "Dockerfile"))
	goStat, _ := os.Stat(filepath.Join(dir, "main.go"))
	goHash, _ := buildcache.HashFile(filepath.Join(dir, "main.go"))

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:abc123",
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix(), Hash: goHash},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "sha256:abc123")
	if err != nil {
		t.Fatal(err)
	}
	if result.NeedsRebuild {
		t.Errorf("expected no rebuild, got reasons: %v", result.Reasons)
	}
}

func TestDetectStale_CustomDockerfilePath(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "docker"), 0755)
	_ = createFile(t, dir, "docker/Dockerfile.dev", "FROM alpine")

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				DockerfileHash: "oldhash",
				ContextFiles:   map[string]buildcache.FileEntry{},
			},
		},
	}

	svc := workspace.Service{Build: ".", Dockerfile: "docker/Dockerfile.dev"}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.NeedsRebuild {
		t.Error("expected needs rebuild (custom Dockerfile changed)")
	}
}

func TestDetectStale_RespectsGitignore(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "Dockerfile", "FROM alpine")
	createFile(t, dir, "main.go", "package main")
	createTestFile(t, dir, "node_modules/pkg/index.js", "module.exports = {}")
	createFile(t, dir, ".gitignore", "node_modules/\n")

	dfHash, _ := buildcache.HashFile(filepath.Join(dir, "Dockerfile"))
	goStat, _ := os.Stat(filepath.Join(dir, "main.go"))
	goHash, _ := buildcache.HashFile(filepath.Join(dir, "main.go"))

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:abc123",
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix(), Hash: goHash},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "sha256:abc123")
	if err != nil {
		t.Fatal(err)
	}
	if result.NeedsRebuild {
		t.Errorf("expected no rebuild (node_modules gitignored), got reasons: %v", result.Reasons)
	}
}

func TestDetectStale_GitignoreFileNotTracked(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "Dockerfile", "FROM alpine")
	createFile(t, dir, "main.go", "package main")
	createFile(t, dir, ".gitignore", "*.log\n")

	dfHash, _ := buildcache.HashFile(filepath.Join(dir, "Dockerfile"))
	goStat, _ := os.Stat(filepath.Join(dir, "main.go"))
	goHash, _ := buildcache.HashFile(filepath.Join(dir, "main.go"))

	cache := &buildcache.Cache{
		Version: 1,
		Services: map[string]buildcache.ServiceCache{
			"api": {
				ImageID:        "sha256:abc123",
				DockerfileHash: dfHash,
				ContextFiles: map[string]buildcache.FileEntry{
					"main.go": {Mtime: goStat.ModTime().Unix(), Hash: goHash},
				},
			},
		},
	}

	svc := workspace.Service{Build: "."}
	result, err := buildcache.DetectStale(cache, "api", svc, dir, "sha256:abc123")
	if err != nil {
		t.Fatal(err)
	}
	if result.NeedsRebuild {
		t.Errorf("expected no rebuild (.gitignore should not be tracked as content), got reasons: %v", result.Reasons)
	}
}

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
