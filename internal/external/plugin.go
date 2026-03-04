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
	manifest  discovery.PluginManifest
	starter   ProcessStarter
	process   Process
	rpc       *rpcClient
	isRuntime bool
	isService bool
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
