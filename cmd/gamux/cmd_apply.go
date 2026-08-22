package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var applyCommand = &cli.Command{
	Name:      "apply",
	Category:  "Step 2: Setup & Integration",
	Usage:     "Apply Goldberg Emulator setup, folder normalization, and Lutris integration for a game",
	ArgsUsage: "[path]",
	Flags:     commonSetupFlags(),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		path := extractPath(c)
		eng := engine.New(activeConfig)

		var status *engine.GameStatus
		if st, err := eng.InspectStatus(c.Context, path); err == nil {
			status = st
		}

		if !isAutoYes(c) && status != nil {
			var uiCands []ui.LaunchCandidateSummary
			for _, lc := range status.LaunchCandidates {
				uiCands = append(uiCands, ui.LaunchCandidateSummary{
					Name:        lc.Name,
					Executable:  lc.Executable,
					Arguments:   lc.Arguments,
					Description: lc.Description,
				})
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
				UntrackedFiles:    status.UntrackedFiles,
				HasUpdate:         status.HasUpdate,
				RemoteManifestID:  status.RemoteManifestID,
				DLCCount:          status.DLCCount,
				AchievementCount:  status.AchievementCount,
				LutrisRegistered:  status.LutrisRegistered,
				RecentPatchNote:   status.RecentPatchNote,
			})
		}

		opts := extractProcessOptions(c, true, activeConfig)
		opts.Path = path

		// If user specified --exe as a numeric index or name, resolve against status.LaunchCandidates
		if opts.ExePath != "" && status != nil && len(status.LaunchCandidates) > 0 {
			if idx, err := strconv.Atoi(opts.ExePath); err == nil && idx >= 1 && idx <= len(status.LaunchCandidates) {
				opts.ExePath = status.LaunchCandidates[idx-1].Executable
				opts.ExeArgs = status.LaunchCandidates[idx-1].Arguments
			} else {
				for _, cand := range status.LaunchCandidates {
					if strings.EqualFold(cand.Name, opts.ExePath) || strings.EqualFold(filepath.Base(cand.Executable), opts.ExePath) {
						opts.ExePath = cand.Executable
						opts.ExeArgs = cand.Arguments
						break
					}
				}
			}
		} else if !isAutoYes(c) && !c.IsSet("exe") && status != nil && len(status.LaunchCandidates) > 1 {
			var uiCands []ui.LaunchCandidateSummary
			for _, lc := range status.LaunchCandidates {
				uiCands = append(uiCands, ui.LaunchCandidateSummary{
					Name:        lc.Name,
					Executable:  lc.Executable,
					Arguments:   lc.Arguments,
					Description: lc.Description,
				})
			}
			if selected, pErr := ui.PromptSelectLaunchOption(status.Name, uiCands); pErr == nil {
				opts.ExePath = selected.Executable
				opts.ExeArgs = selected.Arguments
			}
		}

		ui.RenderStep(1, 1, "Applying configuration for "+opts.Path)
		res, err := eng.ApplyGame(c.Context, opts)
		if err != nil {
			renderCLIError(err, "Check directory path", "Verify file permissions")
			return err
		}

		nextSteps := []string{}
		if opts.AddLutris {
			nextSteps = append(nextSteps, "Launch game via Lutris or application menu")
		}
		nextSteps = append(nextSteps, fmt.Sprintf("Run 'gamux status %s' to view mod files & update status", opts.Path))

		ui.RenderSuccess("Setup applied successfully for "+res.Info.Name, "", nextSteps)
		return nil
	},
}
