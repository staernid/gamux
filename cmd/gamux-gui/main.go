package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/staernid/gamux/frontend"
	"github.com/staernid/gamux/gui"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Version of the gamux-gui application (set at build time via -ldflags)
var Version = "dev"

func main() {
	app := gui.NewApp(nil)
	app.Startup(context.Background())

	initialTarget := gui.ExtractOpenTarget(os.Args[1:])
	if initialTarget != "" {
		app.HandleOpenTarget(initialTarget)
	}

	var mainWindow *application.WebviewWindow
	var tray *gui.TrayManager

	wailsApp := application.New(application.Options{
		Name:        "gamux-gui",
		Description: "gamux - Steam Game Manager",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(frontend.Dist),
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/art/") || strings.HasPrefix(r.URL.Path, "/local-img/") {
						app.AssetHandler().ServeHTTP(w, r)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
		Linux: application.LinuxOptions{
			ProgramName: "gamux-gui",
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "gamux-gui-singleinstance",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				if mainWindow != nil {
					mainWindow.Restore()
					mainWindow.Focus()
				}
				target := gui.ExtractOpenTarget(data.Args)
				if target != "" && app != nil {
					app.HandleOpenTarget(target)
				}
			},
		},
		OnShutdown: func() {
			if tray != nil {
				tray.Close()
			}
			if app != nil {
				app.Shutdown(context.Background())
			}
		},
	})

	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "gamux - Steam Game Manager",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        600,
		URL:              "/",
		BackgroundColour: application.NewRGB(11, 14, 20),
	})
	app.SetMainWindow(mainWindow)

	mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if app != nil && app.IsQuitting() {
			return
		}
		if app != nil && app.Config != nil && app.Config.CloseToTray {
			mainWindow.Hide()
			e.Cancel()
		}
	})

	// Initialize system tray
	var trayErr error
	tray, trayErr = gui.StartTray(gui.TrayCallbacks{
		OnActivate: func() {
			if mainWindow != nil {
				mainWindow.Show()
				mainWindow.Restore()
				mainWindow.Focus()
			}
		},
		OnQuit: func() {
			slog.Info("Quit requested from system tray")
			if app != nil {
				app.SetQuitting(true)
			}
			wailsApp.Quit()
		},
		OnLaunchGame: func(gameDir string) {
			if app != nil {
				go func() {
					slog.Info("Launching game from system tray", "gameDir", gameDir)
					if err := app.QuickPlay(gui.LaunchOptions{GameDir: gameDir}); err != nil {
						slog.Warn("Failed to quick play game from tray", "gameDir", gameDir, "error", err)
					}
				}()
			}
		},
		GetRecentGames: func() []gui.TrayGame {
			if app != nil {
				return app.GetRecentGames(3)
			}
			return nil
		},
	})
	if trayErr != nil {
		slog.Warn("Could not initialize system tray", "error", trayErr)
	} else if app != nil {
		app.SetTrayManager(tray)
	}

	if err := wailsApp.Run(); err != nil {
		slog.Error("Failed to start Wails GUI application", "error", err)
		os.Exit(1)
	}
}
