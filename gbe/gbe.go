package gbe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/ini"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// PlatformConfig maps platform identifiers to their Goldberg Steam Emulator library targets.
var PlatformConfig = map[string]struct {
	Subdir, Target, Additional, Generator, Arch string
}{
	"linux": {
		Subdir:     "linux_release",
		Target:     "libsteam_api.so",
		Additional: "steamclient.so",
		Generator:  "generate_interfaces_x64",
		Arch:       "64",
	},
	"linux32": {
		Subdir:     "linux_release",
		Target:     "libsteam_api.so",
		Additional: "steamclient.so",
		Generator:  "generate_interfaces_x32",
		Arch:       "86",
	},
	"win64": {
		Subdir:     "win_release",
		Target:     "steam_api64.dll",
		Additional: "steamclient64.dll",
		Generator:  "generate_interfaces_x64.exe",
		Arch:       "64",
	},
	"win32": {
		Subdir:     "win_release",
		Target:     "steam_api.dll",
		Additional: "steamclient.dll",
		Generator:  "generate_interfaces_x32.exe",
		Arch:       "86",
	},
}

// Options specifies configuration and target paths for applying Goldberg Steam Emulator.
type Options struct {
	Config        *config.Config
	TargetDir     string
	Platform      string
	AppID         string
	Language      string
	DryRun        bool
	Portable      bool
	WinePrefix    string
	ExePath       string
	CandidateExes []string
}

// EnsureAvailable ensures that Goldberg Emulator release assets are present locally, auto-downloading them if missing.
func EnsureAvailable(ctx context.Context, cfg *config.Config) (string, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	gbeDir := util.CanonicalPath(cfg.GbeDir)
	if _, err := os.Stat(filepath.Join(gbeDir, "linux_release")); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(gbeDir, "win_release")); os.IsNotExist(err) {
			slog.Info("Goldberg Emulator assets not found locally, bootstrapping from GitHub releases...", "dest", gbeDir)
			if err := github.UpdateGBE(ctx, cfg); err != nil {
				return "", fmt.Errorf("auto-bootstrap Goldberg emulator: %w", err)
			}
		}
	}
	return gbeDir, nil
}

// ApplyGBE applies the Goldberg Steam Emulator to the target game directory (alias for Apply).
func ApplyGBE(ctx context.Context, opts Options) error {
	return Apply(ctx, opts)
}

