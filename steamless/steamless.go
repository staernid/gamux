package steamless

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"
)

// EnsureBinary extracts and returns the embedded steamless executable, caching it in cfg.SteamlessDir.
func EnsureBinary(ctx context.Context, cfg *config.Config) (string, error) {
	// 1. If we have embedded binary data for this target architecture
	if len(embeddedBinary) > 0 {
		destDir := cfg.SteamlessDir
		if !filepath.IsAbs(destDir) {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve user home dir: %w", err)
			}
			destDir = filepath.Join(homeDir, destDir)
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return "", fmt.Errorf("create steamless dir %s: %w", destDir, err)
		}

		destPath := filepath.Join(destDir, embeddedBinaryName)

		// Check if already extracted and matching size
		if fi, err := os.Stat(destPath); err == nil && fi.Size() == int64(len(embeddedBinary)) {
			return destPath, nil
		}

		if err := os.WriteFile(destPath, embeddedBinary, 0755); err != nil {
			return "", fmt.Errorf("extract embedded steamless binary to %s: %w", destPath, err)
		}

		slog.Info("Extracted embedded Steamless binary", "path", destPath)
		return destPath, nil
	}

	// 2. Fallback to system PATH
	for _, binName := range []string{"steamless", "steamless-cli"} {
		if path, err := exec.LookPath(binName); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("embedded steamless binary unavailable for this architecture and 'steamless' not found in PATH")
}

// UnpackGameDirectory scans a game directory (or target executable) and attempts SteamStub DRM unpacking on all discovered executables.
func UnpackGameDirectory(ctx context.Context, cfg *config.Config, gameDir string, dryRun bool) (int, error) {
	if strings.TrimSpace(gameDir) == "" || !util.FileExists(gameDir) {
		return 0, nil
	}

	slog.Info("Attempting Steamless DRM unpacking on game directory...", "path", gameDir)

	if dryRun {
		slog.Info("[DRY RUN] Would attempt Steamless DRM unpacking on game directory", "path", gameDir)
		return 0, nil
	}

	steamlessBin, err := EnsureBinary(ctx, cfg)
	if err != nil {
		slog.Warn("Steamless DRM unpacking skipped (binary unavailable)", "error", err)
		return 0, nil
	}

	// Run steamless CLI on the game directory (or target executable)
	cmd := exec.CommandContext(ctx, steamlessBin, gameDir)
	outputBytes, err := cmd.CombinedOutput()
	outputStr := string(outputBytes)
	slog.Info("Steamless inspection completed", "path", gameDir, "output", strings.TrimSpace(outputStr))

	// Walk directory to replace any created .unpacked binaries with their originals (backed up to .ORIGINAL)
	var unpackedCount int
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".unpacked") || strings.HasSuffix(path, ".unpacked.exe") {
			origPath := strings.TrimSuffix(strings.TrimSuffix(path, ".unpacked.exe"), ".unpacked")
			if util.FileExists(origPath) {
				if err := util.BackupAndReplace(path, origPath); err == nil {
					_ = os.Remove(path)
					slog.Info("[SUCCESS] Steamless unpacked SteamStub DRM from executable", "exe", origPath)
					unpackedCount++
				} else {
					slog.Error("Failed to replace original executable with unpacked version", "orig", origPath, "error", err)
				}
			}
		}
		return nil
	})

	return unpackedCount, nil
}

// UnpackExecutable attempts to unpack SteamStub DRM from a target executable or its containing directory.
func UnpackExecutable(ctx context.Context, cfg *config.Config, exePath string, dryRun bool) (bool, error) {
	count, err := UnpackGameDirectory(ctx, cfg, exePath, dryRun)
	return count > 0, err
}
