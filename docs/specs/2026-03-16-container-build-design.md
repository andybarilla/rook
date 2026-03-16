# Container Build Support — Design Spec

**Date:** 2026-03-16
**Status:** Draft

## Overview

Services with local code (like `api` and `worker` in a typical docker-compose setup) need their Docker images built before they can run. Currently Rook only supports pre-built images (`image` field) or local processes (`command` field). This spec adds a `build` field so Rook can build images from a Dockerfile before starting containers.

## Changes

### 1. Service Struct — Add `Build` and `ForceBuild` Fields

Add to `internal/workspace/workspace.go`:

```go
type Service struct {
    // ...existing fields...
    Build      string `yaml:"build,omitempty"`       // build context path (relative to workspace root)
    ForceBuild bool   `yaml:"-"`                     // runtime flag, not serialized — set by --build CLI flag
}
```

`Build` is the build context path from the manifest. `ForceBuild` is a transient runtime field (no yaml tag) set by the CLI's `--build` flag before passing services to the orchestrator. This avoids changing the `Runner` interface signature and keeps the flag propagation explicit on the value type.

A service with `Build` set is a container service — it gets built into an image and run as a container. It can also have a `Command` field which overrides the container's default entrypoint (like docker-compose's `command:`).

### 2. Update `IsContainer()` / `IsProcess()`

```go
func (s Service) IsContainer() bool { return s.Image != "" || s.Build != "" }
func (s Service) IsProcess() bool   { return s.Command != "" && s.Image == "" && s.Build == "" }
```

This ensures build services are treated as containers throughout the system (port allocation, Docker log streaming, status inspection, reconnection).

### 3. DockerRunner.Start — Build Before Run

When `svc.Build` is non-empty, `DockerRunner.Start`:

1. Derives an image tag: `rook-<workspace>-<service>:latest` (e.g., `rook-kern-app-api:latest`). Uses hyphens instead of slashes to avoid ambiguity with registry paths across Docker and Podman.
2. Resolves the build context path: `filepath.Join(workDir, svc.Build)`. The `workDir` parameter is always `ws.Root` which is set by `Manifest.ToWorkspace()` to the manifest's directory path.
3. Checks if a build is needed:
   - If `svc.ForceBuild` is true: always build
   - If image doesn't exist (checked via `<runtime> image inspect <tag>` — exit code non-zero means absent): build
   - Otherwise: skip build, reuse existing image
4. If building: runs `<ContainerRuntime> build -t <tag> <context-path>`. Build output is streamed to stderr so the user can see progress.
5. Sets a local `imageTag` variable to the built tag
6. Proceeds with the existing `docker run` logic, using `imageTag` instead of `svc.Image` for the image argument

If the service has both `Image` and `Build`, `Build` takes precedence.

The `docker run` args construction already handles `svc.Command != ""` by appending `sh -c <command>` — this works for build services with command overrides (like the `worker` example).

### 4. `rook up --build` Flag

Add a `--build` flag to `rook up`. When set, the CLI sets `svc.ForceBuild = true` on each service that has `Build` defined before passing the workspace to the orchestrator. This is done by mutating a copy of `ws.Services` before calling `orch.Up()`.

`rook restart` does **not** get a `--build` flag in this iteration. Restart always reuses the existing image. To rebuild, use `rook up --build`.

### 5. Compose Discoverer — Extract Build Context and Command

Update `internal/discovery/compose.go`:

**Build extraction:** Handle the simple string form of `build`:
```go
if cs.Build != nil {
    if buildStr, ok := cs.Build.(string); ok {
        svc.Build = buildStr
    }
}
```

The map form (`build: {context: ., dockerfile: ...}`) is silently ignored for now — only the string form is supported in this iteration.

**Command extraction (bug fix):** The `composeService` struct already parses `Command any` but never populates `workspace.Service.Command`. Fix this:
```go
if cs.Command != nil {
    switch v := cs.Command.(type) {
    case string:
        svc.Command = v
    case []any:
        parts := make([]string, len(v))
        for i, p := range v { parts[i] = fmt.Sprintf("%v", p) }
        svc.Command = strings.Join(parts, " ")
    }
}
```

This is a pre-existing bug that this spec fixes because the command override is required for build services (e.g., `worker` with `command: ["./server", "-worker"]`).

### 6. Image Naming Convention

Built images use: `rook-<workspace>-<service>:latest`

Examples:
- `rook-kern-app-api:latest`
- `rook-kern-app-worker:latest`

Hyphens instead of slashes avoid ambiguity with registry paths. Both Docker and Podman handle hyphenated local image names consistently.

### 7. Workspace Root Guarantee

`Manifest.ToWorkspace()` already sets `Root` to the manifest directory when `m.Root` is empty. This means `ws.Root` is always a valid absolute path when a workspace is loaded from the registry (which stores the directory path). No additional fallback is needed — but `DockerRunner.Start` should validate that `workDir` is non-empty before resolving build paths and return an error if it is.

## Manifest Example

```yaml
name: kern-app
type: single

services:
  api:
    build: .
    ports: [8080]
    depends_on: [postgres, redis]
    environment:
      DATABASE_URL: "postgres://kern:kern@{{.Host.postgres}}:{{.Port.postgres}}/kern"

  worker:
    build: .
    command: "./server -worker"
    depends_on: [postgres, redis]

  postgres:
    image: postgres:16-alpine
    ports: [5432]
    environment:
      POSTGRES_USER: kern
      POSTGRES_PASSWORD: kern
      POSTGRES_DB: kern

  redis:
    image: redis:7-alpine
    ports: [6379]
```

## Testing Strategy

- **IsContainer/IsProcess:** Update existing workspace tests to verify `Build` field makes a service a container, and that a service with `Build` + `Command` is still a container (not a process).
- **DockerRunner build:** Integration test (requires Docker/Podman, skip if unavailable) that creates a temp dir with a minimal Dockerfile (`FROM alpine:latest`), calls Start with a Build service, verifies the image was built and container started.
- **Compose discoverer:** Test that `build: .` in compose YAML populates `Service.Build`. Test that `command: ["./server", "-worker"]` populates `Service.Command`.
- **`--build` flag:** Verify the flag is registered on `rook up` and that `ForceBuild` is set on services.
