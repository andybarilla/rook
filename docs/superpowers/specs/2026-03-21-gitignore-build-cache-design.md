# Respect .gitignore in Build Cache

## Problem

The build cache context-file hashing walks the entire build context directory and hashes every file not excluded by `.dockerignore`. Files that are git-ignored (e.g., `node_modules/`, `dist/`, build artifacts, local config) but not docker-ignored trigger false-positive stale build detections. This causes unnecessary rebuild prompts on `rook up`.

## Solution

Merge `.gitignore` patterns into the exclusion list alongside `.dockerignore` patterns. Both sets of patterns are additive — a file excluded by either source is skipped during the context walk.

## Design

### New functions in `internal/buildcache/`

**`ParseGitignore(dir string) ([]string, error)`**
- Reads `.gitignore` from the given directory
- Returns empty slice if file doesn't exist
- Same parsing logic as `ParseDockerignore` (skip blank lines and comments)

**`CollectIgnorePatterns(buildCtx, workDir string) ([]string, error)`**
- Combines patterns from all sources in order:
  1. Default exclusions (`.rook/`, `.git/`)
  2. `.dockerignore` from build context
  3. `.gitignore` from build context
  4. `.gitignore` from workspace root (if different from build context)
- When `buildCtx == workDir`, reads `.gitignore` once to avoid duplication

### Changes to existing code

- `DetectStale`: replace `ParseDockerignore(buildCtx)` call with `CollectIgnorePatterns(buildCtx, workDir)`
- `UpdateAfterBuild`: replace `ParseDockerignore(buildCtx)` call with `CollectIgnorePatterns(buildCtx, workDir)`. Requires adding `workDir` parameter (already available at call sites).

### No changes needed

- `MatchesPatterns` — `.gitignore` uses the same glob syntax as `.dockerignore`, so the existing `patternmatcher` handles both
- `ParseDockerignore` — kept as-is, called internally by `CollectIgnorePatterns`
- `HashFile`, `Cache`, `ServiceCache` — unchanged

## File changes

| File | Change |
|------|--------|
| `internal/buildcache/dockerignore.go` | Rename to `ignore.go`. Add `ParseGitignore`, `CollectIgnorePatterns` |
| `internal/buildcache/dockerignore_test.go` | Rename to `ignore_test.go`. Add tests for new functions |
| `internal/buildcache/detect.go` | Replace `ParseDockerignore` with `CollectIgnorePatterns` |
| `internal/buildcache/cache.go` | `UpdateAfterBuild` signature adds `workDir`, replaces `ParseDockerignore` with `CollectIgnorePatterns` |
| Call sites of `UpdateAfterBuild` | Pass `workDir` argument |

## Edge cases

- **No `.gitignore` exists**: no-op, identical to current behavior
- **Negation patterns** (`!important.log`): already supported by `patternmatcher`
- **Build context == workspace root**: `.gitignore` read once, not duplicated
- **Nested `.gitignore` files**: not supported (only root-level). This handles the 99% case; nested gitignores in build contexts are rare.

## Testing

1. `ParseGitignore` reads patterns correctly, returns empty on missing file
2. `CollectIgnorePatterns` merges dockerignore + gitignore, deduplicates when `buildCtx == workDir`
3. `DetectStale` ignores git-ignored files (file in `node_modules/` changes → no rebuild)
4. `UpdateAfterBuild` excludes git-ignored files from the cache
5. Existing `.dockerignore` tests continue to pass
