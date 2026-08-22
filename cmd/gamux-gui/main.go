package main

import (
	"log/slog"
	"os"

	"github.com/staernid/gamux/frontend"
	"github.com/staernid/gamux/gui"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

// Version of the gamux-gui application (set at build time via -ldflags)
var Version = "dev"

func main() {
	app := gui.NewApp(nil)

	err := wails.Run(&options.App{
		Title:     "gamux - Steam Game Manager",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: frontend.Dist,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 14, B: 20, A: 255},
		OnStartup:        app.Startup,
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
