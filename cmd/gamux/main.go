package main

import (
	"errors"
	"io"
	"log/slog"
	"os"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

// Version of the gamux application (set at build time via -ldflags)
var Version = "dev"

var activeConfig *config.Config

func renderCLIError(err error, fallback ...string) {
	var suggestions []string
	if errors.Is(err, detector.ErrGameNotFound) {
		suggestions = append(suggestions, "Verify that the specified directory path exists", "Provide an absolute path or navigate to the game directory")
	} else if errors.Is(err, detector.ErrNoExecutableFound) {
		suggestions = append(suggestions, "Verify that the folder contains a valid .exe or Linux native binary", "Ensure file permissions allow execution (chmod +x)")
	} else if errors.Is(err, detector.ErrInvalidManifest) {
		suggestions = append(suggestions, "Re-generate or repair the manifest via 'gamux apply'", "Verify that the ACF file is not empty or corrupted")
	} else if errors.Is(err, steam.ErrRateLimited) {
		suggestions = append(suggestions, "Steam API rate limit reached; wait a few minutes before retrying")
	} else if errors.Is(err, manifest.ErrDepotKeyNotFound) {
		suggestions = append(suggestions, "No depot key was found for this game on Hubcap or RevoBD", "Supply a custom depot key via lua or manifest folder")
	} else if len(fallback) > 0 {
		suggestions = fallback
	} else {
		suggestions = []string{"Check file permissions and directory path", "Run with --verbose for detailed debug logs"}
	}
	ui.RenderErrorHelp(err, suggestions)
}

func buildApp() *cli.App {
	return &cli.App{
		Name:    "gamux",
		Usage:   "Streamline post-download Steam game management on Linux",
		Version: Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Non-interactive mode (use defaults without prompting)"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without modifying files"},
			&cli.BoolFlag{Name: "json", Usage: "Output status or batch results as JSON"},
			&cli.StringFlag{Name: "config", Usage: "Path to custom configuration file"},
			&cli.BoolFlag{Name: "verbose", Usage: "Enable verbose debug logging output"},
		},

		Before: func(c *cli.Context) error {
			level := slog.LevelWarn
			if c.Bool("verbose") {
				level = slog.LevelDebug
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: level,
			}))
			slog.SetDefault(logger)

			var err error
			activeConfig, err = config.LoadConfig(c.String("config"))
			return err
		},

		Action: func(c *cli.Context) error {
			ui.RenderAppHelp(c.App, Version)
			return nil
		},

		Commands: []*cli.Command{
			downloadCommand,
			workshopCommand,
			applyCommand,
			batchCommand,
			statusCommand,
			newsCommand,
			rollbackCommand,
			updateGBECommand,
			updateSteamlessCommand,
			toolsCommand,
			configCommand,
			notifyLaunchCommand,
			launchCommand,
		},
	}
}

func init() {
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		if app, ok := data.(*cli.App); ok {
			ui.RenderAppHelp(app, Version)
		} else if cmd, ok := data.(*cli.Command); ok {
			ui.RenderCommandHelp(cmd, Version)
		} else {
			ui.RenderAppHelp(nil, Version)
		}
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	slog.SetDefault(logger)

	app := buildApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
