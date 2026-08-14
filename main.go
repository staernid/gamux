package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/ui"
	"github.com/staernid/gamux/util"

	"github.com/urfave/cli/v2"
)

// Version of the gamux application (set at build time via -ldflags)
var Version = "dev"

func commonSetupFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "lutris", Usage: "Add game to Lutris"},
		&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts (non-interactive mode)"},
		&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder instead of loader mode"},
		&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Manifests] manifests"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
		&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
		&cli.BoolFlag{Name: "achievements", Usage: "Generate GBE achievements.json schema & download icon assets"},
	}
}

func commonDownloadFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "lua", Usage: "Path to optional .lua key file (overrides automatic resolution)"},
		&cli.UintFlag{Name: "app", Usage: "Steam AppID to download (if positional argument is a directory)"},
		&cli.StringFlag{Name: "dir", Usage: "Target directory to download game files into"},
		&cli.StringFlag{Name: "hubcap-key", Usage: "Hubcap API key (defaults to hubcap_api_key in ~/.config/gamux/config.json)"},
		&cli.StringFlag{Name: "platform", Usage: "Target platform architecture (win64, linux, all; default: win64)"},
		&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Non-interactive mode (use defaults without prompting)"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be downloaded without writing files"},
		&cli.BoolFlag{Name: "no-setup", Usage: "Skip automatic post-download DRM unpacking & GBE setup"},
		&cli.BoolFlag{Name: "raw", Usage: "Alias for --no-setup"},
	}
}

func extractPath(c *cli.Context) string {
	if c.Args().Len() >= 1 {
		return c.Args().Get(0)
	}
	return "."
}

func resolveAppIDAndTargetDir(ctx context.Context, c *cli.Context) (uint32, string) {
	arg := extractPath(c)
	customDir := c.String("dir")

	var appID uint32
	if parsedID, err := strconv.ParseUint(arg, 10, 32); err == nil && parsedID > 0 {
		appID = uint32(parsedID)
	} else if c.IsSet("app") {
		appID = uint32(c.Uint("app"))
	}

	if customDir != "" {
		return appID, customDir
	}

	if appID > 0 && (arg == "" || arg == strconv.FormatUint(uint64(appID), 10)) {
		title, err := steam.FetchAppName(ctx, fmt.Sprintf("%d", appID))
		if err == nil && title != "" {
			cleanTitle := util.SanitizeFilename(title)
			return appID, filepath.Join(".", cleanTitle)
		}
		return appID, fmt.Sprintf("./%d", appID)
	}

	if arg != "" {
		return appID, arg
	}

	return appID, "."
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
			"Portable mode replaces steam_api.dll directly in the game folder (backed up to .ORIGINAL). Default Loader mode keeps original game files 100% untouched.",
			false,
		)
	}


	return engine.ProcessOptions{
		Path:              extractPath(c),
		AddLutris:         addLutris,
		Runner:            c.String("runner"),
		WinePrefix:        c.String("wine-prefix"),
		Portable:          portable,
		Promote:           c.Bool("promote"),
		DryRun:            c.Bool("dry-run"),
		AutoYes:           autoYes,
		NoSteamless:       c.Bool("no-steamless"),
		FetchAchievements: c.Bool("achievements"),
	}
}

