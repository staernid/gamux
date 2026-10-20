package main

import (
	"fmt"
	"strings"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var modsCommand = &cli.Command{
	Name:     "mods",
	Category: "Step 2: Configuration & Mods",
	Usage:    "Inspect, toggle, and manage installed mods and Steam Workshop items",
	Subcommands: []*cli.Command{
		{
			Name:      "list",
			Usage:     "List all installed mods and detected engine mod topology",
			ArgsUsage: "[target_dir]",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "dir", Usage: "Target game directory (defaults to current directory)"},
			},
			Action: func(c *cli.Context) error {
				targetDir := c.String("dir")
				if targetDir == "" && c.Args().Len() > 0 {
					targetDir = c.Args().Get(0)
				}
				if targetDir == "" {
					targetDir = "."
				}

				eng := engine.New(activeConfig)
				status, err := eng.GetGameMods(c.Context, targetDir)
				if err != nil {
					renderCLIError(err, "Verify game directory path")
					return err
				}

				fmt.Println("=== Mod Engine Topology ===")
				fmt.Printf("  Engine:   %s\n", strings.ToUpper(status.Topology.Engine))
				fmt.Printf("  Mod Dir:  %s\n", status.Topology.ModDir)
				fmt.Printf("  Detected: %t\n", status.Topology.Detected)
				fmt.Println()

				fmt.Printf("=== Installed Mods (%d total, %d outdated) ===\n", len(status.Mods), status.OutdatedCount)
				if len(status.Mods) == 0 {
					fmt.Println("  No mods installed.")
					fmt.Println("  Install a workshop item via: gamux workshop <id-or-url>")
					return nil
				}

				for _, m := range status.Mods {
					statusBadge := "[ENABLED] "
					if !m.Enabled {
						statusBadge = "[DISABLED]"
					}
					updateBadge := ""
					if m.HasUpdate {
						updateBadge = " (Update Available!)"
					}
					fmt.Printf("  %s %s%s\n", statusBadge, m.Name, updateBadge)
					fmt.Printf("             ID: %s | Path: %s\n", m.ID, m.RelativePath)
					if m.WorkshopID > 0 {
						fmt.Printf("             Workshop ID: %d\n", m.WorkshopID)
					}
				}
				return nil
			},
		},
		{
			Name:      "toggle",
			Usage:     "Toggle a mod enabled or disabled without deleting files",
			ArgsUsage: "<mod_id_or_name>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "dir", Usage: "Target game directory (defaults to current directory)"},
				&cli.BoolFlag{Name: "enable", Usage: "Explicitly enable the mod"},
				&cli.BoolFlag{Name: "disable", Usage: "Explicitly disable the mod"},
			},
			Action: func(c *cli.Context) error {
				if c.Args().Len() < 1 {
					err := fmt.Errorf("mod name or ID is required")
					renderCLIError(err, "Example: gamux mods toggle SuperMod.pak")
					return err
				}
				modID := c.Args().Get(0)
				targetDir := c.String("dir")
				if targetDir == "" {
					targetDir = "."
				}

				eng := engine.New(activeConfig)
				status, err := eng.GetGameMods(c.Context, targetDir)
				if err != nil {
					renderCLIError(err, "Verify game directory path")
					return err
				}

				enable := true
				if c.Bool("disable") {
					enable = false
				} else if c.Bool("enable") {
					enable = true
				} else {
					// Auto-toggle based on current state
					for _, m := range status.Mods {
						if m.ID == modID || strings.EqualFold(m.Name, modID) {
							enable = !m.Enabled
							break
						}
					}
				}

				mod, err := eng.ToggleGameMod(targetDir, modID, enable)
				if err != nil {
					renderCLIError(err, "Verify mod ID or name")
					return err
				}

				stateStr := "enabled"
				if !mod.Enabled {
					stateStr = "disabled"
				}
				ui.RenderSuccess(fmt.Sprintf("Mod %q is now %s", mod.Name, stateStr), "", []string{
					fmt.Sprintf("Path: %s", mod.RelativePath),
				})
				return nil
			},
		},
		{
			Name:      "check-updates",
			Usage:     "Check for newer versions of installed workshop mods",
			ArgsUsage: "[target_dir]",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "dir", Usage: "Target game directory (defaults to current directory)"},
			},
			Action: func(c *cli.Context) error {
				targetDir := c.String("dir")
				if targetDir == "" && c.Args().Len() > 0 {
					targetDir = c.Args().Get(0)
				}
				if targetDir == "" {
					targetDir = "."
				}

				ui.RenderStep(1, 1, "Checking Steam Workshop for mod updates...")
				eng := engine.New(activeConfig)
				status, err := eng.CheckGameModUpdates(c.Context, targetDir)
				if err != nil {
					renderCLIError(err, "Check internet connection or Steam Web API status")
					return err
				}

				if status.OutdatedCount == 0 {
					ui.RenderSuccess("All workshop mods are up to date", "", nil)
				} else {
					ui.RenderSuccess(fmt.Sprintf("%d mod update(s) available!", status.OutdatedCount), "", nil)
					for _, m := range status.Mods {
						if m.HasUpdate {
							fmt.Printf("  - %s (Workshop ID: %d)\n", m.Name, m.WorkshopID)
						}
					}
				}
				return nil
			},
		},
	},
}
