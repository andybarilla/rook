package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user preferences for rook behavior.
type Settings struct {
	AutoRebuild bool `json:"autoRebuild"`
}

// defaultSettings returns settings with default values applied.
func defaultSettings() *Settings {
	return &Settings{
		AutoRebuild: true,
	}
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

	// Apply defaults for zero values
	if s.AutoRebuild == false {
		// Check if it was explicitly set to false by looking at raw JSON
		var raw map[string]interface{}
		if json.Unmarshal(data, &raw) == nil {
			if _, ok := raw["autoRebuild"]; !ok {
				s.AutoRebuild = true
			}
		}
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
