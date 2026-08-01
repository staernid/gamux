package gbe

import (
	"context"
	"fmt"
	"gbe_fork_helper/config"
	"gbe_fork_helper/steam"
	"gbe_fork_helper/util"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ApplyGBE applies the GBE patch to a specified platform.
func ApplyGBE(ctx context.Context, platform, appID string, dryRun bool) error {
	platformCfg, ok := config.PlatformConfig[platform]
	if !ok {
		var validPlatforms []string
		for p := range config.PlatformConfig {
			validPlatforms = append(validPlatforms, p)
		}
		return fmt.Errorf("invalid platform: '%s'. Valid platforms: %s", platform, strings.Join(validPlatforms, ", "))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	gbePath := filepath.Join(homeDir, config.GbeDir, platformCfg.Subdir, "experimental", "x"+platformCfg.Arch)

	var targetFiles []string
	walkErr := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == platformCfg.Target {
			targetFiles = append(targetFiles, path)
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to search for files: %w", walkErr)
	}

	if len(targetFiles) == 0 {
		slog.Warn("No target files found", "platform", platform)
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
		slog.Info("Found potential target", "file", file)

		targetHash, err := util.GetHash(file)
		if err != nil {
			slog.Error("Failed to get hash", "file", file, "error", err)
			continue
		}

		if targetHash == sourceHash {
			slog.Info("File is already up-to-date", "file", file)
			continue
		}

		if dryRun {
			slog.Info("[DRY RUN] Would replace file", "file", file)
		} else if err := util.BackupAndReplace(sourceFile, file); err != nil {
			slog.Error("Failed to replace file", "file", file, "error", err)
			continue
		}

		if platformCfg.Additional != "" {
			additionalSource := filepath.Join(gbePath, platformCfg.Additional)
			additionalDest := filepath.Join(filepath.Dir(file), platformCfg.Additional)
			if _, err := os.Stat(additionalSource); err == nil { // Check if source exists
				if _, err := os.Stat(additionalDest); err == nil { // Check if destination exists
					if dryRun {
						slog.Info("[DRY RUN] Would replace additional file", "dest", additionalDest)
					} else if err := util.BackupAndReplace(additionalSource, additionalDest); err != nil {
						slog.Warn("Failed to replace additional file", "dest", additionalDest, "error", err)
					}
				} else {
					slog.Info("Additional file does not exist in destination, skipping replacement", "dest", additionalDest)
				}
			}
		}

		homeDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		generatorPath := filepath.Join(homeDir, config.GbeDir, platformCfg.Subdir, "tools", "generate_interfaces", platformCfg.Generator)
		if _, err := os.Stat(generatorPath); err == nil {
			if dryRun {
				slog.Info("[DRY RUN] Would run generator", "generator", platformCfg.Generator)
			} else {
				slog.Info("Running generator", "generator", platformCfg.Generator)
				if runtime.GOOS != "windows" {
					if err := os.Chmod(generatorPath, 0755); err != nil {
						slog.Warn("Failed to set executable permissions", "path", generatorPath, "error", err)
					}
				}
				cmd := exec.Command(generatorPath, filepath.Base(file))
				cmd.Dir = filepath.Dir(file)
				if out, err := cmd.CombinedOutput(); err != nil {
					slog.Error("Generator failed", "error", err, "output", string(out))
				}
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

	slog.Info("GBE application process completed")
	return nil
}
