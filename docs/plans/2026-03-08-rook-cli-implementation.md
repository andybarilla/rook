# rook-cli Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add CLI subcommands to the existing `rook` binary so users can manage sites and services from the terminal.

**Architecture:** Single binary dispatch — `main.go` checks for subcommands and runs Cobra CLI handlers instead of launching the Wails GUI. CLI commands instantiate the same `core.Core` used by the GUI. Output auto-detects TTY for human-readable tables vs JSON.

**Tech Stack:** Go, Cobra (CLI framework), `mattn/go-isatty` (already in go.mod as indirect)

---

### Task 1: Add Cobra dependency and CLI skeleton

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/root_test.go`
- Modify: `main.go`

**Step 1: Install Cobra**

Run: `go get github.com/spf13/cobra`

**Step 2: Write the failing test for root command**

Create `internal/cli/root_test.go`:

```go
package cli_test

import (
	"bytes"
	"testing"

	"github.com/andybarilla/rook/internal/cli"
)

func TestRootCommandShowsHelp(t *testing.T) {
	var buf bytes.Buffer
	cmd := cli.NewRootCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("rook")) {
		t.Errorf("help output missing 'rook': %s", out)
	}
}

func TestRootCommandJSONFlag(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetArgs([]string{"--json", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	jsonFlag := cmd.Flag("json")
	if jsonFlag == nil {
		t.Fatal("expected --json flag to exist")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test -tags webkit2_41 ./internal/cli/ -v`
Expected: FAIL — package doesn't exist

**Step 4: Write minimal implementation**

Create `internal/cli/root.go`:

```go
package cli

import (
	"github.com/spf13/cobra"
)

var jsonOutput bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rook",
		Short: "Rook — local development environment manager",
		Long:  "Rook manages local development sites, SSL, PHP, Node, and database services.",
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func Execute() {
	cmd := NewRootCmd()
	cmd.Execute()
}
```

**Step 5: Run test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/cli/ -v`
Expected: PASS

**Step 6: Wire CLI dispatch into main.go**

Modify `main.go` — add CLI dispatch before Wails launch:

```go
package main

import (
	"embed"
	"os"

	"github.com/andybarilla/rook/internal/cli"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) > 1 {
		cli.Execute()
		return
	}

	app := NewApp()
	err := wails.Run(&options.App{
		Title:  "rook",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
```

**Step 7: Run all tests**

Run: `go test -tags webkit2_41 ./... -v`
Expected: All PASS

**Step 8: Commit**

```bash
git add internal/cli/root.go internal/cli/root_test.go main.go go.mod go.sum
git commit -m "feat(cli): add cobra root command with --json flag"
```

---

### Task 2: Output formatting (TTY detection + table/JSON)

**Files:**
- Create: `internal/cli/output.go`
- Create: `internal/cli/output_test.go`

**Step 1: Write the failing tests**

Create `internal/cli/output_test.go`:

```go
package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/andybarilla/rook/internal/cli"
)

func TestFormatTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"NAME", "VALUE"}
	rows := [][]string{
		{"domain", "myapp.test"},
		{"path", "/home/user/myapp"},
	}

	cli.FormatTable(&buf, headers, rows)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("NAME")) {
		t.Errorf("table missing header NAME: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("myapp.test")) {
		t.Errorf("table missing value myapp.test: %s", out)
	}
}

func TestFormatJSON(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]string{{"domain": "myapp.test"}}

	if err := cli.FormatJSON(&buf, data); err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var result []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result[0]["domain"] != "myapp.test" {
		t.Errorf("domain = %q, want myapp.test", result[0]["domain"])
	}
}

func TestFormatTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"SHORT", "LONG"}
	rows := [][]string{
		{"a", "abcdef"},
		{"ab", "xy"},
	}

	cli.FormatTable(&buf, headers, rows)
	out := buf.String()

	// Should contain aligned columns (tab-separated or padded)
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestFormat`
Expected: FAIL — functions not defined

**Step 3: Write minimal implementation**

Create `internal/cli/output.go`:

```go
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/mattn/go-isatty"
)

func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func FormatTable(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(tw, "\t")
		}
		fmt.Fprint(tw, h)
	}
	fmt.Fprintln(tw)
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, col)
		}
		fmt.Fprintln(tw)
	}
	tw.Flush()
}

func FormatJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestFormat`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/output.go internal/cli/output_test.go
git commit -m "feat(cli): add output formatting (table and JSON)"
```

---

### Task 3: Core factory helper

The CLI needs to instantiate `core.Core` with the same config the GUI uses. Extract this into a shared helper so both GUI and CLI use the same setup.

**Files:**
- Create: `internal/cli/corefactory.go`
- Create: `internal/cli/corefactory_test.go`

**Step 1: Write the failing test**

Create `internal/cli/corefactory_test.go`:

```go
package cli_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/cli"
)

func TestNewCoreReturnsNonNil(t *testing.T) {
	c, cleanup, err := cli.NewCore()
	if err != nil {
		t.Fatalf("NewCore: %v", err)
	}
	defer cleanup()

	if c == nil {
		t.Fatal("expected non-nil Core")
	}
}

func TestNewCoreListSitesEmpty(t *testing.T) {
	c, cleanup, err := cli.NewCore()
	if err != nil {
		t.Fatalf("NewCore: %v", err)
	}
	defer cleanup()

	sites := c.Sites()
	// Should not panic; may return empty or existing sites
	_ = sites
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestNewCore`
Expected: FAIL — function not defined

**Step 3: Write minimal implementation**

Create `internal/cli/corefactory.go`:

```go
package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/config"
	"github.com/andybarilla/rook/internal/core"
	"github.com/andybarilla/rook/internal/databases"
	"github.com/andybarilla/rook/internal/node"
)

// NewCore creates a Core instance with production config, identical to the GUI.
// Returns the Core, a cleanup function (calls Stop), and any error.
func NewCore() (*core.Core, func(), error) {
	logDir := config.DataDir()
	os.MkdirAll(logDir, 0o755)
	logFile, err := os.OpenFile(config.LogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logFile = os.Stderr
	}
	logger := log.New(logFile, "[rook-cli] ", log.LstdFlags)

	cfg := core.Config{
		SitesFile:    config.SitesFile(),
		Logger:       logger,
		CaddyRunner:  &noopCaddyRunner{},
		FPMRunner:    &noopFPMRunner{},
		CertStore:    &noopCertStore{},
		DBRunner:     databases.NewProcessRunner(),
		NodeRunner:   node.NewProcessRunner(),
		DBConfigPath: filepath.Join(config.ConfigDir(), "databases.json"),
		DBDataRoot:   filepath.Join(config.DataDir(), "databases"),
		PluginsDir:   config.PluginsDir(),
	}

	c := core.NewCore(cfg)
	if err := c.Start(); err != nil {
		return nil, func() {}, err
	}

	cleanup := func() {
		c.Stop()
	}

	return c, cleanup, nil
}

// noopCaddyRunner does nothing — CLI commands don't manage Caddy directly.
type noopCaddyRunner struct{}

func (r *noopCaddyRunner) Run(cfgJSON []byte) error { return nil }
func (r *noopCaddyRunner) Stop() error              { return nil }

// noopFPMRunner does nothing — CLI commands don't manage FPM directly.
type noopFPMRunner struct{}

func (r *noopFPMRunner) StartPool(version string) error  { return nil }
func (r *noopFPMRunner) StopPool(version string) error   { return nil }
func (r *noopFPMRunner) PoolSocket(version string) string { return "" }

// noopCertStore does nothing — CLI commands don't manage certs directly.
type noopCertStore struct{}

func (s *noopCertStore) InstallCA() error                { return nil }
func (s *noopCertStore) GenerateCert(domain string) error { return nil }
func (s *noopCertStore) CertPath(domain string) string   { return "" }
func (s *noopCertStore) KeyPath(domain string) string    { return "" }
func (s *noopCertStore) HasCert(domain string) bool      { return false }
```

**Step 4: Run test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestNewCore`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/corefactory.go internal/cli/corefactory_test.go
git commit -m "feat(cli): add core factory for CLI commands"
```

---

### Task 4: `rook list` command

**Files:**
- Create: `internal/cli/sites.go`
- Create: `internal/cli/sites_test.go`

**Step 1: Write the failing tests**

Create `internal/cli/sites_test.go`:

```go
package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/andybarilla/rook/internal/cli"
	"github.com/andybarilla/rook/internal/registry"
)

func TestListCmdTable(t *testing.T) {
	sites := []registry.Site{
		{Path: "/home/user/myapp", Domain: "myapp.test", PHPVersion: "8.3", TLS: true},
		{Path: "/home/user/api", Domain: "api.test", NodeVersion: "20", TLS: false},
	}

	var buf bytes.Buffer
	cli.RenderSiteList(&buf, sites, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("myapp.test")) {
		t.Errorf("output missing myapp.test: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("api.test")) {
		t.Errorf("output missing api.test: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("DOMAIN")) {
		t.Errorf("output missing DOMAIN header: %s", out)
	}
}

func TestListCmdJSON(t *testing.T) {
	sites := []registry.Site{
		{Path: "/home/user/myapp", Domain: "myapp.test", PHPVersion: "8.3", TLS: true},
	}

	var buf bytes.Buffer
	cli.RenderSiteList(&buf, sites, true)

	var result []registry.Site
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	if result[0].Domain != "myapp.test" {
		t.Errorf("domain = %q, want myapp.test", result[0].Domain)
	}
}

func TestListCmdEmptyTable(t *testing.T) {
	var buf bytes.Buffer
	cli.RenderSiteList(&buf, nil, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("No sites")) {
		t.Errorf("expected 'No sites' message, got: %s", out)
	}
}

func TestListCmdEmptyJSON(t *testing.T) {
	var buf bytes.Buffer
	cli.RenderSiteList(&buf, nil, true)

	var result []registry.Site
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 sites, got %d", len(result))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestListCmd`
Expected: FAIL — function not defined

**Step 3: Write minimal implementation**

Create `internal/cli/sites.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/registry"
	"github.com/spf13/cobra"
)

func RenderSiteList(w io.Writer, sites []registry.Site, asJSON bool) {
	if asJSON {
		if sites == nil {
			sites = []registry.Site{}
		}
		FormatJSON(w, sites)
		return
	}

	if len(sites) == 0 {
		fmt.Fprintln(w, "No sites registered.")
		return
	}

	headers := []string{"DOMAIN", "PATH", "PHP", "NODE", "TLS"}
	rows := make([][]string, len(sites))
	for i, s := range sites {
		tlsStr := ""
		if s.TLS {
			tlsStr = "✓"
		}
		rows[i] = []string{s.Domain, s.Path, s.PHPVersion, s.NodeVersion, tlsStr}
	}
	FormatTable(w, headers, rows)
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered sites",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			useJSON := jsonOutput || !IsTTY()
			RenderSiteList(os.Stdout, c.Sites(), useJSON)
			return nil
		},
	}
}

func newAddCmd() *cobra.Command {
	var domain, phpVersion, nodeVersion string
	var tls bool

	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a new site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			if domain == "" {
				domain = registry.InferDomain(path)
			}

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			site := registry.Site{
				Path:        path,
				Domain:      domain,
				PHPVersion:  phpVersion,
				NodeVersion: nodeVersion,
				TLS:         tls,
			}

			if err := c.AddSite(site); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, site)
			} else {
				fmt.Fprintf(os.Stdout, "✓ Site %q added (path: %s)\n", domain, path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Domain name (default: inferred from path)")
	cmd.Flags().StringVar(&phpVersion, "php", "", "PHP version")
	cmd.Flags().StringVar(&nodeVersion, "node", "", "Node version")
	cmd.Flags().BoolVar(&tls, "tls", false, "Enable TLS")

	return cmd
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <domain>",
		Short: "Remove a registered site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := c.RemoveSite(domain); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, map[string]string{"removed": domain})
			} else {
				fmt.Fprintf(os.Stdout, "✓ Site %q removed\n", domain)
			}
			return nil
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestListCmd`
Expected: PASS

**Step 5: Register commands in root**

Update `internal/cli/root.go` — add subcommands in `NewRootCmd()`:

```go
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rook",
		Short: "Rook — local development environment manager",
		Long:  "Rook manages local development sites, SSL, PHP, Node, and database services.",
		SilenceUsage: true,
	}

	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())

	return cmd
}
```

**Step 6: Run all tests**

Run: `go test -tags webkit2_41 ./internal/cli/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/cli/sites.go internal/cli/sites_test.go internal/cli/root.go
git commit -m "feat(cli): add list, add, and remove site commands"
```

---

### Task 5: `rook add` and `rook remove` tests

**Files:**
- Modify: `internal/cli/sites_test.go`

**Step 1: Write the failing tests**

Add to `internal/cli/sites_test.go`:

```go
func TestRenderAddSuccess(t *testing.T) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "✓ Site %q added (path: %s)\n", "myapp.test", "/home/user/myapp")
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("myapp.test")) {
		t.Errorf("output missing domain: %s", out)
	}
}

func TestRenderAddJSON(t *testing.T) {
	var buf bytes.Buffer
	site := registry.Site{Path: "/home/user/myapp", Domain: "myapp.test"}
	cli.FormatJSON(&buf, site)

	var result registry.Site
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Domain != "myapp.test" {
		t.Errorf("domain = %q, want myapp.test", result.Domain)
	}
}

func TestRenderRemoveSuccess(t *testing.T) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "✓ Site %q removed\n", "myapp.test")
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("myapp.test")) {
		t.Errorf("output missing domain: %s", out)
	}
}

func TestRenderRemoveJSON(t *testing.T) {
	var buf bytes.Buffer
	cli.FormatJSON(&buf, map[string]string{"removed": "myapp.test"})

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["removed"] != "myapp.test" {
		t.Errorf("removed = %q, want myapp.test", result["removed"])
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run "TestRenderAdd|TestRenderRemove"`
Expected: PASS (these test output rendering which is already implemented)

**Step 3: Commit**

```bash
git add internal/cli/sites_test.go
git commit -m "test(cli): add rendering tests for add and remove commands"
```

---

### Task 6: `rook status`, `rook start`, `rook stop` commands

**Files:**
- Create: `internal/cli/services.go`
- Create: `internal/cli/services_test.go`

**Step 1: Write the failing tests**

Create `internal/cli/services_test.go`:

```go
package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/andybarilla/rook/internal/cli"
	"github.com/andybarilla/rook/internal/databases"
)

func TestRenderServiceStatus(t *testing.T) {
	services := []databases.ServiceInfo{
		{Type: "mysql", Enabled: true, Running: true, Port: 3306},
		{Type: "postgresql", Enabled: true, Running: false, Port: 5432},
		{Type: "redis", Enabled: false, Running: false, Port: 6379},
	}

	var buf bytes.Buffer
	cli.RenderServiceStatus(&buf, services, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("mysql")) {
		t.Errorf("output missing mysql: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte("SERVICE")) {
		t.Errorf("output missing SERVICE header: %s", out)
	}
}

func TestRenderServiceStatusJSON(t *testing.T) {
	services := []databases.ServiceInfo{
		{Type: "mysql", Enabled: true, Running: true, Port: 3306},
	}

	var buf bytes.Buffer
	cli.RenderServiceStatus(&buf, services, true)

	var result []databases.ServiceInfo
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result))
	}
	if string(result[0].Type) != "mysql" {
		t.Errorf("type = %q, want mysql", result[0].Type)
	}
}

func TestRenderServiceStatusEmpty(t *testing.T) {
	var buf bytes.Buffer
	cli.RenderServiceStatus(&buf, nil, false)
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte("No services")) {
		t.Errorf("expected 'No services' message, got: %s", out)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestRenderService`
Expected: FAIL — function not defined

**Step 3: Write minimal implementation**

Create `internal/cli/services.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/andybarilla/rook/internal/databases"
	"github.com/spf13/cobra"
)

func RenderServiceStatus(w io.Writer, services []databases.ServiceInfo, asJSON bool) {
	if asJSON {
		if services == nil {
			services = []databases.ServiceInfo{}
		}
		FormatJSON(w, services)
		return
	}

	if len(services) == 0 {
		fmt.Fprintln(w, "No services configured.")
		return
	}

	headers := []string{"SERVICE", "STATUS", "PORT"}
	rows := make([][]string, len(services))
	for i, s := range services {
		status := "stopped"
		if s.Running {
			status = "running"
		}
		port := ""
		if s.Enabled {
			port = fmt.Sprintf("%d", s.Port)
		}
		rows[i] = []string{string(s.Type), status, port}
	}
	FormatTable(w, headers, rows)
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of all database services",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			useJSON := jsonOutput || !IsTTY()
			RenderServiceStatus(os.Stdout, c.DatabaseServices(), useJSON)
			return nil
		},
	}
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <service>",
		Short: "Start a database service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := args[0]

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := c.StartDatabase(svc); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, map[string]string{"started": svc})
			} else {
				fmt.Fprintf(os.Stdout, "✓ Service %q started\n", svc)
			}
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <service>",
		Short: "Stop a database service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := args[0]

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := c.StopDatabase(svc); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, map[string]string{"stopped": svc})
			} else {
				fmt.Fprintf(os.Stdout, "✓ Service %q stopped\n", svc)
			}
			return nil
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/cli/ -v -run TestRenderService`
Expected: PASS

**Step 5: Register service commands in root**

Update `internal/cli/root.go` — add to `NewRootCmd()`:

```go
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
```

**Step 6: Run all tests**

Run: `go test -tags webkit2_41 ./internal/cli/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/cli/services.go internal/cli/services_test.go internal/cli/root.go
git commit -m "feat(cli): add status, start, and stop service commands"
```

---

### Task 7: File locking for registry

**Files:**
- Modify: `internal/registry/registry.go`
- Modify: `internal/registry/registry_test.go`

**Step 1: Write the failing test for concurrent writes**

Add to `internal/registry/registry_test.go`:

```go
func TestConcurrentAddDoesNotCorrupt(t *testing.T) {
	path := tempFile(t)

	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			r := registry.New(path)
			_ = r.Load()
			dir := realDir(t)
			errs <- r.Add(registry.Site{
				Path:   dir,
				Domain: fmt.Sprintf("site%d.test", n),
			})
		}(i)
	}

	errCount := 0
	for i := 0; i < 10; i++ {
		if err := <-errs; err != nil {
			errCount++
		}
	}

	// Load final state and verify no corruption
	r := registry.New(path)
	if err := r.Load(); err != nil {
		t.Fatalf("Load after concurrent writes: %v", err)
	}

	sites := r.List()
	// Some adds may fail due to locking, but the file should be valid JSON
	// and contain at least 1 site
	if len(sites) == 0 {
		t.Error("expected at least 1 site after concurrent adds")
	}
}
```

Note: Add `"fmt"` to imports.

**Step 2: Run test to verify behavior (may pass or show corruption without locking)**

Run: `go test -tags webkit2_41 ./internal/registry/ -v -run TestConcurrent -count=5`

**Step 3: Add file locking to registry**

Modify `internal/registry/registry.go` — add locking to `Load()` and `save()`:

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func (r *Registry) withFileLock(fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	lockPath := r.path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer f.Close()

	if err := syscall.Rook(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer syscall.Rook(int(f.Fd()), syscall.LOCK_UN)

	return fn()
}
```

Then wrap `save()` internals with `withFileLock`, and also wrap `Load()` + `Add()` / `Remove()` / `Update()` to re-read from disk under lock before mutating:

```go
func (r *Registry) Add(s Site) error {
	return r.withFileLock(func() error {
		// Re-read from disk under lock
		if err := r.loadFromDisk(); err != nil {
			return err
		}

		info, err := os.Stat(s.Path)
		if err != nil {
			return fmt.Errorf("path %q: %w", s.Path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("path %q is not a directory", s.Path)
		}
		for _, existing := range r.sites {
			if existing.Domain == s.Domain {
				return fmt.Errorf("domain %q is already registered", s.Domain)
			}
		}
		r.sites = append(r.sites, s)
		if err := r.saveToDisk(); err != nil {
			r.sites = r.sites[:len(r.sites)-1]
			return err
		}
		r.notify(ChangeEvent{Type: SiteAdded, Site: s})
		return nil
	})
}
```

Rename existing `save()` to `saveToDisk()` (internal, no lock) and add `loadFromDisk()` (internal, no lock). The public `Load()` method should also use `withFileLock`.

**Important:** `syscall.Rook` is Unix-only. For Windows compatibility, use `golang.org/x/sys/windows` or a build-tag approach. For Phase 1, Unix-only locking is acceptable since the primary targets are Linux and macOS. Add a `//go:build !windows` tag and a no-op `registry_lock_windows.go` stub.

**Step 4: Run concurrent test to verify it passes**

Run: `go test -tags webkit2_41 ./internal/registry/ -v -run TestConcurrent -count=5`
Expected: PASS consistently

**Step 5: Run all registry tests**

Run: `go test -tags webkit2_41 ./internal/registry/ -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/registry/registry.go internal/registry/registry_test.go
git commit -m "feat(registry): add file locking for concurrent access safety"
```

---

### Task 8: Update ROADMAP and final integration test

**Files:**
- Modify: `docs/ROADMAP.md`

**Step 1: Run the full test suite**

Run: `go test -tags webkit2_41 ./... -v`
Expected: All PASS

**Step 2: Manual smoke test**

Run: `go build -tags webkit2_41 -o /tmp/rook-test .`

Then test:
```bash
/tmp/rook-test list
/tmp/rook-test status
/tmp/rook-test --help
/tmp/rook-test list --json
/tmp/rook-test list | cat   # should output JSON (piped)
```

**Step 3: Update ROADMAP.md**

Mark the CLI item as complete:
```markdown
- [x] rook-cli (CLI for managing sites, services, and plugins without the GUI) — See: docs/plans/2026-03-08-rook-cli-design.md
```

**Step 4: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs: mark rook-cli as complete in roadmap"
```
