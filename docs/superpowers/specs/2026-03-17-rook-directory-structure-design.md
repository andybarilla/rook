# Rook Directory Structure Refactor

## Problem

The `.rook/` directory currently mixes user-owned files with generated/cache files. Users need an easy way to know what to gitignore, and the current structure requires ignoring the entire `.rook/` directory or manually tracking which files are generated.

## Goals

- Separate user-owned files from generated/cache files
- Provide a simple gitignore solution (one line)
- Maintain backwards compatibility during transition (not required for this change since rook is early-stage)

## Proposed Structure

```
.rook/
├── scripts/           # User files (checked into git)
│   └── post-create.sh # Devcontainer scripts copied during init
└── .cache/            # Generated files (gitignored)
    ├── build-cache.json
    └── resolved/
        └── api.env
```

## Behavior Changes

### `rook init`
- Devcontainer scripts now copy to `.rook/scripts/<script>` instead of `.rook/<script>`
- Generate `.rook/.gitignore` with contents: `.cache/`
- Warning message updated to reference new path: `.rook/scripts/<script>`

### `rook up`
- Write resolved env files to `.rook/.cache/resolved/<service>.env`
- Read/write build cache at `.rook/.cache/build-cache.json`
- Create `.cache/` directory if it doesn't exist

### `rook check-builds`
- Read build cache at `.rook/.cache/build-cache.json`

## Files to Modify

1. `internal/cli/init.go` — Update script copy path, generate `.rook/.gitignore`
2. `internal/cli/up.go` — Update resolved dir and build-cache paths
3. `internal/cli/check_builds.go` — Update build-cache path
4. `docs/rook-project-blurb.md` — Update documentation to mention `.rook/.cache/`

## Gitignore Strategy

Generate `.rook/.gitignore` during `rook init`:

```
.cache/
```

This keeps the gitignore rule close to the affected files and doesn't clutter the root `.gitignore`.

## Edge Cases

- **Existing `.rook/` directory**: If `.rook/` exists with old structure, continue to work. New files go to new locations. Old files remain but are no longer updated.
- **Missing `.cache/` directory**: Create on demand (already done for resolved files)
- **Missing scripts directory**: Create when first script is copied

## Success Criteria

- `rook init` creates `.rook/.gitignore` with `.cache/` entry
- `rook up` writes to `.rook/.cache/resolved/` and `.rook/.cache/build-cache.json`
- `rook check-builds` reads from new location
- Scripts copy to `.rook/scripts/`
- User only needs to commit `.rook/scripts/` and `.rook/.gitignore`
