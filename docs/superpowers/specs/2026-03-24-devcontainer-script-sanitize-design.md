# Devcontainer Script Sanitization

## Problem

When `rook init` copies devcontainer scripts to `.rook/scripts/`, the copies are verbatim. Devcontainer scripts commonly contain patterns that are harmful under rook:

- **Wait loops** that block on devcontainer-specific markers (e.g., `while [ ! -f /tmp/.devcontainer-ready ]; do sleep 1; done`) — these block forever since the marker is never created outside devcontainers.
- **Keep-alive commands** (`sleep infinity`, `tail -f /dev/null`) — unnecessary under rook's container management and prevent the script from completing.
- **Backgrounded commands** (`make dev-servers &`) — only needed when followed by a keep-alive; without the keep-alive the script should run the command in the foreground.

## Design

### New function: `SanitizeScript`

Location: `internal/discovery/sanitize.go`

```go
type ScriptChange struct {
    Description string // human-readable description of what was removed/changed
}

func SanitizeScript(content []byte) ([]byte, []ScriptChange)
```

Takes raw script content, returns sanitized content and a list of changes made. Never returns an error — worst case it returns the input unchanged with no changes.

### Sanitization rules (applied in order)

1. **Remove wait loops**: Match single-line `while` loops whose body is only `sleep`, ending with `done`. Only the `while [...]; do` / `while ...; do` single-line form is matched. Contiguous comment lines (`#`) and `echo` lines immediately above the `while` keyword are removed with it.

2. **Remove keep-alive commands**: Lines matching `exec sleep infinity`, `sleep infinity`, `exec tail -f /dev/null`, `tail -f /dev/null`. Contiguous comment lines immediately above are removed with it.

3. **Strip trailing `&`**: Only applied when Rule 2 removed at least one keep-alive. All remaining command lines ending with ` &` have the ` &` stripped. Comment lines containing "in the background" have that phrase removed.

4. **Collapse blank lines**: Runs of 2+ blank lines become a single blank line. Trailing blank lines at end of file removed.

Each rule that fires appends a `ScriptChange` describing what happened.

### Integration in `init.go`

Between reading the script content and writing it to `.rook/scripts/`, call `SanitizeScript`. If changes were made:

- Print each change: `  Sanitized .rook/scripts/<name>: <description>`
- Replace the generic "Review" warning with: `"Verify .rook/scripts/%s — devcontainer patterns were automatically removed"`

If no changes were made, keep the existing warning as-is.

### Example transformation

Input:
```bash
#!/bin/bash
cd /workspaces/emrai

# Wait for post-create.sh to finish (it creates this marker file)
echo "Waiting for post-create to finish..."
while [ ! -f /tmp/.devcontainer-ready ]; do
  sleep 1
done

# Start dev servers in the background
make dev-servers &

# Keep the container alive
exec sleep infinity
```

Output:
```bash
#!/bin/bash
cd /workspaces/emrai

# Start dev servers
make dev-servers
```

Changes reported:
- "Removed wait loop (while [ ! -f /tmp/.devcontainer-ready ])"
- "Removed keep-alive command (exec sleep infinity)"
- "Removed background operator from 'make dev-servers'"

## Testing

- Script with all three patterns (wait loop + backgrounded command + sleep infinity) → all removed
- Script with only a keep-alive → keep-alive removed, other content preserved
- Script with no devcontainer patterns → returned unchanged, no changes reported
- Script with multiple wait loops → all removed
- Script where removal leaves effectively empty script (just shebang + cd) → returned as-is
- Comment association: comments immediately above removed blocks are removed; unrelated comments are preserved
- Trailing `&` only stripped when a keep-alive was also removed (if no keep-alive present, `&` is intentional)

## Scope

- Two new files: `internal/discovery/sanitize.go`, `internal/discovery/sanitize_test.go`
- One modified file: `internal/cli/init.go` (call site)