func buildApp() *cli.App {
	var activeConfig *config.Config

	return &cli.App{
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
			ui.RenderAppHelp(c.App, Version)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:      "add",
				Category:  "Setup & Management",
				Usage:     "Complete post-download setup for a game (GBE + optional Lutris integration)",
				ArgsUsage: "[path]",
				Flags:     commonSetupFlags(),
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
				Category:  "Setup & Management",
				Usage:     "Apply GBE to Steam API files and configure DLCs",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "dry-run", Usage: "Do not move or write files, just show what would be done"},
					&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder instead of loader mode"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Manifests] manifests"},
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
				Category:  "Setup & Management",
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
				Category:  "Setup & Management",
				Usage:     "Inspect game directory status (Original, Loader-Configured, or Portable-Patched)",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "Output status inspection result as JSON"},
					&cli.BoolFlag{Name: "news", Usage: "Display recent Steam patch notes and update history"},
					&cli.BoolFlag{Name: "patch-notes", Usage: "Alias for --news"},
				},
				Action: func(c *cli.Context) error {
					path := extractPath(c)
					eng := engine.New(activeConfig)

					newsCount := 1
					if c.Bool("news") || c.Bool("patch-notes") {
						newsCount = 5
					}

					status, err := eng.InspectStatusEx(c.Context, path, newsCount)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Ensure path points to a valid game directory"})
						return err
					}

					if c.Bool("json") {
						enc := json.NewEncoder(os.Stdout)
						enc.SetIndent("", "  ")
						return enc.Encode(status)
					}

					newsItems := make([]ui.NewsItemSummary, len(status.NewsItems))
					for i, n := range status.NewsItems {
						newsItems[i] = ui.NewsItemSummary{
							Title:     n.Title,
							FeedLabel: n.FeedLabel,
							Date:      n.Date,
							URL:       n.URL,
						}
					}

					ui.RenderDetectionSummary(ui.DetectionInfoSummary{
						Name:             status.Name,
						AppID:            status.AppID,
						Platform:         status.Platform,
						GameDir:          status.GameDir,
						ExePath:          status.ExePath,
						State:            status.State,
						OriginalBackups:  len(status.OriginalBackups),
						ManifestID:       status.ManifestID,
						BuildID:          status.BuildID,
						DiskSizeBytes:    status.DiskSizeBytes,
						FileCount:        status.FileCount,
						DLCCount:         status.DLCCount,
						AchievementCount: status.AchievementCount,
						LutrisRegistered: status.LutrisRegistered,
						RecentPatchNote:  status.RecentPatchNote,
						NewsItems:        newsItems,
					})
					return nil
				},
			},
			{
				Name:      "rollback",
				Category:  "Setup & Management",
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
				Name:     "update",
				Category: "Maintenance",
				Usage:    "Update the GBE fork repository",
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
				Category:  "Setup & Management",
				Usage:     "Add a game to Lutris",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "runner", Usage: "Lutris runner (linux or wine)"},
					&cli.StringFlag{Name: "wine-prefix", Usage: "Path to Wine prefix"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
					&cli.BoolFlag{Name: "portable", Usage: "Use direct DLL replacement instead of configuring loader env vars"},
					&cli.BoolFlag{Name: "promote", Value: true, Usage: "Promote inner common/ game folder to top level and consolidate [Manifests] manifests"},
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
			{
				Name:      "download",
				Category:  "Acquisition",
				Usage:     "Download or update game files directly from Steam CDNs (automatic via RevoBD/Hubcap or .lua key file)",
				ArgsUsage: "[appid_or_target_dir]",
				Flags:     commonDownloadFlags(),
				Action: func(c *cli.Context) error {
					luaPath := c.String("lua")
					hubcapKey := c.String("hubcap-key")
					if hubcapKey == "" && activeConfig != nil {
						hubcapKey = activeConfig.HubcapAPIKey
					}

					appID, targetDir := resolveAppIDAndTargetDir(c.Context, c)

					if appID == 0 {
						// 1. Try detector scan
						if info, err := detector.Detect(c.Context, targetDir); err == nil && info.AppID != "" && info.AppID != "0" {
							if parsedID, err := strconv.ParseUint(info.AppID, 10, 32); err == nil && parsedID > 0 {
								appID = uint32(parsedID)
							}
						}
					}

					if appID == 0 {
						// 2. Fall back to directory basename Store search
						absDir, err := filepath.Abs(targetDir)
						if err == nil {
							folderName := filepath.Base(absDir)
							if folderName != "" && folderName != "." && folderName != "/" {
								slog.Info("Searching Steam Store for AppID from directory name", "query", folderName)
								candidates, err := steam.SearchAppIDCandidates(c.Context, folderName)
								if err == nil && len(candidates) > 0 {
									if len(candidates) == 1 || c.Bool("yes") {
										appID = candidates[0].AppID
										slog.Info("Resolved AppID from directory name search", "appID", appID, "title", candidates[0].Name)
									} else {
										uiCandidates := make([]ui.CandidateItem, len(candidates))
										for i, cand := range candidates {
											uiCandidates[i] = ui.CandidateItem{AppID: cand.AppID, Name: cand.Name}
										}
										if selected, promptErr := ui.PromptSelectCandidate(folderName, uiCandidates); promptErr == nil {
											appID = selected.AppID
										} else {
											appID = candidates[0].AppID
										}
									}
								}
							}
						}
					}

					if appID == 0 {
						err := fmt.Errorf("no AppID provided and could not resolve AppID for target '%s'", targetDir)
						ui.RenderErrorHelp(err, []string{
							"Pass numeric AppID directly: gamux download <appid>",
							"Or pass --app <appid> flag: gamux download . --app <appid>",
							"Set hubcap_api_key in ~/.config/gamux/config.json",
						})
						return err
					}

					platform := c.String("platform")
					if platform == "" && !c.Bool("yes") {
						title := fmt.Sprintf("%d", appID)
						if name, err := steam.FetchAppName(c.Context, fmt.Sprintf("%d", appID)); err == nil {
							title = fmt.Sprintf("%s (%d)", name, appID)
						}
						if selectedPlat, promptErr := ui.PromptSelectPlatform(title, nil); promptErr == nil {
							platform = selectedPlat
						}
					}
					if platform == "" {
						platform = "win64"
					}

					parsedLua, err := manifest.ResolveKeys(c.Context, appID, luaPath, hubcapKey)
					if err != nil {
						ui.RenderErrorHelp(err, []string{
							"Set hubcap_api_key in ~/.config/gamux/config.json",
							"Or pass --lua /path/to/game.lua for manual debug key file",
						})
						return err
					}

					appID = parsedLua.AppID
					ui.RenderStep(1, 1, fmt.Sprintf("Downloading AppID %d (%s) to %s", appID, platform, targetDir))

					if !c.Bool("dry-run") && len(parsedLua.ManifestFiles) > 0 {
						manifestsDir := filepath.Join(targetDir, "[Manifests]")
						_ = os.MkdirAll(manifestsDir, 0755)
						for fname, fcontent := range parsedLua.ManifestFiles {
							outPath := filepath.Join(manifestsDir, fname)
							_ = os.WriteFile(outPath, fcontent, 0644)
							slog.Info("Saved manifest file", "path", outPath)
						}
					}

					dummyManifest := &manifest.Manifest{
						AppID:     appID,
						DepotKeys: make(map[uint32]string),
					}
					for _, d := range parsedLua.Depots {
						dummyManifest.DepotKeys[d.DepotID] = d.DecryptionKey
					}
					if len(parsedLua.Depots) > 0 {
						dummyManifest.DepotID = parsedLua.Depots[0].DepotID
						dummyManifest.DecryptionKey = parsedLua.Depots[0].DecryptionKey
					}

					opts := downloader.DownloadOptions{
						TargetDir: targetDir,
						AppID:     appID,
						LuaPath:   luaPath,
						Platform:  platform,
						DryRun:    c.Bool("dry-run"),
					}

					res, err := downloader.DownloadOrUpdateGame(c.Context, dummyManifest, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Check network connectivity", "Verify depot decryption keys"})
						return err
					}

					if !c.Bool("no-setup") && !c.Bool("raw") && !c.Bool("dry-run") {
						ui.RenderStep(1, 1, "Executing automatic post-download setup for "+targetDir)
						eng := engine.New(activeConfig)
						processOpts := engine.ProcessOptions{
							Path:      targetDir,
							Promote:   true,
							AutoYes:   true,
						}
						if _, setupErr := eng.ProcessGame(c.Context, processOpts); setupErr != nil {
							slog.Warn("Post-download setup encountered warnings", "error", setupErr)
						}
					}

					ui.RenderSuccess(fmt.Sprintf("Download/Update complete (%d files updated)", res.UpdatedFiles), "", nil)
					return nil
				},
			},
			{
				Name:      "workshop",
				Category:  "Acquisition",
				Usage:     "Download a Steam Workshop item or mod directly into a game directory",
				ArgsUsage: "<url-or-id>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "dir", Usage: "Target game directory (defaults to current directory)"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be downloaded without writing files"},
				},
				Action: func(c *cli.Context) error {
					if c.Args().Len() < 1 {
						err := fmt.Errorf("workshop command requires a Workshop item URL or ID")
						ui.RenderErrorHelp(err, []string{"Example: gamux workshop https://steamcommunity.com/sharedfiles/filedetails/?id=315783921"})
						return err
					}

					input := c.Args().Get(0)
					targetDir := c.String("dir")
					if targetDir == "" {
						targetDir = "."
					}

					ui.RenderStep(1, 1, "Fetching Workshop item details: "+input)
					if err := steam.DownloadWorkshopItem(c.Context, input, targetDir, c.Bool("dry-run")); err != nil {
						ui.RenderErrorHelp(err, []string{"Verify Workshop URL or ID", "Check internet connection"})
						return err
					}

					ui.RenderSuccess("Workshop item setup complete", "", nil)
					return nil
				},
			},
			{
				Name:      "check-updates",
				Category:  "Maintenance",
				Usage:     "Scan game library directory for available Steam updates",
				ArgsUsage: "[parent_dir]",
				Action: func(c *cli.Context) error {
					dir := "."
					if c.Args().Len() > 0 {
						dir = c.Args().Get(0)
					}

					eng := engine.New(activeConfig)
					ui.RenderStep(1, 1, "Scanning game library for updates in "+dir)
					statuses, err := eng.CheckLibraryUpdates(c.Context, dir)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Verify target directory existence"})
						return err
					}

					if len(statuses) == 0 {
						fmt.Println("No games with [Manifests] found in directory.")
						return nil
					}

					fmt.Println()
					fmt.Println("🔍 Library Update Status:")
					fmt.Println("------------------------------------------------------------------------")
					for _, s := range statuses {
						fmt.Printf("  • %-25s AppID: %-10s Manifest: %s\n", s.GameTitle, s.AppID, s.LocalManifestID)
					}
					fmt.Println("------------------------------------------------------------------------")
					fmt.Println("💡 Run 'gamux download <appid> --dir <path>' to perform a fast delta update.")
					fmt.Println()
					return nil
				},
			},
			{
				Name:      "news",
				Category:  "Maintenance",
				Usage:     "Read full Steam patch notes and changelogs for a game",
				ArgsUsage: "[path_or_appid] [index]",
				Action: func(c *cli.Context) error {
					targetPath := "."
					index := 1

					if c.Args().Len() >= 1 {
						arg1 := c.Args().Get(0)
						if idx, err := strconv.Atoi(arg1); err == nil && idx > 0 {
							index = idx
						} else {
							targetPath = arg1
						}
					}
					if c.Args().Len() >= 2 {
						if idx, err := strconv.Atoi(c.Args().Get(1)); err == nil && idx > 0 {
							index = idx
						}
					}

					eng := engine.New(activeConfig)
					item, gameName, err := eng.GetPatchNote(c.Context, targetPath, index)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Verify AppID or game directory path", "Check network connectivity"})
						return err
					}

									ui.RenderNewsItem(gameName, index, item.Title, item.FeedLabel, item.Contents, item.URL, item.Date)
					return nil
				},
			},
		},
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		if app, ok := data.(*cli.App); ok {
			ui.RenderAppHelp(app, Version)
		} else {
			ui.RenderAppHelp(nil, Version)
		}
	}

	app := buildApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
