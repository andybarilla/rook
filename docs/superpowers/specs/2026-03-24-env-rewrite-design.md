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
2. Find all services whose `env_file` contains `VAR_NAME`.
3. If no services have it, error.
4. If `SERVICE_NAME` doesn't exist in the workspace, error.
5. Read the var's value from the `.env` file.
6. If the target service has multiple ports, prompt the user to pick which one.
7. Detect the value type and rewrite:
   - **URL** (contains `://`): parse with `net/url`, replace host and port in the original string
   - **Host:Port** (matches `hostname:digits`): replace both components
   - **Bare host** (localhost, 127.0.0.1, 0.0.0.0, IP addresses): replace with `{{.Host.service}}`
   - **Bare port** (numeric string): replace with `{{.Port.service}}`
   - If no host or port is detected, error with the raw value shown.
8. For each service whose `env_file` contains the variable, add the rewritten value to `svc.Environment[VAR_NAME]` in the manifest.
9. Write the updated manifest back to `rook.yaml`.
10. Print what was done, e.g.: `app: DATABASE_URL = postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db`

## Value Detection & Rewriting

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

- Matches a hostname pattern (localhost, 127.0.0.1, 0.0.0.0, IP address): replace with `{{.Host.service}}`
- Is a numeric string: replace with `{{.Port.service}}`

## Multi-Port Disambiguation

When the target service exposes multiple ports, prompt interactively:

```
Service "app" exposes multiple ports: 8080, 3000
Which port does API_URL use?
  [1] 8080
  [2] 3000
>
```

Single-port services skip the prompt.

## Where the Rewritten Value Goes

The rewritten value is added to the `environment` map of every service in `rook.yaml` whose `env_file` contains the variable. At runtime, inline environment values override env_file values, so the template-resolved value takes precedence while the original `.env` stays untouched.

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
- Modify `internal/cli/env.go` — restructure as parent command with `env` (print) and `env rewrite` as subcommands
- Uses existing: `envgen.ParseEnvFile()`, `workspace.ParseManifest()`, `workspace.WriteManifest()`
- The rewrite logic is a pure function: `Rewrite(value string, serviceName string, port int) (string, error)` — takes the raw value and returns the rewritten string. All detection logic lives here.
