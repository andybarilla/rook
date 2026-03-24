# `rook env rewrite` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `rook env rewrite <VAR> <SERVICE>` command that reads a variable from a workspace's `.env` file, replaces host/port values with rook template tags, and writes the result into `rook.yaml`'s environment block.

**Architecture:** Pure rewrite function in `internal/envgen/rewrite.go` (URL-aware parsing with string fallback), CLI command in `internal/cli/env_rewrite.go` that wires workspace loading, env file parsing, rewriting, and manifest update. The existing `env` command gets a subcommand added without breaking `rook env [workspace]`.

**Tech Stack:** Go stdlib (`net/url`, `net`, `strconv`), existing `envgen.ParseEnvFile()`, `workspace.ParseManifest()`/`WriteManifest()`

**Spec:** `docs/superpowers/specs/2026-03-24-env-rewrite-design.md`

---

### Task 1: Rewrite function — URL values

**Files:**
- Create: `internal/envgen/rewrite.go`
- Create: `internal/envgen/rewrite_test.go`

- [ ] **Step 1: Write failing tests for URL rewriting**

```go
package envgen_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/envgen"
)

func TestRewrite_URLWithHostAndPort(t *testing.T) {
	result, err := envgen.Rewrite("postgres://user:pass@localhost:5432/db", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithHostOnly(t *testing.T) {
	result, err := envgen.Rewrite("http://localhost/api", "app")
	if err != nil {
		t.Fatal(err)
	}
	expected := "http://{{.Host.app}}/api"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithIPAndPort(t *testing.T) {
	result, err := envgen.Rewrite("redis://127.0.0.1:6379/0", "redis")
	if err != nil {
		t.Fatal(err)
	}
	expected := "redis://{{.Host.redis}}:{{.Port.redis}}/0"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithNonStandardPort(t *testing.T) {
	result, err := envgen.Rewrite("postgres://user:pass@localhost:9999/db", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithQueryAndFragment(t *testing.T) {
	result, err := envgen.Rewrite("postgres://user:pass@localhost:5432/db?sslmode=disable#pool", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/db?sslmode=disable#pool"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_URLWithEmptyHost(t *testing.T) {
	_, err := envgen.Rewrite("http:///path", "app")
	if err == nil {
		t.Error("expected error for URL with empty host and no port")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/envgen/ -run TestRewrite -v`
Expected: FAIL — `Rewrite` not defined

- [ ] **Step 3: Implement `Rewrite` for URL values**

```go
package envgen

import (
	"fmt"
	"net/url"
	"strings"
)

// Rewrite detects host and port values in a string and replaces them
// with rook template tags for the given service name.
func Rewrite(value string, serviceName string) (string, error) {
	hostTag := fmt.Sprintf("{{.Host.%s}}", serviceName)
	portTag := fmt.Sprintf("{{.Port.%s}}", serviceName)

	if strings.Contains(value, "://") {
		return rewriteURL(value, hostTag, portTag)
	}

	return "", fmt.Errorf("cannot detect host or port in value %q", value)
}

func rewriteURL(value string, hostTag string, portTag string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if host == "" && port == "" {
		return "", fmt.Errorf("cannot detect host or port in value %q", value)
	}

	result := value
	if host != "" && port != "" {
		oldHostPort := host + ":" + port
		newHostPort := hostTag + ":" + portTag
		result = strings.Replace(result, oldHostPort, newHostPort, 1)
	} else if host != "" {
		result = strings.Replace(result, host, hostTag, 1)
	} else if port != "" {
		result = strings.Replace(result, ":"+port, ":"+portTag, 1)
	}

	return result, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/envgen/ -run TestRewrite -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/envgen/rewrite.go internal/envgen/rewrite_test.go
git commit -m "feat: add Rewrite function for URL value detection"
```

---

### Task 2: Rewrite function — host:port and bare values

**Files:**
- Modify: `internal/envgen/rewrite.go`
- Modify: `internal/envgen/rewrite_test.go`

- [ ] **Step 1: Write failing tests for host:port and bare values**

Add to `rewrite_test.go`:

