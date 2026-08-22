package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/ui"
	"github.com/staernid/gamux/util"
	"github.com/urfave/cli/v2"
)

func commonSetupFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "lutris", Usage: "Enable Lutris library registration"},
		&cli.BoolFlag{Name: "no-lutris", Usage: "Disable Lutris library registration"},
		&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts (non-interactive mode)"},
		&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder"},
		&cli.BoolFlag{Name: "loader", Usage: "Perform loader mode setup without modifying game DLLs"},
		&cli.BoolFlag{Name: "normalize", Value: true, Usage: "Normalize directory name to match Steam's official 1:1 installdir"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
		&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
		&cli.BoolFlag{Name: "achievements", Usage: "Generate GBE achievements.json schema & download icon assets"},
		&cli.StringFlag{Name: "exe", Usage: "Specify game executable path or launch option index (e.g. --exe 1 or --exe bin/game_vk.exe)"},
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
	return ""
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

func extractProcessOptions(c *cli.Context, promptIfUnset bool, cfg *config.Config) engine.ProcessOptions {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	autoYes := isAutoYes(c)

	// 1. GBE & Portable/Loader Mode Resolution
	applyGBE := true
	portable := false
	gbeMode := cfg.GbeMode

	if c.IsSet("portable") {
		gbeMode = "portable"
	} else if c.IsSet("loader") {
		gbeMode = "loader"
	} else if c.IsSet("no-gbe") {
		gbeMode = "disabled"
	}

	if strings.EqualFold(gbeMode, "portable") {
		applyGBE = true
		portable = true
	} else if strings.EqualFold(gbeMode, "loader") {
		applyGBE = true
		portable = false
	} else if strings.EqualFold(gbeMode, "disabled") || strings.EqualFold(gbeMode, "none") || strings.EqualFold(gbeMode, "off") {
		applyGBE = false
		portable = false
	} else if promptIfUnset && !autoYes {
		applyGBE = ui.PromptYesNoWithExplanation(
			"Apply Goldberg Emulator & Steamless DRM removal?",
			"Unpacks SteamStub DRM, configures DLCs, and sets up Steam API emulation.",
			true,
		)
		if applyGBE {
			portable = ui.PromptYesNoWithExplanation(
				"Use Portable Mode (Direct DLL/SO replacement)?",
				"Portable mode replaces steam_api.dll directly in the game folder (backed up to .ORIGINAL). Default Loader mode keeps original game files 100% untouched.",
				false,
			)
		}
	}

	// 2. Lutris Integration Resolution
	addLutris := cfg.Lutris
	if c.IsSet("no-lutris") {
		addLutris = false
	} else if c.IsSet("lutris") {
		addLutris = c.Bool("lutris")
	}

	// 3. Operational Toggles
	normalize := cfg.Normalize
	if c.IsSet("normalize") {
		normalize = c.Bool("normalize")
	}

	noSteamless := !cfg.Steamless
	if c.IsSet("no-steamless") {
		noSteamless = c.Bool("no-steamless")
	}

	fetchAchievements := cfg.Achievements
	if c.IsSet("achievements") {
		fetchAchievements = c.Bool("achievements")
	}

	runner := cfg.Runner
	if c.IsSet("runner") && c.String("runner") != "" {
		runner = c.String("runner")
	}

	winePrefix := cfg.WinePrefix
	if c.IsSet("wine-prefix") && c.String("wine-prefix") != "" {
		winePrefix = c.String("wine-prefix")
	}

	path := ""
	if c.Args().Len() >= 1 {
		path = c.Args().Get(0)
	}
	exeFlag := c.String("exe")

	return engine.ProcessOptions{
		Path:              path,
		ApplyGBE:          applyGBE,
		AddLutris:         addLutris,
		Runner:            runner,
		WinePrefix:        winePrefix,
		Portable:          portable,
		NormalizeDir:      normalize,
		DryRun:            c.Bool("dry-run"),
		AutoYes:           autoYes,
		NoSteamless:       noSteamless,
		FetchAchievements: fetchAchievements,
		ExePath:           exeFlag,
	}
}
