package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

var savesCommand = &cli.Command{
	Name:      "saves",
	Category:  "Step 3: Inspection & Maintenance",
	Usage:     "Inspect game save/config paths, create portable snapshots, and restore backups",
	ArgsUsage: "[path]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Output save location inspection or backup info as JSON"},
		&cli.BoolFlag{Name: "backup", Usage: "Create a new save & config backup snapshot archive"},
		&cli.StringFlag{Name: "note", Usage: "Optional note or description to attach to the backup snapshot"},
		&cli.BoolFlag{Name: "list", Usage: "List all existing backup snapshots for this game"},
		&cli.StringFlag{Name: "restore", Usage: "Restore a specific backup snapshot archive ID"},
		&cli.BoolFlag{Name: "update-manifest", Usage: "Force re-download and re-index the Ludusavi save manifest"},
	},
	Action: func(c *cli.Context) error {
		eng := engine.New(activeConfig)

		if c.Bool("update-manifest") {
			fmt.Printf("Updating Ludusavi save manifest from GitHub...\n")
			if err := eng.UpdateSaveManifest(c.Context); err != nil {
				return fmt.Errorf("update save manifest: %w", err)
			}
			fmt.Printf("Successfully updated and indexed Ludusavi save manifest.\n")
			if c.Args().Len() == 0 {
				return nil
			}
		}

		if c.Args().Len() == 0 {
			ui.RenderCommandHelp(c.Command, Version)
			return nil
		}

		path := extractPath(c)

		// 1. Restore Backup
		if restoreID := c.String("restore"); restoreID != "" {
			if err := eng.RestoreSaveBackup(c.Context, path, restoreID); err != nil {
				return fmt.Errorf("restore save backup: %w", err)
			}
			fmt.Printf("Successfully restored save backup %q for %s\n", restoreID, path)
			return nil
		}

		// 2. Create Backup Snapshot
		if c.Bool("backup") {
			note := c.String("note")
			info, err := eng.CreateSaveBackup(c.Context, path, note, "cli")
			if err != nil {
				return fmt.Errorf("create save backup: %w", err)
			}

			if c.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			fmt.Printf("Save snapshot created successfully!\n")
			fmt.Printf("  ID:        %s\n", info.ID)
			fmt.Printf("  Archive:   %s\n", info.ZipPath)
			fmt.Printf("  Files:     %d\n", info.FileCount)
			fmt.Printf("  Size:      %s\n", ui.FormatBytes(info.SizeBytes))
			if info.Note != "" {
				fmt.Printf("  Note:      %s\n", info.Note)
			}
			return nil
		}

		// 3. List Backups
		if c.Bool("list") {
			backups, err := eng.ListSaveBackups(c.Context, path)
			if err != nil {
				return fmt.Errorf("list save backups: %w", err)
			}

			if c.Bool("json") {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(backups)
			}

			if len(backups) == 0 {
				fmt.Printf("No save backups found for %s\n", path)
				return nil
			}

			fmt.Printf("Available Save Snapshots for %s (%d total):\n", path, len(backups))
			for _, b := range backups {
				fmt.Printf("  • [%s] %s (%d files, %s)\n", b.ID, b.ZipPath, b.FileCount, ui.FormatBytes(b.SizeBytes))
				if b.Note != "" {
					fmt.Printf("      Note: %s\n", b.Note)
				}
			}
			return nil
		}

		// 4. Default: Inspect Save Locations
		res, err := eng.InspectSaveLocations(c.Context, path)
		if err != nil {
			return fmt.Errorf("inspect save locations: %w", err)
		}

		if c.Bool("json") {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}

		fmt.Printf("Save & Config Locations for: %s (AppID: %s, Platform: %s)\n", res.GameName, res.AppID, res.Platform)
		fmt.Printf("Total Save Size: %s | Config Files: %d | Cloud Emulation: %v\n\n",
			ui.FormatBytes(res.TotalSaveBytes),
			res.TotalConfigFiles,
			res.HasCloudSaves,
		)

		for _, loc := range res.Locations {
			status := "EXISTS"
			if !loc.Exists {
				status = "NOT FOUND ON DISK"
			}
			categoryBadge := fmt.Sprintf("[%s]", loc.Category)
			fmt.Printf("  %-10s %s (%s)\n", categoryBadge, loc.DisplayPath, status)
			if loc.Exists {
				fmt.Printf("             %s (%d files, source: %s)\n", ui.FormatBytes(loc.SizeBytes), loc.FileCount, loc.Source)
			}
		}

		return nil
	},
}
