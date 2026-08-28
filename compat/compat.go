package compat

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/util"
)

// Apply orchestrates atomic LaunchProfile persistence, on-disk setting synchronization,
// and DRM layer application (portable library replacement, loader symlinks/registry, or clean revert).
func Apply(ctx context.Context, cfg *config.Config, game *domain.Game, profile *domain.LaunchProfile, dryRun bool) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	gameDir := ""
	if profile != nil && profile.GameDir != "" {
		gameDir = profile.GameDir
	} else if game != nil && game.GameDir != "" {
		gameDir = game.GameDir
	}
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	if profile == nil {
		loaded, err := LoadProfile(gameDir)
		if err == nil && loaded != nil {
			profile = loaded
		} else {
			profile = &domain.LaunchProfile{GameDir: gameDir}
		}
	}
	profile.GameDir = gameDir

	// 1. Sync launch profile and basic text configs (steam_appid.txt, force_language.txt)
	if !dryRun {
		if err := SyncSettings(ctx, profile, cfg); err != nil {
			return fmt.Errorf("sync launch profile settings: %w", err)
		}
	} else {
		slog.Info("[DRY RUN] Would sync launch profile and emulator settings", "dir", gameDir)
	}

	// 2. Resolve target parameters
	platform := profile.Platform
	if platform == "" && game != nil {
		platform = game.Platform
	}
	if platform == "" {
		platform = "win64"
	}

	appID := profile.AppID
	if (appID == "" || appID == "0") && game != nil {
		appID = game.AppID
	}

	exePath := profile.ExePath
	if exePath == "" && game != nil {
		exePath = game.EffectiveExePath()
	}

	opMode := profile.OperatingMode
	if opMode == "" && game != nil {
		opMode = game.EffectiveOperatingMode()
	}
	if opMode == "" {
		if cfg.OperatingMode != "" {
			opMode = cfg.OperatingMode
		} else {
			opMode = domain.ModeStandalone
		}
	}

	// 3. Apply or revert DRM according to operating mode
	switch opMode {
	case domain.ModePortable:
		if err := gbe.Apply(ctx, gbe.Options{
			Config:     cfg,
			TargetDir:  gameDir,
			Platform:   platform,
			AppID:      appID,
			Language:   profile.Language,
			Portable:   true,
			WinePrefix: profile.WinePrefix,
			ExePath:    exePath,
			DryRun:     dryRun,
		}); err != nil {
			slog.Warn("Failed to apply portable DRM", "dir", gameDir, "error", err)
			return err
		}
	case domain.ModeLoader:
		if err := gbe.Apply(ctx, gbe.Options{
			Config:     cfg,
			TargetDir:  gameDir,
			Platform:   platform,
			AppID:      appID,
			Language:   profile.Language,
			Portable:   false,
			WinePrefix: profile.WinePrefix,
			ExePath:    exePath,
			DryRun:     dryRun,
		}); err != nil {
			slog.Warn("Failed to apply loader DRM", "dir", gameDir, "error", err)
			return err
		}
	case domain.ModeNone:
		if _, err := gbe.Revert(gameDir, dryRun); err != nil {
			slog.Warn("Failed to revert DRM for ModeNone", "dir", gameDir, "error", err)
			return err
		}
	}

	return nil
}

// Revert restores original backups and removes all Goldberg emulator artifacts.
func Revert(ctx context.Context, gameDir string, dryRun bool) (int, error) {
	return gbe.Revert(gameDir, dryRun)
}
