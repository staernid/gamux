package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/lutris"
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
			LutrisRegistered:  status.LutrisRegistered,
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

		eng := engine.New(activeConfig)
		item, gameName, err := eng.GetPatchNote(c.Context, targetPath, index)
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
	Usage:     "Run pre-launch update checks and launch game via Lutris",
	ArgsUsage: "[path]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "no-notify", Usage: "Disable pre-launch update notification check"},
	},
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		targetPath := extractPath(c)
		eng := engine.New(activeConfig)
		eng.Notifier = ui.NotifyIssue
		if !c.Bool("no-notify") {
			if err := eng.NotifyLaunch(c.Context, targetPath); err != nil {
				slog.Warn("Pre-launch notification check failed", "error", err)
			}
		}
		status, err := eng.InspectStatus(c.Context, targetPath)
		if err != nil {
			return fmt.Errorf("detect game: %w", err)
		}
		slug := lutris.Slugify(status.Name)
		ui.RenderStep(1, 1, "Launching "+status.Name+" via Lutris...")
		lutrisBin, err := exec.LookPath("lutris")
		if err != nil {
			return fmt.Errorf("lutris binary not found in PATH: %w", err)
		}
		cmd := exec.Command(lutrisBin, "lutris:rungame/"+slug)
		return cmd.Run()
	},
}
