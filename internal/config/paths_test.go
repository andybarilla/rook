package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/andybarilla/flock/internal/config"
)

func TestConfigDir(t *testing.T) {
	dir := config.ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir returned empty string")
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			t.Skip("APPDATA not set")
		}
		expected := filepath.Join(appData, "flock")
		if dir != expected {
			t.Errorf("ConfigDir = %q, want %q", dir, expected)
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(home, ".config", "flock")
		if dir != expected {
			t.Errorf("ConfigDir = %q, want %q", dir, expected)
		}
	}
}

func TestDataDir(t *testing.T) {
	dir := config.DataDir()
	if dir == "" {
		t.Fatal("DataDir returned empty string")
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			t.Skip("APPDATA not set")
		}
		expected := filepath.Join(appData, "flock")
		if dir != expected {
			t.Errorf("DataDir = %q, want %q", dir, expected)
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(home, ".local", "share", "flock")
		if dir != expected {
			t.Errorf("DataDir = %q, want %q", dir, expected)
		}
	}
}

func TestSitesFile(t *testing.T) {
	f := config.SitesFile()
	if f == "" {
		t.Fatal("SitesFile returned empty string")
	}
	if !strings.HasSuffix(f, "sites.json") {
		t.Errorf("SitesFile = %q, want suffix 'sites.json'", f)
	}

	configDir := config.ConfigDir()
	expected := filepath.Join(configDir, "sites.json")
	if f != expected {
		t.Errorf("SitesFile = %q, want %q", f, expected)
	}
}

func TestPluginsDir(t *testing.T) {
	dir := config.PluginsDir()
	if !strings.HasSuffix(dir, filepath.Join("flock", "plugins")) {
		t.Fatalf("PluginsDir() = %q, want suffix flock/plugins", dir)
	}
}

func TestLogFile(t *testing.T) {
	f := config.LogFile()
	if f == "" {
		t.Fatal("LogFile returned empty string")
	}
	if !strings.HasSuffix(f, "flock.log") {
		t.Errorf("LogFile = %q, want suffix 'flock.log'", f)
	}

	dataDir := config.DataDir()
	expected := filepath.Join(dataDir, "flock.log")
	if f != expected {
		t.Errorf("LogFile = %q, want %q", f, expected)
	}
}
