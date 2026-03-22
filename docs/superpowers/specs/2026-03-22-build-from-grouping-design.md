# Design: Group `build_from` Services in Rebuild Prompt

**Date:** 2026-03-22
**Status:** Draft

## Problem

When `rook up` detects stale builds, the rebuild prompt lists only source services (those with a `build` field). Services that use `build_from` to share a source's image are invisible in the prompt. The user has no indication that rebuilding `api` also affects `worker`.

Additionally, after a source service is rebuilt, `build_from` consumers may still have running containers with the old image. These stale containers need to be removed before the orchestrator reconnects, or they'll be adopted with the stale image.

## Scope

- `rook up` rebuild prompt only (not `check-builds` command)
- Container removal for `build_from` consumers before orchestrator start when their source is being rebuilt
- Only consumers in the active profile's resolved service list are shown/affected

## Design

### 1. Grouped Rebuild Prompt

After collecting stale services in `up.go`, build a reverse map of `build_from` consumers for each stale source. Only include consumers that are in the resolved service list for the active profile. Display them grouped:

```
2 service(s) need rebuild:
  - api (Dockerfile modified)
    also used by: worker
  - frontend (context files changed)
```

**Details:**
- The count still reflects source services (keeps existing "service(s)" wording)
- Consumer names are sorted alphabetically
- Multiple consumers are comma-separated: `also used by: worker, scheduler`
- If a source has no `build_from` consumers (or none in the active profile), it displays as today — no "also used by" line
- The grouping applies to both the interactive prompt and the auto-rebuild (missing image) output

**Implementation:**
1. After the `staleServices` map is built, iterate the resolved service list to find `build_from` references pointing at stale sources
2. Build a `map[string][]string` of source name -> sorted consumer names
3. Update the display loop to print consumer annotations
4. Extract the display logic into a helper function that accepts an `io.Writer` for testability

### 2. Remove Consumer Containers Before Reconnect

When a source service will be rebuilt (either via interactive prompt or `--build` flag), its `build_from` consumers' existing containers must be removed so they aren't adopted by `orch.Reconnect` with the stale image.

**Implementation:**
1. After `ForceBuild` flags are set (from either the interactive prompt or `--build` flag), identify `build_from` consumers of any service with `ForceBuild=true` that are in the active profile
2. For each consumer, call `runner.StopContainer(containerName)` to stop and remove the existing container
3. This must happen BEFORE `orch.Reconnect` is called — the insertion point is between `ForceBuild` flag setting and `orch.Reconnect`

**Why before Reconnect?**
`Reconnect` adopts running containers. If a consumer container is still running with the old image, it would be adopted rather than recreated. Removing it first ensures a clean start.

**Edge case — missing source image:** If a `build_from` consumer's source is NOT stale but the source image doesn't exist, this is already handled by `DockerRunner.Start` which checks for the image and returns a clear error. No changes needed for this case.

## Files to Modify

- `internal/cli/up.go` — rebuild prompt grouping + consumer container removal before reconnect

## Testing

- Grouped display with single consumer shows "also used by: worker"
- Grouped display with multiple consumers shows "also used by: scheduler, worker" (sorted)
- Source with no consumers displays identically to current behavior (regression)
- `build_from` consumers NOT in active profile are excluded from display
- Consumer containers are removed before reconnect when source has `ForceBuild=true`
- Consumer containers are removed when `--build` flag is used
- Display logic is tested via extracted helper function with `io.Writer`
