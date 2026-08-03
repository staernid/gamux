package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// ProcessOptions holds configuration flags for processing a game directory.
type ProcessOptions struct {
	Path       string
	AddLutris  bool
	Portable   bool
	Promote    bool
	DryRun     bool
	AutoYes    bool
	WinePrefix string
	Runner     string
}

// ProcessResult summarizes the outcome of processing a game.
type ProcessResult struct {
	Info           *detector.GameInfo
	LutrisAdded    bool
	Patched        bool
	ManifestsMoved bool
	Errors         []string
}

// GameStatus describes the current patch & integration state of a game.
type GameStatus struct {
	Name               string
	AppID              string
	GameDir            string
	Platform           string
	ExePath            string
	State              string // "Original", "Loader-Configured", "Portable-Patched"
	OriginalBackups    []string
	SettingsDirExists  bool
	SteamAppIDTxtFound bool
}

// Engine is the core domain service orchestrating game scanning, patching, and launcher integrations.
type Engine struct {
	Config *config.Config
}

// New creates a new Engine instance.
func New(cfg *config.Config) *Engine {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Engine{Config: cfg}
}

// ProcessGame executes the end-to-end post-download setup for a single game directory.
func ProcessGame(ctx context.Context, cfg *config.Config, opts ProcessOptions) (*ProcessResult, error) {
	eng := New(cfg)
	return eng.ProcessGame(ctx, opts)
}

// ProcessGame processes a game directory according to the supplied options.
func (e *Engine) ProcessGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	targetPath := opts.Path
	if targetPath == "" {
		targetPath = "."
	}

	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("auto-detect failed for %s: %w", targetPath, err)
	}

	res := &ProcessResult{
		Info: info,
	}

	if err := detector.ConsolidateManifests(info, opts.Promote); err != nil {
		slog.Warn("Failed to consolidate manifests", "error", err)
		res.Errors = append(res.Errors, fmt.Sprintf("manifest consolidation: %v", err))
	} else {
		res.ManifestsMoved = true
	}

	slog.Info("Auto-detected game", "title", info.Name, "appID", info.AppID, "platform", info.Platform, "exe", info.ExePath)

	// 1. Apply GBE
	if err := gbe.ApplyGBE(ctx, e.Config, info.GameDir, info.Platform, info.AppID, opts.DryRun, opts.Portable); err != nil {
		slog.Warn("GBE application warning/error", "error", err)
		res.Errors = append(res.Errors, fmt.Sprintf("gbe patch: %v", err))
	} else {
		res.Patched = true
	}

	// 2. Add to Lutris
	if opts.AddLutris {
		if strings.TrimSpace(info.ExePath) == "" {
			err := fmt.Errorf("cannot register in Lutris: no executable (.exe or Linux binary) detected in %s", info.GameDir)
			slog.Error(err.Error())
			res.Errors = append(res.Errors, err.Error())
			return res, err
		}

		runner := opts.Runner
		if runner == "" {
			if info.Platform == "linux" {
				runner = "linux"
			} else {
				runner = "wine"
			}
		}

		var env map[string]string
		if !opts.Portable {
			gbeBase := e.Config.GbeDir
			if !filepath.IsAbs(gbeBase) {
				home, _ := os.UserHomeDir()
				gbeBase = filepath.Join(home, gbeBase)
			}
			env = make(map[string]string)
			if runner == "linux" {
				env["LD_PRELOAD"] = filepath.Join(gbeBase, "linux_release", "experimental", "x64", "steamclient.so")
			} else {
				env["SteamClient64Dll"] = filepath.Join(gbeBase, "win_release", "experimental", "x64", "steamclient64.dll")
			}
		}

		lcfg := lutris.Config{
			Name:       info.Name,
			GamePath:   info.ExePath,
			Runner:     runner,
			PrefixPath: opts.WinePrefix,
			Env:        env,
		}

		if opts.DryRun {
			slog.Info("[DRY RUN] Would write Lutris config & register in database", "name", info.Name)
			res.LutrisAdded = true
		} else {
			targetDir := e.Config.LutrisDir
			if !filepath.IsAbs(targetDir) {
				home, err := os.UserHomeDir()
				if err == nil {
					targetDir = filepath.Join(home, targetDir)
				}
			}
			if err := lutris.Write(lcfg, targetDir); err == nil {
				slog.Info("Successfully wrote Lutris game config & updated database", "name", info.Name, "dir", targetDir)
				_, _ = steam.FetchLutrisArtwork(ctx, info.AppID, lutris.Slugify(info.Name), false)
				res.LutrisAdded = true
			} else {
				slog.Error("Failed to write Lutris game config", "error", err)
				res.Errors = append(res.Errors, fmt.Sprintf("lutris register: %v", err))
				return res, fmt.Errorf("failed to register in Lutris: %w", err)
			}
		}
	}

	return res, nil
}

