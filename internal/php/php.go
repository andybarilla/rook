package php

import (
	"fmt"

	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
)

type FPMRunner interface {
	StartPool(version string) error
	StopPool(version string) error
	PoolSocket(version string) string
}

type Plugin struct {
	runner FPMRunner
	host   plugin.Host
	pools  map[string]bool // version → running
	status plugin.ServiceStatus
}

func NewPlugin(runner FPMRunner) *Plugin {
	return &Plugin{
		runner: runner,
		pools:  map[string]bool{},
		status: plugin.ServiceStopped,
	}
}

func (p *Plugin) ID() string   { return "flock-php" }
func (p *Plugin) Name() string { return "Flock PHP" }

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	return nil
}

func (p *Plugin) Start() error {
	versions := map[string]bool{}
	for _, site := range p.host.Sites() {
		if site.PHPVersion != "" {
			versions[site.PHPVersion] = true
		}
	}

	for version := range versions {
		if err := p.runner.StartPool(version); err != nil {
			p.host.Log(p.ID(), "failed to start pool for PHP %s: %v", version, err)
			continue
		}
		p.pools[version] = true
	}

	if len(p.pools) > 0 {
		p.status = plugin.ServiceRunning
	}
	return nil
}

func (p *Plugin) Stop() error {
	for version := range p.pools {
		if err := p.runner.StopPool(version); err != nil {
			p.host.Log(p.ID(), "failed to stop pool for PHP %s: %v", version, err)
		}
		delete(p.pools, version)
	}
	p.status = plugin.ServiceStopped
	return nil
}

func (p *Plugin) Handles(site registry.Site) bool {
	return site.PHPVersion != ""
}

func (p *Plugin) UpstreamFor(site registry.Site) (string, error) {
	if !p.pools[site.PHPVersion] {
		return "", fmt.Errorf("no running pool for PHP %s", site.PHPVersion)
	}
	return "unix/" + p.runner.PoolSocket(site.PHPVersion), nil
}

func (p *Plugin) ServiceStatus() plugin.ServiceStatus {
	return p.status
}

func (p *Plugin) StartService() error { return p.Start() }
func (p *Plugin) StopService() error  { return p.Stop() }
