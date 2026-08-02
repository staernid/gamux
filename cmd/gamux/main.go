package main

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gamux/config"
	"gamux/detector"
	"gamux/gbe"
	"gamux/github"
	"gamux/lutris"
	"gamux/steam"
	"gamux/steamshortcut"
	"gamux/tui"

	"github.com/urfave/cli/v2"
)

// Version of the gamux application
const Version = "v1"

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
			return config.InitConfig(c.String("config"))
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

					info, err := detector.Detect(c.Context, path)
					if err != nil {
						return fmt.Errorf("auto-detect failed: %w", err)
					}
					if err := detector.ConsolidateManifests(info, c.Bool("promote")); err != nil {
						slog.Warn("Failed to consolidate manifests", "error", err)
					}

					slog.Info("Auto-detected game", "title", info.Name, "appID", info.AppID, "platform", info.Platform, "exe", info.ExePath)

					dryRun := c.Bool("dry-run")
					portable := c.Bool("portable")
					autoYes := c.Bool("yes")

					// Determine whether to add to Lutris
					addLutris := false
					if c.IsSet("lutris") {
						addLutris = c.Bool("lutris")
					} else if autoYes {
						addLutris = true
					} else {
						addLutris = promptYesNo("Add game to Lutris?", true)
					}

					// Determine whether to add to Steam
					addSteam := false
					if c.IsSet("steam") {
						addSteam = c.Bool("steam")
					} else if autoYes {
						addSteam = true
					} else {
						addSteam = promptYesNo("Add non-Steam game shortcut to Steam?", true)
					}

					// 1. Apply GBE
					if err := gbe.ApplyGBE(c.Context, info.Platform, info.AppID, dryRun, portable); err != nil {
						slog.Warn("GBE application warning/error", "error", err)
					}

					// 2. Add to Lutris if selected
					if addLutris {
						runner := "linux"
						if info.Platform != "linux" {
							runner = "wine"
						}

						var env map[string]string
						if !portable {
							home, _ := os.UserHomeDir()
							env = make(map[string]string)
							if runner == "linux" {
								env["LD_PRELOAD"] = filepath.Join(home, config.GbeDir, "linux_release", "experimental", "x64", "steamclient.so")
							} else {
								env["SteamClient64Dll"] = filepath.Join(home, config.GbeDir, "win_release", "experimental", "x64", "steamclient64.dll")
							}
						}

						lcfg := lutris.Config{
							Name:     info.Name,
							GamePath: info.ExePath,
							Runner:   runner,
							Env:      env,
						}

						if dryRun {
							slog.Info("[DRY RUN] Would write Lutris YAML & fetch cover art", "name", info.Name)
						} else {
							home, err := os.UserHomeDir()
							if err == nil {
								targetDir := filepath.Join(home, config.LutrisDir)
								if err := lutris.Write(lcfg, targetDir); err == nil {
									slog.Info("Successfully wrote Lutris game config", "name", info.Name, "dir", targetDir)
									_, _ = steam.FetchLutrisArtwork(c.Context, info.AppID, lutris.Slugify(info.Name), false)
								}
							}
						}
					}

					// 3. Add to Steam if selected
					if addSteam {
						launchOpt := ""
						if !portable {
							home, _ := os.UserHomeDir()
							soPath := filepath.Join(home, config.GbeDir, "linux_release", "experimental", "x64", "steamclient.so")
							launchOpt = fmt.Sprintf("LD_PRELOAD=%s %%command%%", soPath)
						}

						scfg := steamshortcut.ShortcutConfig{
							Name:      info.Name,
							ExePath:   info.ExePath,
							AppID:     info.AppID,
							LaunchOpt: launchOpt,
						}
						if err := steamshortcut.RegisterShortcut(c.Context, scfg, dryRun); err != nil {
							slog.Warn("Failed to register Steam shortcut", "error", err)
						}
					}

					slog.Info("Setup completed successfully for "+info.Name)
					return nil
				},
			},
			{
				Name:      "apply",
				Usage:     "Apply GBE to Steam API files and configure DLCs",
				ArgsUsage: "[path] or <platform> <appid>",
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
					var path, platform, appID string
					if c.Args().Len() >= 2 {
						platform = c.Args().Get(0)
						appID = c.Args().Get(1)
						path = "."
					} else {
						if c.Args().Len() == 1 {
							path = c.Args().Get(0)
						} else {
							path = "."
						}
						info, err := detector.Detect(c.Context, path)
						if err != nil {
							return fmt.Errorf("auto-detect failed: %w", err)
						}
						if err := detector.ConsolidateManifests(info, c.Bool("promote")); err != nil {
							slog.Warn("Failed to consolidate manifests", "error", err)
						}
						platform = info.Platform
						appID = info.AppID
						slog.Info("Auto-detected game", "title", info.Name, "appID", appID, "platform", platform, "exe", info.ExePath)
					}
					return gbe.ApplyGBE(c.Context, platform, appID, c.Bool("dry-run"), c.Bool("portable"))
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
				Name:  "tui",
				Usage: "Start the terminal user interface",
				Action: func(c *cli.Context) error {
					return tui.Run()
				},
			},
			{
				Name:      "lutris-add",
				Usage:     "Add a game to Lutris",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Game title"},
					&cli.StringFlag{Name: "appid", Usage: "Steam AppID"},
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

					info, err := detector.Detect(c.Context, path)
					if err != nil {
						return fmt.Errorf("auto-detect failed: %w", err)
					}
					if err := detector.ConsolidateManifests(info, c.Bool("promote")); err != nil {
						slog.Warn("Failed to consolidate manifests", "error", err)
					}

					name := c.String("name")
					if name == "" {
						name = info.Name
					}

					appID := c.String("appid")
					if appID == "" {
						appID = info.AppID
					}

					runner := c.String("runner")
					if runner == "" {
						if info.Platform == "linux" {
							runner = "linux"
						} else {
							runner = "wine"
						}
					}

					var env map[string]string
					if !c.Bool("portable") {
						home, _ := os.UserHomeDir()
						env = make(map[string]string)
						if runner == "linux" {
							env["LD_PRELOAD"] = filepath.Join(home, config.GbeDir, "linux_release", "experimental", "x64", "steamclient.so")
						} else {
							env["SteamClient64Dll"] = filepath.Join(home, config.GbeDir, "win_release", "experimental", "x64", "steamclient64.dll")
						}
					}

					dryRun := c.Bool("dry-run")
					lcfg := lutris.Config{
						Name:       name,
						GamePath:   info.ExePath,
						Runner:     runner,
						PrefixPath: c.String("wine-prefix"),
						Env:        env,
					}

					if dryRun {
						slog.Info("[DRY RUN] Would write Lutris YAML", "name", name, "exe", info.ExePath, "runner", lcfg.Runner)
						_, _ = steam.FetchLutrisArtwork(c.Context, appID, lutris.Slugify(name), true)
						return nil
					}

					home, err := os.UserHomeDir()
					if err != nil {
						return fmt.Errorf("get home dir: %w", err)
					}
					targetDir := filepath.Join(home, config.LutrisDir)
					if err := lutris.Write(lcfg, targetDir); err != nil {
						return fmt.Errorf("lutris install: %w", err)
					}
					slog.Info("Successfully wrote Lutris game config", "name", name, "dir", targetDir)

					// Fetch cover art & banners for Lutris
					_, _ = steam.FetchLutrisArtwork(c.Context, appID, lutris.Slugify(name), false)

					return nil
				},
			},
			{
				Name:      "steam-add",
				Usage:     "Add a non-Steam game shortcut to Steam",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Usage: "Shortcut name"},
					&cli.StringFlag{Name: "appid", Usage: "Steam AppID"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "portable", Usage: "Use direct DLL replacement instead of adding pre-load launch options"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Steam] manifests"},
				},
				Action: func(c *cli.Context) error {
					path := "."
					if c.Args().Len() >= 1 {
						path = c.Args().Get(0)
					}

					info, err := detector.Detect(c.Context, path)
					if err != nil {
						return fmt.Errorf("auto-detect failed: %w", err)
					}
					if err := detector.ConsolidateManifests(info, c.Bool("promote")); err != nil {
						slog.Warn("Failed to consolidate manifests", "error", err)
					}

					name := c.String("name")
					if name == "" {
						name = info.Name
					}

					appID := c.String("appid")
					if appID == "" {
						appID = info.AppID
					}

					launchOpt := ""
					if !c.Bool("portable") {
						home, _ := os.UserHomeDir()
						soPath := filepath.Join(home, config.GbeDir, "linux_release", "experimental", "x64", "steamclient.so")
						launchOpt = fmt.Sprintf("LD_PRELOAD=%s %%command%%", soPath)
					}

					dryRun := c.Bool("dry-run")
					scfg := steamshortcut.ShortcutConfig{
						Name:      name,
						ExePath:   info.ExePath,
						AppID:     appID,
						LaunchOpt: launchOpt,
					}
					return steamshortcut.RegisterShortcut(c.Context, scfg, dryRun)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
