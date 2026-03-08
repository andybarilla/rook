package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "rook"

func ConfigDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}

func DataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", appName)
}

func PluginsDir() string {
	return filepath.Join(ConfigDir(), "plugins")
}

func SitesFile() string {
	return filepath.Join(ConfigDir(), "sites.json")
}

func LogFile() string {
	return filepath.Join(DataDir(), "rook.log")
}
