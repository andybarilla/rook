# Process env_file Support Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Load `.env` file variables into process services, with template resolution and shell expansion, so process services get the same env_file support containers already have.

**Architecture:** Add `ParseEnvFile()` to the `envgen` package, then integrate it into `up.go`'s template resolution phase via a `LoadProcessEnvFile()` helper. Process services with an `env_file` get those variables parsed, expanded, resolved, and merged into `svc.Environment` (inline wins on conflict). `ProcessRunner` stays unchanged.

**Tech Stack:** Go 1.22+, stdlib `testing`

**Spec:** `docs/superpowers/specs/2026-03-16-process-envfile-design.md`

---

## Chunk 1: ParseEnvFile

### Task 1: Add `envgen.ParseEnvFile` with TDD

**Files:**
- Create: `internal/envgen/envfile.go`
- Create: `internal/envgen/envfile_test.go`

- [ ] **Step 1: Write failing tests for ParseEnvFile**

In `internal/envgen/envfile_test.go`:

```go
package envgen_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/envgen"
)

func TestParseEnvFile_BasicKeyValue(t *testing.T) {
	path := writeEnvFile(t, "DB_HOST=localhost\nDB_PORT=5432\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["DB_HOST"] != "localhost" {
		t.Errorf("DB_HOST: got %q", result["DB_HOST"])
	}
	if result["DB_PORT"] != "5432" {
		t.Errorf("DB_PORT: got %q", result["DB_PORT"])
	}
}

func TestParseEnvFile_CommentsAndBlankLines(t *testing.T) {
	path := writeEnvFile(t, "# this is a comment\n\nKEY=value\n\n# another comment\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 key, got %d: %v", len(result), result)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_DoubleQuotes(t *testing.T) {
	path := writeEnvFile(t, `KEY="value with spaces"` + "\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value with spaces" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_SingleQuotes(t *testing.T) {
	path := writeEnvFile(t, `KEY='value with spaces'` + "\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value with spaces" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_ExportPrefix(t *testing.T) {
	path := writeEnvFile(t, "export KEY=value\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "value" {
		t.Errorf("KEY: got %q", result["KEY"])
	}
}

func TestParseEnvFile_EmptyValue(t *testing.T) {
	path := writeEnvFile(t, "KEY=\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := result["KEY"]; !ok || v != "" {
		t.Errorf("KEY: expected empty string, got %q (ok=%v)", v, ok)
	}
}

func TestParseEnvFile_DuplicateKeysLastWins(t *testing.T) {
	path := writeEnvFile(t, "KEY=first\nKEY=second\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["KEY"] != "second" {
		t.Errorf("KEY: got %q, want 'second'", result["KEY"])
	}
}

func TestParseEnvFile_NoEqualsSkipped(t *testing.T) {
	path := writeEnvFile(t, "VALID=yes\nINVALIDLINE\nALSO_VALID=yes\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(result), result)
	}
}

func TestParseEnvFile_ValueWithEquals(t *testing.T) {
	path := writeEnvFile(t, "DATABASE_URL=postgres://u:p@host:5432/db?sslmode=disable\n")
	result, err := envgen.ParseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result["DATABASE_URL"] != "postgres://u:p@host:5432/db?sslmode=disable" {
		t.Errorf("DATABASE_URL: got %q", result["DATABASE_URL"])
	}
}

func TestParseEnvFile_FileNotFound(t *testing.T) {
	_, err := envgen.ParseEnvFile("/nonexistent/.env")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/envgen/ -run TestParseEnvFile -v`
Expected: FAIL — `ParseEnvFile` not defined

- [ ] **Step 3: Implement ParseEnvFile**

In `internal/envgen/envfile.go`:

```go
package envgen

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseEnvFile reads a .env file and returns key-value pairs.
// Supports: KEY=VALUE, comments (#), blank lines, quoted values,
// export prefix. Lines without = are skipped.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		key := line[:idx]
		value := line[idx+1:]

		// Strip matching surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/envgen/ -run TestParseEnvFile -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/envgen/envfile.go internal/envgen/envfile_test.go
git commit -m "feat: add ParseEnvFile to envgen package"
```

---

## Chunk 2: LoadProcessEnvFile helper and up.go integration

### Task 2: Add `envgen.LoadProcessEnvFile` helper with TDD

**Files:**
- Modify: `internal/envgen/envfile.go` (add helper)
- Modify: `internal/envgen/envfile_test.go` (add tests)

- [ ] **Step 1: Write failing tests for LoadProcessEnvFile**

Append to `internal/envgen/envfile_test.go`:

