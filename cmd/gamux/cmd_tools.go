package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/ui"
	"github.com/staernid/gamux/util"
	"github.com/urfave/cli/v2"
)

var rollbackCommand = &cli.Command{
	Name:      "rollback",
	Aliases:   []string{"undo"},
	Category:  "Step 3: Inspection & Maintenance",
	Usage:     "Restore original binaries & remove emulated settings from a game directory",
	ArgsUsage: "[path]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be restored without mutating files"},
	},
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		path := extractPath(c)
		eng := engine.New(activeConfig)
		ui.RenderStep(1, 1, "Rolling back changes for "+path)
		if err := eng.Rollback(c.Context, path, c.Bool("dry-run")); err != nil {
			renderCLIError(err, "Check file permissions")
			return err
		}
		ui.RenderSuccess("Rollback completed successfully for "+path, "", []string{
			fmt.Sprintf("Run 'gamux apply %s' whenever you want to re-enable GBE", path),
		})
		return nil
	},
}

var updateGBECommand = &cli.Command{
	Name:     "update-gbe",
	Category: "Maintenance & Tools",
	Usage:    "Update Goldberg Emulator release assets from GitHub",
	Action: func(c *cli.Context) error {
		ui.RenderStep(1, 1, "Updating Goldberg Emulator release assets from GitHub")
		if err := github.UpdateGBE(c.Context, activeConfig); err != nil {
			renderCLIError(err)
			return err
		}
		ui.RenderSuccess("Goldberg Emulator assets updated", "", []string{
			"Run 'gamux apply <game_dir>' to apply updated emulator files to your games",
		})
		return nil
	},
}

var updateSteamlessCommand = &cli.Command{
	Name:     "update-steamless",
	Category: "Maintenance & Tools",
	Usage:    "Update Steamless DRM unpacker release assets from GitHub",
	Action: func(c *cli.Context) error {
		destDir := util.ExpandPath(activeConfig.SteamlessDir)
		ui.RenderStep(1, 1, "Updating Steamless DRM unpacker release assets from GitHub")
		if err := github.UpdateSteamlessAssets(c.Context, activeConfig, destDir); err != nil {
			renderCLIError(err)
			return err
		}
		ui.RenderSuccess("Steamless assets updated", "", []string{
			"Steamless will automatically unpack DRM when running 'gamux apply <game_dir>'",
		})
		return nil
	},
}

var toolsCommand = &cli.Command{
	Name:      "tools",
	Category:  "Maintenance & Tools",
	Usage:     "Manage, inspect, and update post-processing tools (GBE & Steamless)",
	ArgsUsage: "[status|update]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Output tools status in JSON format"},
	},
	Action: func(c *cli.Context) error {
		arg := c.Args().First()
		if arg == "update" || arg == "upgrade" {
			ui.RenderStep(1, 2, "Updating Goldberg Emulator release assets...")
			if err := github.UpdateGBE(c.Context, activeConfig); err != nil {
				slog.Warn("GBE update failed", "error", err)
			}
			ui.RenderStep(2, 2, "Updating Steamless release assets...")
			destDir := util.ExpandPath(activeConfig.SteamlessDir)
			if err := github.UpdateSteamlessAssets(c.Context, activeConfig, destDir); err != nil {
				slog.Warn("Steamless update failed", "error", err)
			}
			ui.RenderSuccess("Tools update process finished", "", nil)
			return nil
		}

		tools, err := github.GetToolsStatus(c.Context, activeConfig)
		if err != nil {
			renderCLIError(err)
			return err
		}

		if c.Bool("json") {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(tools)
		}

		fmt.Println()
		fmt.Println("🔧 POST-PROCESSING TOOLS & RELEASE NOTES")
		fmt.Println("------------------------------------------------------------------------")
		for _, t := range tools {
			statusTag := "Up to Date"
			if t.HasUpdate {
				statusTag = "⚠️ Update Available"
			}
			fmt.Printf("  • %s (%s)\n", t.Name, statusTag)
			fmt.Printf("    Installed: %s  |  Latest: %s (%s)\n", t.InstalledTag, t.LatestTag, t.PublishedAt)
			fmt.Printf("    Path: %s\n", t.InstalledPath)
			if t.ReleaseNotes != "" {
				notes := strings.TrimSpace(t.ReleaseNotes)
				if len(notes) > 300 {
					notes = notes[:300] + "..."
				}
				fmt.Printf("    Notes:\n      %s\n", strings.ReplaceAll(notes, "\n", "\n      "))
			}
			fmt.Println("------------------------------------------------------------------------")
		}
		return nil
	},
}

var configCommand = &cli.Command{
	Name:      "config",
	Category:  "Maintenance & Tools",
	Usage:     "View, inspect, and interactively configure gamux settings",
	ArgsUsage: "[show|wizard]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "show", Usage: "Display current configuration dashboard"},
		&cli.BoolFlag{Name: "json", Usage: "Output configuration in JSON format"},
	},
	Action: func(c *cli.Context) error {
		configPath := config.GetConfigPath(c.String("config"))
		if activeConfig == nil {
			var err error
			activeConfig, err = config.LoadConfig(c.String("config"))
			if err != nil {
				activeConfig = config.DefaultConfig()
			}
		}

		if c.Bool("json") {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(activeConfig)
		}

		arg := c.Args().First()
		if c.Bool("show") || arg == "show" {
			ui.RenderConfigSummary(activeConfig, configPath)
			return nil
		}

		return ui.RunConfigWizard(activeConfig, configPath)
	},
	Subcommands: []*cli.Command{
		{
			Name:  "show",
			Usage: "Display current vs default configuration dashboard",
			Flags: []cli.Flag{
				&cli.BoolFlag{Name: "json", Usage: "Output configuration in JSON format"},
			},
			Action: func(c *cli.Context) error {
				configPath := config.GetConfigPath(c.String("config"))
				if activeConfig == nil {
					var err error
					activeConfig, err = config.LoadConfig(c.String("config"))
					if err != nil {
						activeConfig = config.DefaultConfig()
					}
				}
				if c.Bool("json") {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(activeConfig)
				}
				ui.RenderConfigSummary(activeConfig, configPath)
				return nil
			},
		},
		{
			Name:  "wizard",
			Usage: "Run the interactive guided configuration setup wizard",
			Action: func(c *cli.Context) error {
				configPath := config.GetConfigPath(c.String("config"))
				if activeConfig == nil {
					var err error
					activeConfig, err = config.LoadConfig(c.String("config"))
					if err != nil {
						activeConfig = config.DefaultConfig()
					}
				}
				return ui.RunConfigWizard(activeConfig, configPath)
			},
		},
	},
}

var notifyLaunchCommand = &cli.Command{
	Name:      "notify-launch",
	Category:  "Maintenance & Tools",
	Usage:     "Pre-launch hook to check game update status and notify via console/desktop notification",
	ArgsUsage: "[path]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "path", Usage: "Path to game directory"},
	},
	Action: func(c *cli.Context) error {
		targetPath := c.String("path")
		if targetPath == "" {
			targetPath = extractPath(c)
		}
		if targetPath == "" {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		eng := engine.New(activeConfig)
		eng.Notifier = ui.NotifyIssue
		return eng.NotifyLaunch(c.Context, targetPath)
	},
}
