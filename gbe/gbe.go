package gbe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	Config     *config.Config
	TargetDir  string
	Platform   string
	AppID      string
	DryRun     bool
	Portable   bool
	WinePrefix string
	ExePath    string
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
		if !d.IsDir() && d.Name() == platformCfg.Target {
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

	sourceFile := filepath.Join(gbePath, platformCfg.Target)
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: '%s'", sourceFile)
	}

	sourceHash, err := util.GetHash(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to get hash of source file: %w", err)
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
			if platform == "win64" || platform == "win32" {
				targetExe := exePath
				if targetExe == "" {
					targetExe = file
				}
				if err := SetupLoaderMode(gbeBase, absTargetDir, platform, appID, targetExe, dryRun); err != nil {
					slog.Warn("Failed to setup Goldberg loader mode", "error", err)
				}
				dllPath := filepath.Join(gbePath, platformCfg.Additional)
				ConfigureWineRegistry(winePrefix, platform, dllPath, dryRun)
			}
		}



		// Determine the source original binary for interface detection
		sourceDll := file
		if util.FileExists(file + ".ORIGINAL") {
			sourceDll = file + ".ORIGINAL"
		}

		if dryRun {
			slog.Info("[DRY RUN] Would extract steam_interfaces.txt into steam_settings", "source", sourceDll)
		} else {
			targetDirs := []string{
				filepath.Join(absTargetDir, "steam_settings"),       // Root steam_settings (for Loader mode)
				filepath.Join(filepath.Dir(file), "steam_settings"), // Subdir steam_settings (for plugin folders)
				filepath.Dir(file),                                  // Alongside steam_api.dll (for portable mode)
			}
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

	// Configure language, persona profile, SteamID, and overlay settings in steam_settings/
	supportedLangs, _ := steam.FetchSupportedLanguages(ctx, appID)
	for _, file := range targetFiles {
		steamSettingsDir := filepath.Join(filepath.Dir(file), "steam_settings")
		if err := ConfigureCustomSettings(steamSettingsDir, cfg, supportedLangs, dryRun); err != nil {
			slog.Warn("Failed to configure custom emulator settings", "dir", steamSettingsDir, "error", err)
		}
	}

	// Ensure steam_appid.txt exists in root directory, steam_settings, and alongside library
	if appID != "" && appID != "0" {
		appidDirs := []string{
			absTargetDir,
			filepath.Join(absTargetDir, "steam_settings"),
		}
		for _, file := range targetFiles {
			appidDirs = append(appidDirs, filepath.Dir(file))
			appidDirs = append(appidDirs, filepath.Join(filepath.Dir(file), "steam_settings"))
		}
		for _, dir := range appidDirs {
			if dryRun {
				slog.Info("[DRY RUN] Would ensure steam_appid.txt", "dir", dir, "appID", appID)
			} else {
				_ = os.MkdirAll(dir, 0755)
				_ = os.WriteFile(filepath.Join(dir, "steam_appid.txt"), []byte(appID+"\n"), 0644)
			}
		}
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
	targetLang := cfg.Language
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
	if cfg.DisableOverlay {
		overlayFile := filepath.Join(steamSettingsDir, "disable_overlay.txt")
		if dryRun {
			slog.Info("[DRY RUN] Would write disable_overlay.txt")
		} else {
			_ = os.WriteFile(overlayFile, []byte("1\n"), 0644)
		}
	}

	// 5. Offline Mode
	if cfg.OfflineMode {
		offlineFile := filepath.Join(steamSettingsDir, "offline.txt")
		if dryRun {
			slog.Info("[DRY RUN] Would write offline.txt")
		} else {
			_ = os.WriteFile(offlineFile, []byte("1\n"), 0644)
		}
	}

	return nil
}

// ConfigureWineRegistry configures the Wine registry to load Goldberg's steamclient DLL for non-portable loader mode.
func ConfigureWineRegistry(winePrefix, platform, dllPath string, dryRun bool) {
	winePath, err := exec.LookPath("wine")
	if err != nil {
		slog.Warn("Wine binary not found in PATH; skipping Wine registry configuration", "platform", platform)
		return
	}

	regKey := "SteamClientDll64"
	if platform == "win32" {
		regKey = "SteamClientDll"
	}

	if dryRun {
		slog.Info("[DRY RUN] Would configure Wine registry key", "key", regKey, "path", dllPath)
		return
	}

	cmd := exec.Command(winePath, "reg", "add", `HKCU\Software\Valve\Steam`, "/v", regKey, "/t", "REG_SZ", "/d", dllPath, "/f")
	if winePrefix != "" {
		cmd.Env = append(os.Environ(), "WINEPREFIX="+winePrefix)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("Failed to set Wine registry key", "key", regKey, "error", err, "output", string(out))
	} else {
		slog.Info("Successfully configured Wine registry key for Goldberg loader", "key", regKey, "path", dllPath)
	}
}

// SetupLoaderMode configures Goldberg's official helper loader (steamclient_loader_x64/x86.exe and ColdClientLoader.ini)
// by creating relative symlinks to central GBE binaries in ~/.local/share/gbe_fork/ and generating ColdClientLoader.ini.
func SetupLoaderMode(gbeBase, absTargetDir, platform, appID, exePath string, dryRun bool) error {
	if platform != "win64" && platform != "win32" {
		return nil
	}

	loaderDir := filepath.Join(gbeBase, "win_release", "steamclient_experimental")
	loaderExeName := "steamclient_loader_x64.exe"
	if platform == "win32" {
		loaderExeName = "steamclient_loader_x86.exe"
	}

	filesToSymlink := []string{
		loaderExeName,
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
	}

	for _, fname := range filesToSymlink {
		src := filepath.Join(loaderDir, fname)
		dst := filepath.Join(absTargetDir, fname)
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

	relExe := exePath
	if filepath.IsAbs(exePath) {
		if r, err := filepath.Rel(absTargetDir, exePath); err == nil {
			relExe = r
		} else {
			relExe = filepath.Base(exePath)
		}
	}

	exeRunDir := ""
	if dir := filepath.Dir(relExe); dir != "." && dir != "" && dir != "/" {
		exeRunDir = dir
	}

	iniContent := fmt.Sprintf(`[SteamClient]
Exe=%s
ExeRunDir=%s
ExeCommandLine=
AppId=%s
SteamClientDll=steamclient.dll
SteamClient64Dll=steamclient64.dll

[Injection]
ForceInjectSteamClient=1
ForceInjectGameOverlayRenderer=0
IgnoreInjectionError=0
IgnoreLoaderArchDifference=0
`, relExe, exeRunDir, appID)

	iniPath := filepath.Join(absTargetDir, "ColdClientLoader.ini")
	if dryRun {
		slog.Info("[DRY RUN] Would write ColdClientLoader.ini", "path", iniPath)
	} else {
		if err := os.WriteFile(iniPath, []byte(iniContent), 0644); err != nil {
			return fmt.Errorf("write ColdClientLoader.ini: %w", err)
		}
		slog.Info("Successfully wrote ColdClientLoader.ini", "path", iniPath, "appID", appID, "exe", relExe)
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
	if dryRun {
		slog.Info("[DRY RUN] Would write achievements.json", "path", achievementsJSONPath, "count", len(achievements))
	} else {
		data, err := json.MarshalIndent(achievements, "", "  ")
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

	return nil
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

	var restoredCount int
	walkErr := filepath.WalkDir(absTargetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		name := d.Name()
		if strings.Contains(name, ".ORIGINAL") {
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
	if walkErr != nil {
		return restoredCount, fmt.Errorf("walk directory for backup restoration: %w", walkErr)
	}

	// Clean up loader helper binaries and artifacts
	loaderFiles := []string{
		"ColdClientLoader.ini",
		"steamclient_loader_x64.exe",
		"steamclient_loader_x86.exe",
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
		"steam_interfaces.txt",
		".steam_interfaces.hash",
	}

	for _, fname := range loaderFiles {
		p := filepath.Join(absTargetDir, fname)
		if util.FileExists(p) {
			if dryRun {
				slog.Info("[DRY RUN] Would remove loader file", "path", p)
			} else {
				_ = os.Remove(p)
				slog.Info("Removed loader file", "path", p)
			}
		}
	}

	// Clean up steam_settings directory
	steamSettingsDir := filepath.Join(absTargetDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_settings directory", "path", steamSettingsDir)
		} else {
			if err := os.RemoveAll(steamSettingsDir); err == nil {
				slog.Info("Removed steam_settings directory", "path", steamSettingsDir)
			}
		}
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

	slog.Info("Goldberg emulator revert completed", "restoredFiles", restoredCount)
	return restoredCount, nil
}



