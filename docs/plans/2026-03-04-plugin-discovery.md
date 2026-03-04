# Plugin Discovery and Loading API — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable Flock to discover and load external plugins at startup via convention-based directory scanning and JSON-RPC 2.0 subprocess communication.

**Architecture:** External plugins are standalone executables in `~/.config/flock/plugins/<name>/`. Each has a `plugin.json` manifest declaring capabilities. An `ExternalPlugin` adapter struct proxies the existing `Plugin`/`RuntimePlugin`/`ServicePlugin` interfaces to the subprocess over JSON-RPC 2.0 on stdin/stdout. The existing Manager sees no difference between built-in and external plugins.

**Tech Stack:** Go stdlib only (encoding/json, os/exec, bufio, context). No new dependencies.

---

### Task 1: PluginsDir Helper

**Files:**
- Modify: `internal/config/paths.go`
- Modify: `internal/config/paths_test.go`

**Step 1: Write the failing test**

```go
func TestPluginsDir(t *testing.T) {
	dir := config.PluginsDir()
	if !strings.HasSuffix(dir, filepath.Join("flock", "plugins")) {
		t.Fatalf("PluginsDir() = %q, want suffix flock/plugins", dir)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/config/ -run TestPluginsDir -v`
Expected: FAIL — `PluginsDir` not defined

**Step 3: Write minimal implementation**

Add to `internal/config/paths.go`:

```go
func PluginsDir() string {
	return filepath.Join(ConfigDir(), "plugins")
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/config/ -run TestPluginsDir -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/paths.go internal/config/paths_test.go
git commit -m "feat: add PluginsDir helper to config package"
```

---

### Task 2: Discovery Package — PluginManifest & Scan

**Files:**
- Create: `internal/discovery/discovery.go`
- Create: `internal/discovery/discovery_test.go`

**Step 1: Write failing tests for manifest parsing and directory scanning**

```go
package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
}

func TestScanNonexistentDir(t *testing.T) {
	manifests, errs := Scan("/nonexistent/path")
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errs))
	}
}

func TestScanValidPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "flock-node")
	os.MkdirAll(pluginDir, 0o755)

	manifest := `{
		"id": "flock-node",
		"name": "Flock Node",
		"version": "0.1.0",
		"executable": "flock-node",
		"capabilities": ["runtime"]
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)

	// Create a fake executable
	exePath := filepath.Join(pluginDir, "flock-node")
	os.WriteFile(exePath, []byte("#!/bin/sh\n"), 0o755)

	manifests, errs := Scan(dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
	m := manifests[0]
	if m.ID != "flock-node" {
		t.Errorf("ID = %q, want flock-node", m.ID)
	}
	if m.Name != "Flock Node" {
		t.Errorf("Name = %q, want Flock Node", m.Name)
	}
	if m.ExePath != exePath {
		t.Errorf("ExePath = %q, want %q", m.ExePath, exePath)
	}
	if len(m.Capabilities) != 1 || m.Capabilities[0] != "runtime" {
		t.Errorf("Capabilities = %v, want [runtime]", m.Capabilities)
	}
}

func TestScanSkipsMissingManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "no-manifest"), 0o755)

	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestScanSkipsMissingExecutable(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "bad-exe")
	os.MkdirAll(pluginDir, 0o755)

	manifest := `{
		"id": "bad-exe",
		"name": "Bad Exe",
		"version": "0.1.0",
		"executable": "nonexistent",
		"capabilities": ["runtime"]
	}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)

	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestScanSkipsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "no-id")
	os.MkdirAll(pluginDir, 0o755)

	manifest := `{"name": "No ID", "version": "0.1.0", "executable": "x", "capabilities": ["runtime"]}`
	os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)
	os.WriteFile(filepath.Join(pluginDir, "x"), []byte("#!/bin/sh\n"), 0o755)

	manifests, errs := Scan(dir)
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests, got %d", len(manifests))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestScanMultiplePlugins(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"plugin-a", "plugin-b"} {
		pluginDir := filepath.Join(dir, name)
		os.MkdirAll(pluginDir, 0o755)
		manifest := `{"id":"` + name + `","name":"` + name + `","version":"0.1.0","executable":"` + name + `","capabilities":["runtime"]}`
		os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644)
		os.WriteFile(filepath.Join(pluginDir, name), []byte("#!/bin/sh\n"), 0o755)
	}

	manifests, errs := Scan(dir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(manifests) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(manifests))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/discovery/ -v`
