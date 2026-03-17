# Auto-Rebuild Detection Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Detect when container images are stale relative to their build context and prompt the user to rebuild before starting services.

**Architecture:** New `internal/buildcache/` package handles cache persistence, `.dockerignore` parsing, and stale detection. `rook check-builds` command provides CI-friendly inspection. `rook up` integrates detection with an interactive prompt.

**Tech Stack:** Go 1.22+, stdlib `testing`, `github.com/moby/patternmatcher` for .dockerignore

**Spec:** `docs/superpowers/specs/2026-03-17-auto-rebuild-detection-design.md`

---

## Chunk 1: Build Cache Core

### Task 1: Add buildcache package with Cache struct and file hashing

**Files:**
- Create: `internal/buildcache/cache.go`
- Create: `internal/buildcache/cache_test.go`
- Create: `internal/buildcache/hash.go`

- [ ] **Step 1: Write failing tests for Cache Load/Save**

In `internal/buildcache/cache_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/buildcache/ -run TestCache -v`
Expected: FAIL — package not defined

- [ ] **Step 3: Implement Cache struct and Load/Save**

In `internal/buildcache/cache.go`:

```go
package buildcache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileEntry stores mtime and content hash for a single file.
type FileEntry struct {
	Mtime int64  `json:"mtime"`
	Hash  string `json:"hash"`
}

// ServiceCache stores build metadata for a single service.
type ServiceCache struct {
	ImageID        string               `json:"image_id"`
	DockerfileHash string               `json:"dockerfile_hash"`
	ContextFiles   map[string]FileEntry `json:"context_files"`
}

// Cache stores build metadata for all services in a workspace.
type Cache struct {
	Version  int                    `json:"version"`
	Services map[string]ServiceCache `json:"services"`
}

// Load reads the cache from disk. Returns empty cache if file doesn't exist.
func Load(path string) (*Cache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Cache{Version: 1, Services: make(map[string]ServiceCache)}, nil
		}
		return nil, fmt.Errorf("reading build cache: %w", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing build cache: %w", err)
	}
	if cache.Services == nil {
		cache.Services = make(map[string]ServiceCache)
	}
	return &cache, nil
}

// Save writes the cache to disk, creating parent directories if needed.
func (c *Cache) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding build cache: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/buildcache/ -run TestCache -v`
Expected: all PASS

- [ ] **Step 5: Add file hashing tests**

Append to `internal/buildcache/cache_test.go`:

```go
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
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/buildcache/ -run TestHash -v`
Expected: FAIL — `HashFile` not defined

- [ ] **Step 7: Implement HashFile**

In `internal/buildcache/hash.go`:

```go
package buildcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// HashFile returns the SHA256 hash of a file's contents.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/buildcache/ -run TestHash -v`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add internal/buildcache/
git commit -m "feat: add buildcache package with Cache struct and file hashing"
```

---

## Chunk 2: .dockerignore Parsing

### Task 2: Add .dockerignore support

**Files:**
- Create: `internal/buildcache/dockerignore.go`
- Create: `internal/buildcache/dockerignore_test.go`

- [ ] **Step 1: Write failing tests for .dockerignore parsing**

In `internal/buildcache/dockerignore_test.go`:

```go
package buildcache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/buildcache"
)

func TestParseDockerignore_MissingReturnsDefault(t *testing.T) {
	dir := t.TempDir()
	patterns, err := buildcache.ParseDockerignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should include default exclusions
	if len(patterns) == 0 {
		t.Error("expected default patterns")
	}
}

func TestParseDockerignore_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte("*.log\ntmp/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := buildcache.ParseDockerignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should include file patterns + defaults
	found := false
	for _, p := range patterns {
		if p == "*.log" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected *.log pattern")
	}
}

