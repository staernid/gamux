package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var statusCommand = &cli.Command{
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
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		path := extractPath(c)
		_, lib, _, _ := engine.NewServices(activeConfig)

		if c.Bool("terse") {
			statuses, err := lib.BatchInspect(c.Context, path)
			if err == nil && len(statuses) > 0 {
				if c.Bool("json") {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(statuses)
				}

				ui.RenderTerseHeader(path, len(statuses))
				var loaderCount, cleanCount, updatesAvail int
				for _, s := range statuses {
					if strings.Contains(strings.ToLower(s.State), "original") {
						cleanCount++
					} else {
						loaderCount++
					}
					if s.HasUpdate || len(s.ModifiedFiles) > 0 || len(s.MissingFiles) > 0 {
						updatesAvail++
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
					})
				}
				ui.RenderTerseSummary(len(statuses), loaderCount, cleanCount, updatesAvail)
				return nil
			}
		}

		newsCount := 1
		if c.Bool("news") {
			newsCount = 5
		}

		status, err := lib.InspectStatusEx(c.Context, path, newsCount)
		if err != nil {
			renderCLIError(err, "Ensure path points to a valid game directory")
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

		uiCands := make([]ui.LaunchCandidateSummary, len(status.LaunchCandidates))
		for i, lc := range status.LaunchCandidates {
			uiCands[i] = ui.LaunchCandidateSummary{
				Name:        lc.Name,
				Executable:  lc.Executable,
				Arguments:   lc.Arguments,
				Description: lc.Description,
			}
		}

		ui.RenderDetectionSummary(ui.DetectionInfoSummary{
			Name:              status.Name,
			AppID:             status.AppID,
			Platform:          status.Platform,
			GameDir:           status.GameDir,
			ExePath:           status.ExePath,
			LaunchCandidates:  uiCands,
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
			RecentPatchNote:   status.RecentPatchNote,
			NewsItems:         newsItems,
		})
		return nil
	},
}

var newsCommand = &cli.Command{
	Name:      "news",
	Category:  "Step 3: Inspection & Maintenance",
	Usage:     "Read full Steam patch notes and changelogs for a game",
	ArgsUsage: "[path_or_appid] [index]",
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		targetPath := "."
		index := 1

		if c.Args().Len() == 1 {
			arg := c.Args().Get(0)
			if num, err := strconv.Atoi(arg); err == nil {
				if num > 100 { // Likely an AppID (e.g. 3904850)
					targetPath = arg
					index = 1
				} else {
					index = num
					targetPath = "."
				}
			} else {
				targetPath = arg
				index = 1
			}
		} else if c.Args().Len() >= 2 {
			arg1 := c.Args().Get(0)
			arg2 := c.Args().Get(1)
			num1, err1 := strconv.Atoi(arg1)
			num2, err2 := strconv.Atoi(arg2)

			if err1 == nil && err2 != nil {
				// "gamux news 1 /path/to/game"
				index = num1
				targetPath = arg2
			} else if err1 != nil && err2 == nil {
				// "gamux news /path/to/game 1"
				targetPath = arg1
				index = num2
			} else if err1 == nil && err2 == nil {
				// "gamux news 3904850 1"
				targetPath = arg1
				index = num2
			} else {
				targetPath = arg1
				index = 1
			}
		}

		_, lib, _, _ := engine.NewServices(activeConfig)
		item, gameName, err := lib.GetPatchNote(c.Context, targetPath, index)
		if err != nil {
			renderCLIError(err, "Verify AppID or game directory path", "Check network connectivity")
			return err
		}

		ui.RenderNewsItem(gameName, index, item.Title, item.FeedLabel, item.Contents, item.URL, item.Date)
		return nil
	},
}

