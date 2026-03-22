# Design: `rook up` from non-inited directory

**Date:** 2026-03-22
**Issue:** #7

## Problem

When a user runs `rook up` in a directory that has a `rook.yaml` but hasn't been registered with `rook init`, the command fails with an unhelpful "not found" error. The user has to know to run `rook init .` first.

## Behavior

When any workspace-resolving command (`up`, `down`, `restart`, `status`, `logs`, `env`, `check-builds`) finds a `rook.yaml` in the current directory but the workspace isn't in the registry, it prompts:

```
Workspace "myapp" is not registered. Initialize it now? [Y/n]
```

- **Yes**: Run init logic (port allocation, registry entry, `.rook/` scaffold), then continue with the original command.
- **No**: Exit with `Run "rook init ." to register this workspace.`
- **Non-TTY stdin**: Fall back to the existing error (no prompt).

## Implementation

### Where the change lives

`internal/cli/context.go` in `loadWorkspace()`. After `resolveWorkspaceName()` succeeds (found `rook.yaml` in cwd), if `registry.Get(name)` returns "not found":

1. Prompt the user for confirmation via stdin.
2. If yes, call `initFromManifest()` — a new helper that performs the subset of init applicable when `rook.yaml` already exists.
3. Reload and continue with the original command.

### `initFromManifest()` helper

Extracted from `init.go`, this helper does:

- Register workspace in registry
- Allocate ports for all services
- Create `.rook/` directory and `.gitignore`
- Append to CLAUDE.md/AGENTS.md if present

### What it skips vs full `rook init`

- **Skips**: Auto-discovery, `rook.yaml` generation, devcontainer script copying
- **Keeps**: Registry entry, port allocation, `.rook/` scaffold, agentmd append

## Scope boundaries

- Only triggers when `rook.yaml` exists in cwd — no auto-discovery from compose files
- Only handles "not registered" — does not handle stale/moved registry paths
- Prompt requires a TTY; non-TTY falls back to the existing error

## Testing

- Unit test for `initFromManifest` helper: verifies port allocation and registry entry
- Unit test confirming prompt is skipped when workspace is already registered
- Unit test confirming non-TTY stdin falls back to error message
