package databases

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SvcConfig holds settings for one database service.
type SvcConfig struct {
	Enabled   bool   `json:"enabled"`
	Autostart bool   `json:"autostart"`
	Port      int    `json:"port"`
	DataDir   string `json:"dataDir"`
}

// Config holds settings for all database services.
type Config struct {
	MySQL    SvcConfig `json:"mysql"`
	Postgres SvcConfig `json:"postgres"`
	Redis    SvcConfig `json:"redis"`
}

// ForType returns the SvcConfig for the given service type.
func (c *Config) ForType(svc ServiceType) SvcConfig {
	switch svc {
	case MySQL:
		return c.MySQL
	case Postgres:
		return c.Postgres
	case Redis:
		return c.Redis
	}
	return SvcConfig{}
}

// SetEnabled sets the enabled flag for the given service type.
func (c *Config) SetEnabled(svc ServiceType, enabled bool) {
	switch svc {
	case MySQL:
		c.MySQL.Enabled = enabled
	case Postgres:
		c.Postgres.Enabled = enabled
	case Redis:
		c.Redis.Enabled = enabled
	}
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(dataRoot string) Config {
	return Config{
		MySQL: SvcConfig{
			Enabled:   true,
			Autostart: false,
			Port:      3306,
			DataDir:   filepath.Join(dataRoot, "mysql"),
		},
		Postgres: SvcConfig{
			Enabled:   true,
			Autostart: false,
			Port:      5432,
			DataDir:   filepath.Join(dataRoot, "postgres"),
		},
		Redis: SvcConfig{
			Enabled:   true,
			Autostart: false,
			Port:      6379,
			DataDir:   filepath.Join(dataRoot, "redis"),
		},
	}
}

// LoadConfig reads config from path. If the file does not exist, writes
// defaults and returns them.
func LoadConfig(path, dataRoot string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := DefaultConfig(dataRoot)
		if saveErr := SaveConfig(path, cfg); saveErr != nil {
			return cfg, saveErr
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig writes config to path as indented JSON.
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
