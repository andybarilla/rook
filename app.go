package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/andybarilla/flock/internal/config"
	"github.com/andybarilla/flock/internal/core"
	"github.com/andybarilla/flock/internal/registry"
)

// App struct
type App struct {
	ctx  context.Context
	core *core.Core
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	logDir := config.DataDir()
	os.MkdirAll(logDir, 0o755)
	logFile, err := os.OpenFile(config.LogFile(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		logFile = os.Stderr
	}
	logger := log.New(logFile, "[flock] ", log.LstdFlags)

	cfg := core.Config{
		SitesFile:   config.SitesFile(),
		Logger:      logger,
		CaddyRunner: &loggingCaddyRunner{logger: logger},
		FPMRunner:   &loggingFPMRunner{logger: logger},
		CertStore:   &loggingCertStore{logger: logger, certsDir: filepath.Join(config.DataDir(), "certs")},
	}

	a.core = core.NewCore(cfg)
	if err := a.core.Start(); err != nil {
		logger.Printf("core start failed: %v", err)
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.core != nil {
		a.core.Stop()
	}
}

// ListSites returns all registered sites
func (a *App) ListSites() []registry.Site {
	return a.core.Sites()
}

// AddSite registers a new site
func (a *App) AddSite(path, domain, phpVersion string, tls bool) error {
	return a.core.AddSite(registry.Site{
		Path:       path,
		Domain:     domain,
		PHPVersion: phpVersion,
		TLS:        tls,
	})
}

// RemoveSite removes a registered site
func (a *App) RemoveSite(domain string) error {
	return a.core.RemoveSite(domain)
}
