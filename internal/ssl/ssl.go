package ssl

import (
	"fmt"

	"github.com/andybarilla/flock/internal/plugin"
)

type CertStore interface {
	InstallCA() error
	GenerateCert(domain string) error
	CertPath(domain string) string
	KeyPath(domain string) string
	HasCert(domain string) bool
}

type Plugin struct {
	store  CertStore
	host   plugin.Host
	status plugin.ServiceStatus
}

func NewPlugin(store CertStore) *Plugin {
	return &Plugin{store: store, status: plugin.ServiceStopped}
}

func (p *Plugin) ID() string   { return "flock-ssl" }
func (p *Plugin) Name() string { return "Flock SSL" }

func (p *Plugin) Init(host plugin.Host) error {
	p.host = host
	if err := p.store.InstallCA(); err != nil {
		p.status = plugin.ServiceDegraded
		return fmt.Errorf("install CA: %w", err)
	}
	return nil
}

func (p *Plugin) Start() error {
	for _, site := range p.host.Sites() {
		if !site.TLS || p.store.HasCert(site.Domain) {
			continue
		}
		if err := p.store.GenerateCert(site.Domain); err != nil {
			p.host.Log(p.ID(), "failed to generate cert for %s: %v", site.Domain, err)
		}
	}
	p.status = plugin.ServiceRunning
	return nil
}

func (p *Plugin) Stop() error {
	p.status = plugin.ServiceStopped
	return nil
}

func (p *Plugin) ServiceStatus() plugin.ServiceStatus {
	return p.status
}

func (p *Plugin) StartService() error { return p.Start() }
func (p *Plugin) StopService() error  { return p.Stop() }

func (p *Plugin) CertPair(domain string) (string, string, error) {
	if !p.store.HasCert(domain) {
		return "", "", fmt.Errorf("no certificate for %s", domain)
	}
	return p.store.CertPath(domain), p.store.KeyPath(domain), nil
}
