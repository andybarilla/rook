# Process env_file Support

## Problem

When a service is discovered from docker-compose with an `env_file` directive, container services correctly pass `--env-file` to the container runtime. Process services ignore `env_file` entirely — the variables from the file are never loaded into the process environment.

## Solution

Load and resolve env_file variables for process services during the `up` command's template resolution phase, merging them into `svc.Environment` before the orchestrator starts services. This keeps `ProcessRunner` unchanged — it continues to use `svc.Environment` as it does today.

## Design

### New function: `envgen.ParseEnvFile(path string) (map[string]string, error)`

Reads a `.env` file and returns key-value pairs.

**Supported syntax:**
- `KEY=VALUE` — basic form, split on first `=`
- Blank lines and `# comment` lines — skipped
- `"quoted values"` and `'quoted values'` — surrounding quotes stripped
- `export KEY=VALUE` — `export ` prefix stripped
- Inline comments (`KEY=value # comment`) — **not supported** (value includes everything after `=`)

**Error handling:**
- Returns error if file doesn't exist or can't be read
- Lines without `=` are skipped (no error)

### Changes to `internal/cli/up.go`

In the template resolution section, after resolving inline `Environment` for process services:

1. If `svc.EnvFile` is set and `svc.IsProcess()`:
   - Call `envgen.ParseEnvFile()` to read the file
   - Run parsed values through `envgen.ExpandShellVars()`
   - Run parsed values through `envgen.ResolveTemplates()` with the same host/port context used for inline vars
   - Merge into `svc.Environment` — **inline values take precedence** over env_file values (matching docker-compose behavior)

### No changes needed

- **Discovery** — already parses `env_file` from compose
- **ProcessRunner** — continues using `svc.Environment` unchanged
- **DockerRunner** — handles `env_file` independently via `--env-file` flag
- **Orchestrator** — no env_file awareness needed

### Precedence order (lowest to highest)

1. Variables from `env_file`
2. Inline `environment` variables from the manifest

### Testing

- `envgen.ParseEnvFile()`: unit tests for all supported syntax (basic, comments, blank lines, quotes, export prefix, no-equals lines)
- `internal/cli/up.go`: integration-level test or verification that process services with env_file get the variables merged into their environment