// Apply applies the Goldberg Steam Emulator to the target game directory.
// If opts.Portable is true, it performs direct DLL/SO replacement (copying emulated libraries into the game dir).
// If opts.Portable is false (loader mode), it leaves original game libraries untouched and sets up Wine registry/loader config.
func Apply(ctx context.Context, opts Options) error {
	cfg := opts.Config
	if cfg == nil {
		cfg = config.DefaultConfig()
	} else {
		// Clone to avoid mutating shared config
		cfgCopy := *cfg
		cfg = &cfgCopy
	}
	if opts.Language != "" {
		cfg.Language = opts.Language
	}

	targetDir := opts.TargetDir
	if targetDir == "" {
		targetDir = "."
	}

	platform := opts.Platform
	appID := opts.AppID
	dryRun := opts.DryRun
	portable := opts.Portable
	winePrefix := opts.WinePrefix
	exePath := opts.ExePath

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolve target dir: %w", err)
	}

	platformCfg, ok := PlatformConfig[platform]
	if !ok {
		var validPlatforms []string
		for p := range PlatformConfig {
			validPlatforms = append(validPlatforms, p)
		}
		return fmt.Errorf("invalid platform: '%s'. Valid platforms: %s", platform, strings.Join(validPlatforms, ", "))
	}

	gbeBase := util.CanonicalPath(cfg.GbeDir)
	if gbeBase == "" {
		gbeBase = util.CanonicalPath(config.DefaultConfig().GbeDir)
	}

	gbePath := filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x"+platformCfg.Arch)
	if !util.FileExists(gbePath) && platformCfg.Arch == "86" {
		fallbackPath := filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x32")
		if util.FileExists(fallbackPath) {
			gbePath = fallbackPath
		}
	}

	var targetFiles []string
	walkErr := filepath.WalkDir(absTargetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			lower := strings.ToLower(d.Name())
			if path != absTargetDir && (lower == "drive_c" || lower == "dosdevices" || lower == "pfx" || lower == ".git" || lower == ".gamux" || lower == ".gamux_backup" || lower == "node_modules" || lower == "gbe_fork" || lower == "win_release" || lower == "linux_release" || lower == "steamclient_experimental") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == platformCfg.Target {
			targetFiles = append(targetFiles, path)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to search for files in %s: %w", absTargetDir, walkErr)
	}

	if len(targetFiles) == 0 {
		slog.Warn("No target files found", "platform", platform, "dir", absTargetDir)
	}

	// For Linux, automatically detect if the target library is 32-bit ELF (ELFCLASS32: header[4] == 1)
	if (platform == "linux" || platform == "linux32") && len(targetFiles) > 0 {
		if f, err := os.Open(targetFiles[0]); err == nil {
			header := make([]byte, 5)
			if n, rErr := f.Read(header); rErr == nil && n >= 5 && string(header[:4]) == "\x7fELF" {
				if header[4] == 1 {
					slog.Info("Detected 32-bit (x86) Linux ELF library, selecting x86 Goldberg emulator", "file", targetFiles[0])
					platformCfg = PlatformConfig["linux32"]
				} else if header[4] == 2 {
					platformCfg = PlatformConfig["linux"]
				}
				gbePath = filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x"+platformCfg.Arch)
				if !util.FileExists(gbePath) && platformCfg.Arch == "86" {
					if fallback := filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x32"); util.FileExists(fallback) {
						gbePath = fallback
					}
				}
			}
			f.Close()
		}
	}

	sourceFile := filepath.Join(gbePath, platformCfg.Target)
	if !util.FileExists(sourceFile) {
		if _, err := EnsureAvailable(ctx, cfg); err == nil {
			gbeBase = util.CanonicalPath(cfg.GbeDir)
			gbePath = filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x"+platformCfg.Arch)
			if !util.FileExists(gbePath) && platformCfg.Arch == "86" {
				if fallback := filepath.Join(gbeBase, platformCfg.Subdir, "experimental", "x32"); util.FileExists(fallback) {
					gbePath = fallback
				}
			}
			sourceFile = filepath.Join(gbePath, platformCfg.Target)
		}
	}

	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: '%s'", sourceFile)
	}

	sourceHash, err := util.GetHash(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to get hash of source file: %w", err)
	}

	candidateExes := opts.CandidateExes
	if len(candidateExes) == 0 && exePath != "" {
		candidateExes = []string{exePath}
	}
	allSettingsDirs := ResolveTargetSettingsDirs(absTargetDir, targetFiles, candidateExes...)

	if portable {
		CleanLoaderArtifacts(absTargetDir, dryRun)
	}

	for _, file := range targetFiles {
		slog.Info("Found target library", "file", file)

		if portable {
			targetHash, err := util.GetHash(file)
			if err != nil {
				slog.Error("Failed to get hash", "file", file, "error", err)
				continue
			}

			if targetHash == sourceHash {
				slog.Info("File is already up-to-date", "file", file)
			} else {
				if dryRun {
					slog.Info("[DRY RUN] Would replace file with Goldberg emulator", "file", file)
				} else if err := util.BackupAndReplace(sourceFile, file); err != nil {
					slog.Error("Failed to replace file", "file", file, "error", err)
					continue
				}
			}

			if platformCfg.Additional != "" {
				additionalSource := filepath.Join(gbePath, platformCfg.Additional)
				additionalDest := filepath.Join(filepath.Dir(file), platformCfg.Additional)
				if _, err := os.Stat(additionalSource); err == nil {
					if _, err := os.Stat(additionalDest); err == nil {
						if dryRun {
							slog.Info("[DRY RUN] Would replace additional file", "dest", additionalDest)
						} else if err := util.BackupAndReplace(additionalSource, additionalDest); err != nil {
							slog.Warn("Failed to replace additional file", "dest", additionalDest, "error", err)
						}
					}
				}
			}
		} else {
			slog.Info("[LOADER MODE] Keeping original library intact (zero file replacement)", "file", file)
			// Restore any existing .ORIGINAL backup to ensure pristine original binary
			RestoreOriginalBackup(file, dryRun)

			if platform == "win64" || platform == "win32" {
				targetExe := exePath
				if targetExe == "" {
					targetExe = file
				}
				if err := SetupLoaderMode(gbeBase, absTargetDir, platform, appID, targetExe, targetFiles, dryRun, candidateExes...); err != nil {
					slog.Warn("Failed to setup Goldberg loader mode", "error", err)
				}
				if winePrefix != "" {
					settingsDir := filepath.Join(absTargetDir, "steam_settings")
					_ = SetupWinePrefixSteamDir(winePrefix, gbeBase, settingsDir, appID, dryRun)
				}
			}
		}

		// Determine the source original binary for interface detection
		sourceDll := file
		dir := filepath.Dir(file)
		base := filepath.Base(file)
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasPrefix(e.Name(), base) && strings.Contains(e.Name(), ".ORIGINAL") {
					sourceDll = filepath.Join(dir, e.Name())
					break
				}
			}
		}

		if dryRun {
			slog.Info("[DRY RUN] Would extract steam_interfaces.txt into steam_settings", "source", sourceDll)
		} else {
			targetDirs := []string{filepath.Dir(file)}
			targetDirs = append(targetDirs, allSettingsDirs...)
			if err := GenerateAndSaveInterfaces(sourceDll, targetDirs...); err != nil {
				slog.Warn("Failed to extract Steam interfaces from binary", "source", sourceDll, "error", err)
			}
		}
	}

	// After applying GBE, fetch and configure DLCs
	for _, file := range targetFiles {
		libraryPath := filepath.Dir(file)
		if err := steam.FetchDLCs(ctx, appID, libraryPath, dryRun); err != nil {
			slog.Warn("Failed to fetch and configure DLCs", "appID", appID, "libraryPath", libraryPath, "error", err)
		}
	}
	if appID != "" && appID != "0" {
		if err := steam.FetchDLCs(ctx, appID, absTargetDir, dryRun); err != nil {
			slog.Debug("Failed to fetch DLCs for root", "appID", appID, "error", err)
		}
	}

	// Synchronize configs.app.ini across all steam_settings directories if present
	var iniData []byte
	for _, sDir := range allSettingsDirs {
		iniPath := filepath.Join(sDir, "configs.app.ini")
		if data, err := os.ReadFile(iniPath); err == nil && len(data) > 0 {
			iniData = data
			break
		}
	}
	if len(iniData) > 0 {
		for _, sDir := range allSettingsDirs {
			iniPath := filepath.Join(sDir, "configs.app.ini")
			if !util.FileExists(iniPath) {
				if dryRun {
					slog.Info("[DRY RUN] Would sync configs.app.ini", "path", iniPath)
				} else {
					_ = os.MkdirAll(sDir, 0755)
					_ = os.WriteFile(iniPath, iniData, 0644)
					slog.Info("Synchronized configs.app.ini", "path", iniPath)
				}
			}
		}
	}

	// Synchronize emulator settings across all discovered steam_settings dirs, libraries, and executables
	supportedLangs, _ := steam.FetchSupportedLanguages(ctx, appID)
	prof := &domain.LaunchProfile{
		GameDir:  absTargetDir,
		AppID:    appID,
		Language: opts.Language,
		ExePath:  exePath,
	}
	var extraPaths []string
	extraPaths = append(extraPaths, candidateExes...)
	extraPaths = append(extraPaths, targetFiles...)
	if err := SyncEmulatorSettings(absTargetDir, prof, cfg, supportedLangs, dryRun, extraPaths...); err != nil {
		slog.Warn("Failed to synchronize emulator settings", "error", err)
	}

	slog.Info("GBE application process completed")
	return nil
}

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

