package compat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/util"
)

// LoadProfile retrieves the active launch profile from <gameDir>/.gamux/launch_profile.json,
// falling back to legacy steam_settings, inspecting on-disk emulator metadata and directory heuristics if absent.
func LoadProfile(gameDir string) (*domain.LaunchProfile, error) {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	profilePath := filepath.Join(gameDir, ".gamux", "launch_profile.json")
	if util.FileExists(profilePath) {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("read launch profile: %w", err)
		}
		var p domain.LaunchProfile
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("unmarshal launch profile: %w", err)
		}
		if p.GameDir == "" {
			p.GameDir = gameDir
		}
		hydrateFallbacks(&p, gameDir)
		return &p, nil
	}

	// Legacy migration: check <gameDir>/steam_settings/launch_profile.json
	legacyPath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(legacyPath) {
		data, err := os.ReadFile(legacyPath)
		if err == nil {
			var p domain.LaunchProfile
			if err := json.Unmarshal(data, &p); err == nil {
				if p.GameDir == "" {
					p.GameDir = gameDir
				}
				hydrateFallbacks(&p, gameDir)
				// Migrate to .gamux and delete legacy file
				_ = SaveProfileDirect(&p)
				_ = os.Remove(legacyPath)
				return &p, nil
			}
		}
	}

	// Fallback to checking nested directories
	var foundPath string
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "launch_profile.json" && (strings.Contains(path, ".gamux") || strings.Contains(path, "steam_settings")) {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})

	if foundPath != "" {
		data, err := os.ReadFile(foundPath)
		if err == nil {
			var p domain.LaunchProfile
			if err := json.Unmarshal(data, &p); err == nil {
				if p.GameDir == "" {
					p.GameDir = gameDir
				}
				hydrateFallbacks(&p, gameDir)
				_ = SaveProfileDirect(&p)
				_ = os.Remove(foundPath)
				return &p, nil
			}
		}
	}

	// No launch_profile.json on disk: build initial baseline from on-disk artifacts
	p := &domain.LaunchProfile{
		GameDir: gameDir,
	}
	hydrateFallbacks(p, gameDir)
	return p, nil
}

// hydrateFallbacks checks on-disk GBE/ACF metadata files if fields are missing.
func hydrateFallbacks(p *domain.LaunchProfile, gameDir string) {
	// Discover all steam_settings dirs in the game tree
	settingsDirs := []string{filepath.Join(gameDir, "steam_settings")}
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			lower := strings.ToLower(d.Name())
			if path != gameDir && (lower == "drive_c" || lower == "dosdevices" || lower == "pfx" || lower == ".git" || lower == ".gamux" || lower == ".gamux_backup" || lower == "node_modules") {
				return filepath.SkipDir
			}
			if lower == "steam_settings" && path != settingsDirs[0] {
				settingsDirs = append(settingsDirs, path)
				return filepath.SkipDir
			}
		}
		return nil
	})

	// Read steam_appid.txt
	if p.AppID == "" {
		for _, sDir := range settingsDirs {
			appIDFile := filepath.Join(sDir, "steam_appid.txt")
			if data, err := os.ReadFile(appIDFile); err == nil {
				val := strings.TrimSpace(string(data))
				if val != "" && val != "0" {
					p.AppID = val
					break
				}
			}
		}
		if p.AppID == "" {
			rootAppIDFile := filepath.Join(gameDir, "steam_appid.txt")
			if data, err := os.ReadFile(rootAppIDFile); err == nil {
				val := strings.TrimSpace(string(data))
				if val != "" && val != "0" {
					p.AppID = val
				}
			}
		}
	}

	// Read force_language.txt
	if p.Language == "" {
		for _, sDir := range settingsDirs {
			langFile := filepath.Join(sDir, "force_language.txt")
			if data, err := os.ReadFile(langFile); err == nil {
				val := strings.TrimSpace(string(data))
				if val != "" {
					p.Language = val
					break
				}
			}
		}
	}

	// Read overlay configuration from configs.overlay.ini (or legacy disable_overlay.txt)
	if p.DisableOverlay == nil {
		for _, sDir := range settingsDirs {
			if enabled, ok := gbe.IsOverlayEnabled(sDir); ok {
				if !enabled {
					p.DisableOverlay = util.Ptr(true)
					break
				}
			}
		}
	}

	// Read offline.txt
	if p.OfflineMode == nil {
		for _, sDir := range settingsDirs {
			if util.FileExists(filepath.Join(sDir, "offline.txt")) {
				p.OfflineMode = util.Ptr(true)
				break
			}
		}
	}

	// Read ColdClientLoader.ini Exe target if ExePath is still empty
	if p.ExePath == "" {
		if realExe := util.ParseColdClientLoaderExe(gameDir); realExe != "" {
			p.ExePath = realExe
		}
	}

	if p.Store == "" && p.MetadataProvider == "" {
		if p.AppID != "" && p.AppID != "0" && p.StoreAppID != "" {
			p.Store = domain.StoreSteam
		} else if p.GOGID != "" {
			p.Store = domain.StoreGOG
		}
	}
}

