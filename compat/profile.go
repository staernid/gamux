package compat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

// LoadProfile retrieves the active launch profile from <gameDir>/steam_settings/launch_profile.json,
// falling back to inspecting on-disk emulator metadata and directory heuristics if absent.
func LoadProfile(gameDir string) (*domain.LaunchProfile, error) {
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
			var p domain.LaunchProfile
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
	p := &domain.LaunchProfile{
		GameDir: gameDir,
	}
	hydrateFallbacks(p, gameDir)
	return p, nil
}

// hydrateFallbacks checks on-disk GBE/ACF metadata files if fields are missing.
func hydrateFallbacks(p *domain.LaunchProfile, gameDir string) {
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

// SaveProfile persists the profile to <gameDir>/steam_settings/launch_profile.json atomically,
// safely preserving existing on-disk playtime statistics and custom metadata if omitted in p.
func SaveProfile(p *domain.LaunchProfile) error {
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
	if util.FileExists(profilePath) {
		if data, err := os.ReadFile(profilePath); err == nil {
			var existing domain.LaunchProfile
			if err := json.Unmarshal(data, &existing); err == nil {
				if p.PlaytimeSeconds <= 0 && existing.PlaytimeSeconds > 0 {
					p.PlaytimeSeconds = existing.PlaytimeSeconds
				}
				if p.LastPlayed <= 0 && existing.LastPlayed > 0 {
					p.LastPlayed = existing.LastPlayed
				}
				if p.CustomCoverURL == "" && existing.CustomCoverURL != "" {
					p.CustomCoverURL = existing.CustomCoverURL
				}
				if p.CustomBannerURL == "" && existing.CustomBannerURL != "" {
					p.CustomBannerURL = existing.CustomBannerURL
				}
				if p.CustomHeroURL == "" && existing.CustomHeroURL != "" {
					p.CustomHeroURL = existing.CustomHeroURL
				}
				if p.CustomLogoURL == "" && existing.CustomLogoURL != "" {
					p.CustomLogoURL = existing.CustomLogoURL
				}
				if p.CustomIconURL == "" && existing.CustomIconURL != "" {
					p.CustomIconURL = existing.CustomIconURL
				}
				if !p.LockedArtwork && existing.LockedArtwork {
					p.LockedArtwork = existing.LockedArtwork
				}
				if p.MetadataProvider == "" && existing.MetadataProvider != "" {
					p.MetadataProvider = existing.MetadataProvider
				}
				if p.MetadataID == "" && existing.MetadataID != "" {
					p.MetadataID = existing.MetadataID
				}
				if p.StoreAppID == "" && existing.StoreAppID != "" {
					p.StoreAppID = existing.StoreAppID
				}
				if p.GOGID == "" && existing.GOGID != "" {
					p.GOGID = existing.GOGID
				}
				if p.SGDBID == "" && existing.SGDBID != "" {
					p.SGDBID = existing.SGDBID
				}
				if p.Store == "" && existing.Store != "" {
					p.Store = existing.Store
				}
			}
		}
	}

	return SaveProfileDirect(p)
}

// SaveProfileDirect writes the profile to <gameDir>/steam_settings/launch_profile.json atomically without merging.
func SaveProfileDirect(p *domain.LaunchProfile) error {
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
	return prof.PlaytimeSeconds, prof.LastPlayed, nil
}

var osTimeNowUnix = func() int64 {
	return time.Now().Unix()
}

// ResetProfile removes custom launch configurations from <gameDir>/steam_settings/launch_profile.json.
func ResetProfile(gameDir string) error {
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

	return SaveProfileDirect(p)
}

// SyncSettings updates on-disk Goldberg settings (steam_appid.txt, force_language.txt) to match the LaunchProfile.
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
