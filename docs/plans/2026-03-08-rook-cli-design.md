# rook-cli Design

CLI interface for Rook — enables developer convenience and scripting automation without the GUI.

## Goals

- Quick commands for common tasks (`rook add .`, `rook list`, `rook start mysql`)
- Scriptable output for automation (CI, dotfiles, shell aliases)
- Single binary: no args launches GUI, subcommands run CLI

## Non-Goals (Phase 1)

- IPC socket to communicate with running GUI (Phase 2)
- Daemon mode
- Plugin management commands

## Command Structure

| Command | Description | Example |
|---------|-------------|---------|
| `rook list` | List all sites | `rook list` |
| `rook add <path>` | Add a site (infer domain) | `rook add .` |
| `rook remove <domain>` | Remove a site | `rook remove myapp.test` |
| `rook status` | Show all service statuses | `rook status` |
| `rook start <service>` | Start a database service | `rook start mysql` |
| `rook stop <service>` | Stop a database service | `rook stop redis` |

### Flags

- `rook add` accepts `--domain`, `--php`, `--node`, `--tls` to override defaults
- Global `--json` flag forces JSON output

### CLI Framework

Cobra — the Go standard for subcommand CLIs. Handles subcommands, flags, help generation.

## Output Formatting

Auto-detect TTY:
- Terminal → human-readable tables with color
- Piped or `--json` → JSON output

Human output:

```
$ rook list
DOMAIN          PATH                      PHP    NODE   TLS
myapp.test      /home/user/projects/myapp  8.3           ✓
api.test        /home/user/projects/api    8.2    20     ✓

$ rook status
SERVICE     STATUS
mysql       running
postgresql  stopped
redis       running

$ rook add .
✓ Site "myapp.test" added (path: /home/user/projects/myapp)
```

JSON output:

```
$ rook list --json
[{"domain":"myapp.test","path":"/home/user/projects/myapp","php":"8.3","node":"","tls":true}]

$ rook add . --json
{"domain":"myapp.test","path":"/home/user/projects/myapp"}
```

Errors go to stderr. Exit code 0 for success, 1 for errors.

## Architecture

### Binary Dispatch

```go
main() {
    if len(os.Args) > 1 {
        cli.Execute()  // CLI mode
    } else {
        runGUI()       // Wails GUI (current behavior)
    }
}
```

### Package Structure

- `internal/cli/root.go` — root command, global flags (`--json`)
- `internal/cli/sites.go` — `list`, `add`, `remove` commands
- `internal/cli/services.go` — `status`, `start`, `stop` commands
- `internal/cli/output.go` — TTY detection, table/JSON formatting

Each command instantiates `core.Core` with the same config as `app.startup`, calls the relevant method, formats output, and exits.

### Core Reuse

The CLI reuses `internal/core.Core` directly — the same business logic layer the GUI uses. No duplication of site management, plugin orchestration, or service control.

## GUI Coexistence

### Phase 1 (This Implementation)

CLI always runs standalone, instantiating its own Core. File locking on the registry prevents corrupt writes if both GUI and CLI run simultaneously.

File locking: add `rook(2)` advisory locking around registry reads/writes in `registry.go`. GUI picks up CLI changes on next registry read.

### Phase 2 (Future)

- GUI writes a lock file (`~/.local/share/rook/rook.lock`) with PID on startup
- GUI exposes a Unix socket (`~/.local/share/rook/rook.sock`)
- CLI checks lock file: if PID alive, connects via socket; otherwise runs standalone

## Testing

**Unit tests:**
- Command output formatting (table and JSON modes)
- TTY detection logic
- Argument parsing and flag handling

**Integration tests:**
- Commands against real Core with temp config directory
- `add` + `list` round-trip
- `remove` on nonexistent domain returns proper error

**File locking tests:**
- Concurrent writes don't corrupt the registry

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- No other new dependencies (Core, registry, config already exist)
