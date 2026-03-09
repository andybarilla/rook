package core

import (
	"fmt"
	"log"

	"github.com/andybarilla/rook/internal/caddy"
	"github.com/andybarilla/rook/internal/databases"
	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/external"
	"github.com/andybarilla/rook/internal/mise"
	"github.com/andybarilla/rook/internal/node"
	"github.com/andybarilla/rook/internal/php"
	"github.com/andybarilla/rook/internal/plugin"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/ssl"
)

type Config struct {
	SitesFile    string
	Logger       *log.Logger
	CaddyRunner  caddy.CaddyRunner
	FPMRunner    php.FPMRunner
	CertStore    ssl.CertStore
	DBRunner     databases.DBRunner
	DBConfigPath string
	DBDataRoot   string
	NodeRunner   node.NodeRunner
	PluginsDir   string
	Resolver     *mise.RuntimeResolver
}

type Core struct {
	registry   *registry.Registry
	pluginMgr  *plugin.Manager
	caddyMgr   *caddy.Manager
	sslPlugin  *ssl.Plugin
	phpPlugin  *php.Plugin
	nodePlugin *node.Plugin
	dbPlugin   *databases.Plugin
	logger     *log.Logger
	resolver   *mise.RuntimeResolver
}

type MiseInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
}

type RuntimeStatus struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	Domain    string `json:"domain"`
}

func NewCore(cfg Config) *Core {
	resolver := cfg.Resolver
	if resolver == nil {
		resolver = mise.New()
	}

	reg := registry.New(cfg.SitesFile)
	pluginMgr := plugin.NewManager(reg, cfg.Logger)

	sslPlugin := ssl.NewPlugin(cfg.CertStore)
	phpPlugin := php.NewPlugin(cfg.FPMRunner)
	nodePlugin := node.NewPlugin(cfg.NodeRunner)
	dbPlugin := databases.NewPlugin(cfg.DBRunner, cfg.DBConfigPath, cfg.DBDataRoot)
	pluginMgr.Register(sslPlugin)
	pluginMgr.Register(phpPlugin)
	pluginMgr.Register(nodePlugin)
	pluginMgr.Register(dbPlugin)

	manifests, scanErrs := discovery.Scan(cfg.PluginsDir)
	for _, err := range scanErrs {
		cfg.Logger.Printf("plugin discovery: %v", err)
	}
	for _, m := range manifests {
		ext := external.NewPlugin(m, external.ExecProcessStarter)
		pluginMgr.Register(ext)
	}

	caddyMgr := caddy.NewManager(cfg.CaddyRunner, pluginMgr, sslPlugin)

	c := &Core{
		registry:   reg,
		pluginMgr:  pluginMgr,
		caddyMgr:   caddyMgr,
		sslPlugin:  sslPlugin,
		phpPlugin:  phpPlugin,
		nodePlugin: nodePlugin,
		dbPlugin:   dbPlugin,
		logger:     cfg.Logger,
		resolver:   resolver,
	}

	reg.OnChange(func(e registry.ChangeEvent) {
		if err := c.caddyMgr.Reload(c.registry.List()); err != nil {
			c.logger.Printf("caddy reload failed: %v", err)
		}
	})

	return c
}

func (c *Core) Start() error {
	if err := c.registry.Load(); err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	c.pluginMgr.InitAll()
	c.pluginMgr.StartAll()

	if err := c.caddyMgr.Start(c.registry.List()); err != nil {
		return fmt.Errorf("start caddy: %w", err)
	}

	return nil
}

func (c *Core) Stop() error {
	if err := c.caddyMgr.Stop(); err != nil {
		c.logger.Printf("caddy stop failed: %v", err)
	}
	c.pluginMgr.StopAll()
	return nil
}

func (c *Core) Sites() []registry.Site {
	return c.registry.List()
}

func (c *Core) GetSite(domain string) (registry.Site, bool) {
	return c.registry.Get(domain)
}

func (c *Core) AddSite(site registry.Site) error {
	return c.registry.Add(site)
}

func (c *Core) RemoveSite(domain string) error {
	return c.registry.Remove(domain)
}

func (c *Core) UpdateSite(domain string, updated registry.Site) error {
	return c.registry.Update(domain, func(s *registry.Site) {
		s.Path = updated.Path
		s.PHPVersion = updated.PHPVersion
		s.NodeVersion = updated.NodeVersion
		s.TLS = updated.TLS
	})
}

func (c *Core) Plugins() []plugin.PluginInfo {
	return c.pluginMgr.Plugins()
}

func (c *Core) DatabaseServices() []databases.ServiceInfo {
	return c.dbPlugin.ServiceStatuses()
}

func (c *Core) StartDatabase(svc string) error {
	return c.dbPlugin.StartSvc(databases.ServiceType(svc))
}

func (c *Core) StopDatabase(svc string) error {
	return c.dbPlugin.StopSvc(databases.ServiceType(svc))
}

func (c *Core) DetectSiteVersions(path string) (map[string]string, error) {
	return c.resolver.Detect(path)
}

func (c *Core) MiseStatus() MiseInfo {
	available, ver := c.resolver.Available()
	return MiseInfo{
		Available: available,
		Version:   ver,
	}
}

func (c *Core) CheckRuntimes() []RuntimeStatus {
	var statuses []RuntimeStatus
	for _, site := range c.registry.List() {
		if site.PHPVersion != "" {
			statuses = append(statuses, RuntimeStatus{
				Tool:      "php",
				Version:   site.PHPVersion,
				Installed: c.resolver.IsInstalled("php", site.PHPVersion),
				Domain:    site.Domain,
			})
		}
		if site.NodeVersion != "" {
			statuses = append(statuses, RuntimeStatus{
				Tool:      "node",
				Version:   site.NodeVersion,
				Installed: c.resolver.IsInstalled("node", site.NodeVersion),
				Domain:    site.Domain,
			})
		}
	}
	return statuses
}

func (c *Core) InstallRuntime(tool, version string) error {
	return c.resolver.Install(tool, version)
}