// SetupWinePrefixSteamDir populates <prefixDir>/drive_c/Program Files (x86)/Steam/ with Goldberg
// Steam client emulation libraries, overlay renderer, and steam_settings symlinks.
// This allows steam_api to discover the emulator out-of-the-box via Proton's default ActiveProcess configuration.
func SetupWinePrefixSteamDir(prefixDir, gbeBase, gameSettingsDir, appID string, dryRun bool) error {
	if prefixDir == "" {
		return nil
	}
	if gbeBase == "" {
		gbeBase = util.CanonicalPath(config.DefaultConfig().GbeDir)
	}

	steamDir := filepath.Join(prefixDir, "drive_c", "Program Files (x86)", "Steam")
	if dryRun {
		slog.Info("[DRY RUN] Would setup Wine prefix Steam directory", "steamDir", steamDir)
		return nil
	}

	if err := os.MkdirAll(steamDir, 0755); err != nil {
		return fmt.Errorf("create prefix steam dir: %w", err)
	}

	loaderDir := filepath.Join(gbeBase, "win_release", "steamclient_experimental")
	loaderFiles := []string{
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
	}

	for _, fname := range loaderFiles {
		src := filepath.Join(loaderDir, fname)
		dst := filepath.Join(steamDir, fname)
		if !util.FileExists(src) {
			continue
		}
		_ = os.Remove(dst)
		if err := os.Symlink(src, dst); err != nil {
			if copyErr := util.CopyFile(src, dst); copyErr != nil {
				slog.Warn("Failed to copy loader DLL to prefix steam dir", "file", fname, "error", copyErr)
			}
		}
	}

	// Symlink steam_settings
	if gameSettingsDir != "" && util.DirExists(gameSettingsDir) {
		prefixSettings := filepath.Join(steamDir, "steam_settings")
		_ = os.Remove(prefixSettings)
		if rel, err := filepath.Rel(steamDir, gameSettingsDir); err == nil {
			_ = os.Symlink(rel, prefixSettings)
		} else {
			_ = os.Symlink(gameSettingsDir, prefixSettings)
		}
	}

	// Write steam_appid.txt
	if appID != "" && appID != "0" {
		_ = os.WriteFile(filepath.Join(steamDir, "steam_appid.txt"), []byte(appID+"\n"), 0644)
	}

	return nil
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

// CleanLoaderArtifacts removes Goldberg loader emulation libraries and loader executables from absTargetDir.
func CleanLoaderArtifacts(absTargetDir string, dryRun bool) {
	loaderFiles := map[string]bool{
		"steamclient64.dll":          true,
		"steamclient.dll":            true,
		"GameOverlayRenderer64.dll":  true,
		"GameOverlayRenderer.dll":    true,
		"ColdClientLoader.ini":       true,
		"steamclient_loader_x64.exe": true,
		"steamclient_loader_x86.exe": true,
	}

	_ = filepath.WalkDir(absTargetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			lower := strings.ToLower(d.Name())
			if path != absTargetDir && (lower == "drive_c" || lower == "dosdevices" || lower == "pfx" || lower == ".git" || lower == ".gamux" || lower == ".gamux_backup" || lower == "node_modules" || lower == "gbe_fork" || lower == "win_release" || lower == "linux_release" || lower == "steamclient_experimental") {
				return filepath.SkipDir
			}
			return nil
		}
		if loaderFiles[d.Name()] {
			if dryRun {
				slog.Info("[DRY RUN] Would remove loader artifact", "path", path)
			} else {
				_ = os.Remove(path)
				slog.Info("Removed loader artifact", "path", path)
			}
		}
		return nil
	})
}

