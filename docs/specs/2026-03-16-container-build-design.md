# Container Build Support — Design Spec

**Date:** 2026-03-16
**Status:** Draft

## Overview

Services with local code (like `api` and `worker` in a typical docker-compose setup) need their Docker images built before they can run. Currently Rook only supports pre-built images (`image` field) or local processes (`command` field). This spec adds a `build` field so Rook can build images from a Dockerfile before starting containers.

## Changes

### 1. Service Struct — Add `Build` Field

Add to `internal/workspace/workspace.go`:

```go
type Service struct {
    // ...existing fields...
    Build string `yaml:"build,omitempty"` // build context path (relative to workspace root)
}
```

A service with `Build` set is a container service — it gets built into an image and run as a container. It can also have a `Command` field which overrides the container's default entrypoint (like docker-compose's `command:`).

### 2. Update `IsContainer()` / `IsProcess()`

```go
func (s Service) IsContainer() bool { return s.Image != "" || s.Build != "" }
func (s Service) IsProcess() bool   { return s.Command != "" && s.Image == "" && s.Build == "" }
```

This ensures build services are treated as containers throughout the system (port allocation, Docker log streaming, status inspection, reconnection).

### 3. DockerRunner.Start — Build Before Run

When a service has a `Build` context path (passed via a new field on `workspace.Service`), `DockerRunner.Start`:

1. Derives an image tag: `rook/<workspace>/<service>:latest` (e.g., `rook/kern-app/api:latest`)
2. Resolves the build context path relative to `workDir` (the workspace root)
3. Runs `podman build -t <tag> <context-path>` (or `docker build` depending on `ContainerRuntime`)
4. Uses the built tag as the image for `podman run`

The build step is skipped if the image already exists AND the `--build` flag is not set. Image existence is checked via `podman image inspect <tag>`.

If the service also has an `Image` field set, `Build` takes precedence — the image is built, not pulled.

### 4. `rook up --build` Flag

Add a `--build` flag to `rook up`. When set, forces a rebuild of all services that have `Build` defined, even if the image already exists. This matches `docker compose up --build` behavior.

The flag is passed through to the orchestrator/runner layer. The simplest approach: set `Service.Image` to the built tag before calling `Start`, and pass a `forceBuild bool` parameter or use a context value.

Implementation detail: The `--build` flag sets a package-level or context variable that `DockerRunner.Start` checks. If true, it always runs the build step for services with `Build` set.

### 5. Compose Discoverer — Extract Build Context

The `composeService` struct already parses `Build any`. Update the compose discoverer to handle the simple string form:

```go
// In Discover(), when processing each service:
if cs.Build != nil {
    if buildStr, ok := cs.Build.(string); ok {
        svc.Build = buildStr
    }
}
```

This populates `Service.Build` with the context path (e.g., `.` for current directory).

### 6. Image Naming Convention

Built images use: `rook/<workspace>/<service>:latest`

Examples:
- `rook/kern-app/api:latest`
- `rook/kern-app/worker:latest`

This avoids collisions between workspaces and makes it easy to identify rook-built images.

### 7. Command Override for Build Services

A service with both `Build` and `Command` (like the `worker` example: `build: .` + `command: ["./server", "-worker"]`) builds the image and runs it with the command override. The `DockerRunner` already appends the command to `docker run` args when `svc.Command` is set.

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

- **IsContainer/IsProcess:** Update existing workspace tests to verify `Build` field makes a service a container.
- **DockerRunner build:** Integration test (requires Docker/Podman) that creates a minimal Dockerfile, calls Start with a Build service, verifies the image was built and container started.
- **Compose discoverer:** Test that `build: .` in compose YAML populates `Service.Build`.
- **`--build` flag:** Verify the flag is registered and passed through.
