# `rook env rewrite` Design Spec

## Overview

A new CLI subcommand that reads an environment variable from a workspace's `.env` file, replaces host and port values with rook template tags (`{{.Host.x}}`/`{{.Port.x}}`), and writes the rewritten value into `rook.yaml`'s `environment` block. This keeps the original `.env` file untouched — rook's inline environment overrides at runtime.

## Command

```
rook env rewrite <VAR_NAME> <SERVICE_NAME> [workspace]
```

- `VAR_NAME` — the env var to rewrite (e.g., `DATABASE_URL`)
- `SERVICE_NAME` — the rook service whose host/port template tags to use (e.g., `postgres`)
- `workspace` — optional, inferred from cwd if omitted

## Behavior

1. Resolve the workspace (from arg or cwd).
2. Validate `SERVICE_NAME` exists in the workspace; error if not.
3. Find all services whose `env_file` contains `VAR_NAME`. The `env_file` path is resolved relative to the workspace root directory.
4. If no services have it, error.
5. Read the var's value from the `.env` file.
6. Detect the value type and rewrite (see Value Detection below).
7. For each service whose `env_file` contains the variable, set `svc.Environment[VAR_NAME]` to the rewritten value in the manifest. Initialize the map if nil. Overwrites silently if the key already exists.
8. Write the updated manifest back to `rook.yaml`.
9. Print what was done, e.g.: `app: DATABASE_URL = postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db`

The command is idempotent — running it twice with the same arguments produces the same result.

## Value Detection & Rewriting

The rewrite function is pure: `Rewrite(value string, serviceName string) (string, error)`. It produces template tags, not resolved values.

Three cases, checked in order:

### URL (value contains `://`)

Parse with `net/url`. Extract host and port from the authority section. Replace via string substitution on the original value (preserving scheme, userinfo, path, query, fragment).

```
postgres://user:pass@localhost:5432/db
→ postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db
```

If the URL has a host but no port, only replace the host. If it has a port but no recognizable host (unlikely), only replace the port.

### Host:Port pair

If the value matches a `hostname:digits` pattern (not a URL), replace both.

```
localhost:5432 → {{.Host.postgres}}:{{.Port.postgres}}
```

### Bare value

- Matches a hostname pattern (localhost, 127.0.0.1, 0.0.0.0, or any IPv4 address): replace with `{{.Host.service}}`
- Is a numeric string: replace with `{{.Port.service}}`

Note: Docker-compose service names used as hostnames (e.g., `postgres`, `db`) are not detected as bare hosts — the user should use the URL or host:port forms for those cases, which will match on the port component.

## Where the Rewritten Value Goes

The rewritten value is added to the `environment` map of every service in `rook.yaml` whose `env_file` contains the variable. At runtime, inline environment values override env_file values, so the template-resolved value takes precedence while the original `.env` stays untouched.

## CLI Restructuring

The existing `rook env [workspace]` command becomes a parent command with two children:

- `rook env [workspace]` — still works as before (default action prints resolved env vars, using Cobra's `RunE` on the parent)
- `rook env rewrite <VAR_NAME> <SERVICE_NAME> [workspace]` — new subcommand

No breaking change to existing usage.

## YAML Formatting

`WriteManifest` uses `yaml.Marshal` which rewrites the entire file. This may lose comments and custom formatting from hand-edited `rook.yaml` files. This is an existing limitation. A future improvement could use `yaml.Node` for targeted edits, but that's out of scope for this feature.

## Error Cases

| Condition | Behavior |
|-----------|----------|
| Var not found in any service's env_file | Error: `"DATABASE_URL" not found in any service's env_file` |
| Service name doesn't exist in workspace | Error: `service "redis" not found in workspace "myapp"` |
| No detectable host or port in value | Error: `cannot detect host or port in value "some_string"` |
| No services have env_file set | Error: `no services in workspace "myapp" have an env_file` |

## Implementation Notes

- New file: `internal/cli/env_rewrite.go` — command definition
- New file: `internal/envgen/rewrite.go` — value detection and rewriting logic (pure function, testable)
- Modify `internal/cli/env.go` — add `rewrite` subcommand while keeping existing behavior as default
- Uses existing: `envgen.ParseEnvFile()`, `workspace.ParseManifest()`, `workspace.WriteManifest()`
- The rewrite logic is a pure function: `Rewrite(value string, serviceName string) (string, error)` — takes the raw value and returns the rewritten string with template tags. All detection logic lives here.
- Initialize `svc.Environment` map before writing if nil.
