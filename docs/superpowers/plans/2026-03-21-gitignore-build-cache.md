# .gitignore-Aware Build Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the build cache respect `.gitignore` patterns so git-ignored files (node_modules, dist, etc.) don't trigger false-positive stale build detections.

**Architecture:** Add `ParseGitignore` and `CollectIgnorePatterns` functions to the existing `internal/buildcache/` package. `CollectIgnorePatterns` merges default exclusions + `.dockerignore` + `.gitignore` patterns additively. Replace direct `ParseDockerignore` calls in `DetectStale` and `UpdateAfterBuild` with `CollectIgnorePatterns`. Also skip `.gitignore` files from the context walk (like `.dockerignore` already is).

**Tech Stack:** Go 1.22+, `github.com/moby/patternmatcher` (already a dependency)

**Spec:** `docs/superpowers/specs/2026-03-21-gitignore-build-cache-design.md`

---

### File Structure

| File | Responsibility |
|------|---------------|
| `internal/buildcache/ignore.go` | Renamed from `dockerignore.go`. Contains `ParseDockerignore`, `ParseGitignore`, `CollectIgnorePatterns`, `MatchesPatterns`, `normalizePattern` |
| `internal/buildcache/ignore_test.go` | Renamed from `dockerignore_test.go`. All ignore pattern tests |
| `internal/buildcache/detect.go` | `DetectStale` — uses `CollectIgnorePatterns`, skips `.gitignore` |
| `internal/buildcache/detect_test.go` | Stale detection tests including gitignore integration |
| `internal/buildcache/cache.go` | `UpdateAfterBuild` — uses `CollectIgnorePatterns`, skips `.gitignore` |
| `internal/buildcache/cache_test.go` | Cache tests including gitignore integration for `UpdateAfterBuild` |

---

### Task 1: Rename files and add `ParseGitignore`

**Files:**
- Rename: `internal/buildcache/dockerignore.go` → `internal/buildcache/ignore.go`
- Rename: `internal/buildcache/dockerignore_test.go` → `internal/buildcache/ignore_test.go`
- Modify: `internal/buildcache/ignore.go` (add `ParseGitignore`)
- Modify: `internal/buildcache/ignore_test.go` (add tests)

- [ ] **Step 1: Write the failing tests for `ParseGitignore`**

Add to `internal/buildcache/ignore_test.go` (after renaming):

```go
func TestParseGitignore_MissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	patterns, err := buildcache.ParseGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns) != 0 {
		t.Errorf("expected empty patterns for missing .gitignore, got %d", len(patterns))
	}
}

func TestParseGitignore_ReadsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\ndist/\n# comment\n\n*.log\n"), 0644); err != nil {
		t.Fatal(err)
	}

	patterns, err := buildcache.ParseGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"node_modules/", "dist/", "*.log"}
	if len(patterns) != len(expected) {
		t.Fatalf("expected %d patterns, got %d: %v", len(expected), len(patterns), patterns)
	}
	for i, p := range expected {
		if patterns[i] != p {
			t.Errorf("pattern[%d]: got %q, want %q", i, patterns[i], p)
		}
	}
}
```

- [ ] **Step 2: Rename files**

```bash
cd /home/andy/dev/andybarilla/rook
git mv internal/buildcache/dockerignore.go internal/buildcache/ignore.go
git mv internal/buildcache/dockerignore_test.go internal/buildcache/ignore_test.go
```

- [ ] **Step 3: Run tests to verify rename didn't break anything**

```bash
go test ./internal/buildcache/ -v -run 'TestParseDockerignore|TestMatchesPatterns'
```

Expected: all existing tests PASS (package name unchanged, only file renamed)

- [ ] **Step 4: Run new tests to verify they fail**

```bash
go test ./internal/buildcache/ -v -run 'TestParseGitignore'
```

Expected: FAIL — `ParseGitignore` not defined

- [ ] **Step 5: Implement `ParseGitignore`**

Add to `internal/buildcache/ignore.go`:

```go
// ParseGitignore reads .gitignore from the given directory.
// Returns empty slice if file doesn't exist (unlike ParseDockerignore,
// does not include default exclusions — those are added by CollectIgnorePatterns).
func ParseGitignore(dir string) ([]string, error) {
	path := filepath.Join(dir, ".gitignore")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading .gitignore: %w", err)
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading .gitignore: %w", err)
	}
	return patterns, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./internal/buildcache/ -v -run 'TestParseGitignore'
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/buildcache/ignore.go internal/buildcache/ignore_test.go
git commit -m "feat(buildcache): rename dockerignore files to ignore, add ParseGitignore"
```

