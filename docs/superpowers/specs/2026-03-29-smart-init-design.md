# Smart Init Design

## Problem

`rook init` today is a one-shot, non-interactive process that:

1. Searches a fixed list of compose file paths, takes the first match, and ignores the rest
2. Treats devcontainer Dockerfiles as opaque single-service blobs
3. Cannot discover local application services (the node/go/python thing you're actually developing)
4. Never asks the user anything — it guesses and writes `rook.yaml`

This works for simple repos with a single `docker-compose.yml` of infrastructure services. It falls short for repos with multiple compose files, devcontainer setups that bundle several concerns into one Dockerfile, or repos where the primary service is a local process that isn't described in any compose file.

## Goals

- Discover more: find all compose files, not just the first match
- Be interactive: show what was found, ask which sources to use, let the user refine
- Help add local services: detect repo languages/frameworks and help scaffold `command`-based services
- Smarter devcontainer handling: analyze Dockerfiles to understand what services they represent
- Optional LLM assistance: for ambiguous cases (Dockerfile analysis, repo structure inference)

## Design

### Phase 1: Multi-source discovery and interactive selection

#### Find all compose files

Replace the current "first match wins" logic with a scan that finds all compose-like files:

```
docker-compose.yml, docker-compose.yaml
compose.yml, compose.yaml
.devcontainer/docker-compose.yml, .devcontainer/docker-compose.yaml
docker-compose.*.yml (e.g., docker-compose.dev.yml, docker-compose.override.yml)
```

Each file is parsed independently. The user is shown a summary of what was found in each and asked which to base the manifest on. Multiple can be selected and merged.

#### Interactive flow

```
$ rook init .

Found compose files:
  1. docker-compose.yml (postgres, redis, nginx)
  2. docker-compose.dev.yml (app — build from ./Dockerfile)
  3. .devcontainer/docker-compose.yml (app, postgres)

Also detected:
  - .devcontainer/devcontainer.json
  - mise.toml (node 22, go 1.22)
  - go.mod (Go module: github.com/example/myapp)
  - package.json (name: myapp-frontend, scripts: dev, build, test)

Which compose file(s) should rook use? [1,2,3 or none]: 1

Add local services? These are the application services you develop locally.
  Detected: Go module (go.mod), Node project (package.json)
  [a]dd a service, [s]kip: a

Service name: api
  Command (how to run it): go run ./cmd/api
  Depends on (comma-separated, or empty): postgres, redis

Service name: frontend
  Command: npm run dev
  Depends on: api

[a]dd another, [d]one: d

Generated rook.yaml with 4 services: postgres, redis, api, frontend
```

#### Non-interactive mode

`rook init --non-interactive` preserves the current behavior: first compose file wins, no prompts. This keeps scripted/CI usage working.

### Phase 2: Local service detection

Scan the repo for signals that indicate runnable services:

| Signal | Inference |
|--------|-----------|
| `go.mod` | Go service — suggest `go run ./cmd/<name>` |
| `package.json` with `scripts.dev` | Node service — suggest `npm run dev` |
| `package.json` with `scripts.start` | Node service — suggest `npm start` |
| `Makefile` with `dev` or `run` target | Suggest `make dev` or `make run` |
| `Procfile` | Parse process types directly |
| `pyproject.toml` / `setup.py` | Python service |
| `Cargo.toml` | Rust service |

These are presented as suggestions the user can accept, modify, or skip. The detection is heuristic — it proposes, the user decides.

For Go modules with multiple `cmd/` directories, each is suggested as a separate service.

### Phase 3: Devcontainer Dockerfile analysis

Many devcontainer Dockerfiles bundle multiple concerns:

- Install system dependencies (postgres client, redis tools)
- Install language runtimes
- Copy application code and build
- Run multiple processes via a start script

#### Without LLM

Parse the Dockerfile for structured signals:

- `EXPOSE` directives → port mappings
- `apt-get install` / `apk add` of known packages → infer services (e.g., `postgresql-client` suggests a postgres dependency)
- Multi-stage builds → each named stage could be a service
- `CMD` / `ENTRYPOINT` → primary service command
- Start scripts (already parsed by sanitizer) → multiple `make`/`npm`/`go run` commands suggest multiple services

#### With LLM

For Dockerfiles that resist structured parsing, offer an LLM-assisted analysis:

```
$ rook init .

Found .devcontainer/Dockerfile (complex — 87 lines)
  Analyze with LLM to suggest service breakdown? [y/n]: y

  LLM analysis:
    This Dockerfile sets up:
    - Go 1.22 development environment
    - PostgreSQL 16 client tools (suggests postgres dependency)
    - Node 22 for frontend tooling

    Suggested services:
    1. postgres (image: postgres:16, from apt-get install context)
    2. api (command: go run ./cmd/api, from start.sh analysis)
    3. frontend (command: npm run dev, from start.sh analysis)

  Accept suggestions? [y]es, [e]dit, [s]kip: y
```

The LLM receives: the Dockerfile content, any start scripts found, the repo's file tree (top 2 levels), and any compose files for cross-reference. It returns structured JSON with suggested services.

### Phase 4: LLM integration architecture

#### Provider abstraction

```go
// internal/llm/llm.go
type Provider interface {
    Complete(ctx context.Context, req Request) (Response, error)
}

type Request struct {
    System string
    Prompt string
}

type Response struct {
    Content string
}
```

Initial implementation: Anthropic API (`claude-haiku-4-5-20251001` for cost/speed). Provider selected via `ROOK_LLM_PROVIDER` env var or `~/.config/rook/settings.json`. No API key bundled — user provides their own or skips LLM features.

#### LLM-assisted operations in init

1. **Dockerfile analysis**: "Given this Dockerfile and start script, what services does this represent? Return JSON."
2. **Repo structure inference**: "Given this file tree and these config files, what runnable services exist? Return JSON."
3. **Command suggestion**: "Given this go.mod/package.json/Makefile, what's the likely dev command?"

All LLM calls are optional — the user is always asked before any call is made, and every suggestion can be accepted, edited, or skipped.

#### Structured output

LLM responses are parsed as JSON matching a defined schema:

```go
type LLMServiceSuggestion struct {
    Name        string            `json:"name"`
    Type        string            `json:"type"` // "container" or "process"
    Image       string            `json:"image,omitempty"`
    Command     string            `json:"command,omitempty"`
    Ports       []int             `json:"ports,omitempty"`
    DependsOn   []string          `json:"depends_on,omitempty"`
    Reasoning   string            `json:"reasoning"`
}
```

If the LLM returns unparseable output, fall back to showing the raw text and asking the user to add services manually.

## Non-goals

- Rewriting existing `rook.yaml` files (init is for new workspaces only; editing is the manifest editor's job)
- Auto-running `rook up` after init
- Supporting LLM providers other than Anthropic in the initial implementation
- Generating Dockerfiles or compose files

## Migration

Existing `rook init` behavior is preserved behind `--non-interactive`. The new interactive flow becomes the default. If stdin is not a terminal, non-interactive mode is used automatically.

## Decisions

1. **LLM accuracy**: Start with Haiku, evaluate against real-world Dockerfiles, upgrade model if needed.
2. **Repo scanning depth**: Top-level only. No recursion into subdirectories (monorepo support can come later).
3. **Merge conflicts**: When the user selects multiple compose files and service names collide, prompt the user to pick which definition wins for each conflict.
4. **Re-running init**: Two flags:
   - `--force`: Blow away existing `rook.yaml` and redo the full init process
   - `--add`: Keep existing services, run discovery/interactive flow to add new ones incrementally
