package databases

import (
	"github.com/andybarilla/flock/internal/plugin"
)

// Plugin manages MySQL, PostgreSQL, and Redis services.
type Plugin struct {
	runner        DBRunner
	host          plugin.Host
	configPath    string
	dataRoot      string
	config        Config
	running       map[ServiceType]bool
	binaryChecker func(string) bool
}

// NewPlugin creates a databases plugin.
func NewPlugin(runner DBRunner, configPath, dataRoot string) *Plugin {
	return &Plugin{
		runner:        runner,
		configPath:    configPath,
		dataRoot:      dataRoot,
		running:       map[ServiceType]bool{},
		binaryChecker: CheckBinary,
	}
}

// SetBinaryChecker overrides the default binary detection function.
func (p *Plugin) SetBinaryChecker(fn func(string) bool) {
	p.binaryChecker = fn
}

func (p *Plugin) ID() string   { return "flock-databases" }
func (p *Plugin) Name() string { return "Flock Databases" }

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	cfg, err := LoadConfig(p.configPath, p.dataRoot)
	if err != nil {
		return err
	}
	p.config = cfg
	for _, svc := range AllServiceTypes {
		if !p.binaryChecker(BinaryFor(svc)) {
			p.config.SetEnabled(svc, false)
			p.host.Log(p.ID(), "%s binary not found on PATH", svc)
		}
	}
	return nil
}

func (p *Plugin) Start() error {
	for _, svc := range AllServiceTypes {
		svcCfg := p.config.ForType(svc)
		if !svcCfg.Enabled || !svcCfg.Autostart {
			continue
		}
		if err := p.runner.Start(svc, ServiceConfig{Port: svcCfg.Port, DataDir: svcCfg.DataDir}); err != nil {
			p.host.Log(p.ID(), "autostart %s failed: %v", svc, err)
			continue
		}
		p.running[svc] = true
	}
	return nil
}

func (p *Plugin) Stop() error {
	for svc := range p.running {
		if err := p.runner.Stop(svc); err != nil {
			p.host.Log(p.ID(), "stop %s failed: %v", svc, err)
		}
		delete(p.running, svc)
	}
	return nil
}

// isRunning checks whether a service is actually running by querying the
// runner and reconciling the in-memory map with real process state.
func (p *Plugin) isRunning(svc ServiceType) bool {
	if !p.running[svc] {
		return false
	}
	if p.runner.Status(svc) != StatusRunning {
		delete(p.running, svc)
		return false
	}
	return true
}

func (p *Plugin) ServiceStatus() plugin.ServiceStatus {
	for _, svc := range AllServiceTypes {
		if p.isRunning(svc) {
			return plugin.ServiceRunning
		}
	}
	return plugin.ServiceStopped
}

func (p *Plugin) StartService() error { return p.Start() }
func (p *Plugin) StopService() error  { return p.Stop() }

// StartSvc starts a specific database service.
func (p *Plugin) StartSvc(svc ServiceType) error {
	svcCfg := p.config.ForType(svc)
	if err := p.runner.Start(svc, ServiceConfig{Port: svcCfg.Port, DataDir: svcCfg.DataDir}); err != nil {
		return err
	}
	p.running[svc] = true
	return nil
}

// StopSvc stops a specific database service.
func (p *Plugin) StopSvc(svc ServiceType) error {
	if err := p.runner.Stop(svc); err != nil {
		return err
	}
	delete(p.running, svc)
	return nil
}

// ServiceStatuses returns status info for all services.
func (p *Plugin) ServiceStatuses() []ServiceInfo {
	var infos []ServiceInfo
	for _, svc := range AllServiceTypes {
		svcCfg := p.config.ForType(svc)
		infos = append(infos, ServiceInfo{
			Type:      svc,
			Enabled:   svcCfg.Enabled,
			Running:   p.isRunning(svc),
			Autostart: svcCfg.Autostart,
			Port:      svcCfg.Port,
		})
	}
	return infos
}
