package main

import (
	"context"
	"log/slog"
	"os"
	"time"

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
	initialTarget := gui.ExtractOpenTarget(os.Args[1:])
	if initialTarget != "" {
		app.HandleOpenTarget(initialTarget)
	}

	guard, isPrimary := (*gui.SingleInstanceGuard)(nil), true
	if os.Getenv("WAILS_GENERATE_MODULE") != "1" {
		guard, isPrimary = gui.AcquireOrActivate(os.Args[1:], func() context.Context {
			return runtimeCtx
		}, func(target string) {
			app.HandleOpenTarget(target)
		})
	}
	if !isPrimary {
		return
	}
	if guard != nil {
		defer guard.Release()
	}

	var tray *gui.TrayManager
	onActivate := func() {
		if runtimeCtx != nil {
			wailsRuntime.WindowShow(runtimeCtx)
			wailsRuntime.WindowUnminimise(runtimeCtx)
		}
	}
	onQuit := func() {
		slog.Info("Quit requested from system tray")
		if app != nil {
			app.SetQuitting(true)
		}
		if runtimeCtx != nil {
			wailsRuntime.Quit(runtimeCtx)
			go func() {
				time.Sleep(2 * time.Second)
				slog.Warn("Wails quit timed out, terminating process")
				if tray != nil {
					tray.Close()
				}
				if guard != nil {
					guard.Release()
				}
				os.Exit(0)
			}()
		} else {
			if tray != nil {
				tray.Close()
			}
			if guard != nil {
				guard.Release()
			}
			os.Exit(0)
		}
	}
	onLaunchGame := func(gameDir string) {
		if app != nil {
			go func() {
				slog.Info("Launching game from system tray", "gameDir", gameDir)
				if err := app.QuickPlay(gui.LaunchOptions{GameDir: gameDir}); err != nil {
					slog.Warn("Failed to quick play game from tray", "gameDir", gameDir, "error", err)
				}
			}()
		}
	}
	getRecentGames := func() []gui.TrayGame {
		if app != nil {
			return app.GetRecentGames(3)
		}
		return nil
	}

	var trayErr error
	tray, trayErr = gui.StartTray(gui.TrayCallbacks{
		OnActivate:     onActivate,
		OnQuit:         onQuit,
		OnLaunchGame:   onLaunchGame,
		GetRecentGames: getRecentGames,
	})
	if trayErr != nil {
		slog.Warn("Could not initialize system tray", "error", trayErr)
	} else if app != nil {
		app.SetTrayManager(tray)
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
			if tray != nil {
				tray.Close()
			}
			if app != nil {
				app.Shutdown(ctx)
			}
			if guard != nil {
				guard.Release()
			}
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if app != nil && app.IsQuitting() {
				return false
			}
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
