package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/pipeline"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var applyCommand = &cli.Command{
	Name:      "apply",
	Category:  "Step 2: Setup & Integration",
	Usage:     "Apply Goldberg Emulator setup and folder normalization for a game",
	ArgsUsage: "[path]",
	Flags:     commonSetupFlags(),
	Action: func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}
		path := extractPath(c)
		eng := engine.New(activeConfig)

		var status *domain.Game
		if st, err := eng.InspectStatus(c.Context, path); err == nil {
			status = st
		}

		if !isAutoYes(c) && status != nil {
			ui.RenderDetectionSummary(status)
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
			if selected, pErr := ui.PromptSelectLaunchOption(status.Name, status.LaunchCandidates); pErr == nil {
				opts.ExePath = selected.Executable
				opts.ExeArgs = selected.Arguments
			}
		}

		ui.RenderStep(1, 1, "Applying configuration for "+opts.Path)
		gameName := ""
		if status != nil {
			gameName = status.Name
		}
		pctx := &pipeline.Context{
			Options:  opts,
			Game:     domain.NewGame(gameName, opts.Path, "", ""),
			Config:   eng.Config,
			EventBus: eng.EventBus,
		}
		applyOp := pipeline.NewSetupOperation(gameName, opts.Path, pctx)
		err := eng.Tasks.RunSyncOp(c.Context, applyOp)
		if err != nil {
			renderCLIError(err, "Check directory path", "Verify file permissions")
			return err
		}

		nextSteps := []string{
			fmt.Sprintf("Run 'gamux launch %s' to start the game", opts.Path),
			fmt.Sprintf("Run 'gamux status %s' to view mod files & update status", opts.Path),
		}

		successName := opts.Path
		if pctx.Game != nil && pctx.Game.Name != "" {
			successName = pctx.Game.Name
		}
		ui.RenderSuccess("Setup applied successfully for "+successName, "", nextSteps)
		return nil
	},
}