// BatchProcess scans a parent directory containing multiple game folders and processes each discovered game.
func (e *Engine) BatchProcess(ctx context.Context, parentDir string, opts ProcessOptions) ([]*ProcessResult, error) {
	absParent, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve batch directory: %w", err)
	}

	entries, err := os.ReadDir(absParent)
	if err != nil {
		return nil, fmt.Errorf("read batch directory %s: %w", absParent, err)
	}

	var results []*ProcessResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gamePath := filepath.Join(absParent, entry.Name())

		// Skip hidden folders
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		slog.Info("Batch scanning candidate game directory", "path", gamePath)
		subOpts := opts
		subOpts.Path = gamePath

		res, err := e.ProcessGame(ctx, subOpts)
		if err != nil {
			slog.Warn("Failed to batch process game directory", "path", gamePath, "error", err)
			continue
		}
		results = append(results, res)
	}

	return results, nil
}

// InspectStatus inspects a game directory to determine its patch and configuration state.
func (e *Engine) InspectStatus(ctx context.Context, targetPath string) (*GameStatus, error) {
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("detect game: %w", err)
	}

	status := &GameStatus{
		Name:     info.Name,
		AppID:    info.AppID,
		GameDir:  info.GameDir,
		Platform: info.Platform,
		ExePath:  info.ExePath,
		State:    "Original",
	}

	// Walk game directory to check for backup files or settings
	_ = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.Contains(name, ".ORIGINAL") {
			status.OriginalBackups = append(status.OriginalBackups, path)
		}
		if name == "steam_appid.txt" {
			status.SteamAppIDTxtFound = true
		}
		return nil
	})

	steamSettingsDir := filepath.Join(info.GameDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		status.SettingsDirExists = true
	}

	if len(status.OriginalBackups) > 0 {
		status.State = "Portable-Patched"
	} else if status.SettingsDirExists || status.SteamAppIDTxtFound {
		status.State = "Loader-Configured"
	}

	return status, nil
}

// Rollback restores original backup files and cleans up generated configuration files in a game directory.
func (e *Engine) Rollback(ctx context.Context, targetPath string, dryRun bool) error {
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("detect game for rollback: %w", err)
	}

	slog.Info("Starting rollback for game", "name", info.Name, "dir", info.GameDir)

	var restoredCount int
	err = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(d.Name(), ".ORIGINAL") {
			idx := strings.Index(path, ".20")
			origPath := path
			if idx != -1 {
				origPath = path[:idx]
			} else {
				origPath = strings.TrimSuffix(path, ".ORIGINAL")
			}

			if dryRun {
				slog.Info("[DRY RUN] Would restore original file", "from", path, "to", origPath)
			} else {
				if err := os.Rename(path, origPath); err != nil {
					slog.Error("Failed to restore original file", "from", path, "to", origPath, "error", err)
				} else {
					slog.Info("Restored original file", "restored", origPath)
					restoredCount++
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir during rollback: %w", err)
	}

	steamSettingsDir := filepath.Join(info.GameDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_settings directory", "path", steamSettingsDir)
		} else {
			if err := os.RemoveAll(steamSettingsDir); err == nil {
				slog.Info("Removed steam_settings directory", "path", steamSettingsDir)
			}
		}
	}

	appIDTxt := filepath.Join(info.GameDir, "steam_appid.txt")
	if util.FileExists(appIDTxt) {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_appid.txt", "path", appIDTxt)
		} else {
			_ = os.Remove(appIDTxt)
			slog.Info("Removed steam_appid.txt", "path", appIDTxt)
		}
	}

	slog.Info("Rollback completed", "restoredFiles", restoredCount)
	return nil
}
