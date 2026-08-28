package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/util"
)

// LoadLaunchProfile retrieves the active launch profile from <gameDir>/steam_settings/launch_profile.json,
// falling back to inspecting on-disk emulator metadata and directory heuristics if absent.
func LoadLaunchProfile(gameDir string) (*LaunchProfile, error) {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	profilePath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(profilePath) {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("read launch profile: %w", err)
		}
		var p LaunchProfile
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("unmarshal launch profile: %w", err)
		}
		if p.GameDir == "" {
			p.GameDir = gameDir
		}
		hydrateFallbacks(&p, gameDir)
		return &p, nil
	}

	// Fallback to checking nested steam_settings directory
	var foundPath string
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "launch_profile.json" && strings.Contains(path, "steam_settings") {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})

	if foundPath != "" {
		data, err := os.ReadFile(foundPath)
		if err == nil {
			var p LaunchProfile
			if err := json.Unmarshal(data, &p); err == nil {
				if p.GameDir == "" {
					p.GameDir = gameDir
				}
				hydrateFallbacks(&p, gameDir)
				return &p, nil
			}
		}
	}

	// No launch_profile.json on disk: build initial baseline from on-disk artifacts
	p := &LaunchProfile{
		GameDir: gameDir,
	}
	hydrateFallbacks(p, gameDir)
	return p, nil
}

// GetLaunchProfile reads custom launch configurations from <gameDir>/steam_settings/launch_profile.json (alias for LoadLaunchProfile).
func GetLaunchProfile(gameDir string) (*LaunchProfile, error) {
	return LoadLaunchProfile(gameDir)
}

// hydrateFallbacks checks on-disk GBE/ACF metadata files if fields are missing.
func hydrateFallbacks(p *LaunchProfile, gameDir string) {
	settingsDir := filepath.Join(gameDir, "steam_settings")

	// Read steam_appid.txt
	if p.AppID == "" {
		appIDFile := filepath.Join(settingsDir, "steam_appid.txt")
		if !util.FileExists(appIDFile) {
			appIDFile = filepath.Join(gameDir, "steam_appid.txt")
		}
		if data, err := os.ReadFile(appIDFile); err == nil {
			p.AppID = strings.TrimSpace(string(data))
		}
	}

	// Read force_language.txt
	if p.Language == "" {
		langFile := filepath.Join(settingsDir, "force_language.txt")
		if data, err := os.ReadFile(langFile); err == nil {
			p.Language = strings.TrimSpace(string(data))
		}
	}

	// Read ColdClientLoader.ini Exe target if ExePath is still empty
	if p.ExePath == "" {
		iniPath := filepath.Join(gameDir, "ColdClientLoader.ini")
		if data, err := os.ReadFile(iniPath); err == nil {
			reIniExe := regexp.MustCompile(`(?i)^\s*Exe\s*=\s*(.+)`)
			for _, line := range strings.Split(string(data), "\n") {
				if m := reIniExe.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
					realExe := strings.TrimSpace(m[1])
					if !filepath.IsAbs(realExe) {
						realExe = filepath.Join(gameDir, realExe)
					}
					if util.FileExists(realExe) {
						p.ExePath = realExe
						break
					}
				}
			}
		}
	}

	if p.Store == "" {
		if p.AppID != "" && p.AppID != "0" {
			p.Store = domain.StoreSteam
		} else if p.GOGID != "" {
			p.Store = domain.StoreGOG
		}
	}
}

// SaveLaunchProfile writes custom launch configurations to <gameDir>/steam_settings/launch_profile.json atomically.
func SaveLaunchProfile(gameDir string, opts LaunchProfile) error {
	opts.GameDir = gameDir
	return SaveProfile(&opts)
}

// SaveProfile persists the profile to <gameDir>/steam_settings/launch_profile.json atomically.
func SaveProfile(p *LaunchProfile) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	p.GameDir = util.CanonicalPath(p.GameDir)
	if p.GameDir == "" {
		return fmt.Errorf("profile GameDir cannot be empty")
	}

	settingsDir := filepath.Join(p.GameDir, "steam_settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("create steam_settings directory: %w", err)
	}

	profilePath := filepath.Join(settingsDir, "launch_profile.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal launch profile: %w", err)
	}

	return os.WriteFile(profilePath, data, 0644)
}

// ResetLaunchProfile removes custom launch configurations from <gameDir>/steam_settings/launch_profile.json.
func ResetLaunchProfile(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	profilePath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(profilePath) {
		if err := os.Remove(profilePath); err != nil {
			return fmt.Errorf("remove launch profile: %w", err)
		}
	}

	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "launch_profile.json" && strings.Contains(path, "steam_settings") {
			_ = os.Remove(path)
		}
		return nil
	})

	return nil
}

// UnlinkMetadata removes custom metadata bindings and artwork locks from a game's launch profile.
func UnlinkMetadata(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	p, err := LoadLaunchProfile(gameDir)
	if err != nil || p == nil {
		return nil
	}

	p.MetadataProvider = ""
	p.MetadataID = ""
	p.StoreAppID = ""
	p.GOGID = ""
	p.SGDBID = ""
	p.Store = ""
	p.CustomCoverURL = ""
	p.CustomBannerURL = ""
	p.CustomHeroURL = ""
	p.CustomLogoURL = ""
	p.CustomIconURL = ""
	p.LockedArtwork = false

	return SaveProfile(p)
}

// SyncLaunchProfile updates on-disk Goldberg settings to match the LaunchProfile.
func SyncLaunchProfile(ctx context.Context, p *LaunchProfile, cfg *config.Config) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// 1. Persist the launch profile JSON
	if err := SaveProfile(p); err != nil {
		return fmt.Errorf("save profile: %w", err)
	}

	settingsDir := filepath.Join(p.GameDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)

	// 2. Sync AppID to steam_appid.txt if set
	if p.AppID != "" && p.AppID != "0" {
		_ = os.WriteFile(filepath.Join(settingsDir, "steam_appid.txt"), []byte(p.AppID), 0644)
		_ = os.WriteFile(filepath.Join(p.GameDir, "steam_appid.txt"), []byte(p.AppID), 0644)
	}

	// 3. Sync Language to force_language.txt
	langFile := filepath.Join(settingsDir, "force_language.txt")
	if p.Language != "" && p.Language != "auto" {
		_ = os.WriteFile(langFile, []byte(strings.ToLower(p.Language)+"\n"), 0644)
	} else {
		_ = os.Remove(langFile)
	}

	return nil
}

// ApplyCompatibility orchestrates atomic LaunchProfile persistence, on-disk setting synchronization (steam_appid.txt, force_language.txt),
// and DRM layer application (portable library replacement, loader symlinks/registry, or clean revert).
func ApplyCompatibility(ctx context.Context, cfg *config.Config, game *domain.Game, profile *LaunchProfile, dryRun bool) error {
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
		loaded, err := LoadLaunchProfile(gameDir)
		if err == nil && loaded != nil {
			profile = loaded
		} else {
			profile = &LaunchProfile{GameDir: gameDir}
		}
	}
	profile.GameDir = gameDir

	// 1. Sync launch profile and basic text configs (steam_appid.txt, force_language.txt)
	if !dryRun {
		if err := SyncLaunchProfile(ctx, profile, cfg); err != nil {
			return fmt.Errorf("sync launch profile: %w", err)
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

// RevertCompatibility restores original backups and removes all Goldberg emulator artifacts.
func RevertCompatibility(ctx context.Context, gameDir string, dryRun bool) (int, error) {
	return gbe.Revert(gameDir, dryRun)
}
