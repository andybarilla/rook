package plugin

import (
	"fmt"
	"log"

	"github.com/andybarilla/flock/internal/registry"
)

type pluginEntry struct {
	plugin Plugin
	status PluginStatus
}

type Manager struct {
	registry SiteSource
	logger   *log.Logger
	plugins  []pluginEntry
}

func NewManager(src SiteSource, logger *log.Logger) *Manager {
	return &Manager{registry: src, logger: logger}
}

func (m *Manager) Register(p Plugin) {
	m.plugins = append(m.plugins, pluginEntry{plugin: p, status: PluginReady})
}

func (m *Manager) InitAll() {
	for i := range m.plugins {
		e := &m.plugins[i]
		if err := e.plugin.Init(m); err != nil {
			m.logger.Printf("plugin %q init failed: %v", e.plugin.ID(), err)
			e.status = PluginDegraded
		}
	}
}

func (m *Manager) StartAll() {
	for i := range m.plugins {
		e := &m.plugins[i]
		if e.status == PluginDegraded {
			continue
		}
		if err := e.plugin.Start(); err != nil {
			m.logger.Printf("plugin %q start failed: %v", e.plugin.ID(), err)
			e.status = PluginDegraded
		}
	}
}

func (m *Manager) StopAll() {
	for i := len(m.plugins) - 1; i >= 0; i-- {
		e := &m.plugins[i]
		if err := e.plugin.Stop(); err != nil {
			m.logger.Printf("plugin %q stop failed: %v", e.plugin.ID(), err)
		}
	}
}

func (m *Manager) ResolveUpstream(site registry.Site) (string, error) {
	for _, e := range m.plugins {
		rp, ok := e.plugin.(RuntimePlugin)
		if !ok {
			continue
		}
		if rp.Handles(site) {
			return rp.UpstreamFor(site)
		}
	}
	return "", nil
}

func (m *Manager) Plugins() []PluginInfo {
	out := make([]PluginInfo, len(m.plugins))
	for i, e := range m.plugins {
		out[i] = PluginInfo{
			ID:     e.plugin.ID(),
			Name:   e.plugin.Name(),
			Status: e.status,
		}
	}
	return out
}

// Host interface implementation

func (m *Manager) Sites() []registry.Site {
	return m.registry.List()
}

func (m *Manager) GetSite(domain string) (registry.Site, bool) {
	return m.registry.Get(domain)
}

func (m *Manager) Log(pluginID string, msg string, args ...any) {
	m.logger.Printf("[%s] %s", pluginID, fmt.Sprintf(msg, args...))
}
