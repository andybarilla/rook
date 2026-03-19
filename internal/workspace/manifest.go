package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func ParseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validating manifest: %w", err)
	}
	return &m, nil
}

func WriteManifest(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manifest) ToWorkspace(manifestDir string) (*Workspace, error) {
	root := manifestDir
	if m.Root != "" {
		expanded := m.Root
		if expanded[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("expanding home dir: %w", err)
			}
			expanded = filepath.Join(home, expanded[1:])
		}
		root = expanded
	}
	if m.Name == "" {
		return nil, fmt.Errorf("manifest missing required field: name")
	}
	return &Workspace{
		Name:     m.Name,
		Type:     m.Type,
		Root:     root,
		Services: m.Services,
		Groups:   m.Groups,
		Profiles: m.Profiles,
	}, nil
}