---

### Task 2: Add `CollectIgnorePatterns`

**Files:**
- Modify: `internal/buildcache/ignore.go` (add `CollectIgnorePatterns`)
- Modify: `internal/buildcache/ignore_test.go` (add tests)

- [ ] **Step 1: Write the failing tests**

Add to `internal/buildcache/ignore_test.go`:

```go
func TestCollectIgnorePatterns_DefaultsOnly(t *testing.T) {
	dir := t.TempDir()
	// No .dockerignore, no .gitignore
	patterns, err := buildcache.CollectIgnorePatterns(dir, dir)
	if err != nil {
		t.Fatal(err)
	}
	// Should have default exclusions (.rook/, .git/)
	if len(patterns) < 2 {
		t.Errorf("expected at least default patterns, got %d", len(patterns))
	}
}

func TestCollectIgnorePatterns_MergesDockerignoreAndGitignore(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte("*.log\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644)

	patterns, err := buildcache.CollectIgnorePatterns(dir, dir)
	if err != nil {
		t.Fatal(err)
	}

	hasLog := false
	hasNodeModules := false
	for _, p := range patterns {
		if p == "*.log" {
			hasLog = true
		}
		if p == "node_modules/" {
			hasNodeModules = true
		}
	}
	if !hasLog {
		t.Error("expected *.log from .dockerignore")
	}
	if !hasNodeModules {
		t.Error("expected node_modules/ from .gitignore")
	}
}

func TestCollectIgnorePatterns_WorkDirGitignore(t *testing.T) {
	// Build context is a subdirectory; .gitignore is in workspace root
	workDir := t.TempDir()
	buildCtx := filepath.Join(workDir, "server")
	os.MkdirAll(buildCtx, 0755)
	os.WriteFile(filepath.Join(workDir, ".gitignore"), []byte("node_modules/\ndist/\n"), 0644)

	patterns, err := buildcache.CollectIgnorePatterns(buildCtx, workDir)
	if err != nil {
		t.Fatal(err)
	}

	hasNodeModules := false
	for _, p := range patterns {
		if p == "node_modules/" {
			hasNodeModules = true
		}
	}
	if !hasNodeModules {
		t.Error("expected node_modules/ from workspace root .gitignore")
	}
}

func TestCollectIgnorePatterns_NoDuplicateWhenSameDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644)

	patterns, err := buildcache.CollectIgnorePatterns(dir, dir)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for _, p := range patterns {
		if p == "node_modules/" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected node_modules/ once, got %d times", count)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/buildcache/ -v -run 'TestCollectIgnorePatterns'
```

Expected: FAIL — `CollectIgnorePatterns` not defined

- [ ] **Step 3: Implement `CollectIgnorePatterns`**

Add to `internal/buildcache/ignore.go`:

```go
// CollectIgnorePatterns merges ignore patterns from all sources:
// default exclusions, .dockerignore (from build context), and
// .gitignore (from build context and workspace root).
func CollectIgnorePatterns(buildCtx, workDir string) ([]string, error) {
	// Start with .dockerignore (includes default exclusions)
	patterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return nil, err
	}

	// Add .gitignore from build context
	gitPatterns, err := ParseGitignore(buildCtx)
	if err != nil {
		return nil, err
	}
	patterns = append(patterns, gitPatterns...)

	// Add .gitignore from workspace root if different from build context
	absBuildCtx, err := filepath.Abs(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("resolving build context path: %w", err)
	}
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace root path: %w", err)
	}
	if absBuildCtx != absWorkDir {
		rootGitPatterns, err := ParseGitignore(workDir)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, rootGitPatterns...)
	}

	return patterns, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/buildcache/ -v -run 'TestCollectIgnorePatterns'
```

Expected: PASS

- [ ] **Step 5: Run all buildcache tests to verify no regressions**

```bash
go test ./internal/buildcache/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/buildcache/ignore.go internal/buildcache/ignore_test.go
git commit -m "feat(buildcache): add CollectIgnorePatterns merging dockerignore and gitignore"
```

---

### Task 3: Wire `CollectIgnorePatterns` into `DetectStale` and skip `.gitignore`

**Files:**
- Modify: `internal/buildcache/detect.go:77-81` (replace `ParseDockerignore` call)
- Modify: `internal/buildcache/detect.go:120` (add `.gitignore` skip)
- Modify: `internal/buildcache/detect_test.go` (add integration test)

- [ ] **Step 1: Write the failing test**