func TestMatchesPatterns_Simple(t *testing.T) {
	patterns := []string{"*.log", "tmp/"}

	if buildcache.MatchesPatterns("test.log", patterns) != true {
		t.Error("test.log should match")
	}
	if buildcache.MatchesPatterns("src/test.log", patterns) != true {
		t.Error("src/test.log should match")
	}
	if buildcache.MatchesPatterns("tmp/file.txt", patterns) != true {
		t.Error("tmp/file.txt should match")
	}
	if buildcache.MatchesPatterns("main.go", patterns) != false {
		t.Error("main.go should not match")
	}
}

func TestMatchesPatterns_Negation(t *testing.T) {
	patterns := []string{"*.log", "!important.log"}

	if buildcache.MatchesPatterns("test.log", patterns) != true {
		t.Error("test.log should match")
	}
	if buildcache.MatchesPatterns("important.log", patterns) != false {
		t.Error("important.log should not match (negated)")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/buildcache/ -run TestDockerignore -v`
Expected: FAIL — functions not defined

- [ ] **Step 3: Implement .dockerignore parsing**

In `internal/buildcache/dockerignore.go`:

```go
package buildcache

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/patternmatcher"
)

// Default exclusion patterns (always applied, even without .dockerignore).
var defaultExclusions = []string{".rook/", ".git/"}

// ParseDockerignore reads .dockerignore from the given directory.
// Returns default exclusions if file doesn't exist.
func ParseDockerignore(dir string) ([]string, error) {
	path := filepath.Join(dir, ".dockerignore")
	patterns := append([]string{}, defaultExclusions...)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, fmt.Errorf("reading .dockerignore: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading .dockerignore: %w", err)
	}
	return patterns, nil
}

// MatchesPatterns checks if a file path matches any of the patterns.
// Supports negation patterns (those starting with !).
func MatchesPatterns(path string, patterns []string) bool {
	// Convert path to forward slashes for consistent matching
	path = filepath.ToSlash(path)
	matcher, err := patternmatcher.New(patterns)
	if err != nil {
		// If we can't parse patterns, be conservative and don't exclude
		return false
	}
	matches, _ := matcher.MatchesOrParentMatches(path)
	return matches
}
```

- [ ] **Step 4: Add moby/patternmatcher dependency**

Run: `go get github.com/moby/patternmatcher`

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/buildcache/ -run TestDockerignore -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/buildcache/dockerignore.go internal/buildcache/dockerignore_test.go go.mod go.sum
git commit -m "feat: add .dockerignore parsing to buildcache package"
```

---

## Chunk 3: Stale Detection

### Task 3: Implement DetectStale

**Files:**
- Create: `internal/buildcache/detect.go`
- Create: `internal/buildcache/detect_test.go`

- [ ] **Step 1: Write failing tests for DetectStale**

In `internal/buildcache/detect_test.go`:

```go
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
	// Current image ID is different from cached
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
	// Current image ID is empty (image was deleted)
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
	dfPath := createFile(t, dir, "Dockerfile", "FROM alpine\nRUN echo changed")
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
				ContextFiles:   map[string]buildcache.FileEntry{}, // empty = no files cached
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
	createFile(t, dir, "test.log", "log content") // should be ignored
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
	dfPath := createFile(t, dir, "docker/Dockerfile.dev", "FROM alpine")

	dfHash, _ := buildcache.HashFile(dfPath)

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

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/buildcache/ -run TestDetect -v`
Expected: FAIL — `DetectStale` not defined

- [ ] **Step 3: Implement DetectStale**

In `internal/buildcache/detect.go`:

```go
package buildcache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
)

// StaleResult describes whether a service needs rebuilding and why.
type StaleResult struct {
	NeedsRebuild bool
	Reasons      []string
}

// DetectStale checks if a service's image is stale relative to its build context.
// workDir is the workspace root path. svc.Build is the build context path relative to workDir.
// currentImageID is the current Docker image ID (optional - if empty, image ID check is skipped).
func DetectStale(cache *Cache, service string, svc workspace.Service, workDir, currentImageID string) (*StaleResult, error) {
	result := &StaleResult{}

	if svc.Build == "" {
		return result, nil // no build context, nothing to check
	}

	buildCtx := filepath.Join(workDir, svc.Build)
	cached, hasCache := cache.Services[service]

	// No cache entry = needs rebuild
	if !hasCache {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "no build cache")
		return result, nil
	}

	// Check if image was deleted (cached exists but current doesn't)
	if currentImageID == "" && cached.ImageID != "" {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "image missing")
	}

	// Check if image was rebuilt externally (IDs differ)
	if currentImageID != "" && cached.ImageID != "" && currentImageID != cached.ImageID {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "image rebuilt externally")
	}

	// Determine Dockerfile path
	dockerfile := "Dockerfile"
	if svc.Dockerfile != "" {
		dockerfile = svc.Dockerfile
	}
	dockerfilePath := filepath.Join(workDir, dockerfile)

	// Check Dockerfile
	dockerfileHash, err := HashFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("hashing Dockerfile: %w", err)
	}
	if dockerfileHash != cached.DockerfileHash {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "Dockerfile modified")
	}

	// Parse .dockerignore
	ignorePatterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("parsing .dockerignore: %w", err)
	}

	// Walk build context
	newFiles := make(map[string]FileEntry)
	err = filepath.Walk(buildCtx, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(buildCtx, path)
		if err != nil {
			return err
		}

		// Skip .dockerignore patterns
		if MatchesPatterns(relPath, ignorePatterns) {
			return nil
		}

		// Check mtime first
		cachedEntry, wasCached := cached.ContextFiles[relPath]
		mtime := info.ModTime().Unix()

		if wasCached && cachedEntry.Mtime == mtime {
			// mtime unchanged, file is unchanged
			newFiles[relPath] = cachedEntry
			return nil
		}

		// mtime changed or new file, compute hash
		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", relPath, err)
		}

		newFiles[relPath] = FileEntry{Mtime: mtime, Hash: hash}

		if !wasCached {
			result.NeedsRebuild = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s added", relPath))
		} else if hash != cachedEntry.Hash {
			result.NeedsRebuild = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s modified", relPath))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking build context: %w", err)
	}

	// Check for deleted files
	for relPath := range cached.ContextFiles {
		if _, exists := newFiles[relPath]; !exists {
			result.NeedsRebuild = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s deleted", relPath))
		}
	}

	// Deduplicate and summarize reasons
	result.Reasons = summarizeReasons(result.Reasons)

	return result, nil
}

// summarizeReasons consolidates multiple file changes into a summary.
func summarizeReasons(reasons []string) []string {
	if len(reasons) <= 3 {
		return reasons
	}
	return []string{fmt.Sprintf("%d files changed", len(reasons))}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/buildcache/ -run TestDetect -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/buildcache/detect.go internal/buildcache/detect_test.go
git commit -m "feat: add DetectStale for build context change detection"
```

---

## Chunk 4: UpdateAfterBuild

### Task 4: Add UpdateAfterBuild to refresh cache after build

**Files:**
- Modify: `internal/buildcache/cache.go`
- Modify: `internal/buildcache/cache_test.go`

- [ ] **Step 1: Write failing tests for UpdateAfterBuild**

Append to `internal/buildcache/cache_test.go`:

```go
func TestCache_UpdateAfterBuild(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "Dockerfile", "FROM alpine")
	createFile(t, dir, "main.go", "package main")

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
	createFile(t, dir, "docker/Dockerfile.dev", "FROM alpine") // custom dockerfile path
	createFile(t, dir, "app/main.go", "package main")          // nested build context

	cache := &buildcache.Cache{Version: 1, Services: map[string]buildcache.ServiceCache{}}

	err := cache.UpdateAfterBuild("api", dir, "app", "docker/Dockerfile.dev", "sha256:newimage")
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/buildcache/ -run TestCache_UpdateAfterBuild -v`
Expected: FAIL — `UpdateAfterBuild` not defined

- [ ] **Step 3: Implement UpdateAfterBuild**

Append to `internal/buildcache/cache.go`:

