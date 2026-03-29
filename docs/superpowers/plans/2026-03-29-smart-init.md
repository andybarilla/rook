# Smart Init Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the non-interactive `rook init` with a multi-source, interactive init that discovers more, asks questions, helps add local services, and optionally uses LLM analysis for ambiguous setups.

**Spec:** `docs/superpowers/specs/2026-03-29-smart-init-design.md`

**Tech Stack:** Go, Cobra, stdlib `testing`, `bufio` (interactive prompts), `net/http` (LLM API calls)

---

### File Map

**Create:**
- `internal/discovery/scanner.go` — find all compose files in a directory
- `internal/discovery/scanner_test.go`
- `internal/discovery/local.go` — detect local service signals (go.mod, package.json, etc.)
- `internal/discovery/local_test.go`
- `internal/discovery/dockerfile.go` — parse Dockerfile for structured signals
- `internal/discovery/dockerfile_test.go`
- `internal/prompt/prompt.go` — interactive prompt helpers (select, confirm, input)
- `internal/prompt/prompt_test.go`
- `internal/llm/llm.go` — Provider interface, Request/Response types
- `internal/llm/anthropic.go` — Anthropic API implementation
- `internal/llm/anthropic_test.go`
- `internal/llm/analyze.go` — structured prompts for Dockerfile/repo analysis
- `internal/llm/analyze_test.go`

**Modify:**
- `internal/discovery/discovery.go` — new `ScanResult` type for multi-source results
- `internal/discovery/compose.go` — accept a specific file path instead of searching
- `internal/cli/init.go` — rewrite to interactive flow, add `--non-interactive`, `--force`, `--add` flags

---

### Task 1: Compose file scanner

Find all compose files in a directory instead of taking the first match. This is the foundation for letting the user choose.

**Files:**
- Create: `internal/discovery/scanner_test.go`
- Create: `internal/discovery/scanner.go`

- [ ] **Step 1: Write failing test**

```go
package discovery_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/discovery"
)

func TestScanComposeFiles(t *testing.T) {
	t.Run("finds_multiple_compose_files", func(t *testing.T) {
		dir := t.TempDir()
		// Create several compose files
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  postgres:\n    image: postgres:16\n"), 0644)
		os.WriteFile(filepath.Join(dir, "docker-compose.dev.yml"), []byte("services:\n  app:\n    build: .\n"), 0644)
		os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0755)
		os.WriteFile(filepath.Join(dir, ".devcontainer", "docker-compose.yml"), []byte("services:\n  app:\n    build:\n      context: ..\n"), 0644)

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 3 {
			t.Fatalf("expected 3 compose files, got %d", len(results))
		}
	})

	t.Run("returns_service_names_per_file", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services:\n  postgres:\n    image: postgres:16\n  redis:\n    image: redis:7\n"), 0644)

		results := discovery.ScanComposeFiles(dir)

		if len(results) != 1 {
			t.Fatalf("expected 1 compose file, got %d", len(results))
		}
		if len(results[0].ServiceNames) != 2 {
			t.Fatalf("expected 2 service names, got %d", len(results[0].ServiceNames))
		}
	})

	t.Run("returns_empty_for_no_compose_files", func(t *testing.T) {
		dir := t.TempDir()
		results := discovery.ScanComposeFiles(dir)
		if len(results) != 0 {
			t.Fatalf("expected 0 compose files, got %d", len(results))
		}
	})
}
```

- [ ] **Step 2: Implement ScanComposeFiles**

```go
// ComposeFileInfo summarizes a compose file found during scanning.
type ComposeFileInfo struct {
	Path         string   // absolute path
	RelPath      string   // relative to project dir
	ServiceNames []string // service names found in the file
}

// ScanComposeFiles finds all compose files in dir and returns a summary of each.
func ScanComposeFiles(dir string) []ComposeFileInfo { ... }
```

Search for: fixed names (`docker-compose.yml`, `docker-compose.yaml`, `compose.yml`, `compose.yaml`), glob `docker-compose.*.yml`/`.yaml`, and `.devcontainer/docker-compose.{yml,yaml}`. Parse each just enough to extract service names (unmarshal to get `services` keys). Sort results: `.devcontainer/` files first, then root files alphabetically.

- [ ] **Step 3: Run tests, verify pass**

Run: `go test ./internal/discovery/ -run TestScanComposeFiles -v`

- [ ] **Step 4: Commit**

---

### Task 2: Refactor ComposeDiscoverer to accept a file path

