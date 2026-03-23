# Design: `rook init` agentmd update + `rook agentmd` command

## Problem

`ensureAgentMDRookSection` in `internal/cli/agentmd.go` skips updating the rook section if `<!-- rook -->` tags already exist. Re-running `rook init` after adding/removing services never refreshes the section. There is also no standalone way to update the section without re-running init.

## Solution

1. Change `ensureAgentMDRookSection` to **upsert**: if `<!-- rook -->...<!-- /rook -->` tags exist, replace everything between them (inclusive) with freshly generated content. If no tags exist, append as before.

2. Add a `rook agentmd` CLI command that loads the workspace manifest and calls the same upsert logic.

## Changes

### `internal/cli/agentmd.go`

- Change signature to `ensureAgentMDRookSection(dir string, m *workspace.Manifest) (action string, err error)`.
- Replace the early-return on existing tags with a replacement: find the start of `<!-- rook -->` and end of `<!-- /rook -->\n`, splice in the new section.
- If `<!-- rook -->` exists without a matching `<!-- /rook -->`, return an error (malformed file the user should fix).
- No change to `buildRookSection` — it already generates the full tagged block.

### `internal/cli/context.go`

- Update the call in `initFromManifest` to handle the new return signature. Print a warning on error, matching the pattern used by `ensureRookGitignore`.

### `internal/cli/agentmd_cmd.go` (new)

- `newAgentMDCmd()` returns a `*cobra.Command` for `rook agentmd [workspace]`.
- Resolves workspace (by name or current directory), loads manifest from `rook.yaml`, calls `ensureAgentMDRookSection`.
- Prints "Updated rook section in <filename>" or "Added rook section to <filename>" depending on whether tags existed.
- To support this feedback, `ensureAgentMDRookSection` should return `(action string, err error)` instead of being silent.

### `internal/cli/root.go`

- Register `newAgentMDCmd()` as a subcommand.

### Test changes in `internal/cli/init_test.go`

- **Update** `TestEnsureAgentMD_SkipsIfTagExists` → rename to `TestEnsureAgentMD_ReplacesExistingSection`, verify that old content between tags is replaced with new content reflecting current services.
- **Add** `TestEnsureAgentMD_ReplacesWithDifferentServices` — tags exist with service A, manifest has service B, verify replacement shows only service B.
- **Add** `TestEnsureAgentMD_PreservesContentOutsideTags` — content before and after the rook section is preserved exactly, including when tags are at the very start of the file.
- **Add** `TestEnsureAgentMD_ErrorsOnMissingClosingTag` — `<!-- rook -->` without `<!-- /rook -->` returns an error.
- **Add** `TestAgentMDCmd` — integration test for the CLI command.

## Behavior Matrix

| Scenario | Before | After |
|----------|--------|-------|
| No CLAUDE.md/AGENTS.md | No-op | No-op |
| File exists, no tags | Append section | Append section |
| File exists, tags present | Skip | Replace between tags |
| File exists, opening tag without closing | Skip | Error |

## Scope Exclusions

- No creation of CLAUDE.md/AGENTS.md if neither exists (existing behavior, unchanged).
- No interactive confirmation before overwriting — content between tags is fully rook-owned.
- Everything between `<!-- rook -->` and `<!-- /rook -->` is replaced without preservation.