```go
// UpdateAfterBuild refreshes the cache entry for a service after a successful build.
// workDir is the workspace root path.
// buildCtx is the build context directory path (relative or absolute).
// dockerfile is the relative path to the Dockerfile from workDir (or "Dockerfile" if default).
// imageID is the Docker image ID of the built image.
func (c *Cache) UpdateAfterBuild(service, workDir, buildCtx, dockerfile, imageID string) error {
	// Hash Dockerfile (path is relative to workDir)
	dockerfilePath := filepath.Join(workDir, dockerfile)
	dockerfileHash, err := HashFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("hashing Dockerfile: %w", err)
	}

	// Resolve build context to absolute path
	if !filepath.IsAbs(buildCtx) {
		buildCtx = filepath.Join(workDir, buildCtx)
	}

	// Parse .dockerignore
	ignorePatterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return fmt.Errorf("parsing .dockerignore: %w", err)
	}

	// Walk build context
	contextFiles := make(map[string]FileEntry)
	err = filepath.Walk(buildCtx, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(buildCtx, path)
		if err != nil {
			return err
		}

		// Skip .dockerignore patterns
		if MatchesPatterns(relPath, ignorePatterns) {
			return nil
		}

		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", relPath, err)
		}

		contextFiles[relPath] = FileEntry{
			Mtime: info.ModTime().Unix(),
			Hash:  hash,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking build context: %w", err)
	}

	c.Services[service] = ServiceCache{
		ImageID:        imageID,
		DockerfileHash: dockerfileHash,
		ContextFiles:   contextFiles,
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/buildcache/ -run TestCache_UpdateAfterBuild -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/buildcache/cache.go internal/buildcache/cache_test.go
git commit -m "feat: add UpdateAfterBuild to refresh cache after build"
```

---

## Chunk 5: DockerRunner.GetImageID

### Task 5: Add GetImageID method to DockerRunner

**Files:**
- Modify: `internal/runner/docker.go`

- [ ] **Step 1: Add GetImageID method**

Append to `internal/runner/docker.go`:

```go
// GetImageID returns the Docker image ID for a service's built image.
// Used by the build cache to detect external image changes.
func (r *DockerRunner) GetImageID(serviceName string) (string, error) {
	imageTag := fmt.Sprintf("rook-%s-%s:latest", strings.TrimPrefix(r.prefix, "rook_"), serviceName)
	output, err := exec.Command(ContainerRuntime, "inspect", "--format", "{{.Id}}", imageTag).Output()
	if err != nil {
		return "", fmt.Errorf("inspecting image %s: %w", imageTag, err)
	}
	return strings.TrimSpace(string(output)), nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/runner/docker.go
git commit -m "feat: add GetImageID method to DockerRunner"
```

---

## Chunk 6: rook check-builds Command

### Task 6: Implement rook check-builds command

**Files:**
- Create: `internal/cli/check_builds.go`
- Create: `internal/cli/check_builds_test.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Write failing tests for check-builds command**

In `internal/cli/check_builds_test.go`:

```go
package cli_test

import (
	"testing"
)

func TestCheckBuildsCmd_Help(t *testing.T) {
	// Verify command is registered and has help
	cmd := newCheckBuildsCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != "check-builds [workspace]" {
		t.Errorf("Use: got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected non-empty Short")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestCheckBuilds -v`
Expected: FAIL — `newCheckBuildsCmd` not defined

- [ ] **Step 3: Write the check-builds command**

In `internal/cli/check_builds.go`:

```go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/buildcache"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newCheckBuildsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "check-builds [workspace]",
		Short: "Check which services need rebuilding",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}

			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}

			cachePath := filepath.Join(ws.Root, ".rook", "build-cache.json")
			cache, err := buildcache.Load(cachePath)
			if err != nil {
				return fmt.Errorf("loading build cache: %w", err)
			}

			results := make(map[string]buildcache.StaleResult)
			hasStale := false

			docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))

			for name, svc := range ws.Services {
				if svc.Build == "" {
					results[name] = buildcache.StaleResult{}
					continue
				}

				// Get current image ID (optional - may not exist)
				currentImageID, _ := docker.GetImageID(name)

				result, err := buildcache.DetectStale(cache, name, svc, ws.Root, currentImageID)
				if err != nil {
					return fmt.Errorf("checking %s: %w", name, err)
				}
				results[name] = *result
				if result.NeedsRebuild {
					hasStale = true
				}
			}

			if jsonOutput {
				return printCheckBuildsJSON(results, ws.Services)
			}
			return printCheckBuildsText(results, ws.Services, hasStale)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func printCheckBuildsText(results map[string]buildcache.StaleResult, services map[string]workspace.Service, hasStale bool) error {
	for name, svc := range services {
		result := results[name]
		if svc.Build == "" {
			fmt.Printf("%s: no build context (uses image)\n", name)
		} else if result.NeedsRebuild {
			if len(result.Reasons) > 0 {
				fmt.Printf("%s: needs rebuild (%s)\n", name, result.Reasons[0])
			} else {
				fmt.Printf("%s: needs rebuild\n", name)
			}
		} else {
			fmt.Printf("%s: up to date\n", name)
		}
	}

	if hasStale {
		os.Exit(1)
	}
	return nil
}

