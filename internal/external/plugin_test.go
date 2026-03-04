package external

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/andybarilla/flock/internal/discovery"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

// pipeProcess connects stdin/stdout via pipes for testing
type pipeProcess struct {
	stdinW  *io.PipeWriter
	stdoutR *io.PipeReader
}

func (p *pipeProcess) Stdin() io.WriteCloser  { return p.stdinW }
func (p *pipeProcess) Stdout() io.ReadCloser  { return p.stdoutR }
func (p *pipeProcess) Kill() error             { p.stdinW.Close(); p.stdoutR.Close(); return nil }
func (p *pipeProcess) Wait() error             { return nil }

// fakeProcessStarter returns a ProcessStarter that simulates a plugin subprocess
func fakeProcessStarter(handler func(method string, params json.RawMessage) (any, error)) ProcessStarter {
	return func(exePath string) (Process, error) {
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

// mockHost implements plugin.Host for tests
type mockHost struct {
	sites []registry.Site
}

func (h *mockHost) Sites() []registry.Site                       { return h.sites }
func (h *mockHost) GetSite(domain string) (registry.Site, bool)  { return registry.Site{}, false }
func (h *mockHost) Log(pluginID string, msg string, args ...any) {}

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
		ID:           "fail-plugin",
		Name:         "Fail",
		ExePath:      "/fake/fail",
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

func TestExternalPluginIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Build the test plugin
	dir := t.TempDir()
	exeName := "echo-plugin"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	exePath := filepath.Join(dir, exeName)
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
