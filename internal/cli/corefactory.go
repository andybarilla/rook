package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/andybarilla/flock/internal/config"
	"github.com/andybarilla/flock/internal/core"
	"github.com/andybarilla/flock/internal/databases"
	"github.com/andybarilla/flock/internal/node"
)

func NewCore() (*core.Core, func(), error) {
	logDir := config.DataDir()
	os.MkdirAll(logDir, 0o755)
	logFile, err := os.OpenFile(config.LogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logFile = os.Stderr
	}
	logger := log.New(logFile, "[flock-cli] ", log.LstdFlags)

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

type noopCaddyRunner struct{}

func (r *noopCaddyRunner) Run(cfgJSON []byte) error { return nil }
func (r *noopCaddyRunner) Stop() error              { return nil }

type noopFPMRunner struct{}

func (r *noopFPMRunner) StartPool(version string) error   { return nil }
func (r *noopFPMRunner) StopPool(version string) error    { return nil }
func (r *noopFPMRunner) PoolSocket(version string) string { return "" }

type noopCertStore struct{}

func (s *noopCertStore) InstallCA() error                 { return nil }
func (s *noopCertStore) GenerateCert(domain string) error { return nil }
func (s *noopCertStore) CertPath(domain string) string    { return "" }
func (s *noopCertStore) KeyPath(domain string) string     { return "" }
func (s *noopCertStore) HasCert(domain string) bool       { return false }