```go
func TestRewrite_HostPort(t *testing.T) {
	result, err := envgen.Rewrite("localhost:5432", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	expected := "{{.Host.postgres}}:{{.Port.postgres}}"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_HostPortWithIP(t *testing.T) {
	result, err := envgen.Rewrite("127.0.0.1:6379", "redis")
	if err != nil {
		t.Fatal(err)
	}
	expected := "{{.Host.redis}}:{{.Port.redis}}"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRewrite_BareHost(t *testing.T) {
	result, err := envgen.Rewrite("localhost", "app")
	if err != nil {
		t.Fatal(err)
	}
	if result != "{{.Host.app}}" {
		t.Errorf("got %q", result)
	}
}

func TestRewrite_BareIP(t *testing.T) {
	result, err := envgen.Rewrite("127.0.0.1", "app")
	if err != nil {
		t.Fatal(err)
	}
	if result != "{{.Host.app}}" {
		t.Errorf("got %q", result)
	}
}

func TestRewrite_BareZeroIP(t *testing.T) {
	result, err := envgen.Rewrite("0.0.0.0", "app")
	if err != nil {
		t.Fatal(err)
	}
	if result != "{{.Host.app}}" {
		t.Errorf("got %q", result)
	}
}

func TestRewrite_BarePort(t *testing.T) {
	result, err := envgen.Rewrite("5432", "postgres")
	if err != nil {
		t.Fatal(err)
	}
	if result != "{{.Port.postgres}}" {
		t.Errorf("got %q", result)
	}
}

func TestRewrite_Unrecognized(t *testing.T) {
	_, err := envgen.Rewrite("some_random_string", "app")
	if err == nil {
		t.Error("expected error for unrecognized value")
	}
}
```

- [ ] **Step 2: Run tests to verify the new ones fail**

Run: `go test ./internal/envgen/ -run "TestRewrite_(HostPort|Bare|Unrecognized)" -v`
Expected: FAIL — host:port and bare value branches not implemented

- [ ] **Step 3: Implement host:port and bare value detection**

Update `Rewrite` in `rewrite.go`. The full file after this step:

```go
package envgen

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Rewrite detects host and port values in a string and replaces them
// with rook template tags for the given service name.
func Rewrite(value string, serviceName string) (string, error) {
	hostTag := fmt.Sprintf("{{.Host.%s}}", serviceName)
	portTag := fmt.Sprintf("{{.Port.%s}}", serviceName)

	// URL
	if strings.Contains(value, "://") {
		return rewriteURL(value, hostTag, portTag)
	}

	// Host:Port — split on last colon, validate port side is numeric
	if host, port, ok := splitHostPort(value); ok {
		_ = host // validated by splitHostPort
		_ = port
		return hostTag + ":" + portTag, nil
	}

	// Bare port (numeric string)
	if _, err := strconv.Atoi(value); err == nil {
		return portTag, nil
	}

	// Bare host (localhost or IPv4)
	if isKnownHost(value) {
		return hostTag, nil
	}

	return "", fmt.Errorf("cannot detect host or port in value %q", value)
}

func rewriteURL(value string, hostTag string, portTag string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()

	if host == "" && port == "" {
		return "", fmt.Errorf("cannot detect host or port in value %q", value)
	}

	result := value
	if host != "" && port != "" {
		oldHostPort := host + ":" + port
		newHostPort := hostTag + ":" + portTag
		result = strings.Replace(result, oldHostPort, newHostPort, 1)
	} else if host != "" {
		result = strings.Replace(result, host, hostTag, 1)
	} else if port != "" {
		result = strings.Replace(result, ":"+port, ":"+portTag, 1)
	}

	return result, nil
}

// splitHostPort splits "host:port" where port is numeric.
// Returns false if the value doesn't match this pattern.
func splitHostPort(value string) (string, string, bool) {
	idx := strings.LastIndex(value, ":")
	if idx < 1 || idx == len(value)-1 {
		return "", "", false
	}
	host := value[:idx]
	port := value[idx+1:]
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", false
	}
	return host, port, true
}

func isKnownHost(value string) bool {
	if value == "localhost" {
		return true
	}
	ip := net.ParseIP(value)
	return ip != nil && ip.To4() != nil
}
```