Add to `internal/buildcache/detect_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/buildcache/ -v -run 'TestDetectStale_RespectsGitignore|TestDetectStale_GitignoreFileNotTracked'
```

Expected: FAIL — `node_modules/pkg/index.js added` and `.gitignore added`

- [ ] **Step 3: Update `DetectStale` in `detect.go`**

In `internal/buildcache/detect.go`, make two changes:

**Change 1** — Replace lines 77-81 (the `ParseDockerignore` call):

```go
	// Parse .dockerignore
	ignorePatterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("parsing .dockerignore: %w", err)
	}
```

With:

```go
	// Collect ignore patterns from .dockerignore and .gitignore
	ignorePatterns, err := CollectIgnorePatterns(buildCtx, workDir)
	if err != nil {
		return nil, fmt.Errorf("collecting ignore patterns: %w", err)
	}
```

**Change 2** — After the `.dockerignore` skip (line 120-122), add `.gitignore` skip:

```go
		// Skip .dockerignore - it's metadata, not build content
		if relPath == ".dockerignore" {
			return nil
		}

		// Skip .gitignore - it's metadata, not build content
		if relPath == ".gitignore" {
			return nil
		}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/buildcache/ -v -run 'TestDetectStale_RespectsGitignore|TestDetectStale_GitignoreFileNotTracked'
```

Expected: PASS

- [ ] **Step 5: Run all buildcache tests for regressions**

```bash
go test ./internal/buildcache/ -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/buildcache/detect.go internal/buildcache/detect_test.go
git commit -m "feat(buildcache): DetectStale respects .gitignore patterns"
```

---

### Task 4: Wire `CollectIgnorePatterns` into `UpdateAfterBuild` and skip `.gitignore`

**Files:**
- Modify: `internal/buildcache/cache.go:94-98` (replace `ParseDockerignore` call)
- Modify: `internal/buildcache/cache.go:137` (add `.gitignore` skip)
- Modify: `internal/buildcache/cache_test.go` (add integration test)

- [ ] **Step 1: Write the failing test**

Add to `internal/buildcache/cache_test.go`:

```go
func TestCache_UpdateAfterBuild_RespectsGitignore(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "Dockerfile", "FROM alpine")
	createTestFile(t, dir, "main.go", "package main")
	createTestFile(t, dir, "node_modules/pkg/index.js", "module.exports = {}")
	createTestFile(t, dir, ".gitignore", "node_modules/\n")

	cache := &buildcache.Cache{Version: 1, Services: map[string]buildcache.ServiceCache{}}

	err := cache.UpdateAfterBuild("api", dir, dir, "Dockerfile", "sha256:newimage")
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := cache.Services["api"].ContextFiles["node_modules/pkg/index.js"]; exists {
		t.Error("node_modules files should be excluded by .gitignore")
	}
	if _, exists := cache.Services["api"].ContextFiles["main.go"]; !exists {
		t.Error("main.go should still be tracked")
	}
	if _, exists := cache.Services["api"].ContextFiles[".gitignore"]; exists {
		t.Error(".gitignore should not be tracked as content")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/buildcache/ -v -run 'TestCache_UpdateAfterBuild_RespectsGitignore'
```

Expected: FAIL — `node_modules/pkg/index.js` and `.gitignore` appear in context files

- [ ] **Step 3: Update `UpdateAfterBuild` in `cache.go`**

**Change 1** — Replace lines 94-98 (the `ParseDockerignore` call):

```go
	// Parse .dockerignore
	ignorePatterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return fmt.Errorf("parsing .dockerignore: %w", err)
	}
```

With:

```go
	// Collect ignore patterns from .dockerignore and .gitignore
	ignorePatterns, err := CollectIgnorePatterns(buildCtx, workDir)
	if err != nil {
		return fmt.Errorf("collecting ignore patterns: %w", err)
	}
```

**Change 2** — After the `.dockerignore` skip (line 137-139), add `.gitignore` skip:

```go
		// Skip .dockerignore - it's metadata, not build content
		if relPath == ".dockerignore" {
			return nil
		}

		// Skip .gitignore - it's metadata, not build content
		if relPath == ".gitignore" {
			return nil
		}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/buildcache/ -v -run 'TestCache_UpdateAfterBuild_RespectsGitignore'
```

Expected: PASS

- [ ] **Step 5: Run all tests for regressions**

```bash
go test ./internal/buildcache/ -v
```

Expected: all PASS

- [ ] **Step 6: Run full test suite**

```bash
go test ./...
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/buildcache/cache.go internal/buildcache/cache_test.go
git commit -m "feat(buildcache): UpdateAfterBuild respects .gitignore patterns"
```
