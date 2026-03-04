package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PluginManifest struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Executable      string   `json:"executable"`
	Capabilities    []string `json:"capabilities"`
	MinFlockVersion string   `json:"minFlockVersion,omitempty"`
	ExePath         string   `json:"-"`
}

func Scan(dir string) ([]PluginManifest, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}

	var manifests []PluginManifest
	var errs []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m, err := loadManifest(filepath.Join(dir, entry.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("plugin %s: %w", entry.Name(), err))
			continue
		}
		manifests = append(manifests, m)
	}

	return manifests, errs
}

func loadManifest(pluginDir string) (PluginManifest, error) {
	data, err := os.ReadFile(filepath.Join(pluginDir, "plugin.json"))
	if err != nil {
		return PluginManifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return PluginManifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	if err := validate(m); err != nil {
		return PluginManifest{}, err
	}

	exePath := filepath.Join(pluginDir, m.Executable)
	info, err := os.Stat(exePath)
	if err != nil {
		return PluginManifest{}, fmt.Errorf("executable %q not found", m.Executable)
	}
	if info.IsDir() {
		return PluginManifest{}, fmt.Errorf("executable %q is a directory", m.Executable)
	}

	m.ExePath = exePath
	return m, nil
}

func validate(m PluginManifest) error {
	if m.ID == "" {
		return fmt.Errorf("missing required field: id")
	}
	if m.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if m.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if m.Executable == "" {
		return fmt.Errorf("missing required field: executable")
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("missing required field: capabilities")
	}
	return nil
}
