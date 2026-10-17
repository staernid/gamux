package main

import (
	"context"
	"strconv"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/ui"
	"github.com/urfave/cli/v2"
)

func commonSetupFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Automatic yes to all prompts (non-interactive mode)"},
		&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder"},
		&cli.BoolFlag{Name: "loader", Usage: "Perform loader mode setup without modifying game DLLs"},
		&cli.BoolFlag{Name: "no-gbe", Usage: "Disable Goldberg Emulator patching"},
		&cli.BoolFlag{Name: "normalize", Value: true, Usage: "Normalize directory name to match Steam's official 1:1 installdir"},
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be done without writing"},
		&cli.BoolFlag{Name: "no-steamless", Usage: "Disable automatic Steamless SteamStub DRM executable unpacking"},
		&cli.BoolFlag{Name: "achievements", Usage: "Generate GBE achievements.json schema & download icon assets"},
		&cli.StringFlag{Name: "lang", Aliases: []string{"language"}, Usage: "Force Goldberg emulator language (e.g. english, german, french, spanish, japanese, schinese)"},
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
		&cli.BoolFlag{Name: "setup", Usage: "Automatically run post-download GBE setup after downloading"},
		&cli.BoolFlag{Name: "portable", Usage: "Perform direct DLL/SO replacement in game folder during setup"},
		&cli.BoolFlag{Name: "loader", Usage: "Perform loader mode setup without modifying game DLLs during setup"},
		&cli.BoolFlag{Name: "no-gbe", Usage: "Disable Goldberg Emulator patching during post-download setup"},
		&cli.BoolFlag{Name: "no-steamless", Usage: "Disable Steamless SteamStub DRM unpacking during setup"},
		&cli.BoolFlag{Name: "achievements", Usage: "Generate GBE achievements schema during setup"},
		&cli.StringFlag{Name: "lang", Aliases: []string{"language"}, Usage: "Voice/audio language pack preference (defaults to config language or english)"},
		&cli.UintSliceFlag{Name: "depot", Usage: "Explicit Steam depot ID(s) to include (repeatable)"},
		&cli.UintSliceFlag{Name: "exclude-depot", Usage: "Explicit Steam depot ID(s) to exclude (repeatable)"},
		&cli.BoolFlag{Name: "include-optional", Usage: "Include optional HD texture packs and bonus depots"},
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

	targetDir := customDir
	if targetDir == "" && arg != "" && (appID == 0 || arg != strconv.FormatUint(uint64(appID), 10)) {
		targetDir = arg
	}

	platform := c.String("platform")
	resolved, err := engine.ResolveDownloadTarget(ctx, activeConfig, appID, targetDir, platform)
	if err == nil && resolved != "" {
		return appID, resolved
	}

	if targetDir != "" {
		return appID, targetDir
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

	// 1. Operating Mode Resolution
	opMode := cfg.OperatingMode
	if opMode == "" {
		opMode = domain.ModeLoader
	}

	if c.IsSet("portable") {
		opMode = domain.ModePortable
	} else if c.IsSet("loader") {
		opMode = domain.ModeLoader
	} else if c.IsSet("no-gbe") {
		opMode = domain.ModeNone
	} else if promptIfUnset && !autoYes {
		opMode = ui.PromptSelectOperatingMode(opMode)
	}

	// 2. Operational Toggles
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

	winePrefix := cfg.WinePrefix
	if c.IsSet("wine-prefix") && c.String("wine-prefix") != "" {
		winePrefix = c.String("wine-prefix")
	}

	lang := cfg.Language
	if c.IsSet("lang") {
		lang = c.String("lang")
	}

	path := ""
	if c.Args().Len() >= 1 {
		path = c.Args().Get(0)
	}
	exeFlag := c.String("exe")

	return engine.ProcessOptions{
		Path:              path,
		OperatingMode:     opMode,
		WinePrefix:        winePrefix,
		NormalizeDir:      normalize,
		DryRun:            c.Bool("dry-run"),
		AutoYes:           autoYes,
		NoSteamless:       noSteamless,
		FetchAchievements: fetchAchievements,
		Language:          lang,
		ExePath:           exeFlag,
	}
}
