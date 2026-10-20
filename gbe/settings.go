package gbe

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/ini"
	"github.com/staernid/gamux/util"
)

// ConfigureCustomSettings populates Goldberg emulator profile configurations into steam_settings/.
func ConfigureCustomSettings(steamSettingsDir string, cfg *config.Config, supportedLanguages []string, dryRun bool) error {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if !dryRun {
		_ = os.MkdirAll(steamSettingsDir, 0755)
	}

	// 1. Language Resolution
	langFile := filepath.Join(steamSettingsDir, "force_language.txt")
	targetLang := strings.TrimSpace(cfg.Language)
	if targetLang != "" && targetLang != "auto" {
		// Validate against supported languages if available
		if len(supportedLanguages) > 0 {
			isSupported := false
			for _, sl := range supportedLanguages {
				if strings.EqualFold(sl, targetLang) {
					isSupported = true
					break
				}
			}
			if !isSupported {
				slog.Info("Preferred language not in game's supported list, falling back", "preferred", targetLang, "fallback", supportedLanguages[0])
				targetLang = supportedLanguages[0]
			}
		}

		if dryRun {
			slog.Info("[DRY RUN] Would write force_language.txt", "language", targetLang)
		} else {
			_ = os.WriteFile(langFile, []byte(strings.ToLower(targetLang)+"\n"), 0644)
			slog.Info("Configured emulator language", "language", targetLang, "path", langFile)
		}
	} else if !dryRun {
		_ = os.Remove(langFile)
	}

	// Clean up any legacy per-game force files if present (Goldberg reads global configs.user.ini)
	if !dryRun {
		_ = os.Remove(filepath.Join(steamSettingsDir, "force_account_name.txt"))
		_ = os.Remove(filepath.Join(steamSettingsDir, "force_steamid.txt"))
	}

	// 4. Disable Overlay
	overlayFile := filepath.Join(steamSettingsDir, "disable_overlay.txt")
	if cfg.DisableOverlay {
		if dryRun {
			slog.Info("[DRY RUN] Would write disable_overlay.txt")
		} else {
			_ = os.WriteFile(overlayFile, []byte("1\n"), 0644)
		}
	} else if !dryRun {
		_ = os.Remove(overlayFile)
	}

	// 5. Offline Mode
	offlineFile := filepath.Join(steamSettingsDir, "offline.txt")
	if cfg.OfflineMode {
		if dryRun {
			slog.Info("[DRY RUN] Would write offline.txt")
		} else {
			_ = os.WriteFile(offlineFile, []byte("1\n"), 0644)
		}
	} else if !dryRun {
		_ = os.Remove(offlineFile)
	}

	return nil
}

// SyncEmulatorSettings synchronizes emulator text configuration files (steam_appid.txt, force_language.txt,
// disable_overlay.txt, offline.txt) across all target steam_settings directories and root/exe directories.
func SyncEmulatorSettings(targetDir string, p *domain.LaunchProfile, cfg *config.Config, supportedLanguages []string, dryRun bool, extraPaths ...string) error {
	absTargetDir := util.CanonicalPath(targetDir)
	if absTargetDir == "" {
		return fmt.Errorf("target directory cannot be empty")
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	appID := ""
	targetLang := strings.TrimSpace(cfg.Language)
	disableOverlay := cfg.DisableOverlay
	offlineMode := cfg.OfflineMode
	var candidateExes []string

	if p != nil {
		if p.AppID != "" && p.AppID != "0" {
			appID = p.AppID
		}
		if p.Language != "" {
			targetLang = strings.TrimSpace(p.Language)
		}
		if p.DisableOverlay != nil {
			disableOverlay = *p.DisableOverlay
		}
		if p.OfflineMode != nil {
			offlineMode = *p.OfflineMode
		}
		if p.ExePath != "" {
			candidateExes = append(candidateExes, p.ExePath)
		}
	}
	for _, ep := range extraPaths {
		if ep != "" {
			candidateExes = append(candidateExes, ep)
		}
	}

	allSettingsDirs := ResolveTargetSettingsDirs(absTargetDir, nil, candidateExes...)

	var langBytes []byte
	writeLang := false
	if targetLang != "" && targetLang != "auto" {
		if len(supportedLanguages) > 0 {
			isSupported := false
			for _, sl := range supportedLanguages {
				if strings.EqualFold(sl, targetLang) {
					isSupported = true
					break
				}
			}
			if !isSupported {
				slog.Info("Preferred language not in game's supported list, falling back", "preferred", targetLang, "fallback", supportedLanguages[0])
				targetLang = supportedLanguages[0]
			}
		}
		langBytes = []byte(strings.ToLower(targetLang) + "\n")
		writeLang = true
	}

	for _, sDir := range allSettingsDirs {
		if dryRun {
			if appID != "" {
				slog.Info("[DRY RUN] Would write steam_appid.txt", "dir", sDir, "appID", appID)
			}
			if writeLang {
				slog.Info("[DRY RUN] Would write force_language.txt", "dir", sDir, "language", targetLang)
			}
			if disableOverlay {
				slog.Info("[DRY RUN] Would configure configs.overlay.ini", "dir", sDir, "overlayEnabled", false)
			} else {
				slog.Info("[DRY RUN] Would configure configs.overlay.ini", "dir", sDir, "overlayEnabled", true)
			}
			if offlineMode {
				slog.Info("[DRY RUN] Would write offline.txt", "dir", sDir)
			}
			continue
		}

		_ = os.MkdirAll(sDir, 0755)

		// 1. steam_appid.txt
		if appID != "" {
			_ = os.WriteFile(filepath.Join(sDir, "steam_appid.txt"), []byte(appID), 0644)
		}

		// 2. force_language.txt
		langFile := filepath.Join(sDir, "force_language.txt")
		if writeLang {
			_ = os.WriteFile(langFile, langBytes, 0644)
			slog.Info("Configured emulator language", "language", targetLang, "path", langFile)
		} else {
			_ = os.Remove(langFile)
		}

		// Clean up legacy per-game force files
		_ = os.Remove(filepath.Join(sDir, "force_account_name.txt"))
		_ = os.Remove(filepath.Join(sDir, "force_steamid.txt"))

		// 3. configs.overlay.ini (Goldberg experimental overlay)
		_ = SetOverlayEnabled(sDir, !disableOverlay, dryRun)
		_ = os.Remove(filepath.Join(sDir, "disable_overlay.txt"))

		// 4. offline.txt
		offlineFile := filepath.Join(sDir, "offline.txt")
		if offlineMode {
			_ = os.WriteFile(offlineFile, []byte("1\n"), 0644)
		} else {
			_ = os.Remove(offlineFile)
		}
	}

	// Ensure steam_appid.txt exists in root directory and alongside candidate executables / target files
	if appID != "" {
		appIDDirs := map[string]bool{
			absTargetDir: true,
		}
		for _, ep := range candidateExes {
			if ep == "" {
				continue
			}
			var full string
			if filepath.IsAbs(ep) {
				full = ep
			} else {
				full = filepath.Join(absTargetDir, ep)
			}
			dir := full
			if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
				dir = filepath.Dir(full)
			}
			appIDDirs[dir] = true
		}
		for dir := range appIDDirs {
			if dryRun {
				slog.Info("[DRY RUN] Would ensure steam_appid.txt", "dir", dir, "appID", appID)
			} else {
				_ = os.MkdirAll(dir, 0755)
				_ = os.WriteFile(filepath.Join(dir, "steam_appid.txt"), []byte(appID), 0644)
			}
		}
	}

	return nil
}

