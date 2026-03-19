# Shared Build / Image Alias (`build_from`)

## Problem

When multiple services share the same build context and Dockerfile (e.g., `server` and `worker` both build from `.` with `Dockerfile.go`), rook builds the identical image twice. This wastes time and is unnecessary.

## Solution

Add a `build_from` field to `Service` that references another service's built image. The consumer service skips building entirely and uses the source service's image tag.

## Manifest / YAML

New `build_from` field on `Service`:

```yaml
services:
    server:
        build: .
        dockerfile: .rook/Dockerfile.go
        command: /app/.rook/server-start.sh
        ports:
            - 8080
    worker:
        build_from: server
        command: /app/.rook/worker-start.sh
```

### Validation Rules

- `build_from` is mutually exclusive with `build` and `image`
- Referenced service must exist and must have a `build` field
- No chains: a `build_from` target cannot itself use `build_from`
- `build_from` with `command` is valid (overrides image's default CMD)
- `build_from` without `command` is valid (uses image's default CMD)

## Runner / Build Pipeline

In `DockerRunner.Start()`, when a service has `build_from`:

1. Resolve the image tag from the referenced service: `rook-{workspace}-{referenced}:latest`
2. Use that tag directly — no build step
3. If the image doesn't exist, return an error (the referenced service should have been built first)

The runner constructs the tag from the `build_from` value and does not re-validate the reference at runtime. Validation happens at manifest load time only.

### Build-Order Dependency

`build_from` implies a **build-order dependency** but not a runtime dependency:

- The orchestrator ensures the source service's build completes before starting the `build_from` consumer
- `build_from` does NOT add to `depends_on` — the worker doesn't need the server running, just its image built
- Users can still add explicit `depends_on` for runtime dependencies

The orchestrator's topo-sort should treat `build_from` references as edges for ordering purposes. If the source service is not in the active profile, it is implicitly pulled in (the image must be built regardless).

## Build Cache

- `build_from` services are **not tracked** in the build cache — they don't build anything
- `DetectStale()` skips services with `build_from` (no build context to check)
- The `rook up` rebuild prompt only shows source services, not their consumers

## Discovery

When `ComposeDiscoverer` finds two services with identical `(build, dockerfile)` tuple:

- Empty strings match each other (two services with `build: .` and no explicit dockerfile are considered identical)
- The **first alphabetically** keeps `build`/`dockerfile`
- Subsequent matches get `build_from: <first>` instead, with `build`/`dockerfile` cleared
- Existing `depends_on` from compose is preserved as-is

## CLI Output

During `rook up`, when a `build_from` service starts:

```
  worker: using image from server
```

## Changes Required

### `internal/workspace/workspace.go`
- Add `BuildFrom string` field to `Service` struct with `yaml:"build_from,omitempty"` tag
- Update `IsContainer()` to return `true` when `BuildFrom` is set
- Update `IsProcess()` to return `false` when `BuildFrom` is set
- Add `Validate()` method on `Manifest` for `build_from` rules

### `internal/runner/docker.go`
- In `Start()`: resolve `build_from` to source service's image tag
- Print "using image from {source}" message

### `internal/orchestrator/orchestrator.go`
- Add `build_from` edges to topo-sort for build ordering

### `internal/discovery/compose.go`
- Detect duplicate build+dockerfile combos
- Set `build_from` on subsequent matches

### `internal/buildcache/detect.go`
- Skip `build_from` services in staleness detection

### `internal/cli/up.go`
- Skip `build_from` services in rebuild prompts
- Skip cache updates for `build_from` services

### `internal/cli/check_builds.go`
- Skip `build_from` services in build check output