// SetupLoaderMode configures Goldberg's Steam client emulation libraries (steamclient64/steamclient.dll and overlay renderer)
// by creating relative symlinks to central GBE binaries in ~/.local/share/gbe_fork/ across the root, exe dir, and library dirs.
func SetupLoaderMode(gbeBase, absTargetDir, platform, appID, exePath string, targetFiles []string, dryRun bool, candidateExes ...string) error {
	if platform != "win64" && platform != "win32" {
		return nil
	}

	loaderDir := filepath.Join(gbeBase, "win_release", "steamclient_experimental")

	filesToSymlink := []string{
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
	}

	targetDirs := map[string]bool{
		absTargetDir: true,
	}
	if exePath != "" {
		if filepath.IsAbs(exePath) {
			targetDirs[filepath.Dir(exePath)] = true
		} else {
			targetDirs[filepath.Join(absTargetDir, filepath.Dir(exePath))] = true
		}
	}
	for _, f := range targetFiles {
		targetDirs[filepath.Dir(f)] = true
	}
	for _, exe := range candidateExes {
		if exe != "" {
			if filepath.IsAbs(exe) {
				targetDirs[filepath.Dir(exe)] = true
			} else {
				targetDirs[filepath.Join(absTargetDir, filepath.Dir(exe))] = true
			}
		}
	}

	for dstDir := range targetDirs {
		for _, fname := range filesToSymlink {
			src := filepath.Join(loaderDir, fname)
			dst := filepath.Join(dstDir, fname)
			if _, err := os.Stat(src); os.IsNotExist(err) {
				slog.Warn("GBE loader asset not found", "src", src)
				continue
			}

			if dryRun {
				slog.Info("[DRY RUN] Would symlink loader asset", "from", src, "to", dst)
				continue
			}

			_ = os.Remove(dst)
			if err := os.Symlink(src, dst); err != nil {
				slog.Warn("Failed to symlink loader asset, falling back to copy", "from", src, "to", dst, "error", err)
				if copyErr := util.CopyFile(src, dst); copyErr != nil {
					slog.Error("Failed to copy loader asset fallback", "from", src, "to", dst, "error", copyErr)
				}
			} else {
				slog.Info("Symlinked GBE loader asset", "file", fname, "dst", dst)
			}
		}

		if appID != "" && appID != "0" {
			appIDFile := filepath.Join(dstDir, "steam_appid.txt")
			if !dryRun {
				_ = os.WriteFile(appIDFile, []byte(appID+"\n"), 0644)
			}
		}
	}

	return nil
}