Expected: FAIL — package doesn't exist

**Step 3: Write implementation**

```go
package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PluginManifest struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Executable      string   `json:"executable"`
	Capabilities    []string `json:"capabilities"`
	MinFlockVersion string   `json:"minFlockVersion,omitempty"`
	ExePath         string   `json:"-"`
}

func Scan(dir string) ([]PluginManifest, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}

	var manifests []PluginManifest
	var errs []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m, err := loadManifest(filepath.Join(dir, entry.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("plugin %s: %w", entry.Name(), err))
			continue
		}
		manifests = append(manifests, m)
	}

	return manifests, errs
}

func loadManifest(pluginDir string) (PluginManifest, error) {
	data, err := os.ReadFile(filepath.Join(pluginDir, "plugin.json"))
	if err != nil {
		return PluginManifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return PluginManifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	if err := validate(m); err != nil {
		return PluginManifest{}, err
	}

	exePath := filepath.Join(pluginDir, m.Executable)
	info, err := os.Stat(exePath)
	if err != nil {
		return PluginManifest{}, fmt.Errorf("executable %q not found", m.Executable)
	}
	if info.IsDir() {
		return PluginManifest{}, fmt.Errorf("executable %q is a directory", m.Executable)
	}

	m.ExePath = exePath
	return m, nil
}

func validate(m PluginManifest) error {
	if m.ID == "" {
		return fmt.Errorf("missing required field: id")
	}
	if m.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if m.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if m.Executable == "" {
		return fmt.Errorf("missing required field: executable")
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("missing required field: capabilities")
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/discovery/ -v`
Expected: PASS (all 7 tests)

**Step 5: Commit**

```bash
git add internal/discovery/
git commit -m "feat: add plugin discovery with manifest scanning"
```

---

### Task 3: JSON-RPC 2.0 Client

**Files:**
- Create: `internal/external/jsonrpc.go`
- Create: `internal/external/jsonrpc_test.go`

**Step 1: Write failing tests for JSON-RPC request/response encoding**