type checkBuildsJSONOutput struct {
	Services map[string]checkBuildsServiceStatus `json:"services"`
}

type checkBuildsServiceStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func printCheckBuildsJSON(results map[string]buildcache.StaleResult, services map[string]workspace.Service) error {
	output := checkBuildsJSONOutput{Services: make(map[string]checkBuildsServiceStatus)}

	for name, svc := range services {
		result := results[name]
		status := checkBuildsServiceStatus{}

		if svc.Build == "" {
			status.Status = "no_build_context"
		} else if result.NeedsRebuild {
			status.Status = "needs_rebuild"
			if len(result.Reasons) > 0 {
				status.Reason = result.Reasons[0]
			}
		} else {
			status.Status = "up_to_date"
		}

		output.Services[name] = status
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
```

- [ ] **Step 3: Register command in root.go**

In `internal/cli/root.go`, add to the `init()` function:

```go
rootCmd.AddCommand(newCheckBuildsCmd())
```

- [ ] **Step 4: Run tests to verify compilation**

Run: `go build ./cmd/rook`
Expected: builds successfully

- [ ] **Step 5: Test the command manually**

Run: `./rook check-builds --help`
Expected: shows help text

- [ ] **Step 6: Commit**

```bash
git add internal/cli/check_builds.go internal/cli/root.go
git commit -m "feat: add rook check-builds command"
```

---

## Chunk 7: `rook up` Integration

### Task 7: Integrate detection and prompt into rook up

**Files:**
- Modify: `internal/cli/up.go`

- [ ] **Step 1: Add buildcache import and detection logic**

Add to imports in `internal/cli/up.go`:

```go
import (
	...
	"github.com/andybarilla/rook/internal/buildcache"
)
```

- [ ] **Step 2: Add detection and prompt after workspace load, before port allocation**

In `internal/cli/up.go`, add after loading workspace (around line 45) and before port allocation:

```go
			// Check for stale builds
			cachePath := filepath.Join(ws.Root, ".rook", "build-cache.json")
			cache, err := buildcache.Load(cachePath)
			if err != nil {
				return fmt.Errorf("loading build cache: %w", err)
			}

			docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))
			staleServices := make(map[string][]string)
			for name, svc := range ws.Services {
				if svc.Build == "" {
					continue
				}
				// Get current image ID (optional - may not exist yet)
				currentImageID, _ := docker.GetImageID(name)
				result, err := buildcache.DetectStale(cache, name, svc, ws.Root, currentImageID)
				if err != nil {
					return fmt.Errorf("checking %s: %w", name, err)
				}
				if result.NeedsRebuild {
					staleServices[name] = result.Reasons
				}
			}

			// Prompt to rebuild if any stale services
			if len(staleServices) > 0 && !build {
				fmt.Println("Checking for stale builds...")
				fmt.Printf("\n%d service(s) need rebuild:\n", len(staleServices))
				for name, reasons := range staleServices {
					if len(reasons) > 0 {
						fmt.Printf("  - %s (%s)\n", name, reasons[0])
					} else {
						fmt.Printf("  - %s\n", name)
					}
				}

				// Check which services have missing images (must rebuild)
				var missingImages, staleFiles []string
				for name, reasons := range staleServices {
					for _, r := range reasons {
						if r == "image missing" {
							missingImages = append(missingImages, name)
							break
						}
					}
					if _, isMissing := contains(reasons, "image missing"); !isMissing {
						staleFiles = append(staleFiles, name)
					}
				}

				// Auto-rebuild missing images
				if len(missingImages) > 0 {
					for _, name := range missingImages {
						svc := ws.Services[name]
						svc.ForceBuild = true
						ws.Services[name] = svc
					}
					fmt.Printf("\nAuto-rebuilding %d service(s) with missing images...\n", len(missingImages))
				}

				// Prompt for file changes only in interactive mode
				if len(staleFiles) > 0 {
					if !isTerminal(os.Stdin) {
						fmt.Println("\nNon-interactive mode: skipping rebuild for stale files. Use --build to force.")
					} else {
						fmt.Print("\nRebuild all? [Y/n]: ")

						reader := bufio.NewReader(os.Stdin)
						input, _ := reader.ReadString('\n')
						input = strings.TrimSpace(strings.ToLower(input))

						if input == "n" || input == "no" {
							fmt.Println("Proceeding with existing images...")
						} else {
							// Mark stale services for rebuild
							for name := range staleServices {
								svc := ws.Services[name]
								svc.ForceBuild = true
								ws.Services[name] = svc
							}
						}
					}
				}
			}
