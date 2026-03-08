package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/andybarilla/rook/internal/config"
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
		expected := filepath.Join(appData, "rook")
		if dir != expected {
			t.Errorf("ConfigDir = %q, want %q", dir, expected)
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(home, ".config", "rook")
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
		expected := filepath.Join(appData, "rook")
		if dir != expected {
			t.Errorf("DataDir = %q, want %q", dir, expected)
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(home, ".local", "share", "rook")
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
	if !strings.HasSuffix(dir, filepath.Join("rook", "plugins")) {
		t.Fatalf("PluginsDir() = %q, want suffix rook/plugins", dir)
	}
}

func TestLogFile(t *testing.T) {
	f := config.LogFile()
	if f == "" {
		t.Fatal("LogFile returned empty string")
	}
	if !strings.HasSuffix(f, "rook.log") {
		t.Errorf("LogFile = %q, want suffix 'rook.log'", f)
	}

	dataDir := config.DataDir()
	expected := filepath.Join(dataDir, "rook.log")
	if f != expected {
		t.Errorf("LogFile = %q, want %q", f, expected)
	}
}
