package main

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/staernid/gamux/assets"
	"github.com/staernid/gamux/frontend"
	"github.com/staernid/gamux/gui"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Version of the gamux-gui application (set at build time via -ldflags)
var Version = "dev"

func init() {
	// WebKitGTK 2.40+ on Linux (especially with Mesa AMD/Intel drivers under Wayland)
	// has a known regression where the hardware-accelerated DMA-BUF renderer leaks textures
	// unboundedly (at ~220MB/s) and hangs ThreadedCompositor, freezing the UI and causing OOM.
	// Disabling the DMA-BUF renderer forces the stable shared-memory compositing pipeline.
	if _, exists := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); !exists {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
}

type profilerState struct {
	cpuFile   *os.File
	traceFile *os.File
	memPath   string
}

func startProfiler(args []string) *profilerState {
	ps := &profilerState{}

	var cpuPath, tracePath, httpAddr string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--cpuprofile=") {
			cpuPath = strings.TrimPrefix(arg, "--cpuprofile=")
		} else if strings.HasPrefix(arg, "--memprofile=") {
			ps.memPath = strings.TrimPrefix(arg, "--memprofile=")
		} else if strings.HasPrefix(arg, "--trace=") {
			tracePath = strings.TrimPrefix(arg, "--trace=")
		} else if strings.HasPrefix(arg, "--pprof-http=") {
			httpAddr = strings.TrimPrefix(arg, "--pprof-http=")
		}
	}

	if cpuPath == "" {
		cpuPath = os.Getenv("GAMUX_PPROF_CPU")
	}
	if ps.memPath == "" {
		ps.memPath = os.Getenv("GAMUX_PPROF_MEM")
	}
	if tracePath == "" {
		tracePath = os.Getenv("GAMUX_TRACE")
	}
	if httpAddr == "" {
		httpAddr = os.Getenv("GAMUX_PPROF_HTTP")
	}

	if httpAddr != "" {
		go func() {
			slog.Info("Starting pprof HTTP server", "addr", httpAddr)
			if err := http.ListenAndServe(httpAddr, nil); err != nil && err != http.ErrServerClosed {
				slog.Warn("pprof HTTP server error", "error", err)
			}
		}()
	}

	if cpuPath != "" {
		f, err := os.Create(cpuPath)
		if err != nil {
			slog.Error("Failed to create CPU profile file", "path", cpuPath, "error", err)
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				slog.Error("Failed to start CPU profile", "error", err)
				_ = f.Close()
			} else {
				ps.cpuFile = f
				slog.Info("Started CPU profiling", "path", cpuPath)
			}
		}
	}

	if tracePath != "" {
		f, err := os.Create(tracePath)
		if err != nil {
			slog.Error("Failed to create trace output file", "path", tracePath, "error", err)
		} else {
			if err := trace.Start(f); err != nil {
				slog.Error("Failed to start runtime trace", "error", err)
				_ = f.Close()
			} else {
				ps.traceFile = f
				slog.Info("Started execution trace", "path", tracePath)
			}
		}
	}

	return ps
}

func (ps *profilerState) stop() {
	if ps == nil {
		return
	}
	if ps.cpuFile != nil {
		pprof.StopCPUProfile()
		_ = ps.cpuFile.Close()
		slog.Info("CPU profile written", "path", ps.cpuFile.Name())
		ps.cpuFile = nil
	}
	if ps.traceFile != nil {
		trace.Stop()
		_ = ps.traceFile.Close()
		slog.Info("Execution trace written", "path", ps.traceFile.Name())
		ps.traceFile = nil
	}
	if ps.memPath != "" {
		f, err := os.Create(ps.memPath)
		if err != nil {
			slog.Error("Failed to create memory profile file", "path", ps.memPath, "error", err)
		} else {
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				slog.Error("Failed to write memory profile", "error", err)
			} else {
				slog.Info("Memory profile written", "path", ps.memPath)
			}
			_ = f.Close()
		}
		ps.memPath = ""
	}
}

func main() {
	profiler := startProfiler(os.Args[1:])
	defer profiler.stop()

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
		Icon:        assets.AppIconPNG,
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
			ProgramName:   "io.github.staernid.gamux",
			ApplicationID: "io.github.staernid.gamux",
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
			profiler.stop()
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
		Icon: assets.AppIconPNG,
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