```go
package external

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestRPCClientCall(t *testing.T) {
	// Simulate a plugin subprocess: reads request from stdin, writes response to stdout
	reqBuf := &bytes.Buffer{}
	respJSON := `{"jsonrpc":"2.0","id":1,"result":{"handles":true}}` + "\n"
	respReader := strings.NewReader(respJSON)

	client := newRPCClient(respReader, reqBuf)

	var result struct {
		Handles bool `json:"handles"`
	}
	err := client.Call("plugin.handles", map[string]any{"site": "test"}, &result)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if !result.Handles {
		t.Fatal("expected handles=true")
	}

	// Verify the request was written correctly
	var req rpcRequest
	if err := json.NewDecoder(bytes.NewReader(reqBuf.Bytes())).Decode(&req); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if req.Method != "plugin.handles" {
		t.Errorf("method = %q, want plugin.handles", req.Method)
	}
}

func TestRPCClientCallError(t *testing.T) {
	respJSON := `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"plugin error"}}` + "\n"
	client := newRPCClient(strings.NewReader(respJSON), io.Discard)

	var result struct{}
	err := client.Call("plugin.init", nil, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "plugin error") {
		t.Errorf("error = %q, want contains 'plugin error'", err.Error())
	}
}

func TestRPCClientCallNilResult(t *testing.T) {
	respJSON := `{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"
	client := newRPCClient(strings.NewReader(respJSON), io.Discard)

	err := client.Call("plugin.start", nil, nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -run TestRPC -v`
Expected: FAIL — package doesn't exist

**Step 3: Write implementation**

```go
package external

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcClient struct {
	mu      sync.Mutex
	nextID  int
	reader  *bufio.Reader
	writer  io.Writer
}

func newRPCClient(r io.Reader, w io.Writer) *rpcClient {
	return &rpcClient{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

func (c *rpcClient) Call(method string, params any, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	if _, err := c.writer.Write(data); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var resp rpcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -run TestRPC -v`
Expected: PASS (all 3 tests)

**Step 5: Commit**

```bash
git add internal/external/
git commit -m "feat: add JSON-RPC 2.0 client for external plugins"
```

---

### Task 4: ExternalPlugin Adapter

**Files:**
- Create: `internal/external/plugin.go`
- Create: `internal/external/plugin_test.go`

**Step 1: Write failing tests for ExternalPlugin lifecycle and methods**

Tests use a fake subprocess by piping stdin/stdout. The `ExternalPlugin` is constructed with a `ProcessStarter` interface so tests can inject a mock process.

```go
package external

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/andybarilla/flock/internal/discovery"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// mockProcess simulates a plugin subprocess
type mockProcess struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	killed bool
}

func (p *mockProcess) Kill() error { p.killed = true; return nil }
func (p *mockProcess) Wait() error { return nil }

// fakeProcessStarter returns a mockProcess with pre-wired pipes
// and a goroutine that handles JSON-RPC requests with the given handler
func fakeProcessStarter(handler func(method string, params json.RawMessage) (any, error)) ProcessStarter {
	return func(exePath string) (Process, error) {
		// plugin reads from stdinR, writes to stdoutW
		stdinR, stdinW := io.Pipe()
		stdoutR, stdoutW := io.Pipe()

		go func() {
			decoder := json.NewDecoder(stdinR)
			encoder := json.NewEncoder(stdoutW)
			for {
				var req rpcRequest
				if err := decoder.Decode(&req); err != nil {
					return
				}

				raw, _ := json.Marshal(req.Params)
				result, err := handler(req.Method, raw)
				if err != nil {
					encoder.Encode(map[string]any{
						"jsonrpc": "2.0",
						"id":      req.ID,
						"error":   map[string]any{"code": -32000, "message": err.Error()},
					})
				} else {
					resultJSON, _ := json.Marshal(result)
					encoder.Encode(map[string]any{
						"jsonrpc": "2.0",
						"id":      req.ID,
						"result":  json.RawMessage(resultJSON),
					})
				}
			}
		}()

		return &pipeProcess{
			stdinW:  stdinW,
			stdoutR: stdoutR,
		}, nil
	}
}

type pipeProcess struct {
	stdinW  *io.PipeWriter
	stdoutR *io.PipeReader
}

func (p *pipeProcess) Stdin() io.WriteCloser  { return p.stdinW }
func (p *pipeProcess) Stdout() io.ReadCloser  { return p.stdoutR }
func (p *pipeProcess) Kill() error             { p.stdinW.Close(); p.stdoutR.Close(); return nil }
func (p *pipeProcess) Wait() error             { return nil }

func TestExternalPluginIDAndName(t *testing.T) {
	manifest := discovery.PluginManifest{
		ID:           "flock-node",
		Name:         "Flock Node",
		Capabilities: []string{"runtime"},
	}
	p := NewPlugin(manifest, nil)
	if p.ID() != "flock-node" {
		t.Errorf("ID() = %q, want flock-node", p.ID())
	}
	if p.Name() != "Flock Node" {
		t.Errorf("Name() = %q, want Flock Node", p.Name())
	}
}

func TestExternalPluginInitStartStop(t *testing.T) {
	var methods []string
	starter := fakeProcessStarter(func(method string, params json.RawMessage) (any, error) {
		methods = append(methods, method)
		return map[string]any{}, nil
	})

	manifest := discovery.PluginManifest{
		ID:           "test-plugin",
		Name:         "Test Plugin",
		Capabilities: []string{"runtime"},
		ExePath:      "/fake/test-plugin",
	}
	p := NewPlugin(manifest, starter)

	host := &mockHost{sites: []registry.Site{{Domain: "example.test"}}}

	if err := p.Init(host); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	expected := []string{"plugin.init", "plugin.start", "plugin.stop"}
	if len(methods) != len(expected) {
		t.Fatalf("methods = %v, want %v", methods, expected)
	}
	for i, m := range methods {
		if m != expected[i] {
			t.Errorf("methods[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestExternalPluginHandles(t *testing.T) {
	starter := fakeProcessStarter(func(method string, params json.RawMessage) (any, error) {
		if method == "plugin.handles" {
			return map[string]bool{"handles": true}, nil
		}
		return map[string]any{}, nil
	})

	manifest := discovery.PluginManifest{
		ID:           "test-rt",
		Name:         "Test RT",
		Capabilities: []string{"runtime"},
		ExePath:      "/fake/test-rt",
	}
	p := NewPlugin(manifest, starter)
	host := &mockHost{}
	p.Init(host)

	site := registry.Site{Domain: "test.test"}
	if !p.Handles(site) {
		t.Error("expected Handles() = true")
	}
}

func TestExternalPluginUpstreamFor(t *testing.T) {
	starter := fakeProcessStarter(func(method string, params json.RawMessage) (any, error) {
		if method == "plugin.upstreamFor" {
			return map[string]string{"upstream": "localhost:3000"}, nil
		}
		return map[string]any{}, nil
	})

	manifest := discovery.PluginManifest{
		ID:           "test-rt",
		Name:         "Test RT",
		Capabilities: []string{"runtime"},
		ExePath:      "/fake/test-rt",
	}
	p := NewPlugin(manifest, starter)
	host := &mockHost{}
	p.Init(host)

	upstream, err := p.UpstreamFor(registry.Site{Domain: "test.test"})
	if err != nil {
		t.Fatalf("UpstreamFor failed: %v", err)
	}
	if upstream != "localhost:3000" {
		t.Errorf("upstream = %q, want localhost:3000", upstream)
	}
}

func TestExternalPluginServiceStatus(t *testing.T) {
	starter := fakeProcessStarter(func(method string, params json.RawMessage) (any, error) {
		if method == "plugin.serviceStatus" {
			return map[string]int{"status": int(plugin.ServiceRunning)}, nil
		}
		return map[string]any{}, nil
	})

	manifest := discovery.PluginManifest{
		ID:           "test-svc",
		Name:         "Test Svc",
		Capabilities: []string{"service"},
		ExePath:      "/fake/test-svc",
	}
	p := NewPlugin(manifest, starter)
	host := &mockHost{}
	p.Init(host)

	status := p.ServiceStatus()
	if status != plugin.ServiceRunning {
		t.Errorf("ServiceStatus() = %v, want ServiceRunning", status)
	}
}

func TestExternalPluginInitError(t *testing.T) {
	starter := fakeProcessStarter(func(method string, params json.RawMessage) (any, error) {
		if method == "plugin.init" {
			return nil, fmt.Errorf("init failed")
		}
		return map[string]any{}, nil
	})

	manifest := discovery.PluginManifest{
		ID:      "fail-plugin",
		Name:    "Fail",
		ExePath: "/fake/fail",
		Capabilities: []string{"runtime"},
	}
	p := NewPlugin(manifest, starter)
	host := &mockHost{}

	err := p.Init(host)
	if err == nil {
		t.Fatal("expected Init error")
	}
	if !strings.Contains(err.Error(), "init failed") {
		t.Errorf("error = %q, want contains 'init failed'", err.Error())
	}
}

// mockHost implements plugin.Host for tests
type mockHost struct {
	sites []registry.Site
}

func (h *mockHost) Sites() []registry.Site                        { return h.sites }
func (h *mockHost) GetSite(domain string) (registry.Site, bool)   { return registry.Site{}, false }
func (h *mockHost) Log(pluginID string, msg string, args ...any)  {}
```

**Step 2: Run tests to verify they fail**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -v`
Expected: FAIL — types not defined

**Step 3: Write implementation**

```go
package external

import (
	"fmt"
	"io"

	"github.com/andybarilla/flock/internal/discovery"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// Process represents a running plugin subprocess.
type Process interface {
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Kill() error
	Wait() error
}

// ProcessStarter launches a plugin executable and returns a Process.
type ProcessStarter func(exePath string) (Process, error)

// ExternalPlugin adapts a subprocess-based plugin to the Plugin/RuntimePlugin/ServicePlugin interfaces.
type ExternalPlugin struct {
	manifest   discovery.PluginManifest
	starter    ProcessStarter
	process    Process
	rpc        *rpcClient
	isRuntime  bool
	isService  bool
}

func NewPlugin(manifest discovery.PluginManifest, starter ProcessStarter) *ExternalPlugin {
	p := &ExternalPlugin{
		manifest: manifest,
		starter:  starter,
	}
	for _, cap := range manifest.Capabilities {
		switch cap {
		case "runtime":
			p.isRuntime = true
		case "service":
			p.isService = true
		}
	}
	return p
}

func (p *ExternalPlugin) ID() string   { return p.manifest.ID }
func (p *ExternalPlugin) Name() string { return p.manifest.Name }

func (p *ExternalPlugin) Init(host plugin.Host) error {
	proc, err := p.starter(p.manifest.ExePath)
	if err != nil {
		return fmt.Errorf("start plugin process: %w", err)
	}
	p.process = proc
	p.rpc = newRPCClient(proc.Stdout(), proc.Stdin())

	params := map[string]any{
		"sites": host.Sites(),
	}
	return p.rpc.Call("plugin.init", params, nil)
}

func (p *ExternalPlugin) Start() error {
	return p.rpc.Call("plugin.start", nil, nil)
}

func (p *ExternalPlugin) Stop() error {
	err := p.rpc.Call("plugin.stop", nil, nil)
	if p.process != nil {
		p.process.Kill()
		p.process.Wait()
	}
	return err
}

func (p *ExternalPlugin) Handles(site registry.Site) bool {
	if !p.isRuntime {
		return false
	}
	var result struct {
		Handles bool `json:"handles"`
	}
	if err := p.rpc.Call("plugin.handles", map[string]any{"site": site}, &result); err != nil {
		return false
	}
	return result.Handles
}

func (p *ExternalPlugin) UpstreamFor(site registry.Site) (string, error) {
	var result struct {
		Upstream string `json:"upstream"`
	}
	if err := p.rpc.Call("plugin.upstreamFor", map[string]any{"site": site}, &result); err != nil {
		return "", err
	}
	return result.Upstream, nil
}

func (p *ExternalPlugin) ServiceStatus() plugin.ServiceStatus {
	if !p.isService {
		return plugin.ServiceStopped
	}
	var result struct {
		Status plugin.ServiceStatus `json:"status"`
	}
	if err := p.rpc.Call("plugin.serviceStatus", nil, &result); err != nil {
		return plugin.ServiceDegraded
	}
	return result.Status
}

func (p *ExternalPlugin) StartService() error {
	return p.rpc.Call("plugin.startService", nil, nil)
}

func (p *ExternalPlugin) StopService() error {
	return p.rpc.Call("plugin.stopService", nil, nil)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -v`
Expected: PASS (all tests)

**Step 5: Commit**

```bash
git add internal/external/plugin.go internal/external/plugin_test.go
git commit -m "feat: add ExternalPlugin adapter with JSON-RPC subprocess proxy"
```

---

### Task 5: Real Process Starter (os/exec)

**Files:**
- Create: `internal/external/process.go`
- Create: `internal/external/process_test.go`

**Step 1: Write failing test for real process starter**

```go
package external

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecProcessStarter(t *testing.T) {
	// Create a tiny script that reads one line from stdin and echoes a JSON-RPC response
	dir := t.TempDir()
	var script, scriptPath string
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	script = `#!/bin/sh
read line
echo '{"jsonrpc":"2.0","id":1,"result":{}}'
`
	scriptPath = filepath.Join(dir, "test-plugin")
	os.WriteFile(scriptPath, []byte(script), 0o755)

	proc, err := ExecProcessStarter(scriptPath)
	if err != nil {
		t.Fatalf("ExecProcessStarter failed: %v", err)
	}

	rpc := newRPCClient(proc.Stdout(), proc.Stdin())
	err = rpc.Call("plugin.init", nil, nil)
	if err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}

	proc.Kill()
	proc.Wait()
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -run TestExecProcessStarter -v`
Expected: FAIL — `ExecProcessStarter` not defined

**Step 3: Write implementation**

```go
package external

import (
	"io"
	"os/exec"
)

type execProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *execProcess) Stdin() io.WriteCloser  { return p.stdin }
func (p *execProcess) Stdout() io.ReadCloser  { return p.stdout }
func (p *execProcess) Kill() error             { return p.cmd.Process.Kill() }
func (p *execProcess) Wait() error             { return p.cmd.Wait() }

func ExecProcessStarter(exePath string) (Process, error) {
	cmd := exec.Command(exePath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execProcess{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -run TestExecProcessStarter -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/external/process.go internal/external/process_test.go
git commit -m "feat: add os/exec-based process starter for external plugins"
```

---

### Task 6: Core Integration

**Files:**
- Modify: `internal/core/core.go`
- Modify: `internal/core/core_test.go`

**Step 1: Write failing test for external plugin registration**

Add to `internal/core/core_test.go`:

```go
func TestExternalPluginsRegistered(t *testing.T) {
	cfg := testConfig(t)

	// Create a plugins directory with a fake plugin manifest
	pluginsDir := filepath.Join(t.TempDir(), "plugins")
	os.MkdirAll(filepath.Join(pluginsDir, "test-ext"), 0o755)

	manifest := `{"id":"test-ext","name":"Test External","version":"0.1.0","executable":"test-ext","capabilities":["runtime"]}`
	os.WriteFile(filepath.Join(pluginsDir, "test-ext", "plugin.json"), []byte(manifest), 0o644)

	// Create a fake executable (won't be called since we won't start)
	os.WriteFile(filepath.Join(pluginsDir, "test-ext", "test-ext"), []byte("#!/bin/sh\n"), 0o755)

	cfg.PluginsDir = pluginsDir

	c := core.NewCore(cfg)
	plugins := c.Plugins()

	// 3 built-in + 1 external
	if len(plugins) != 4 {
		t.Fatalf("expected 4 plugins, got %d: %v", len(plugins), plugins)
	}

	found := false
	for _, p := range plugins {
		if p.ID == "test-ext" {
			found = true
			break
		}
	}
	if !found {
		t.Error("external plugin 'test-ext' not found in plugins list")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/core/ -run TestExternalPluginsRegistered -v`
Expected: FAIL — `PluginsDir` field doesn't exist on Config

**Step 3: Write implementation**

Update `internal/core/core.go` — add `PluginsDir string` to Config, scan and register external plugins in `NewCore()`:

```go
// Add to Config struct:
PluginsDir string

// Add to NewCore(), after registering built-in plugins:
manifests, errs := discovery.Scan(cfg.PluginsDir)
for _, err := range errs {
    cfg.Logger.Printf("plugin discovery: %v", err)
}
for _, m := range manifests {
    ext := external.NewPlugin(m, external.ExecProcessStarter)
    pluginMgr.Register(ext)
}
```

Update `app.go` — pass `config.PluginsDir()` to `core.Config.PluginsDir`.

**Step 4: Run test to verify it passes**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/core/ -run TestExternalPluginsRegistered -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./...`
Expected: PASS (all existing tests still pass)

**Step 6: Commit**

```bash
git add internal/core/core.go internal/core/core_test.go app.go
git commit -m "feat: integrate external plugin discovery into core startup"
```

---

### Task 7: Integration Test with Real Plugin Executable

**Files:**
- Create: `internal/external/testdata/echo-plugin.go` (test helper — tiny Go program)
- Modify: `internal/external/plugin_test.go` (add integration test)

**Step 1: Create a test plugin executable**

Write a tiny Go program that implements the JSON-RPC protocol as a test fixture:

```go
//go:build ignore

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		var result any
		switch req.Method {
		case "plugin.init":
			result = map[string]any{}
		case "plugin.start":
			result = map[string]any{}
		case "plugin.stop":
			result = map[string]any{}
		case "plugin.handles":
			result = map[string]bool{"handles": true}
		case "plugin.upstreamFor":
			result = map[string]string{"upstream": "localhost:3000"}
		case "plugin.serviceStatus":
			result = map[string]int{"status": 1}
		case "plugin.startService":
			result = map[string]any{}
		case "plugin.stopService":
			result = map[string]any{}
		default:
			resp, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]any{"code": -32601, "message": "method not found"},
			})
			fmt.Fprintln(os.Stdout, string(resp))
			continue
		}

		resp, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		})
		fmt.Fprintln(os.Stdout, string(resp))
	}
}
```

**Step 2: Write integration test**

```go
func TestExternalPluginIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the test plugin
	dir := t.TempDir()
	exePath := filepath.Join(dir, "echo-plugin")
	cmd := exec.Command("go", "build", "-o", exePath, "./testdata/echo-plugin.go")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test plugin: %v\n%s", err, out)
	}

	manifest := discovery.PluginManifest{
		ID:           "echo-plugin",
		Name:         "Echo Plugin",
		Version:      "0.1.0",
		Capabilities: []string{"runtime", "service"},
		ExePath:      exePath,
	}
	p := NewPlugin(manifest, ExecProcessStarter)

	host := &mockHost{sites: []registry.Site{{Domain: "test.test", TLS: true}}}

	// Full lifecycle
	if err := p.Init(host); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !p.Handles(registry.Site{Domain: "test.test"}) {
		t.Error("expected Handles() = true")
	}

	upstream, err := p.UpstreamFor(registry.Site{Domain: "test.test"})
	if err != nil {
		t.Fatalf("UpstreamFor: %v", err)
	}
	if upstream != "localhost:3000" {
		t.Errorf("upstream = %q, want localhost:3000", upstream)
	}

	status := p.ServiceStatus()
	if status != plugin.ServiceRunning {
		t.Errorf("ServiceStatus = %v, want ServiceRunning", status)
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
```

**Step 3: Run integration test**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./internal/external/ -run TestExternalPluginIntegration -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/external/testdata/ internal/external/plugin_test.go
git commit -m "test: add integration test with real plugin subprocess"
```

---

### Task 8: Final Verification & Cleanup

**Files:**
- Modify: `docs/ROADMAP.md`
- Modify: `docs/tasks/010-plugin-discovery.md`

**Step 1: Run full test suite**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go test ./... -v`
Expected: ALL PASS

**Step 2: Run go vet**

Run: `cd /home/andy/dev/andybarilla/flock.git/agent-1 && go vet ./...`
Expected: No issues

**Step 3: Update roadmap**

Mark "Plugin discovery and loading API" as complete in `docs/ROADMAP.md`.

**Step 4: Commit**

```bash
git add docs/ROADMAP.md docs/tasks/010-plugin-discovery.md
git commit -m "docs: mark plugin discovery and loading API as complete"
```