- [ ] **Step 4: Run all Rewrite tests**

Run: `go test ./internal/envgen/ -run TestRewrite -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/envgen/rewrite.go internal/envgen/rewrite_test.go
git commit -m "feat: add host:port and bare value rewriting"
```

---

### Task 3: CLI command with tests — `rook env rewrite`

CLI tests in this project use `package cli` (white-box), `t.Setenv("XDG_CONFIG_HOME", cfgDir)` for isolation, and call individual command constructors directly. The workspace must be registered via `newInitCmd()` before commands that use the registry.

**Files:**
- Create: `internal/cli/env_rewrite.go`
- Create: `internal/cli/env_rewrite_test.go`
- Modify: `internal/cli/env.go`

- [ ] **Step 1: Write failing CLI integration tests**

Create `internal/cli/env_rewrite_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/rook/internal/workspace"
)

func setupEnvRewriteWorkspace(t *testing.T, envContent string, manifest *workspace.Manifest) (wsDir string, cfgDir string) {
	t.Helper()
	cfgDir = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)

	wsDir = t.TempDir()
	if envContent != "" {
		if err := os.WriteFile(filepath.Join(wsDir, ".env"), []byte(envContent), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := workspace.WriteManifest(filepath.Join(wsDir, "rook.yaml"), manifest); err != nil {
		t.Fatal(err)
	}

	// Register the workspace via init
	initCmd := newInitCmd()
	initCmd.SetArgs([]string{wsDir})
	if err := initCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	return wsDir, cfgDir
}

func TestEnvRewriteCmd_RewritesURLInManifest(t *testing.T) {
	wsDir, _ := setupEnvRewriteWorkspace(t,
		"DATABASE_URL=postgres://user:pass@localhost:5432/mydb\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app": {
					Command: "node server.js",
					Ports:   []int{3000},
					EnvFile: ".env",
				},
				"postgres": {
					Image: "postgres:16",
					Ports: []int{5432},
				},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	updated, err := workspace.ParseManifest(filepath.Join(wsDir, "rook.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/mydb"
	if updated.Services["app"].Environment["DATABASE_URL"] != expected {
		t.Errorf("got %q, want %q", updated.Services["app"].Environment["DATABASE_URL"], expected)
	}
}

func TestEnvRewriteCmd_ErrorOnMissingVar(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"OTHER_VAR=something\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app":      {Command: "node server.js", EnvFile: ".env"},
				"postgres": {Image: "postgres:16", Ports: []int{5432}},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing var")
	}
}

func TestEnvRewriteCmd_ErrorOnMissingService(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"DATABASE_URL=postgres://localhost:5432/db\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app": {Command: "node server.js", EnvFile: ".env"},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing service")
	}
}

func TestEnvRewriteCmd_ErrorOnNoEnvFile(t *testing.T) {
	setupEnvRewriteWorkspace(t,
		"", // no .env file needed
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app":      {Command: "node server.js"},
				"postgres": {Image: "postgres:16", Ports: []int{5432}},
			},
		},
	)

	cmd := newEnvCmd()
	cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error when no services have env_file")
	}
}

func TestEnvRewriteCmd_Idempotent(t *testing.T) {
	wsDir, _ := setupEnvRewriteWorkspace(t,
		"DATABASE_URL=postgres://user:pass@localhost:5432/mydb\n",
		&workspace.Manifest{
			Name: "testws",
			Type: workspace.TypeSingle,
			Services: map[string]workspace.Service{
				"app": {
					Command: "node server.js",
					Ports:   []int{3000},
					EnvFile: ".env",
				},
				"postgres": {
					Image: "postgres:16",
					Ports: []int{5432},
				},
			},
		},
	)

	// Run twice
	for i := 0; i < 2; i++ {
		cmd := newEnvCmd()
		cmd.SetArgs([]string{"rewrite", "DATABASE_URL", "postgres", "testws"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	updated, err := workspace.ParseManifest(filepath.Join(wsDir, "rook.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expected := "postgres://user:pass@{{.Host.postgres}}:{{.Port.postgres}}/mydb"
	if updated.Services["app"].Environment["DATABASE_URL"] != expected {
		t.Errorf("got %q, want %q", updated.Services["app"].Environment["DATABASE_URL"], expected)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestEnvRewrite -v`
