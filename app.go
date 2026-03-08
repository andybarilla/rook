package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/config"
	"github.com/andybarilla/rook/internal/core"
	"github.com/andybarilla/rook/internal/databases"
	"github.com/andybarilla/rook/internal/node"
	"github.com/andybarilla/rook/internal/registry"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
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
	logger := log.New(logFile, "[rook] ", log.LstdFlags)

	cfg := core.Config{
		SitesFile:    config.SitesFile(),
		Logger:       logger,
		CaddyRunner:  &loggingCaddyRunner{logger: logger},
		FPMRunner:    &loggingFPMRunner{logger: logger},
		CertStore:    &loggingCertStore{logger: logger, certsDir: filepath.Join(config.DataDir(), "certs")},
		DBRunner:     databases.NewProcessRunner(),
		NodeRunner:   node.NewProcessRunner(),
		DBConfigPath: filepath.Join(config.ConfigDir(), "databases.json"),
		DBDataRoot:   filepath.Join(config.DataDir(), "databases"),
		PluginsDir:   config.PluginsDir(),
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
func (a *App) AddSite(path, domain, phpVersion, nodeVersion string, tls bool) error {
	return a.core.AddSite(registry.Site{
		Path:        path,
		Domain:      domain,
		PHPVersion:  phpVersion,
		NodeVersion: nodeVersion,
		TLS:         tls,
	})
}

// SelectDirectory opens a native directory picker and returns the selected path
func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Project Directory",
	})
}

// RemoveSite removes a registered site
func (a *App) RemoveSite(domain string) error {
	return a.core.RemoveSite(domain)
}

// DatabaseServices returns status of all database services
func (a *App) DatabaseServices() []databases.ServiceInfo {
	return a.core.DatabaseServices()
}

// StartDatabase starts a specific database service
func (a *App) StartDatabase(svc string) error {
	return a.core.StartDatabase(svc)
}

// StopDatabase stops a specific database service
func (a *App) StopDatabase(svc string) error {
	return a.core.StopDatabase(svc)
}

// DetectSiteVersions scans a project directory for mise config files
// and returns detected runtime versions.
func (a *App) DetectSiteVersions(path string) (map[string]string, error) {
	return a.core.DetectSiteVersions(path)
}

// MiseStatus returns whether mise is available and its version.
func (a *App) MiseStatus() core.MiseInfo {
	return a.core.MiseStatus()
}

// CheckRuntimes returns installation status for all site runtimes.
func (a *App) CheckRuntimes() []core.RuntimeStatus {
	return a.core.CheckRuntimes()
}

// InstallRuntime installs a specific tool version via mise.
func (a *App) InstallRuntime(tool, version string) error {
	return a.core.InstallRuntime(tool, version)
}