Currently `Discover()` searches for the first compose file. Refactor so it can parse a specific file, enabling the interactive flow to pass the user's chosen file(s).

**Files:**
- Modify: `internal/discovery/compose.go`
- Modify: existing compose tests

- [ ] **Step 1: Add `DiscoverFile(dir, filePath string)` method**

Extract the core parsing logic from `Discover` into `DiscoverFile(dir, composePath string) (*DiscoveryResult, error)` where `dir` is the project root and `composePath` is the specific compose file to parse. The existing `Discover` method calls `DiscoverFile` with the first match (preserving current behavior).

- [ ] **Step 2: Run existing tests to verify no regression**

Run: `go test ./internal/discovery/ -v`

- [ ] **Step 3: Write test for DiscoverFile with explicit path**

Test that calling `DiscoverFile` with a specific path parses that file regardless of what other compose files exist.

- [ ] **Step 4: Commit**

---

### Task 3: Interactive prompt helpers

Build a small prompt library for the interactive init flow. These are thin wrappers around `bufio.Scanner` reading from stdin.

**Files:**
- Create: `internal/prompt/prompt.go`
- Create: `internal/prompt/prompt_test.go`

- [ ] **Step 1: Define the Prompter interface and types**

```go
// Prompter handles interactive user prompts.
// The interface exists so init can be tested with a mock.
type Prompter interface {
	// Select shows numbered options and returns selected indices (1-based input, 0-based output).
	// Empty input returns nil (skip). Accepts comma-separated numbers.
	Select(message string, options []string) ([]int, error)
	// Confirm asks a yes/no question. Returns true for y/yes.
	Confirm(message string, defaultYes bool) (bool, error)
	// Input asks for free-form text. Returns empty string if skipped.
	Input(message string, defaultValue string) (string, error)
	// InputList asks for comma-separated values. Returns nil if skipped.
	InputList(message string) ([]string, error)
}
```

- [ ] **Step 2: Implement `StdinPrompter`**

Reads from a `bufio.Scanner` wrapping `os.Stdin`. Each method prints the prompt to a provided `io.Writer` (stdout), reads one line, parses it.

- [ ] **Step 3: Write tests using a `bufio.Scanner` over a `strings.Reader`**

Test `Select` with single choice, multiple choices, empty input. Test `Confirm` with y/n/default. Test `Input` with value and empty.

- [ ] **Step 4: Commit**

---

### Task 4: Local service detection

Scan the repo root for signals that indicate runnable application services.

**Files:**
- Create: `internal/discovery/local_test.go`
- Create: `internal/discovery/local.go`

- [ ] **Step 1: Define types and write failing tests**

```go
// LocalSignal describes a detected local service signal.
type LocalSignal struct {
	Type       string // "go", "node", "python", "rust", "makefile", "procfile"
	File       string // the file that triggered detection
	Name       string // suggested service name
	Command    string // suggested run command
}

// ScanLocalSignals looks for language/framework files at the top level of dir.
func ScanLocalSignals(dir string) []LocalSignal { ... }
```

Tests:
- Directory with `go.mod` and `cmd/api/main.go` → Go signal suggesting `go run ./cmd/api`
- Directory with `go.mod` and multiple `cmd/` dirs → one signal per cmd dir
- Directory with `package.json` containing `scripts.dev` → Node signal suggesting `npm run dev`
- Directory with `package.json` containing `scripts.start` but no `dev` → suggests `npm start`
- Directory with `Makefile` containing `dev:` target → suggests `make dev`
- Directory with `Procfile` → one signal per process type
- Directory with none of these → empty slice
- `pyproject.toml` → Python signal
- `Cargo.toml` → Rust signal

- [ ] **Step 2: Implement ScanLocalSignals**

Check for each file type at the top level only. For Go, also scan `cmd/` subdirectories. For Node, parse `package.json` to read `scripts`. For Makefile, scan for `dev:` or `run:` targets. For Procfile, parse `name: command` lines.

- [ ] **Step 3: Run tests, verify pass**

Run: `go test ./internal/discovery/ -run TestScanLocalSignals -v`

- [ ] **Step 4: Commit**

---

### Task 5: Dockerfile signal parser

Extract structured information from Dockerfiles without an LLM. This provides basic analysis for the non-LLM path and context for the LLM path.

**Files:**
- Create: `internal/discovery/dockerfile_test.go`
- Create: `internal/discovery/dockerfile.go`

