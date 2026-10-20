package gbe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

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
