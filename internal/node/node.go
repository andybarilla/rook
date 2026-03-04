package node

import (
	"fmt"
	"sort"

	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

type NodeRunner interface {
	StartApp(siteDir string, port int) error
	StopApp(siteDir string) error
	IsRunning(siteDir string) bool
	AppPort(siteDir string) int
}

type Plugin struct {
	runner   NodeRunner
	host     plugin.Host
	basePort int
	portMap  map[string]int  // domain -> port
	apps     map[string]bool // siteDir -> running
	status   plugin.ServiceStatus
}

func NewPlugin(runner NodeRunner) *Plugin {
	return &Plugin{
		runner:   runner,
		basePort: 3100,
		portMap:  map[string]int{},
		apps:     map[string]bool{},
		status:   plugin.ServiceStopped,
	}
}

func (p *Plugin) ID() string   { return "flock-node" }
func (p *Plugin) Name() string { return "Flock Node" }

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	return nil
}

func (p *Plugin) Start() error {
	type nodeSite struct {
		domain  string
		siteDir string
	}

	var sites []nodeSite
	for _, site := range p.host.Sites() {
		if site.NodeVersion != "" {
			sites = append(sites, nodeSite{domain: site.Domain, siteDir: site.Path})
		}
	}

	sort.Slice(sites, func(i, j int) bool {
		return sites[i].domain < sites[j].domain
	})

	for i, s := range sites {
		port := p.basePort + i
		if err := p.runner.StartApp(s.siteDir, port); err != nil {
			p.host.Log(p.ID(), "failed to start app for %s: %v", s.domain, err)
			continue
		}
		p.portMap[s.domain] = port
		p.apps[s.siteDir] = true
	}

	if len(p.apps) > 0 {
		p.status = plugin.ServiceRunning
	}
	return nil
}

func (p *Plugin) Stop() error {
	for siteDir := range p.apps {
		if err := p.runner.StopApp(siteDir); err != nil {
			p.host.Log(p.ID(), "failed to stop app at %s: %v", siteDir, err)
		}
		delete(p.apps, siteDir)
	}
	p.portMap = map[string]int{}
	p.status = plugin.ServiceStopped
	return nil
}

func (p *Plugin) Handles(site registry.Site) bool {
	return site.NodeVersion != ""
}

func (p *Plugin) UpstreamFor(site registry.Site) (string, error) {
	port, ok := p.portMap[site.Domain]
	if !ok {
		return "", fmt.Errorf("no running app for %s", site.Domain)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

func (p *Plugin) ServiceStatus() plugin.ServiceStatus {
	return p.status
}

func (p *Plugin) StartService() error { return p.Start() }
func (p *Plugin) StopService() error  { return p.Stop() }
