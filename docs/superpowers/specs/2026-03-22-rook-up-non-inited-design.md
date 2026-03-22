# Design: `rook up` from non-inited directory

**Date:** 2026-03-22
**Issue:** #7

## Problem

When a user runs `rook up` in a directory that has a `rook.yaml` but hasn't been registered with `rook init`, the command fails with an unhelpful "not found" error. The user has to know to run `rook init .` first.

## Behavior

When any workspace-resolving command that loads a workspace (`up`, `restart`, `status`, `logs`, `env`, `check-builds`) infers the workspace name from a `rook.yaml` in the current directory but the workspace isn't in the registry, it prompts:

```
Workspace "myapp" is not registered. Initialize it now? [Y/n]
```

- **Yes**: Run init logic (port allocation, registry entry, `.rook/` scaffold), then continue with the original command.
- **No**: Exit with `Run "rook init ." to register this workspace.`
- **Non-TTY stdin**: Fall back to the existing error (no prompt).

**Excluded commands:** `down` (works via container scanning, never calls `loadWorkspace`).

**Explicit name case:** When the user passes a workspace name as an argument (`rook up myapp`) and it's not registered, the existing error is returned with no prompt — the prompt only fires when the name was inferred from a local `rook.yaml`.

## Implementation

### New method: `resolveAndLoadWorkspace`

Introduce `resolveAndLoadWorkspace(args []string) (*workspace.Workspace, error)` on `cliContext`. This method combines `resolveWorkspaceName` + `loadWorkspace` and preserves whether the name came from cwd inference or an explicit argument:

1. Call `resolveWorkspaceName(args)` to get the name and track whether it came from cwd.
2. Call `loadWorkspace(name)`.
3. If `loadWorkspace` fails with "not found" **and** the name was inferred from cwd:
   - Check if stdin is a TTY. If not, return the existing error.
   - Prompt the user for confirmation.
   - If yes, call `initFromManifest(cwd, registry, allocator)`.
   - Reload and return the workspace.
4. If the name was from an explicit argument, return the error as-is.

Commands that currently call `resolveWorkspaceName` + `loadWorkspace` separately switch to `resolveAndLoadWorkspace`.

### `initFromManifest` helper

```go
func initFromManifest(dir string, reg *registry.FileRegistry, alloc *ports.FileAllocator) error
```

Extracted from `init.go`, this helper does:

- Parse `rook.yaml` from `dir`
- Register workspace in registry via `reg.Register(name, dir)`
- Allocate ports for all services (respecting `pin_port`)
- Create `.rook/` directory and `.gitignore`
- Append to CLAUDE.md/AGENTS.md if present

### What it skips vs full `rook init`

- **Skips**: Auto-discovery, `rook.yaml` generation, devcontainer script copying
- **Keeps**: Registry entry, port allocation, `.rook/` scaffold, agentmd append

## Scope boundaries

- Only triggers when `rook.yaml` exists in cwd and name was inferred from it — no auto-discovery
- Only handles "not registered" — does not handle stale/moved registry paths
- `reg.Register` already errors on duplicate names, which guards against double-registration
- Prompt requires a TTY; non-TTY falls back to the existing error

## Testing

- Unit test for `initFromManifest`: verifies port allocation, registry entry, and `.rook/` scaffold creation
- Unit test for `resolveAndLoadWorkspace` happy path: cwd has `rook.yaml`, not registered, prompt answered yes, workspace loads successfully
- Unit test confirming prompt is skipped when workspace is already registered (normal path)
- Unit test confirming explicit name argument returns error without prompting when not found
- Unit test confirming non-TTY stdin falls back to error message (no prompt)
