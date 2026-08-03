package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/ui"

	"github.com/urfave/cli/v2"
)

// Version of the gamux application (set at build time via -ldflags)
var Version = "dev"

func extractPath(c *cli.Context) string {
	if c.Args().Len() >= 1 {
		return c.Args().Get(0)
	}
	return "."
}

func extractProcessOptions(c *cli.Context, promptIfUnset bool) engine.ProcessOptions {
	autoYes := c.Bool("yes")
	addLutris := false
	if c.IsSet("lutris") {
		addLutris = c.Bool("lutris")
	} else if autoYes {
		addLutris = true
	} else if promptIfUnset {
		addLutris = ui.PromptYesNoWithExplanation(
			"Register game in Lutris?",
			"Creates a Lutris YAML configuration in ~/.config/lutris/ so the game appears in your Lutris library.",
			true,
		)
	}

	portable := c.Bool("portable")
	if !c.IsSet("portable") && promptIfUnset && !autoYes {
		portable = ui.PromptYesNoWithExplanation(
			"Use Portable Mode (Direct DLL/SO replacement)?",
			"Portable mode replaces steam_api.dll directly in the game folder (backed up to .ORIGINAL). Default mode uses environment loader flags.",
			false,
		)
	}

	return engine.ProcessOptions{
		Path:        extractPath(c),
		AddLutris:   addLutris,
		Runner:      c.String("runner"),
		WinePrefix:  c.String("wine-prefix"),
		Portable:    portable,
		Promote:     c.Bool("promote"),
		DryRun:      c.Bool("dry-run"),
		AutoYes:     autoYes,
		NoSteamless: c.Bool("no-steamless"),
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		ui.RenderAppHelp(Version)
	}

	var activeConfig *config.Config

	app := &cli.App{
		Name:    "gamux",
		Usage:   "Streamline post-download Steam game management on Linux",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Load configuration from `FILE`",
			},
		},
		Before: func(c *cli.Context) error {
			var err error
			activeConfig, err = config.LoadConfig(c.String("config"))
			return err
		},
		Action: func(c *cli.Context) error {
			ui.RenderAppHelp(Version)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Complete post-download setup for a game (GBE + optional Lutris integration)",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "lutris", Usage: "Add game to Lutris"},
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts (non-interactive mode)"},
					&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder instead of loader mode"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
				},
				Action: func(c *cli.Context) error {
					path := extractPath(c)
					eng := engine.New(activeConfig)

					if !c.Bool("yes") {
						status, err := eng.InspectStatus(c.Context, path)
						if err == nil {
							ui.RenderDetectionSummary(ui.DetectionInfoSummary{
								Name:            status.Name,
								AppID:           status.AppID,
								Platform:        status.Platform,
								GameDir:         status.GameDir,
								ExePath:         status.ExePath,
								State:           status.State,
								OriginalBackups: len(status.OriginalBackups),
							})
						}
					}

					opts := extractProcessOptions(c, true)

					ui.RenderStep(1, 1, "Executing setup for "+path)
					res, err := eng.ProcessGame(c.Context, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Check directory path", "Verify file permissions"})
						return err
					}

					nextSteps := []string{}
					if opts.AddLutris {
						nextSteps = append(nextSteps, "Launch the game via Lutris")
					}
					ui.RenderSuccess("Setup completed successfully for "+res.Info.Name, "", nextSteps)
					return nil
				},
			},
			{
				Name:      "apply",
				Usage:     "Apply GBE to Steam API files and configure DLCs",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dry-run", Usage: "Do not move or write files, just show what would be done"},
					&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder instead of loader mode"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests"},
					&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
				},
				Action: func(c *cli.Context) error {
					opts := extractProcessOptions(c, false)
					eng := engine.New(activeConfig)
					ui.RenderStep(1, 1, "Applying Goldberg Emulator to "+opts.Path)
					res, err := eng.ProcessGame(c.Context, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Verify target folder contains Steam API files"})
						return err
					}
					ui.RenderSuccess("Goldberg Emulator applied for "+res.Info.Name, "", nil)
					return nil
				},
			},
			{
				Name:      "batch",
				Usage:     "Scan a parent directory containing multiple games and apply post-download setup to each",
				ArgsUsage: "<dir>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "lutris", Usage: "Add all discovered games to Lutris"},
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts"},
					&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement instead of loader mode"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folders to top level"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "json", Usage: "Output batch results as JSON"},
					&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() < 1 {
						err := fmt.Errorf("batch command requires a parent directory path")
						ui.RenderErrorHelp(err, []string{"Example: gamux batch /path/to/downloads"})
						return err
					}
					parentDir := c.Args().Get(0)

					opts := extractProcessOptions(c, false)
					eng := engine.New(activeConfig)
					if !c.Bool("json") {
						ui.RenderStep(1, 1, "Scanning parent directory: "+parentDir)
					}
					results, err := eng.BatchProcess(c.Context, parentDir, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Verify directory exists"})
						return err
					}

					if c.Bool("json") {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						return enc.Encode(results)
					}

					ui.RenderSuccess(fmt.Sprintf("Batch processing complete (%d games processed)", len(results)), "", nil)
					return nil
				},
			},
			{
				Name:      "status",
				Usage:     "Inspect game directory status (Original, Loader-Configured, or Portable-Patched)",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "Output status inspection result as JSON"},
				},
				Action: func(c *cli.Context) error {
					path := extractPath(c)
					eng := engine.New(activeConfig)
					status, err := eng.InspectStatus(c.Context, path)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Ensure path points to a valid game directory"})
						return err
					}

					if c.Bool("json") {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						return enc.Encode(status)
					}

					ui.RenderDetectionSummary(ui.DetectionInfoSummary{
						Name:            status.Name,
						AppID:           status.AppID,
						Platform:        status.Platform,
						GameDir:         status.GameDir,
						ExePath:         status.ExePath,
						State:           status.State,
						OriginalBackups: len(status.OriginalBackups),
					})
					return nil
				},
			},
			{
				Name:      "rollback",
				Usage:     "Restore .ORIGINAL files and remove emulated settings from a game directory",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be restored without mutating files"},
				},
				Action: func(c *cli.Context) error {
					path := extractPath(c)
					eng := engine.New(activeConfig)
					ui.RenderStep(1, 1, "Rolling back changes for "+path)
					if err := eng.Rollback(c.Context, path, c.Bool("dry-run")); err != nil {
						ui.RenderErrorHelp(err, []string{"Check file permissions"})
						return err
					}
					ui.RenderSuccess("Rollback completed successfully for "+path, "", nil)
					return nil
				},
			},
			{
				Name:  "update",
				Usage: "Update the GBE fork repository",
				Action: func(c *cli.Context) error {
					ui.RenderStep(1, 1, "Updating Goldberg Emulator release assets")
					if err := github.UpdateGBE(c.Context); err != nil {
						ui.RenderErrorHelp(err, []string{"Check network connectivity"})
						return err
					}
					ui.RenderSuccess("Goldberg Emulator assets updated", "", nil)
					return nil
				},
			},
			{
				Name:      "lutris-add",
				Usage:     "Add a game to Lutris",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "runner", Usage: "Lutris runner (linux or wine)"},
					&cli.StringFlag{Name: "wine-prefix", Usage: "Path to Wine prefix"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "portable", Usage: "Use direct DLL replacement instead of configuring loader env vars"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests"},
					&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
				},
				Action: func(c *cli.Context) error {
					opts := extractProcessOptions(c, false)
					opts.AddLutris = true
					eng := engine.New(activeConfig)
					ui.RenderStep(1, 1, "Registering with Lutris")
					res, err := eng.ProcessGame(c.Context, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Check Lutris configuration path"})
						return err
					}
					ui.RenderSuccess("Added to Lutris: "+res.Info.Name, "", []string{"Launch via Lutris"})
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