// GenerateAchievementsJSON fetches achievement schema via steam.FetchAchievementSchema, writes steam_settings/achievements.json and downloads icons to steam_settings/img.
func GenerateAchievementsJSON(ctx context.Context, appID uint32, apiKey string, targetDir string, dryRun bool) error {
	slog.Info("Fetching and generating achievements.json for GBE", "appID", appID, "targetDir", targetDir)

	steamSettingsDir := filepath.Join(targetDir, "steam_settings")
	if !dryRun {
		if err := os.MkdirAll(steamSettingsDir, 0755); err != nil {
			return fmt.Errorf("failed to create steam_settings directory: %w", err)
		}
	}

	achievements, err := steam.FetchAchievementSchema(ctx, appID, apiKey, "english")
	if err != nil {
		slog.Warn("Failed to fetch achievement schema", "appID", appID, "error", err)
		return err
	}

	if len(achievements) == 0 {
		slog.Info("No achievements found for AppID", "appID", appID)
		return nil
	}

	achievementsJSONPath := filepath.Join(steamSettingsDir, "achievements.json")
	var data []byte
	if dryRun {
		slog.Info("[DRY RUN] Would write achievements.json", "path", achievementsJSONPath, "count", len(achievements))
	} else {
		var err error
		data, err = json.MarshalIndent(achievements, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal achievements.json: %w", err)
		}
		if err := os.WriteFile(achievementsJSONPath, data, 0644); err != nil {
			return fmt.Errorf("write achievements.json: %w", err)
		}
		slog.Info("Wrote achievements.json", "path", achievementsJSONPath, "count", len(achievements))
	}

	// Collect icon names for downloading
	imgDir := filepath.Join(steamSettingsDir, "img")
	iconNames := make([]string, 0, len(achievements)*2)
	for _, ach := range achievements {
		if ach.Icon != "" && strings.HasPrefix(ach.Icon, "img/") {
			iconNames = append(iconNames, strings.TrimPrefix(ach.Icon, "img/"))
		}
		if ach.IconGray != "" && strings.HasPrefix(ach.IconGray, "img/") {
			iconNames = append(iconNames, strings.TrimPrefix(ach.IconGray, "img/"))
		}
	}

	if len(iconNames) > 0 && !dryRun {
		if err := steam.DownloadAchievementImages(ctx, int(appID), iconNames, imgDir); err != nil {
			slog.Warn("Failed downloading some achievement icons", "appID", appID, "error", err)
		}
	}

	// Synchronize achievements.json and img/ across all discovered steam_settings dirs
	allDirs := FindSteamSettingsDirs(targetDir)
	for _, sDir := range allDirs {
		if sDir == steamSettingsDir {
			continue
		}
		if dryRun {
			slog.Info("[DRY RUN] Would sync achievements to directory", "dir", sDir)
			continue
		}
		_ = os.MkdirAll(sDir, 0755)
		if len(data) > 0 {
			_ = os.WriteFile(filepath.Join(sDir, "achievements.json"), data, 0644)
		}
		targetImg := filepath.Join(sDir, "img")
		if !util.FileExists(targetImg) {
			relPath, relErr := filepath.Rel(sDir, imgDir)
			if relErr == nil {
				_ = os.Symlink(relPath, targetImg)
			}
		}
	}

	return nil
}

