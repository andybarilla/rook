package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/api"
	"github.com/andybarilla/rook/internal/discovery"
	"github.com/andybarilla/rook/internal/orchestrator"
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/registry"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func configDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.ExpandEnv("$HOME/.config")
	}
	return filepath.Join(dir, "rook")
}

func main() {
	runner.DetectRuntime()
	cfgDir := configDir()
	os.MkdirAll(cfgDir, 0755)

	reg, _ := registry.NewFileRegistry(filepath.Join(cfgDir, "workspaces.json"))
	alloc, _ := ports.NewFileAllocator(filepath.Join(cfgDir, "ports.json"), 10000, 60000)

	processRunner := runner.NewProcessRunner()
	dockerRunner := runner.NewDockerRunner("rook")
	orch := orchestrator.New(dockerRunner, processRunner, alloc)

	discoverers := []discovery.Discoverer{
		discovery.NewComposeDiscoverer(),
		discovery.NewDevcontainerDiscoverer(),
		discovery.NewMiseDiscoverer(),
	}

	wsAPI := api.NewWorkspaceAPI(reg, alloc, orch, discoverers)

	err := wails.Run(&options.App{
		Title:     "Rook",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			wsAPI.SetEmitter(&wailsEmitter{ctx: ctx})
		},
		Bind: []interface{}{
			wsAPI,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

type wailsEmitter struct {
	ctx context.Context
}

func (e *wailsEmitter) Emit(eventName string, data ...interface{}) {
	wailsruntime.EventsEmit(e.ctx, eventName, data...)
}
