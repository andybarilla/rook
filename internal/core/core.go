package core

import (
	"fmt"
	"log"

	"github.com/andybarilla/flock/internal/caddy"
	"github.com/andybarilla/flock/internal/databases"
	"github.com/andybarilla/flock/internal/discovery"
	"github.com/andybarilla/flock/internal/external"
	"github.com/andybarilla/flock/internal/php"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
	"github.com/andybarilla/flock/internal/ssl"
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
	PluginsDir   string
}

type Core struct {
	registry  *registry.Registry
	pluginMgr *plugin.Manager
	caddyMgr  *caddy.Manager
	sslPlugin *ssl.Plugin
	phpPlugin *php.Plugin
	dbPlugin  *databases.Plugin
	logger    *log.Logger
}

func NewCore(cfg Config) *Core {
	reg := registry.New(cfg.SitesFile)
	pluginMgr := plugin.NewManager(reg, cfg.Logger)

	sslPlugin := ssl.NewPlugin(cfg.CertStore)
	phpPlugin := php.NewPlugin(cfg.FPMRunner)
	dbPlugin := databases.NewPlugin(cfg.DBRunner, cfg.DBConfigPath, cfg.DBDataRoot)
	pluginMgr.Register(sslPlugin)
	pluginMgr.Register(phpPlugin)
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
		registry:  reg,
		pluginMgr: pluginMgr,
		caddyMgr:  caddyMgr,
		sslPlugin: sslPlugin,
		phpPlugin: phpPlugin,
		dbPlugin:  dbPlugin,
		logger:    cfg.Logger,
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

func (c *Core) AddSite(site registry.Site) error {
	return c.registry.Add(site)
}

func (c *Core) RemoveSite(domain string) error {
	return c.registry.Remove(domain)
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
