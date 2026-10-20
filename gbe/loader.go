package gbe

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"
)

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