// RestoreOriginalBackup checks for any .ORIGINAL backup file corresponding to targetFile and restores it.
func RestoreOriginalBackup(targetFile string, dryRun bool) bool {
	dir := filepath.Dir(targetFile)
	base := filepath.Base(targetFile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, base) && strings.Contains(name, ".ORIGINAL") {
			srcPath := filepath.Join(dir, name)
			if dryRun {
				slog.Info("[DRY RUN] Would restore original file for loader mode", "from", srcPath, "to", targetFile)
				return true
			}
			_ = os.Remove(targetFile)
			if err := os.Rename(srcPath, targetFile); err == nil {
				slog.Info("Restored original file for loader mode", "from", srcPath, "to", targetFile)
				return true
			}
		}
	}
	return false
}

// Rollback restores original backup files and cleans up Goldberg emulator artifacts (alias for Revert).
func Rollback(targetDir string, dryRun bool) (int, error) {
	return Revert(targetDir, dryRun)
}

// Revert restores all original backup files and cleans up Goldberg emulator artifacts in targetDir.
func Revert(targetDir string, dryRun bool) (int, error) {
	absTargetDir := util.CanonicalPath(targetDir)
	if absTargetDir == "" {
		return 0, fmt.Errorf("target directory cannot be empty")
	}

	if _, err := os.Stat(absTargetDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("target directory not found: %s", absTargetDir)
	}

	slog.Info("Starting Goldberg emulator revert", "dir", absTargetDir, "dryRun", dryRun)

	loaderFileNames := map[string]bool{
		"ColdClientLoader.ini":       true,
		"steamclient_loader_x64.exe": true,
		"steamclient_loader_x86.exe": true,
		"steamclient64.dll":          true,
		"steamclient.dll":            true,
		"GameOverlayRenderer64.dll":  true,
		"GameOverlayRenderer.dll":    true,
		"steam_interfaces.txt":       true,
		".steam_interfaces.hash":     true,
	}

	var restoredCount int
	walkErr := filepath.WalkDir(absTargetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "steam_settings" {
				if dryRun {
					slog.Info("[DRY RUN] Would remove directory", "path", path)
				} else {
					_ = os.RemoveAll(path)
					slog.Info("Removed directory", "path", path)
				}
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if origPath, isBak := util.ResolveOriginalBackupPath(path); isBak {
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
		} else if loaderFileNames[name] {
			if dryRun {
				slog.Info("[DRY RUN] Would remove loader artifact", "path", path)
			} else {
				_ = os.Remove(path)
				slog.Info("Removed loader artifact", "path", path)
			}
		}
		return nil
	})
	if walkErr != nil {
		return restoredCount, fmt.Errorf("walk directory for backup restoration: %w", walkErr)
	}

	// Clean up root steam_appid.txt
	appIDTxt := filepath.Join(absTargetDir, "steam_appid.txt")
	if util.FileExists(appIDTxt) {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_appid.txt", "path", appIDTxt)
		} else {
			_ = os.Remove(appIDTxt)
			slog.Info("Removed steam_appid.txt", "path", appIDTxt)
		}
	}

	// Clean up .gamux/launch.sh and legacy root launch.sh
	for _, scriptRel := range []string{filepath.Join(".gamux", "launch.sh"), "launch.sh"} {
		launchSh := filepath.Join(absTargetDir, scriptRel)
		if util.FileExists(launchSh) {
			if dryRun {
				slog.Info("[DRY RUN] Would remove launch script", "path", launchSh)
			} else {
				_ = os.Remove(launchSh)
				slog.Info("Removed launch script", "path", launchSh)
			}
		}
	}

	slog.Info("Goldberg emulator revert completed", "restoredFiles", restoredCount)
	return restoredCount, nil
}
