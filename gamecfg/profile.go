package gamecfg

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
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/util"
)

// Operating Mode Constants
const (
	ModeLutris     = string(domain.ModeLutris)
	ModeStandalone = string(domain.ModeStandalone)
)

// Runner Type Constants
const (
	RunnerAuto   = domain.RunnerAuto
	RunnerUMU    = domain.RunnerUMU
	RunnerWine   = domain.RunnerWine
	RunnerProton = domain.RunnerProton
	RunnerLinux  = domain.RunnerLinux
	RunnerLutris = domain.RunnerLutris
)

// Profile is the canonical launch configuration model from domain.
type Profile = domain.LaunchProfile

// Load retrieves the active game profile from <gameDir>/steam_settings/launch_profile.json,
// falling back to inspecting on-disk emulator metadata and directory heuristics if absent.
func Load(gameDir string) (*Profile, error) {
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
		var p Profile
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
			var p Profile
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
	p := &Profile{
		GameDir: gameDir,
	}
	hydrateFallbacks(p, gameDir)
	return p, nil
}

// hydrateFallbacks checks on-disk GBE/ACF metadata files if fields are missing.
func hydrateFallbacks(p *Profile, gameDir string) {
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
		} else {
			p.Store = domain.StoreStandalone
		}
	}
}

// Save persists the profile to <gameDir>/steam_settings/launch_profile.json atomically.
func Save(p *Profile) error {
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

// Reset removes custom launch configurations from <gameDir>/steam_settings/launch_profile.json.
func Reset(gameDir string) error {
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

// Sync updates on-disk Goldberg settings and Lutris registration to match the Profile.
func Sync(ctx context.Context, p *Profile, cfg *config.Config) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// 1. Persist the launch profile JSON
	if err := Save(p); err != nil {
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

	// 4. Sync ColdClientLoader.ini Exe line if loader mode is used
	coldClientIni := filepath.Join(p.GameDir, "ColdClientLoader.ini")
	if util.FileExists(coldClientIni) && p.ExePath != "" {
		relExe := p.ExePath
		if filepath.IsAbs(relExe) {
			if rel, err := filepath.Rel(p.GameDir, relExe); err == nil {
				relExe = rel
			}
		}
		if iniData, err := os.ReadFile(coldClientIni); err == nil {
			reExe := regexp.MustCompile(`(?i)(Exe\s*=\s*)(.+)`)
			var updated string
			if reExe.Match(iniData) {
				updated = reExe.ReplaceAllString(string(iniData), fmt.Sprintf("${1}%s", relExe))
			} else {
				updated = string(iniData) + fmt.Sprintf("\nExe = %s\n", relExe)
			}
			_ = os.WriteFile(coldClientIni, []byte(updated), 0644)
		}
	}

	// 5. Sync Lutris Integration if OperatingMode is Lutris or Lutris YAML exists
	effective := ResolveEffective(cfg, p)
	if effective.OperatingMode == ModeLutris && p.ExePath != "" {
		lutrisDir := util.ExpandPath(cfg.LutrisDir)
		slug := lutris.Slugify(p.Name)
		if slug == "" {
			slug = lutris.Slugify(filepath.Base(p.GameDir))
		}

		lutrisRunner := "wine"
		if p.Platform == "linux" || effective.Runner == RunnerLinux {
			lutrisRunner = "linux"
		}

		prefix := p.WinePrefix
		if prefix == "" {
			prefix = cfg.WinePrefix
		}
		if prefix != "" {
			prefix = util.ExpandPath(prefix)
		}

		targetExe := p.ExePath
		envMap := make(map[string]string)
		for k, v := range p.CustomEnv {
			envMap[k] = v
		}

		if p.Platform != "linux" {
			loaderX64 := filepath.Join(p.GameDir, "steamclient_loader_x64.exe")
			loaderX86 := filepath.Join(p.GameDir, "steamclient_loader_x86.exe")
			coldIni := filepath.Join(p.GameDir, "ColdClientLoader.ini")

			if util.FileExists(coldIni) {
				if p.Platform == "win32" && util.FileExists(loaderX86) {
					targetExe = loaderX86
					envMap["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
				} else if util.FileExists(loaderX64) {
					targetExe = loaderX64
					envMap["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
				}
			}
		}

		lutrisCfg := lutris.Config{
			Name:       p.Name,
			Slug:       slug,
			GamePath:   targetExe,
			PrefixPath: prefix,
			Runner:     lutrisRunner,
			Args:       p.ExeArgs,
			Env:        envMap,
		}

		if p.MangoHud != nil && *p.MangoHud {
			if lutrisCfg.System == nil {
				lutrisCfg.System = &lutris.SystemConfig{}
			}
			lutrisCfg.System.MangoHud = true
		}

		if err := lutris.Write(lutrisCfg, lutrisDir); err != nil {
			slog.Warn("Failed to sync Lutris config YAML", "error", err)
		}
	}

	return nil
}

// ResolveEffective merges global user configuration with game-specific profile settings.
func ResolveEffective(cfg *config.Config, p *Profile) Profile {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if p == nil {
		p = &Profile{}
	}
	res := *p

	if res.OperatingMode == "" {
		if cfg.OperatingMode != "" {
			res.OperatingMode = cfg.OperatingMode
		} else {
			res.OperatingMode = ModeStandalone
		}
	}

	if res.Runner == "" || res.Runner == RunnerAuto {
		if res.Platform == "linux" {
			res.Runner = RunnerLinux
		} else if res.OperatingMode == ModeLutris {
			res.Runner = RunnerLutris
		} else if cfg.DefaultRunner != "" {
			res.Runner = cfg.DefaultRunner
		} else {
			res.Runner = RunnerUMU
		}
	}

	if res.WinePrefix == "" && res.Runner != RunnerLinux {
		if cfg.WinePrefix != "" {
			res.WinePrefix = cfg.WinePrefix
		}
	}

	if res.MangoHud == nil && cfg.MangoHud {
		res.MangoHud = util.Ptr(cfg.MangoHud)
	}
	if res.WineFullscreenFSR == nil && cfg.WineFullscreenFSR {
		res.WineFullscreenFSR = util.Ptr(cfg.WineFullscreenFSR)
	}
	if res.ProtonNVAPI == nil && cfg.ProtonNVAPI {
		res.ProtonNVAPI = util.Ptr(cfg.ProtonNVAPI)
	}
	if res.DXVKHud == "" && cfg.DXVKHud != "" {
		res.DXVKHud = cfg.DXVKHud
	}

	if res.WorkingDir == "" && res.GameDir != "" {
		res.WorkingDir = res.GameDir
	}
	if res.ProtonPath == "" && cfg.ProtonPath != "" {
		res.ProtonPath = cfg.ProtonPath
	}

	// Merge Environment Variables (Global < Custom Profile)
	mergedEnv := make(map[string]string)
	if cfg.GlobalEnv != nil {
		for k, v := range cfg.GlobalEnv {
			mergedEnv[k] = v
		}
	}
	if res.CustomEnv != nil {
		for k, v := range res.CustomEnv {
			mergedEnv[k] = v
		}
	}
	res.CustomEnv = mergedEnv

	return res
}