// SaveProfile persists the profile to <gameDir>/.gamux/launch_profile.json atomically,
// safely preserving existing on-disk playtime statistics and custom metadata if omitted in p.
func SaveProfile(p *domain.LaunchProfile) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	p.GameDir = util.CanonicalPath(p.GameDir)
	if p.GameDir == "" {
		return fmt.Errorf("profile GameDir cannot be empty")
	}

	// Check existing profile in .gamux or legacy steam_settings
	existingPaths := []string{
		filepath.Join(p.GameDir, ".gamux", "launch_profile.json"),
		filepath.Join(p.GameDir, "steam_settings", "launch_profile.json"),
	}

	for _, profilePath := range existingPaths {
		if util.FileExists(profilePath) {
			if data, err := os.ReadFile(profilePath); err == nil {
				var existing domain.LaunchProfile
				if err := json.Unmarshal(data, &existing); err == nil {
					p.MergeMetadataAndStatsFrom(&existing)
					break
				}
			}
		}
	}

	return SaveProfileDirect(p)
}

// SaveProfileDirect writes the profile to <gameDir>/.gamux/launch_profile.json atomically without merging.
func SaveProfileDirect(p *domain.LaunchProfile) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	p.GameDir = util.CanonicalPath(p.GameDir)
	if p.GameDir == "" {
		return fmt.Errorf("profile GameDir cannot be empty")
	}

	gamuxDir := filepath.Join(p.GameDir, ".gamux")
	if err := os.MkdirAll(gamuxDir, 0755); err != nil {
		return fmt.Errorf("create .gamux directory: %w", err)
	}

	profilePath := filepath.Join(gamuxDir, "launch_profile.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal launch profile: %w", err)
	}

	if err := util.WriteFileAtomic(profilePath, data, 0644); err != nil {
		return err
	}

	// Invalidate warm library cache
	cache.InvalidateGameCache(p.GameDir)

	return nil
}

// IncrementPlaytime atomically adds delta seconds to a game's accumulated playtime and updates last played timestamp.
func IncrementPlaytime(gameDir string, deltaSec int64) (int64, int64, error) {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return 0, 0, fmt.Errorf("game directory cannot be empty")
	}

	prof, err := LoadProfile(gameDir)
	if err != nil || prof == nil {
		prof = &domain.LaunchProfile{GameDir: gameDir}
	}

	if deltaSec > 0 {
		prof.PlaytimeSeconds += deltaSec
	}
	prof.LastPlayed = osTimeNowUnix()

	if err := SaveProfile(prof); err != nil {
		return 0, 0, err
	}
	cache.InvalidateGameCache(gameDir)
	return prof.PlaytimeSeconds, prof.LastPlayed, nil
}

var osTimeNowUnix = func() int64 {
	return time.Now().Unix()
}

// ResetProfile removes custom launch configurations from <gameDir>/.gamux/launch_profile.json and cleans up launch.sh.
func ResetProfile(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	cache.InvalidateGameCache(gameDir)

	profilePath := filepath.Join(gameDir, ".gamux", "launch_profile.json")
	if util.FileExists(profilePath) {
		if err := os.Remove(profilePath); err != nil {
			return fmt.Errorf("remove launch profile: %w", err)
		}
	}

	// Remove legacy profile if present
	legacyPath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(legacyPath) {
		_ = os.Remove(legacyPath)
	}

	// Remove .gamux/launch.sh and legacy root launch.sh
	for _, scriptRel := range []string{filepath.Join(".gamux", "launch.sh"), "launch.sh"} {
		launchScriptPath := filepath.Join(gameDir, scriptRel)
		if util.FileExists(launchScriptPath) {
			_ = os.Remove(launchScriptPath)
		}
	}

	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "launch_profile.json" && (strings.Contains(path, ".gamux") || strings.Contains(path, "steam_settings")) {
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

	p, err := LoadProfile(gameDir)
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

	cache.InvalidateGameCache(gameDir)
	return SaveProfileDirect(p)
}

// SyncSettings updates on-disk Goldberg settings (steam_appid.txt, force_language.txt, etc.) across all steam_settings dirs.
func SyncSettings(ctx context.Context, p *domain.LaunchProfile, cfg *config.Config) error {
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

	// 2. Synchronize emulator settings across steam_settings directories and root
	return gbe.SyncEmulatorSettings(p.GameDir, p, cfg, nil, false)
}