- [ ] **Step 1: Define types and write failing tests**

```go
// DockerfileSignals holds structured information extracted from a Dockerfile.
type DockerfileSignals struct {
	ExposedPorts   []int             // from EXPOSE directives
	AptPackages    []string          // from apt-get install / apk add
	Stages         []string          // named FROM stages
	EntryCmd       string            // CMD or ENTRYPOINT value
	InferredDeps   []string          // e.g., "postgres" inferred from postgresql-client
}

// ParseDockerfile extracts structured signals from a Dockerfile.
func ParseDockerfile(content []byte) DockerfileSignals { ... }
```

Tests:
- Dockerfile with `EXPOSE 8080 3000` → ports [8080, 3000]
- Dockerfile with `RUN apt-get install -y postgresql-client redis-tools` → packages, inferred deps ["postgres", "redis"]
- Multi-stage Dockerfile with `FROM golang:1.22 AS builder` → stages ["builder"]
- Dockerfile with `CMD ["go", "run", "./cmd/api"]` → EntryCmd "go run ./cmd/api"
- Dockerfile with `ENTRYPOINT ["/app/start.sh"]` → EntryCmd "/app/start.sh"
- Minimal Dockerfile with no signals → empty struct

- [ ] **Step 2: Implement ParseDockerfile**

Line-by-line parsing. Map known apt/apk packages to service dependencies (e.g., `postgresql-client` → postgres, `redis-tools` → redis, `mysql-client` → mysql).

- [ ] **Step 3: Run tests, verify pass**

- [ ] **Step 4: Commit**

---

### Task 6: LLM provider abstraction and Anthropic implementation

**Files:**
- Create: `internal/llm/llm.go`
- Create: `internal/llm/anthropic.go`
- Create: `internal/llm/anthropic_test.go`

- [ ] **Step 1: Define Provider interface**

```go
package llm

import "context"

// Provider sends prompts to an LLM and returns responses.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Request is a prompt to send to the LLM.
type Request struct {
	System string
	Prompt string
}

// Response is the LLM's reply.
type Response struct {
	Content string
}
```

- [ ] **Step 2: Implement AnthropicProvider**

Uses `net/http` to call the Anthropic Messages API. Model: `claude-haiku-4-5-20251001`. API key from `ANTHROPIC_API_KEY` env var. Max tokens: 4096. Returns the first text content block.

- [ ] **Step 3: Write test with HTTP test server**

Stand up an `httptest.Server` that mimics the Anthropic API response format. Verify the provider sends the correct request shape and parses the response.

- [ ] **Step 4: Add `NewProvider() (Provider, error)` factory**

Reads `ANTHROPIC_API_KEY`. Returns error if not set (callers use this to decide whether LLM features are available).

- [ ] **Step 5: Commit**

---

### Task 7: LLM analysis prompts

Structured prompts for Dockerfile and repo analysis that return parseable JSON.

**Files:**
- Create: `internal/llm/analyze.go`
- Create: `internal/llm/analyze_test.go`

- [ ] **Step 1: Define suggestion types**

```go
// ServiceSuggestion is a single service suggested by LLM analysis.
type ServiceSuggestion struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"` // "container" or "process"
	Image     string   `json:"image,omitempty"`
	Command   string   `json:"command,omitempty"`
	Ports     []int    `json:"ports,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Reasoning string   `json:"reasoning"`
}
```

- [ ] **Step 2: Implement AnalyzeDockerfile**

```go
// AnalyzeDockerfile asks the LLM to analyze a Dockerfile and suggest services.
func AnalyzeDockerfile(ctx context.Context, p Provider, dockerfile string, startScript string, fileTree string) ([]ServiceSuggestion, error)
```

Builds a system prompt explaining rook's service model. Sends the Dockerfile content, optional start script, and top-level file tree. Asks for JSON array of `ServiceSuggestion`. Parses the response, returns error on unparseable output.

- [ ] **Step 3: Implement AnalyzeRepo**

```go
// AnalyzeRepo asks the LLM to suggest local services based on repo structure.
func AnalyzeRepo(ctx context.Context, p Provider, fileTree string, configFiles map[string]string) ([]ServiceSuggestion, error)
```

`configFiles` is a map of filename → content for files like `go.mod`, `package.json`, `Makefile`. The LLM suggests runnable services.

- [ ] **Step 4: Write tests with mock provider**

