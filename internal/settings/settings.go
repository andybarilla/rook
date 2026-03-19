package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user preferences for rook behavior.
type Settings struct {
	AutoRebuild *bool `json:"autoRebuild"`
}

// defaultSettings returns settings with default values applied.
func defaultSettings() *Settings {
	t := true
	return &Settings{
		AutoRebuild: &t,
	}
}

// GetAutoRebuild returns the AutoRebuild setting, defaulting to true if unset.
func (s *Settings) GetAutoRebuild() bool {
	if s.AutoRebuild == nil {
		return true
	}
	return *s.AutoRebuild
}

// SetAutoRebuild sets the AutoRebuild value.
func (s *Settings) SetAutoRebuild(v bool) {
	s.AutoRebuild = &v
}

// Load reads settings from disk. Returns defaults if file doesn't exist.
func Load(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultSettings(), nil
		}
		return nil, err
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	// Apply defaults for nil values
	if s.AutoRebuild == nil {
		t := true
		s.AutoRebuild = &t
	}

	return &s, nil
}

// Save writes settings to disk, creating parent directories if needed.
func (s *Settings) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
