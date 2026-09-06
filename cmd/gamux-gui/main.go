package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/staernid/gamux/frontend"
	"github.com/staernid/gamux/gui"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version of the gamux-gui application (set at build time via -ldflags)
var Version = "dev"

func main() {
	app := gui.NewApp(nil)

	var runtimeCtx context.Context
	guard, isPrimary := gui.AcquireOrActivate(func() context.Context {
		return runtimeCtx
	})
	if !isPrimary {
		return
	}
	if guard != nil {
		defer guard.Release()
	}

	err := wails.Run(&options.App{
		Title:     "gamux - Steam Game Manager",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  frontend.Dist,
			Handler: app.AssetHandler(),
		},
		BackgroundColour: &options.RGBA{R: 11, G: 14, B: 20, A: 255},
		OnStartup: func(ctx context.Context) {
			runtimeCtx = ctx
			app.Startup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			if app != nil {
				app.Shutdown(ctx)
			}
			if guard != nil {
				guard.Release()
			}
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if app != nil && app.Config != nil && app.Config.CloseToTray {
				wailsRuntime.WindowHide(ctx)
				return true
			}
			return false
		},
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			ProgramName:      "gamux-gui",
			WebviewGpuPolicy: linux.WebviewGpuPolicyAlways,
		},
	})

	if err != nil {
		slog.Error("Failed to start Wails GUI application", "error", err)
		os.Exit(1)
	}
}