Expected: FAIL — `newEnvRewriteCmd` not defined

- [ ] **Step 3: Restructure `env.go` to support subcommands**

Replace `internal/cli/env.go`. Keep `RunE` on the parent so `rook env [workspace]` still works. Keep `Short` descriptive for the print behavior:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env [workspace]",
		Short: "Print generated environment variables",
		RunE:  runEnvPrint,
	}
	cmd.AddCommand(newEnvRewriteCmd())
	return cmd
}

func runEnvPrint(cmd *cobra.Command, args []string) error {
	cctx, err := newCLIContext()
	if err != nil {
		return err
	}

	ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
	if err != nil {
		return err
	}

	portMap := make(map[string]int)
	for name := range ws.Services {
		if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
			portMap[name] = result.Port
		}
	}
	for name, svc := range ws.Services {
		resolved, err := envgen.ResolveTemplates(svc.Environment, portMap)
		if err != nil {
			return err
		}
		for k, v := range resolved {
			fmt.Printf("%s.%s: %s=%s\n", ws.Name, name, k, v)
		}
	}
	return nil
}
```

- [ ] **Step 4: Create `env_rewrite.go`**

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newEnvRewriteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rewrite <var> <service> [workspace]",
		Short: "Rewrite an env var with rook template tags",
		Long:  "Reads a variable from a service's .env file, replaces host/port with {{.Host.x}}/{{.Port.x}} template tags, and adds it to rook.yaml's environment block.",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  runEnvRewrite,
	}
}

func runEnvRewrite(cmd *cobra.Command, args []string) error {
	varName := args[0]
	serviceName := args[1]
	wsArgs := args[2:]

	cctx, err := newCLIContext()
	if err != nil {
		return err
	}

	ws, err := cctx.resolveAndLoadWorkspace(wsArgs, os.Stdin)
	if err != nil {
		return err
	}

	// Validate target service exists
	if _, ok := ws.Services[serviceName]; !ok {
		return fmt.Errorf("service %q not found in workspace %q", serviceName, ws.Name)
	}

	// Find services whose env_file contains the variable
	type envFileMatch struct {
		svcName string
		value   string
	}
	var matches []envFileMatch
	hasEnvFile := false

	for name, svc := range ws.Services {
		if svc.EnvFile == "" {
			continue
		}
		hasEnvFile = true
		envPath := filepath.Join(ws.Root, svc.EnvFile)
		vars, err := envgen.ParseEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("parsing env file for %s: %w", name, err)
		}
		if val, ok := vars[varName]; ok {
			matches = append(matches, envFileMatch{svcName: name, value: val})
		}
	}

	if !hasEnvFile {
		return fmt.Errorf("no services in workspace %q have an env_file", ws.Name)
	}

	if len(matches) == 0 {
		return fmt.Errorf("%q not found in any service's env_file", varName)
	}

	// Rewrite
	rewritten, err := envgen.Rewrite(matches[0].value, serviceName)
	if err != nil {
		return err
	}

	// Load manifest, update environment blocks, write back
	entry, regErr := cctx.registry.Get(ws.Name)
	if regErr != nil {
		return regErr
	}
	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	manifest, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return err
	}

	for _, m := range matches {
		svc := manifest.Services[m.svcName]
		if svc.Environment == nil {
			svc.Environment = make(map[string]string)
		}
		svc.Environment[varName] = rewritten
		manifest.Services[m.svcName] = svc
		fmt.Printf("%s: %s = %s\n", m.svcName, varName, rewritten)
	}

	if err := workspace.WriteManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}
```

- [ ] **Step 5: Run env rewrite tests**

Run: `go test ./internal/cli/ -run TestEnvRewrite -v`
Expected: PASS

- [ ] **Step 6: Run existing CLI tests to ensure nothing breaks**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 7: Run full test suite**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add internal/cli/env.go internal/cli/env_rewrite.go internal/cli/env_rewrite_test.go
git commit -m "feat: add rook env rewrite command"
```
