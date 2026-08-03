package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"

	"github.com/urfave/cli/v2"
)

// Version of the gamux application
const Version = "v0.3.1"

func promptYesNo(promptText string, defaultYes bool) bool {
	hint := " [Y/n]: "
	if !defaultYes {
		hint = " [y/N]: "
	}
	fmt.Print(promptText + hint)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return defaultYes
	}
	if input == "y" || input == "yes" {
		return true
	}
	if input == "n" || input == "no" {
		return false
	}
	return defaultYes
}

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

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
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Complete post-download setup for a game (GBE + optional Lutris & Steam integration)",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "lutris", Usage: "Add game to Lutris"},
					&cli.BoolFlag{Name: "steam", Usage: "Add non-Steam shortcut to Steam"},
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts (non-interactive mode)"},
					&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder instead of loader mode"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
				},
				Action: func(c *cli.Context) error {
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					autoYes := c.Bool("yes")
					addLutris := false
					if c.IsSet("lutris") {
						addLutris = c.Bool("lutris")
					} else if autoYes {
						addLutris = true
					} else {
						addLutris = promptYesNo("Add game to Lutris?", true)
					}

					addSteam := false
					if c.IsSet("steam") {
						addSteam = c.Bool("steam")
					} else if autoYes {
						addSteam = true
					} else {
						addSteam = promptYesNo("Add non-Steam game shortcut to Steam?", true)
					}

					eng := engine.New(activeConfig)
					opts := engine.ProcessOptions{
						Path:      path,
						AddLutris: addLutris,
						AddSteam:  addSteam,
						Portable:  c.Bool("portable"),
						Promote:   c.Bool("promote"),
						DryRun:    c.Bool("dry-run"),
						AutoYes:   autoYes,
					}

					res, err := eng.ProcessGame(c.Context, opts)
					if err != nil {
						return err
					}

					slog.Info("Setup completed successfully for " + res.Info.Name)
					return nil
				},
			},
			{
				Name:      "apply",
				Usage:     "Apply GBE to Steam API files and configure DLCs",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "Do not move or write files, just show what would be done",
					},
					&cli.BoolFlag{
						Name:  "portable",
						Usage: "Perform direct DLL/SO replacement in game folder instead of loader mode",
					},
					&cli.BoolFlag{
						Name:  "promote",
						Value: true,
						Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests",
					},
				},
				Action: func(c *cli.Context) error {
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					eng := engine.New(activeConfig)
					opts := engine.ProcessOptions{
						Path:     path,
						Portable: c.Bool("portable"),
						Promote:  c.Bool("promote"),
						DryRun:   c.Bool("dry-run"),
					}
					_, err := eng.ProcessGame(c.Context, opts)
					return err
				},
			},
			{
				Name:      "batch",
				Usage:     "Scan a parent directory containing multiple games and apply post-download setup to each",
				ArgsUsage: "<dir>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "lutris", Usage: "Add all discovered games to Lutris"},
					&cli.BoolFlag{Name: "steam", Usage: "Add non-Steam shortcuts for all discovered games to Steam"},
					&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts"},
					&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement instead of loader mode"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folders to top level"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "json", Usage: "Output batch results as JSON"},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() < 1 {
						return fmt.Errorf("batch command requires a parent directory path")
					}
					parentDir := c.Args().Get(0)

					eng := engine.New(activeConfig)
					opts := engine.ProcessOptions{
						AddLutris: c.Bool("lutris"),
						AddSteam:  c.Bool("steam"),
						Portable:   c.Bool("portable"),
						Promote:    c.Bool("promote"),
						DryRun:     c.Bool("dry-run"),
						AutoYes:    c.Bool("yes"),
					}

					results, err := eng.BatchProcess(c.Context, parentDir, opts)
					if err != nil {
						return err
					}

					if c.Bool("json") {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						return enc.Encode(results)
					}

					slog.Info("Batch processing complete", "totalGames", len(results))
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
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					eng := engine.New(activeConfig)
					status, err := eng.InspectStatus(c.Context, path)
					if err != nil {
						return err
					}

					if c.Bool("json") {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						return enc.Encode(status)
					}

					fmt.Printf("Game Title:      %s\n", status.Name)
					fmt.Printf("AppID:           %s\n", status.AppID)
					fmt.Printf("Platform:        %s\n", status.Platform)
					fmt.Printf("Directory:       %s\n", status.GameDir)
					fmt.Printf("Executable:      %s\n", status.ExePath)
					fmt.Printf("Patch State:     %s\n", status.State)
					fmt.Printf("Original Backups: %d file(s)\n", len(status.OriginalBackups))
					for _, b := range status.OriginalBackups {
						fmt.Printf("  - %s\n", b)
					}
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
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					eng := engine.New(activeConfig)
					return eng.Rollback(c.Context, path, c.Bool("dry-run"))
				},
			},
			{
				Name:  "update",
				Usage: "Update the GBE fork repository",
				Action: func(c *cli.Context) error {
					return github.UpdateGBE(c.Context)
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
				},
				Action: func(c *cli.Context) error {
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					eng := engine.New(activeConfig)
					opts := engine.ProcessOptions{
						Path:       path,
						AddLutris:  true,
						Runner:     c.String("runner"),
						WinePrefix: c.String("wine-prefix"),
						Portable:   c.Bool("portable"),
						Promote:    c.Bool("promote"),
						DryRun:     c.Bool("dry-run"),
					}
					_, err := eng.ProcessGame(c.Context, opts)
					return err
				},
			},
			{
				Name:      "steam-add",
				Usage:     "Add a non-Steam game shortcut to Steam",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "portable", Usage: "Use direct DLL replacement instead of adding pre-load launch options"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests"},
				},
				Action: func(c *cli.Context) error {
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					eng := engine.New(activeConfig)
					opts := engine.ProcessOptions{
						Path:     path,
						AddSteam: true,
						Portable: c.Bool("portable"),
						Promote:  c.Bool("promote"),
						DryRun:   c.Bool("dry-run"),
					}
					_, err := eng.ProcessGame(c.Context, opts)
					return err
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