Create a `mockProvider` that returns canned JSON responses. Test that `AnalyzeDockerfile` and `AnalyzeRepo` correctly parse valid JSON, handle malformed JSON gracefully (return error with raw content), and handle empty responses.

- [ ] **Step 5: Commit**

---

### Task 8: Rewrite init command — interactive flow

The big integration task. Rewrite `init.go` to use all the new pieces.

**Files:**
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Add flags to init command**

```go
var nonInteractive bool
var force bool
var add bool

cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Skip interactive prompts (legacy behavior)")
cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing rook.yaml")
cmd.Flags().BoolVar(&add, "add", false, "Add services to existing rook.yaml")
```

Auto-detect non-interactive: if stdin is not a terminal, set `nonInteractive = true`.

- [ ] **Step 2: Implement the interactive flow**

High-level structure of the new `RunE`:

```
1. Resolve dir, check rook.yaml exists
   - Exists + no flags → error "already initialized, use --force or --add"
   - Exists + --force → delete rook.yaml, continue
   - Exists + --add → load existing manifest, continue to service addition
   - Not exists → continue

2. Scan for compose files (ScanComposeFiles)
3. Scan for local signals (ScanLocalSignals)
4. Scan for devcontainer, mise (existing discoverers, informational)

5. If non-interactive:
   - Use first compose file (existing behavior)
   - Skip local service prompts
   - Write manifest and register

6. If interactive:
   a. Display what was found (compose files with service lists, local signals, other detections)
   b. Prompt: which compose file(s) to use (Select)
   c. For each selected compose file, run DiscoverFile and merge services
   d. If devcontainer Dockerfile found and LLM available:
      - Prompt: analyze with LLM? (Confirm)
      - If yes: run AnalyzeDockerfile, show suggestions, let user accept/edit/skip
   e. Display local service signals as suggestions
   f. Prompt: add local services? (Confirm)
      - If yes: loop of service name, command, depends_on inputs
      - Pre-fill from signals where possible
   g. Write manifest
   h. Run existing script copy + sanitization for devcontainer services
   i. Register workspace, allocate ports
```

- [ ] **Step 3: Extract the script-copy logic into a helper**

Move the devcontainer script copy/sanitize block from the current `init.go` into a function `copyDevcontainerScripts(dir string, services map[string]workspace.Service) (map[string]workspace.Service, warnings)` so the new flow can call it cleanly.

- [ ] **Step 4: Wire up --add flag**

When `--add` is set:
- Load existing manifest
- Run the interactive flow but skip compose file selection (those services already exist)
- Only show local signal detection and service addition prompts
- Merge new services into existing manifest
- Re-register (ports for new services)

- [ ] **Step 5: Test the non-interactive path still works**

Run existing init-related tests (if any) and manual test: `rook init --non-interactive <path>`

- [ ] **Step 6: Commit**

---

### Task 9: Integration tests

End-to-end tests for the new init flow using a mock `Prompter`.

**Files:**
- Create or modify: `internal/cli/init_test.go`

- [ ] **Step 1: Test non-interactive init (backward compat)**

Set up a temp dir with a `docker-compose.yml`. Run init with `--non-interactive`. Verify `rook.yaml` is created with the expected services.

- [ ] **Step 2: Test interactive init with mock prompter**

Inject a mock `Prompter` that returns predetermined selections. Verify:
- Selecting one of multiple compose files uses only that file's services
- Adding a local service creates a process service with the given command and depends_on
- Skipping all prompts produces a minimal manifest

- [ ] **Step 3: Test --force re-init**

Create a rook.yaml, run init with `--force`, verify it's overwritten.

- [ ] **Step 4: Test --add incremental**

Create a rook.yaml with postgres. Run init with `--add`, add a local service. Verify postgres is preserved and new service is added.

- [ ] **Step 5: Test auto-detection of non-interactive (no tty)**

Verify that when stdin is not a terminal, non-interactive mode is used automatically.

- [ ] **Step 6: Commit**

---

### Task 10: Update CLAUDE.md and roadmap

**Files:**
- Modify: `CLAUDE.md` — update CLI usage section with new flags
- Modify: `docs/ROADMAP.md` — add smart init task

- [ ] **Step 1: Update CLAUDE.md**

Add `--non-interactive`, `--force`, `--add` flags to the `rook init` usage section. Mention LLM features require `ANTHROPIC_API_KEY`.

- [ ] **Step 2: Update roadmap**

Add smart init under a new section or mark as in-progress.

- [ ] **Step 3: Commit**