// SetOverlayEnabled writes or updates configs.overlay.ini in settingsDir with enable_experimental_overlay.
func SetOverlayEnabled(settingsDir string, enabled bool, dryRun bool) error {
	iniPath := filepath.Join(settingsDir, "configs.overlay.ini")
	val := "0"
	if enabled {
		val = "1"
	}

	if dryRun {
		slog.Info("[DRY RUN] Would update configs.overlay.ini", "dir", settingsDir, "enabled", enabled)
		return nil
	}

	f, err := ini.LoadFile(iniPath)
	if err != nil {
		f = ini.NewFile()
	}

	f.Section("overlay::general").Set("enable_experimental_overlay", val)

	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	if err := f.SaveFile(iniPath); err != nil {
		return fmt.Errorf("save configs.overlay.ini: %w", err)
	}
	return nil
}

// IsOverlayEnabled reads configs.overlay.ini in settingsDir and returns whether experimental overlay is enabled.
// If configs.overlay.ini is absent or does not specify enable_experimental_overlay, it falls back to checking disable_overlay.txt.
func IsOverlayEnabled(settingsDir string) (bool, bool) {
	iniPath := filepath.Join(settingsDir, "configs.overlay.ini")
	if f, err := ini.LoadFile(iniPath); err == nil {
		if sec, ok := f.GetSection("overlay::general"); ok {
			if val, ok := sec.GetBool("enable_experimental_overlay"); ok {
				return val, true
			}
		}
	}

	// Legacy fallback: disable_overlay.txt
	if util.FileExists(filepath.Join(settingsDir, "disable_overlay.txt")) {
		return false, true
	}

	return false, false
}

// FindSteamSettingsDirs scans gameDir for all existing directories named steam_settings.
func FindSteamSettingsDirs(gameDir string) []string {
	var dirs []string
	absDir := util.CanonicalPath(gameDir)
	if absDir == "" {
		return nil
	}
	_ = filepath.WalkDir(absDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if p != absDir && (name == "drive_c" || name == "dosdevices" || name == "pfx" || name == ".git" || name == ".gamux" || name == ".gamux_backup" || name == "node_modules") {
				return filepath.SkipDir
			}
			if name == "steam_settings" {
				dirs = append(dirs, p)
				return filepath.SkipDir
			}
		}
		return nil
	})
	return dirs
}

// ResolveTargetSettingsDirs computes all paths where steam_settings should be maintained.
func ResolveTargetSettingsDirs(absTargetDir string, targetFiles []string, candidateExes ...string) []string {
	seen := make(map[string]bool)
	var result []string

	addDir := func(d string) {
		clean := util.CanonicalPath(d)
		if clean != "" && !seen[clean] {
			seen[clean] = true
			result = append(result, clean)
		}
	}

	// 1. Root steam_settings
	addDir(filepath.Join(absTargetDir, "steam_settings"))

	// 2. Alongside each target library (steam_api.dll / libsteam_api.so)
	for _, f := range targetFiles {
		addDir(filepath.Join(filepath.Dir(f), "steam_settings"))
	}

	// 3. Alongside each candidate executable
	for _, exe := range candidateExes {
		if exe != "" {
			if filepath.IsAbs(exe) {
				addDir(filepath.Join(filepath.Dir(exe), "steam_settings"))
			} else {
				addDir(filepath.Join(absTargetDir, filepath.Dir(exe), "steam_settings"))
			}
		}
	}

	// 4. Any existing steam_settings found in the game directory tree
	for _, existing := range FindSteamSettingsDirs(absTargetDir) {
		addDir(existing)
	}

	return result
}