```

- [ ] **Step 3: Add isTerminal helper, contains helper, and imports**

Add `"bufio"` and buildcache to imports in `internal/cli/up.go`:

```go
import (
	...
	"bufio"
	...
	"github.com/andybarilla/rook/internal/buildcache"
)
```
(Note: `"os"` and `"strings"` are already imported in up.go)

Add helper functions after the command definition:

```go
// isTerminal checks if a file descriptor is a terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) (int, bool) {
	for i, s := range slice {
		if s == item {
			return i, true
		}
	}
	return -1, false
}
```

- [ ] **Step 4: Add cache update after builds complete**

After `orch.Up()` succeeds, update the build cache for all services that have a build context. Reuse the docker runner created earlier:

```go
			// Update build cache for all services with build contexts
			for name, svc := range ws.Services {
				if svc.Build == "" {
					continue
				}
				imageID, err := docker.GetImageID(name)
				if err != nil {
					// Image might not exist yet if build failed or was skipped
					continue
				}
				buildCtx := filepath.Join(ws.Root, svc.Build)
				dockerfile := "Dockerfile"
				if svc.Dockerfile != "" {
					dockerfile = svc.Dockerfile
				}
				if err := cache.UpdateAfterBuild(name, ws.Root, buildCtx, dockerfile, imageID); err != nil {
					warns.add("cannot update build cache for %s: %v", name, err)
					continue
				}
			}
			if err := cache.Save(cachePath); err != nil {
				warns.add("cannot save build cache: %v", err)
			}
```

- [ ] **Step 5: Run tests to verify nothing is broken**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat: integrate build detection and prompt into rook up"
```

---

## Chunk 8: Final Verification

### Task 8: Run full test suite and manual verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -count=1 -v`
Expected: all PASS

- [ ] **Step 2: Build CLI**

Run: `make build-cli`
Expected: builds successfully

- [ ] **Step 3: Manual test - check-builds**

In a workspace with build-context services:
```bash
./rook check-builds
```
Expected: Lists services and their rebuild status

- [ ] **Step 4: Manual test - up with stale detection**

```bash
./rook up <workspace>
```
Expected: Shows stale build prompt if any services need rebuild

- [ ] **Step 5: Update CLAUDE.md**

Remove "Auto-rebuild detection" from "What's Not Yet Implemented" section in CLAUDE.md.

- [ ] **Step 6: Final commit**

```bash
git add CLAUDE.md
git commit -m "docs: remove auto-rebuild detection from not-yet-implemented list"
```
