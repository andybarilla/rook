package databases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andybarilla/flock/internal/databases"
)

func TestDefaultConfig(t *testing.T) {
	cfg := databases.DefaultConfig("/data")

	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL port = %d, want 3306", cfg.MySQL.Port)
	}
	if cfg.Postgres.Port != 5432 {
		t.Errorf("Postgres port = %d, want 5432", cfg.Postgres.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis port = %d, want 6379", cfg.Redis.Port)
	}
	if cfg.MySQL.Autostart {
		t.Error("MySQL autostart should default to false")
	}
	wantDataDir := filepath.Join("/data", "mysql")
	if cfg.MySQL.DataDir != wantDataDir {
		t.Errorf("MySQL DataDir = %q, want %q", cfg.MySQL.DataDir, wantDataDir)
	}
}

func TestLoadConfigCreatesDefaultIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")

	cfg, err := databases.LoadConfig(path, "/data")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL port = %d, want 3306", cfg.MySQL.Port)
	}

	// File should have been created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}
}

func TestLoadConfigReadsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")

	content := `{
		"mysql": {"enabled": true, "autostart": true, "port": 3307, "dataDir": "/custom/mysql"},
		"postgres": {"enabled": false, "autostart": false, "port": 5432, "dataDir": "/custom/pg"},
		"redis": {"enabled": true, "autostart": false, "port": 6380, "dataDir": "/custom/redis"}
	}`
	os.WriteFile(path, []byte(content), 0o644)

	cfg, err := databases.LoadConfig(path, "/data")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MySQL.Port != 3307 {
		t.Errorf("MySQL port = %d, want 3307", cfg.MySQL.Port)
	}
	if !cfg.MySQL.Autostart {
		t.Error("MySQL autostart should be true")
	}
	if cfg.MySQL.DataDir != "/custom/mysql" {
		t.Errorf("MySQL DataDir = %q, want /custom/mysql", cfg.MySQL.DataDir)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")

	cfg := databases.DefaultConfig("/data")
	cfg.MySQL.Port = 3307

	if err := databases.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := databases.LoadConfig(path, "/data")
	if err != nil {
		t.Fatalf("LoadConfig after save: %v", err)
	}
	if loaded.MySQL.Port != 3307 {
		t.Errorf("MySQL port = %d, want 3307", loaded.MySQL.Port)
	}
}
