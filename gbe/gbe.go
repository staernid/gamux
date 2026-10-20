package gbe

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/github"
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
