package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var batchCommand = &cli.Command{
	Name:      "batch",
	Category:  "Step 2: Setup & Integration",
	Usage:     "Run operations (status, apply) across a parent directory containing multiple game folders",
	ArgsUsage: "[status|apply] <dir>",
	Flags: append(commonSetupFlags(),
		&cli.BoolFlag{Name: "json", Usage: "Output batch results as JSON"},
	),
	Action: func(c *cli.Context) error {
		if c.Args().Len() < 1 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}

		verb := "apply"
		parentDir := c.Args().Get(0)
		if c.Args().Len() >= 2 {
			verb = strings.ToLower(c.Args().Get(0))
			parentDir = c.Args().Get(1)
		}

		eng := engine.New(activeConfig)

		if verb == "status" {
			statuses, err := eng.BatchInspect(c.Context, parentDir)
			if err != nil {
				renderCLIError(err, "Verify directory exists")
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

		opts := extractProcessOptions(c, false, activeConfig)
		if !c.Bool("json") {
			ui.RenderStep(1, 1, "Scanning parent directory: "+parentDir)
		}
		results, err := eng.BatchProcess(c.Context, parentDir, opts)
		if err != nil {
			renderCLIError(err, "Verify directory exists")
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
}