var launchCommand = &cli.Command{
	Name:      "launch",
	Category:  "Step 3: Inspection & Maintenance",
	Usage:     "Run pre-launch update checks and launch game (via UMU/Wine/Proton or Linux native)",
	ArgsUsage: "[path]",
	Aliases:   []string{"run", "play"},
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "runner", Usage: "Runner override ('umu', 'wine', 'proton', 'linux')"},
		&cli.StringFlag{Name: "prefix", Usage: "Custom Wine prefix path"},
		&cli.StringFlag{Name: "proton-path", Usage: "Custom Proton compatibility tool path"},
		&cli.BoolFlag{Name: "mangohud", Usage: "Enable MangoHud performance overlay (MANGOHUD=1)"},
		&cli.StringFlag{Name: "dxvk-hud", Usage: "Set DXVK HUD overlay elements (e.g. fps,compiler)"},
		&cli.BoolFlag{Name: "fsr", Usage: "Enable Wine fullscreen AMD FSR scaling"},
		&cli.BoolFlag{Name: "nvapi", Usage: "Enable Proton NVAPI support (DLSS)"},
		&cli.BoolFlag{Name: "setup", Usage: "Auto-apply GBE setup before launching if game is unconfigured"},
		&cli.BoolFlag{Name: "no-notify", Usage: "Disable pre-launch update notification check"},
	},
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		targetPath := extractPath(c)
		sc, lib, game, _ := engine.NewServices(activeConfig)
		sc.Notifier = ui.NotifyIssue
		if !c.Bool("no-notify") {
			if err := game.NotifyLaunch(c.Context, targetPath); err != nil {
				slog.Warn("Pre-launch notification check failed", "error", err)
			}
		}
		status, err := lib.InspectStatus(c.Context, targetPath)
		if err != nil {
			return fmt.Errorf("detect game: %w", err)
		}

		if c.Bool("setup") || status.IsClean() || !status.IsReady {
			if status.IsClean() || !status.IsReady {
				ui.RenderStep(1, 2, fmt.Sprintf("Auto-applying setup for %s before launch...", status.Name))
				applyOpts := engine.ProcessOptions{
					Path:       targetPath,
					ApplyGBE:   true,
					Portable:   true,
					WinePrefix: status.WinePrefix,
					ExePath:    status.ExePath,
					ExeArgs:    status.ExeArgs,
				}
				if _, applyErr := game.ApplyGame(c.Context, applyOpts); applyErr != nil {
					slog.Warn("Auto-apply before launch encountered warning", "error", applyErr)
				}
				if refreshed, rErr := lib.InspectStatus(c.Context, targetPath); rErr == nil && refreshed != nil {
					status = refreshed
				}
			}
		}

		launchOpts := runner.LaunchOptions{
			GameDir:       status.GameDir,
			ExePath:       status.ExePath,
			ExeArgs:       status.ExeArgs,
			AppID:         status.AppID,
			Name:          status.Name,
			Platform:      status.Platform,
			OperatingMode: status.OperatingMode,
			Runner:        status.Runner,
			WinePrefix:    status.WinePrefix,
		}

		if status.LaunchProfile != nil {
			launchOpts = *status.LaunchProfile
		}

		// Apply CLI flag overrides if explicitly specified
		if c.IsSet("runner") {
			launchOpts.Runner = c.String("runner")
		}
		if c.IsSet("prefix") {
			launchOpts.WinePrefix = c.String("prefix")
		}
		if c.IsSet("proton-path") {
			launchOpts.ProtonPath = c.String("proton-path")
		}
		if c.IsSet("mangohud") {
			b := c.Bool("mangohud")
			launchOpts.MangoHud = &b
		}
		if c.IsSet("dxvk-hud") {
			launchOpts.DXVKHud = c.String("dxvk-hud")
		}
		if c.IsSet("fsr") {
			b := c.Bool("fsr")
			launchOpts.WineFullscreenFSR = &b
		}
		if c.IsSet("nvapi") {
			b := c.Bool("nvapi")
			launchOpts.ProtonNVAPI = &b
		}

		effectiveOpts := runner.ResolveEffectiveOptions(activeConfig, launchOpts)
		ui.RenderStep(1, 1, fmt.Sprintf("Launching %s (runner: %s)...", status.Name, effectiveOpts.Runner))

		return game.RunGame(c.Context, effectiveOpts)
	},
}

