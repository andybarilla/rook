package plugin

import (
	"github.com/andybarilla/flock/internal/registry"
)

type Plugin interface {
	ID() string
	Name() string
	Init(host Host) error
	Start() error
	Stop() error
}

type RuntimePlugin interface {
	Plugin
	Handles(site registry.Site) bool
	UpstreamFor(site registry.Site) (string, error)
}

type ServicePlugin interface {
	Plugin
	ServiceStatus() ServiceStatus
	StartService() error
	StopService() error
}

type Host interface {
	Sites() []registry.Site
	GetSite(domain string) (registry.Site, bool)
	Log(pluginID string, msg string, args ...any)
}

type SiteSource interface {
	List() []registry.Site
	Get(domain string) (registry.Site, bool)
}

type ServiceStatus int

const (
	ServiceStopped ServiceStatus = iota
	ServiceRunning
	ServiceDegraded
)

type PluginStatus int

const (
	PluginReady PluginStatus = iota
	PluginDegraded
)

type PluginInfo struct {
	ID     string
	Name   string
	Status PluginStatus
}
