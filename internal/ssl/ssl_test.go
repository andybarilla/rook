package ssl_test

import (
	"testing"

	"github.com/andybarilla/rook/internal/plugin"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/ssl"
)

// --- Mock CertStore ---

type mockCertStore struct {
	installCACalls int
	generateCalls  map[string]int
	certs          map[string]bool
	installCAErr   error
	generateErr    error
}

func newMockCertStore() *mockCertStore {
	return &mockCertStore{
		generateCalls: map[string]int{},
		certs:         map[string]bool{},
	}
}

func (m *mockCertStore) InstallCA() error {
	m.installCACalls++
	return m.installCAErr
}

func (m *mockCertStore) GenerateCert(domain string) error {
	m.generateCalls[domain]++
	if m.generateErr != nil {
		return m.generateErr
	}
	m.certs[domain] = true
	return nil
}

func (m *mockCertStore) CertPath(domain string) string {
	return "/certs/" + domain + ".pem"
}

func (m *mockCertStore) KeyPath(domain string) string {
	return "/certs/" + domain + "-key.pem"
}

func (m *mockCertStore) HasCert(domain string) bool {
	return m.certs[domain]
}

// --- Mock Host ---

type mockHost struct {
	sites []registry.Site
}

func (m *mockHost) Sites() []registry.Site {
	return m.sites
}

func (m *mockHost) GetSite(domain string) (registry.Site, bool) {
	for _, s := range m.sites {
		if s.Domain == domain {
			return s, true
		}
	}
	return registry.Site{}, false
}

func (m *mockHost) Log(pluginID string, msg string, args ...any) {}

// --- Tests ---

func TestPluginIDAndName(t *testing.T) {
	p := ssl.NewPlugin(newMockCertStore())
	if p.ID() != "rook-ssl" {
		t.Errorf("ID = %q, want rook-ssl", p.ID())
	}
	if p.Name() != "Rook SSL" {
		t.Errorf("Name = %q, want Rook SSL", p.Name())
	}
}

func TestInitInstallsCA(t *testing.T) {
	store := newMockCertStore()
	p := ssl.NewPlugin(store)

	host := &mockHost{}
	if err := p.Init(host); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if store.installCACalls != 1 {
		t.Errorf("installCACalls = %d, want 1", store.installCACalls)
	}
}

func TestStartGeneratesCertsForTLSSites(t *testing.T) {
	store := newMockCertStore()
	p := ssl.NewPlugin(store)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app", Domain: "app.test", TLS: true},
		{Path: "/docs", Domain: "docs.test", TLS: false},
		{Path: "/api", Domain: "api.test", TLS: true},
	}}
	_ = p.Init(host)
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if store.generateCalls["app.test"] != 1 {
		t.Errorf("generateCalls[app.test] = %d, want 1", store.generateCalls["app.test"])
	}
	if store.generateCalls["docs.test"] != 0 {
		t.Errorf("generateCalls[docs.test] = %d, want 0", store.generateCalls["docs.test"])
	}
	if store.generateCalls["api.test"] != 1 {
		t.Errorf("generateCalls[api.test] = %d, want 1", store.generateCalls["api.test"])
	}
}

func TestStartSkipsExistingCerts(t *testing.T) {
	store := newMockCertStore()
	store.certs["app.test"] = true
	p := ssl.NewPlugin(store)

	host := &mockHost{sites: []registry.Site{
		{Path: "/app", Domain: "app.test", TLS: true},
	}}
	_ = p.Init(host)
	if err := p.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if store.generateCalls["app.test"] != 0 {
		t.Errorf("generateCalls[app.test] = %d, want 0 (cert already exists)", store.generateCalls["app.test"])
	}
}

func TestCertPairReturnsPaths(t *testing.T) {
	store := newMockCertStore()
	store.certs["app.test"] = true
	p := ssl.NewPlugin(store)

	cert, key, err := p.CertPair("app.test")
	if err != nil {
		t.Fatalf("CertPair: %v", err)
	}
	if cert != "/certs/app.test.pem" {
		t.Errorf("cert = %q, want /certs/app.test.pem", cert)
	}
	if key != "/certs/app.test-key.pem" {
		t.Errorf("key = %q, want /certs/app.test-key.pem", key)
	}
}

func TestCertPairErrorsIfNoCert(t *testing.T) {
	store := newMockCertStore()
	p := ssl.NewPlugin(store)

	_, _, err := p.CertPair("missing.test")
	if err == nil {
		t.Error("expected error for missing cert")
	}
}

func TestServiceStatus(t *testing.T) {
	store := newMockCertStore()
	p := ssl.NewPlugin(store)

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Errorf("initial status = %d, want ServiceStopped", p.ServiceStatus())
	}

	host := &mockHost{}
	_ = p.Init(host)
	_ = p.Start()

	if p.ServiceStatus() != plugin.ServiceRunning {
		t.Errorf("after start status = %d, want ServiceRunning", p.ServiceStatus())
	}

	_ = p.Stop()

	if p.ServiceStatus() != plugin.ServiceStopped {
		t.Errorf("after stop status = %d, want ServiceStopped", p.ServiceStatus())
	}
}

func TestStopIsNoOp(t *testing.T) {
	store := newMockCertStore()
	p := ssl.NewPlugin(store)
	host := &mockHost{}
	_ = p.Init(host)

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