```go
func TestLoadProcessEnvFile_MergesIntoEnvironment(t *testing.T) {
	path := writeEnvFile(t, "FROM_FILE=file_val\nSHARED=from_file\n")
	env := map[string]string{"SHARED": "from_inline", "INLINE_ONLY": "yes"}
	portMap := map[string]int{"web": 10000}

	result, err := envgen.LoadProcessEnvFile(path, env, portMap)
	if err != nil {
		t.Fatal(err)
	}
	// Inline wins on conflict
	if result["SHARED"] != "from_inline" {
		t.Errorf("SHARED: got %q, want 'from_inline'", result["SHARED"])
	}
	// File-only var is present
	if result["FROM_FILE"] != "file_val" {
		t.Errorf("FROM_FILE: got %q", result["FROM_FILE"])
	}
	// Inline-only var is preserved
	if result["INLINE_ONLY"] != "yes" {
		t.Errorf("INLINE_ONLY: got %q", result["INLINE_ONLY"])
	}
}

func TestLoadProcessEnvFile_ResolvesTemplates(t *testing.T) {
	path := writeEnvFile(t, "API_URL=http://{{.Host.api}}:{{.Port.api}}/v1\n")
	portMap := map[string]int{"api": 10001}

	result, err := envgen.LoadProcessEnvFile(path, nil, portMap)
	if err != nil {
		t.Fatal(err)
	}
	if result["API_URL"] != "http://localhost:10001/v1" {
		t.Errorf("API_URL: got %q", result["API_URL"])
	}
}

func TestLoadProcessEnvFile_ExpandsShellVars(t *testing.T) {
	path := writeEnvFile(t, "DB_USER=${ROOK_TEST_UNSET_USER:-defaultuser}\n")
	portMap := map[string]int{}

	result, err := envgen.LoadProcessEnvFile(path, nil, portMap)
	if err != nil {
		t.Fatal(err)
	}
	if result["DB_USER"] != "defaultuser" {
		t.Errorf("DB_USER: got %q", result["DB_USER"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/envgen/ -run TestLoadProcessEnvFile -v`
Expected: FAIL — `LoadProcessEnvFile` not defined

- [ ] **Step 3: Implement LoadProcessEnvFile**

Append to `internal/envgen/envfile.go`:

```go
// LoadProcessEnvFile reads an env file, expands shell vars, resolves
// templates (using localhost + allocated ports for process services),
// and merges with inline environment. Inline values take precedence.
func LoadProcessEnvFile(path string, inlineEnv map[string]string, portMap map[string]int) (map[string]string, error) {
	fileVars, err := ParseEnvFile(path)
	if err != nil {
		return nil, err
	}

	// Expand shell vars in each value
	for k, v := range fileVars {
		fileVars[k] = ExpandShellVars(v)
	}

	// Resolve templates (process services use localhost)
	fileVars, err = ResolveTemplates(fileVars, portMap, false)
	if err != nil {
		return nil, fmt.Errorf("resolving env file templates: %w", err)
	}

	// Merge: start with file vars, overlay inline (inline wins)
	result := fileVars
	for k, v := range inlineEnv {
		result[k] = v
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/envgen/ -run TestLoadProcessEnvFile -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/envgen/envfile.go internal/envgen/envfile_test.go
git commit -m "feat: add LoadProcessEnvFile helper for merge and resolution"
```

### Task 3: Integrate into up.go

**Files:**
- Modify: `internal/cli/up.go:92-109` (add env_file loading for process services)

- [ ] **Step 1: Add env_file loading to the template resolution loop**

In `internal/cli/up.go`, replace the template resolution loop (lines 92-109) with:

```go
		for name, svc := range ws.Services {
			if len(svc.Environment) > 0 {
				var resolved map[string]string
				var err error
				if svc.IsContainer() {
					// Container services use container networking
					resolved, err = envgen.ResolveWithHostMap(svc.Environment, containerPortMap, containerHostMap)
				} else {
					// Process services use localhost + allocated ports
					resolved, err = envgen.ResolveTemplates(svc.Environment, portMap, false)
				}
				if err != nil {
					return fmt.Errorf("resolving env for %s: %w", name, err)
				}
				svc.Environment = resolved
				ws.Services[name] = svc
			}

			// Load env_file for process services
			if svc.IsProcess() && svc.EnvFile != "" {
				envFilePath := filepath.Join(ws.Root, svc.EnvFile)
				merged, err := envgen.LoadProcessEnvFile(envFilePath, svc.Environment, portMap)
				if err != nil {
					return fmt.Errorf("loading env_file for %s: %w", name, err)
				}
				svc.Environment = merged
				ws.Services[name] = svc
			}
		}
```

- [ ] **Step 2: Run all tests to verify nothing is broken**

Run: `go test ./... -count=1`
Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cli/up.go
git commit -m "feat: load env_file for process services during up"
```

### Task 4: Run full test suite and verify

- [ ] **Step 1: Run all tests**

Run: `go test ./... -count=1 -v`
Expected: all PASS

- [ ] **Step 2: Build CLI to verify compilation**

Run: `make build-cli`
Expected: builds successfully
