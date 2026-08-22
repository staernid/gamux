package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var downloadCommand = &cli.Command{
	Name:      "download",
	Category:  "Step 1: Acquisition",
	Usage:     "Download pristine game files directly from Steam CDNs",
	ArgsUsage: "[appid_or_target_dir]",
	Flags:     commonDownloadFlags(),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 && !c.IsSet("dir") && !c.IsSet("app") {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
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
			renderCLIError(err,
				"Pass numeric AppID directly: gamux download <appid>",
				"Or pass --app <appid> flag: gamux download . --app <appid>",
				"Set hubcap_api_key in ~/.config/gamux/config.json",
			)
			return err
		}

		platform := c.String("platform")
		if platform == "" && activeConfig != nil && activeConfig.Platform != "" {
			platform = activeConfig.Platform
		}
		if platform == "" && !isAutoYes(c) {
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

		opts := downloader.DownloadOptions{
			TargetDir: targetDir,
			AppID:     appID,
			LuaPath:   luaPath,
			Platform:  platform,
			DryRun:    c.Bool("dry-run"),
		}

		ui.RenderStep(1, 1, fmt.Sprintf("Downloading AppID %d (%s) to %s", appID, platform, targetDir))
		res, err := downloader.DownloadGame(c.Context, activeConfig, opts)
		if err != nil {
			renderCLIError(err,
				fmt.Sprintf("Place official binary depot .manifest file into '%s/[Manifests]/'", targetDir),
				"Or set hubcap_api_key in ~/.config/gamux/config.json to fetch manifest files automatically",
				"Verify depot decryption keys and network connectivity",
			)
			return err
		}

		if c.Bool("setup") && !c.Bool("dry-run") {
			ui.RenderStep(1, 1, "Executing post-download setup for "+targetDir)
			eng := engine.New(activeConfig)
			processOpts := extractProcessOptions(c, false, activeConfig)
			processOpts.Path = targetDir
			processOpts.AutoYes = true
			if _, setupErr := eng.ProcessGame(c.Context, processOpts); setupErr != nil {
				slog.Warn("Post-download setup encountered warnings", "error", setupErr)
			}
		}

		nextSteps := []string{}
		if !c.Bool("setup") {
			nextSteps = append(nextSteps,
				fmt.Sprintf("Run 'gamux apply %s' to set up Goldberg Emulator & Lutris", targetDir),
				fmt.Sprintf("Or run 'gamux status %s' to inspect game state", targetDir),
			)
		} else {
			nextSteps = append(nextSteps, fmt.Sprintf("Launch via Lutris or run 'gamux status %s'", targetDir))
		}
		ui.RenderSuccess(fmt.Sprintf("Download complete (%d files updated)", res.UpdatedFiles), "", nextSteps)
		return nil
	},
}

var workshopCommand = &cli.Command{
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
			renderCLIError(err, "Example: gamux workshop https://steamcommunity.com/sharedfiles/filedetails/?id=315783921")
			return err
		}

		input := c.Args().Get(0)
		targetDir := c.String("dir")
		if targetDir == "" {
			targetDir = "."
		}

		ui.RenderStep(1, 1, "Fetching Workshop item details: "+input)
		if err := steam.DownloadWorkshopItem(c.Context, input, targetDir, c.Bool("dry-run")); err != nil {
			renderCLIError(err, "Verify Workshop URL or ID", "Check internet connection")
			return err
		}

		ui.RenderSuccess("Workshop item setup complete", "", []string{
			fmt.Sprintf("Run 'gamux apply %s' to refresh game configuration", targetDir),
		})
		return nil
	},
}
