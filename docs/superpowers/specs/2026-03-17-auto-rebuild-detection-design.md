# Auto-Rebuild Detection

## Problem

When Dockerfiles or files in the build context change, `rook up` continues using stale images. Users must remember to run `rook up --build` manually, which is error-prone.

## Solution

Detect when images are stale relative to their build context and prompt the user to rebuild before starting services.

## User Experience

### `rook up` Integration

When running `rook up` with services that have build contexts:

```
$ rook up myproject
Checking for stale builds...

3 services need rebuild:
  - api (Dockerfile modified)
  - worker (src/main.go, go.mod changed)
  - frontend (package.json changed)

Rebuild all? [Y/n]: y
Rebuilding api...
Rebuilding worker...
Rebuilding frontend...

Starting myproject (profile: default)...
```

If user answers `n`, proceed with existing images (may fail if images don't exist).

If no services need rebuild, skip the prompt entirely and proceed directly.

### New `rook check-builds` Command

```
$ rook check-builds myproject
api: needs rebuild (Dockerfile modified)
worker: needs rebuild (3 files changed)
frontend: up to date
db: no build context (uses image)

$ rook check-builds myproject --json
{
  "services": {
    "api": {"status": "needs_rebuild", "reason": "Dockerfile modified"},
    "worker": {"status": "needs_rebuild", "reason": "3 files changed"},
    "frontend": {"status": "up_to_date"},
    "db": {"status": "no_build_context"}
  }
}
```

Exit codes:
- 0: All services up-to-date
- 1: One or more services need rebuild
- 2: Error occurred

### Interaction with Existing Flags

- `--build`: Force rebuild all build-context services, skip detection and prompt
- `-d` (detach): Run detection; if stale builds exist, prompt before detaching (unless non-interactive, then skip rebuild)

### Non-Interactive Terminal

When stdin is not a TTY:
- Default to 'n' (do not rebuild stale services)
- Print a warning listing services that need rebuild
- Proceed with existing images

## Technical Design

### Build Cache File

Location: `.rook/build-cache.json` in workspace root

Schema:
```json
{
  "version": 1,
  "services": {
    "api": {
      "image_id": "sha256:abc123...",
      "dockerfile_hash": "e3b0c442...",
      "context_files": {
        "src/main.go": {"mtime": 1709123456, "hash": "a1b2c3..."},
        "go.mod": {"mtime": 1709123400, "hash": "d4e5f6..."}
      }
    }
  }
}
```

Fields:
- `version`: Schema version for future migrations
- `services`: Map of service name to build metadata
- `image_id`: Docker image ID (from `docker inspect --format='{{.Id}}'`)
- `dockerfile_hash`: SHA256 hash of Dockerfile content
- `context_files`: Map of relative file path to mtime/hash pair

### Change Detection Algorithm

1. **Load cache** from `.rook/build-cache.json` (empty if missing)
2. **Check image exists**: If image missing, needs rebuild
3. **Check image ID match**: If cached `image_id` differs from current image, needs rebuild (image was rebuilt externally)
4. **Parse `.dockerignore`**: Load ignore patterns from build context
5. **Hash Dockerfile**: Compare to cached hash, rebuild if changed
6. **Walk build context**:
   - Skip files matching `.dockerignore` patterns
   - For each file:
     - If not in cache: needs rebuild (new file)
     - If mtime matches cache: unchanged
     - If mtime differs: compute hash, compare to cached hash
     - If hash differs: needs rebuild
7. **Check for deleted files**: If cached file no longer exists, needs rebuild

### `.dockerignore` Handling

Use `github.com/moby/patternmatcher` for parsing `.dockerignore` (same library Docker uses).

If `.dockerignore` is missing, scan all files in build context.

Default exclusions (even without `.dockerignore`):
- `.rook/` directory
- `.git/` directory

### New Package: `internal/buildcache/`

```
internal/buildcache/
  cache.go         # Cache struct, Load/Save, UpdateAfterBuild
  detect.go        # DetectStale function
  dockerignore.go  # parseDockerignore function
```

#### `cache.go`

```go
type FileEntry struct {
    Mtime int64  `json:"mtime"`
    Hash  string `json:"hash"`
}

type ServiceCache struct {
    ImageID        string                `json:"image_id"`
    DockerfileHash string                `json:"dockerfile_hash"`
    ContextFiles   map[string]FileEntry  `json:"context_files"`
}

type Cache struct {
    Version  int                    `json:"version"`
    Services map[string]ServiceCache `json:"services"`
}

func Load(path string) (*Cache, error)
func (c *Cache) Save(path string) error
func (c *Cache) UpdateAfterBuild(service, buildCtx, dockerfile, imageID string) error
```

#### `detect.go`

```go
type StaleResult struct {
    NeedsRebuild bool
    Reasons      []string
}

// workDir is the workspace root path (for resolving relative build context paths)
// svc.Build is the build context path relative to workDir
func DetectStale(cache *Cache, service string, svc workspace.Service, workDir string) (*StaleResult, error)

// shared with UpdateAfterBuild for walking context and hashing files
func walkBuildContext(buildCtx string, ignorePatterns []string) (map[string]FileEntry, error)
```

#### `dockerignore.go`

```go
func parseDockerignore(buildCtx string) ([]string, error)
func matchesPatterns(path string, patterns []string) bool
```

### Integration Points

#### `internal/cli/up.go`

1. After loading workspace, before port allocation
2. Call `buildcache.DetectStale` for each service with `Build != ""`
3. If any stale and TTY is available, show prompt
4. If user confirms, set `ForceBuild = true` for stale services
5. After builds complete, call `cache.UpdateAfterBuild` for each built service

#### `internal/cli/check_builds.go`

New command:
```go
func newCheckBuildsCmd() *cobra.Command {
    var jsonOutput bool
    // ...
}
```

#### `internal/runner/docker.go`

Add a method to get the image ID for a built service (used by `up.go` to update cache after build):

```go
// GetImageID returns the Docker image ID for a service's built image.
// Called after successful build to update the build cache.
func (r *DockerRunner) GetImageID(serviceName string) (string, error)
```

This avoids modifying the `Runner` interface.

### Edge Cases

| Case | Behavior |
|------|----------|
| No cache file | Treat all build services as needs rebuild |
| Image missing | Auto-rebuild without prompt (must build to proceed) |
| `.dockerignore` missing | Scan all files |
| Build context path invalid | Return error |
| Dockerfile path invalid | Return error |
| Non-interactive terminal, stale images | Print warning, skip rebuild, proceed with existing images |
| Non-interactive terminal, missing images | Auto-rebuild (no prompt needed, image doesn't exist) |
| Service has no build context | Skip detection (uses image) |

### File Structure

```
internal/buildcache/
  cache.go
  cache_test.go
  detect.go
  detect_test.go
  dockerignore.go
  dockerignore_test.go

internal/cli/
  check_builds.go
  check_builds_test.go
  up.go              # modified
```

## Out of Scope

- Partial rebuilds (rebuilding only changed layers)
- Build cache invalidation across workspaces
- Remote cache storage
- Watching for changes during runtime
