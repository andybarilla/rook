package core

import (
	"fmt"
	"log"

	"github.com/andybarilla/flock/internal/caddy"
	"github.com/andybarilla/flock/internal/php"
	"github.com/andybarilla/flock/internal/plugin"
	"github.com/andybarilla/flock/internal/registry"
	"github.com/andybarilla/flock/internal/ssl"
)

type Config struct {
	SitesFile   string
	Logger      *log.Logger
	CaddyRunner caddy.CaddyRunner
	FPMRunner   php.FPMRunner
	CertStore   ssl.CertStore
}

type Core struct {
	registry  *registry.Registry
	pluginMgr *plugin.Manager
	caddyMgr  *caddy.Manager
	sslPlugin *ssl.Plugin
	phpPlugin *php.Plugin
	logger    *log.Logger
}

func NewCore(cfg Config) *Core {
	reg := registry.New(cfg.SitesFile)
	pluginMgr := plugin.NewManager(reg, cfg.Logger)

	sslPlugin := ssl.NewPlugin(cfg.CertStore)
	phpPlugin := php.NewPlugin(cfg.FPMRunner)
	pluginMgr.Register(sslPlugin)
	pluginMgr.Register(phpPlugin)

	caddyMgr := caddy.NewManager(cfg.CaddyRunner, pluginMgr, sslPlugin)

	c := &Core{
		registry:  reg,
		pluginMgr: pluginMgr,
		caddyMgr:  caddyMgr,
		sslPlugin: sslPlugin,
		phpPlugin: phpPlugin,
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
