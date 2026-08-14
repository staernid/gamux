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
	"strings"


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
		&cli.BoolFlag{Name: "normalize", Value: true, Usage: "Normalize directory name to match Steam's official 1:1 installdir"},
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
		&cli.BoolFlag{Name: "setup", Usage: "Automatically run post-download GBE & Lutris setup after downloading"},
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
			cleanTitle := util.SanitizeInstallDir(title)
			return appID, filepath.Join(".", cleanTitle)
		}
		return appID, fmt.Sprintf("./%d", appID)
	}

	if arg != "" {
		return appID, arg
	}

	return appID, "."
}

func isAutoYes(c *cli.Context) bool {
	if c.Bool("yes") || c.IsSet("yes") || c.IsSet("y") {
		return true
	}
	for _, arg := range c.Args().Slice() {
		if arg == "-y" || arg == "--yes" || strings.HasPrefix(arg, "-y=") || strings.HasPrefix(arg, "--yes=") {
			return true
		}
	}
	return false
}


func extractProcessOptions(c *cli.Context, promptIfUnset bool) engine.ProcessOptions {
	autoYes := isAutoYes(c)



	// Prompt 1: Apply Goldberg Emulator & DRM Removal
	applyGBE := true
	if promptIfUnset && !autoYes {
		applyGBE = ui.PromptYesNoWithExplanation(
			"Apply Goldberg Emulator & Steamless DRM removal?",
			"Unpacks SteamStub DRM, configures DLCs, and sets up Steam API emulation.",
			true,
		)
	}

	// Prompt 2: Portable Mode vs Loader Mode (only if GBE is applied)
	portable := c.Bool("portable")
	if applyGBE && !c.IsSet("portable") && promptIfUnset && !autoYes {
		portable = ui.PromptYesNoWithExplanation(
			"Use Portable Mode (Direct DLL/SO replacement)?",
			"Portable mode replaces steam_api.dll directly in the game folder (backed up to .ORIGINAL). Default Loader mode keeps original game files 100% untouched.",
			false,
		)
	}

	// Prompt 3: Register in Lutris
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

	normalize := true
	if c.IsSet("normalize") {
		normalize = c.Bool("normalize")
	}

	return engine.ProcessOptions{
		Path:              extractPath(c),
		ApplyGBE:          applyGBE,
		AddLutris:         addLutris,
		Runner:            c.String("runner"),
		WinePrefix:        c.String("wine-prefix"),
		Portable:          portable,
		Promote:           c.Bool("promote"),
		NormalizeDir:      normalize,
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
			{
				Name:      "download",
				Category:  "Step 1: Acquisition",
				Usage:     "Download pristine game depot files directly from Steam CDNs",
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
						if info, err := detector.Detect(c.Context, targetDir); err == nil && info.AppID != "" && info.AppID != "0" {
							if parsedID, err := strconv.ParseUint(info.AppID, 10, 32); err == nil && parsedID > 0 {
								appID = uint32(parsedID)
							}
						}
					}

					if appID == 0 {
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
						TargetDir:  targetDir,
						AppID:      appID,
						LuaPath:    luaPath,
						Platform:   platform,
						DryRun:     c.Bool("dry-run"),
						AuditTrace: parsedLua.AuditTrace,
					}


					res, err := downloader.DownloadOrUpdateGame(c.Context, dummyManifest, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{
							fmt.Sprintf("Place official binary depot .manifest file into '%s/[Manifests]/'", targetDir),
							"Or set hubcap_api_key in ~/.config/gamux/config.json to fetch manifest files automatically",
							"Verify depot decryption keys and network connectivity",
						})
						return err
					}


					if c.Bool("setup") && !c.Bool("dry-run") {
						ui.RenderStep(1, 1, "Executing post-download setup for "+targetDir)
						eng := engine.New(activeConfig)
						processOpts := extractProcessOptions(c, false)
						processOpts.Path = targetDir
						processOpts.AutoYes = true
						if _, setupErr := eng.ProcessGame(c.Context, processOpts); setupErr != nil {
							slog.Warn("Post-download setup encountered warnings", "error", setupErr)
						}
					}

					nextSteps := []string{}
					if !c.Bool("setup") {
						nextSteps = append(nextSteps,
							fmt.Sprintf("Run 'gamux sync %s' to set up Goldberg Emulator & Lutris", targetDir),
							fmt.Sprintf("Or run 'gamux status %s' to inspect game state", targetDir),
						)
					} else {
						nextSteps = append(nextSteps, fmt.Sprintf("Launch via Lutris or run 'gamux status %s'", targetDir))
					}
					ui.RenderSuccess(fmt.Sprintf("Download complete (%d files updated)", res.UpdatedFiles), "", nextSteps)
					return nil
				},
			},
			{
				Name:      "workshop",
				Category:  "Step 1: Acquisition",
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

					ui.RenderSuccess("Workshop item setup complete", "", []string{
						fmt.Sprintf("Run 'gamux sync %s' to refresh game configuration", targetDir),
					})
					return nil
				},
			},
			{
				Name:      "sync",
				Category:  "Step 2: Setup & Integration",
				Usage:     "One-shot setup, directory normalization, ACF synthesis, and Lutris integration for a game",
				ArgsUsage: "[path]",
				Flags:     commonSetupFlags(),
				Action: func(c *cli.Context) error {
					path := extractPath(c)
					eng := engine.New(activeConfig)

					if !isAutoYes(c) {


						status, err := eng.InspectStatus(c.Context, path)
						if err == nil {
							ui.RenderDetectionSummary(ui.DetectionInfoSummary{
								Name:              status.Name,
								AppID:             status.AppID,
								Platform:          status.Platform,
								GameDir:           status.GameDir,
								ExePath:           status.ExePath,
								State:             status.State,
								OriginalBackups:   len(status.OriginalBackups),
								ManifestID:        status.ManifestID,
								BuildID:           status.BuildID,
								DiskSizeBytes:     status.DiskSizeBytes,
								FileCount:         status.FileCount,
								OfficialFileCount: status.OfficialFileCount,
								UntrackedFiles:    status.UntrackedFiles,
								HasUpdate:         status.HasUpdate,
								RemoteManifestID:  status.RemoteManifestID,
								DLCCount:          status.DLCCount,
								AchievementCount:  status.AchievementCount,
								LutrisRegistered:  status.LutrisRegistered,
								RecentPatchNote:   status.RecentPatchNote,
							})
						}
					}

					opts := extractProcessOptions(c, true)
					opts.Path = path

					ui.RenderStep(1, 1, "Executing unified sync for "+opts.Path)
					res, err := eng.SyncGame(c.Context, opts)
					if err != nil {
						ui.RenderErrorHelp(err, []string{"Check directory path", "Verify file permissions"})
						return err
					}

					nextSteps := []string{}
					if opts.AddLutris {
						nextSteps = append(nextSteps, "Launch game via Lutris or application menu")
					}
					nextSteps = append(nextSteps, fmt.Sprintf("Run 'gamux status %s' to view mod files & update status", opts.Path))

					ui.RenderSuccess("Sync completed successfully for "+res.Info.Name, "", nextSteps)
					return nil
				},
			},

			{
				Name:      "batch",
				Category:  "Step 2: Setup & Integration",
				Usage:     "Run operations (status, sync) across a parent directory containing multiple game folders",
				ArgsUsage: "[status|sync] <dir>",
				Flags: append(commonSetupFlags(),
					&cli.BoolFlag{Name: "json", Usage: "Output batch results as JSON"},
				),
				Action: func(c *cli.Context) error {
					if c.Args().Len() < 1 {
						err := fmt.Errorf("batch command requires a parent directory path")
						ui.RenderErrorHelp(err, []string{"Example: gamux batch /path/to/downloads", "Example: gamux batch status /path/to/downloads"})
						return err
					}

					verb := "sync"
					parentDir := c.Args().Get(0)
					if c.Args().Len() >= 2 {
						verb = strings.ToLower(c.Args().Get(0))
						parentDir = c.Args().Get(1)
					}

					eng := engine.New(activeConfig)

					if verb == "status" {
						statuses, err := eng.BatchInspect(c.Context, parentDir)
						if err != nil {
							ui.RenderErrorHelp(err, []string{"Verify directory exists"})
							return err
						}

						if c.Bool("json") {
							enc := json.NewEncoder(os.Stdout)
							enc.SetIndent("", "  ")
							return enc.Encode(statuses)
						}

						ui.RenderTerseHeader(parentDir, len(statuses))
						var loaderCount, cleanCount, updatesAvail, lutrisCount int
						for _, s := range statuses {
							if strings.Contains(strings.ToLower(s.State), "original") {
								cleanCount++
							} else {
								loaderCount++
							}
							if s.HasUpdate || len(s.ModifiedFiles) > 0 || len(s.MissingFiles) > 0 {
								updatesAvail++
							}
							if s.LutrisRegistered {
								lutrisCount++
							}

							ui.RenderTerseStatus(ui.DetectionInfoSummary{
								Name:             s.Name,
								AppID:            s.AppID,
								Store:            s.Store,
								Platform:         s.Platform,
								GameDir:          s.GameDir,

								State:            s.State,
								ManifestID:       s.ManifestID,
								ModifiedFiles:    s.ModifiedFiles,
								MissingFiles:     s.MissingFiles,
								UntrackedFiles:   s.UntrackedFiles,
								HasUpdate:        s.HasUpdate,
								RemoteManifestID: s.RemoteManifestID,
								LutrisRegistered: s.LutrisRegistered,
							})
						}
						ui.RenderTerseSummary(len(statuses), loaderCount, cleanCount, updatesAvail, lutrisCount)
						return nil
					}

					opts := extractProcessOptions(c, false)
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

					ui.RenderSuccess(fmt.Sprintf("Batch processing complete (%d games processed)", len(results)), "", []string{
						fmt.Sprintf("Run 'gamux status %s/<game_folder>' to inspect individual game state", parentDir),
					})
					return nil
				},
			},
			{
				Name:      "status",
				Category:  "Step 3: Inspection & Maintenance",
				Usage:     "Inspect game directory status (Original, Loader-Configured, or Portable-Patched)",
				ArgsUsage: "[path]",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "json", Usage: "Output status inspection result as JSON"},
					&cli.BoolFlag{Name: "news", Usage: "Display recent Steam patch notes and update history"},
					&cli.BoolFlag{Name: "terse", Aliases: []string{"t"}, Usage: "Display single-line compact status summary per game"},
				},
				Action: func(c *cli.Context) error {
					path := extractPath(c)
					eng := engine.New(activeConfig)

					if c.Bool("terse") {
						statuses, err := eng.BatchInspect(c.Context, path)
						if err == nil && len(statuses) > 0 {
							if c.Bool("json") {
								enc := json.NewEncoder(os.Stdout)
								enc.SetIndent("", "  ")
								return enc.Encode(statuses)
							}

							ui.RenderTerseHeader(path, len(statuses))
							var loaderCount, cleanCount, updatesAvail, lutrisCount int
							for _, s := range statuses {
								if strings.Contains(strings.ToLower(s.State), "original") {
									cleanCount++
								} else {
									loaderCount++
								}
								if s.HasUpdate || len(s.ModifiedFiles) > 0 || len(s.MissingFiles) > 0 {
									updatesAvail++
								}
								if s.LutrisRegistered {
									lutrisCount++
								}

								ui.RenderTerseStatus(ui.DetectionInfoSummary{
									Name:             s.Name,
									AppID:            s.AppID,
									Platform:         s.Platform,
									GameDir:          s.GameDir,
									State:            s.State,
									ManifestID:       s.ManifestID,
									ModifiedFiles:    s.ModifiedFiles,
									MissingFiles:     s.MissingFiles,
									UntrackedFiles:   s.UntrackedFiles,
									HasUpdate:        s.HasUpdate,
									RemoteManifestID: s.RemoteManifestID,
									LutrisRegistered: s.LutrisRegistered,
								})
							}
							ui.RenderTerseSummary(len(statuses), loaderCount, cleanCount, updatesAvail, lutrisCount)
							return nil
						}
					}

					newsCount := 1
					if c.Bool("news") {
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
						Name:              status.Name,
						AppID:             status.AppID,
						Platform:          status.Platform,
						GameDir:           status.GameDir,
						ExePath:           status.ExePath,
						State:             status.State,
						OriginalBackups:   len(status.OriginalBackups),
						ManifestID:        status.ManifestID,
						BuildID:           status.BuildID,
						DiskSizeBytes:     status.DiskSizeBytes,
						FileCount:         status.FileCount,
						OfficialFileCount: status.OfficialFileCount,
						ModifiedFiles:     status.ModifiedFiles,
						MissingFiles:      status.MissingFiles,
						UntrackedFiles:    status.UntrackedFiles,
						HasUpdate:         status.HasUpdate,

						RemoteManifestID:  status.RemoteManifestID,
						DLCCount:          status.DLCCount,
						AchievementCount:  status.AchievementCount,
						LutrisRegistered:  status.LutrisRegistered,
						RecentPatchNote:   status.RecentPatchNote,
						NewsItems:         newsItems,
					})
					return nil

				},
			},
			{
				Name:      "news",
				Category:  "Step 3: Inspection & Maintenance",
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
			{
				Name:      "rollback",
				Category:  "Step 3: Inspection & Maintenance",
				Usage:     "Restore original binaries & remove emulated settings from a game directory",
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
					ui.RenderSuccess("Rollback completed successfully for "+path, "", []string{
						fmt.Sprintf("Run 'gamux sync %s' whenever you want to re-enable GBE", path),
					})
					return nil
				},
			},
			{
				Name:     "update-gbe",
				Category: "Maintenance & Tools",
				Usage:    "Update Goldberg Emulator release assets from GitHub",
				Action: func(c *cli.Context) error {
					ui.RenderStep(1, 1, "Updating Goldberg Emulator release assets from GitHub")
					if err := github.UpdateGBE(c.Context); err != nil {
						ui.RenderErrorHelp(err, []string{"Check network connectivity"})
						return err
					}
					ui.RenderSuccess("Goldberg Emulator assets updated", "", []string{
						"Run 'gamux sync <game_dir>' to apply updated emulator files to your games",
					})
					return nil
				},
			},
		},
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
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
