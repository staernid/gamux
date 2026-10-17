package steamless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/util"
)

// FileResult represents the structured JSON output from steamless for a single file.
type FileResult struct {
	File        string `json:"file"`
	Unpacked    bool   `json:"unpacked"`
	StubVersion string `json:"stub_version,omitempty"`
	OutputFile  string `json:"output_file,omitempty"`
	Error       string `json:"error,omitempty"`
}

// DirectoryResult represents the structured JSON output from steamless for a directory scan.
type DirectoryResult struct {
	Scanned       int          `json:"scanned"`
	UnpackedCount int          `json:"unpacked_count"`
	CleanCount    int          `json:"clean_count"`
	ErrorCount    int          `json:"error_count"`
	Results       []FileResult `json:"results"`
}

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

	// 3. Fallback to cached binary in cfg.SteamlessDir or auto-bootstrap from GitHub Releases
	destDir := util.CanonicalPath(cfg.SteamlessDir)
	for _, binName := range []string{"steamless-cli", "steamless"} {
		p := filepath.Join(destDir, binName)
		if util.FileExists(p) {
			return p, nil
		}
	}

	slog.Info("Steamless binary missing locally, bootstrapping from GitHub releases...", "dest", destDir)
	if err := github.UpdateSteamlessAssets(ctx, cfg, destDir); err == nil {
		for _, binName := range []string{"steamless-cli", "steamless"} {
			p := filepath.Join(destDir, binName)
			if util.FileExists(p) {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("steamless binary unavailable locally and could not be bootstrapped from GitHub")
}

// UnpackFile attempts to unpack SteamStub DRM from a specific target executable.
func UnpackFile(ctx context.Context, cfg *config.Config, exePath string, dryRun bool) (bool, error) {
	if strings.TrimSpace(exePath) == "" || !util.FileExists(exePath) {
		return false, nil
	}

	fi, err := os.Stat(exePath)
	if err != nil || fi.IsDir() {
		return false, nil
	}

	if dryRun {
		slog.Info("[DRY RUN] Would attempt Steamless DRM unpacking on executable", "exe", exePath)
		return false, nil
	}

	steamlessBin, err := EnsureBinary(ctx, cfg)
	if err != nil {
		slog.Warn("Steamless DRM unpacking skipped (binary unavailable)", "error", err)
		return false, nil
	}

	slog.Info("Attempting Steamless DRM unpacking on executable...", "exe", exePath)

	cmd := exec.CommandContext(ctx, steamlessBin, "--in-place", "--backup", "--json", exePath)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	execErr := cmd.Run()
	var exitCode int
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return false, fmt.Errorf("execute steamless: %w", execErr)
		}
	}

	var res FileResult
	_ = json.Unmarshal(stdoutBuf.Bytes(), &res)

	switch exitCode {
	case 0:
		if res.StubVersion != "" {
			slog.Info("[SUCCESS] Steamless unpacked SteamStub DRM from executable", "exe", exePath, "variant", res.StubVersion)
		} else {
			slog.Info("[SUCCESS] Steamless unpacked SteamStub DRM from executable", "exe", exePath)
		}
		return true, nil
	case 2:
		slog.Debug("Executable is clean (not packed with SteamStub)", "exe", exePath)
		return false, nil
	default:
		errMsg := strings.TrimSpace(stderrBuf.String())
		if res.Error != "" {
			errMsg = res.Error
		} else if errMsg == "" {
			errMsg = strings.TrimSpace(stdoutBuf.String())
		}
		slog.Warn("Steamless unpacking failed", "exe", exePath, "error", errMsg, "exit_code", exitCode)
		return false, fmt.Errorf("steamless failed (%d): %s", exitCode, errMsg)
	}
}

// UnpackGameDirectory scans a game directory (or target executable) and attempts SteamStub DRM unpacking on all discovered executables.
func UnpackGameDirectory(ctx context.Context, cfg *config.Config, gameDir string, dryRun bool) (int, error) {
	if strings.TrimSpace(gameDir) == "" || !util.FileExists(gameDir) {
		return 0, nil
	}

	fi, err := os.Stat(gameDir)
	if err != nil {
		return 0, err
	}
	if !fi.IsDir() {
		unpacked, err := UnpackFile(ctx, cfg, gameDir, dryRun)
		if unpacked {
			return 1, err
		}
		return 0, err
	}

	if dryRun {
		slog.Info("[DRY RUN] Would scan game directory for Steamless DRM unpacking", "dir", gameDir)
		return 0, nil
	}

	steamlessBin, err := EnsureBinary(ctx, cfg)
	if err != nil {
		slog.Warn("Steamless DRM unpacking skipped (binary unavailable)", "error", err)
		return 0, nil
	}

	slog.Info("Scanning game directory for executables to unpack...", "dir", gameDir)

	cmd := exec.CommandContext(ctx, steamlessBin, "--in-place", "--backup", "--recursive", "--json", gameDir)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	execErr := cmd.Run()
	var exitCode int
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 0, fmt.Errorf("execute steamless: %w", execErr)
		}
	}

	var dirRes DirectoryResult
	if err := json.Unmarshal(stdoutBuf.Bytes(), &dirRes); err == nil {
		for _, r := range dirRes.Results {
			if r.Unpacked {
				if r.StubVersion != "" {
					slog.Info("[SUCCESS] Steamless unpacked SteamStub DRM from executable", "exe", r.File, "variant", r.StubVersion)
				} else {
					slog.Info("[SUCCESS] Steamless unpacked SteamStub DRM from executable", "exe", r.File)
				}
			} else if r.Error != "" {
				slog.Warn("Steamless error for executable", "exe", r.File, "error", r.Error)
			}
		}

		slog.Info("Steamless scan complete", "dir", gameDir, "scanned", dirRes.Scanned, "unpacked", dirRes.UnpackedCount, "clean", dirRes.CleanCount, "errors", dirRes.ErrorCount)
		if exitCode == 1 && dirRes.UnpackedCount == 0 && dirRes.ErrorCount > 0 {
			return 0, fmt.Errorf("steamless scan failed on %d file(s)", dirRes.ErrorCount)
		}
		return dirRes.UnpackedCount, nil
	}

	// Fallback if JSON parsing fails
	if exitCode == 0 {
		return 1, nil
	} else if exitCode == 2 {
		return 0, nil
	}

	errMsg := strings.TrimSpace(stderrBuf.String())
	if errMsg == "" {
		errMsg = strings.TrimSpace(stdoutBuf.String())
	}
	return 0, fmt.Errorf("steamless failed (%d): %s", exitCode, errMsg)
}

// UnpackExecutable attempts to unpack SteamStub DRM from a target executable or its containing directory.
func UnpackExecutable(ctx context.Context, cfg *config.Config, exePath string, dryRun bool) (bool, error) {
	count, err := UnpackGameDirectory(ctx, cfg, exePath, dryRun)
	return count > 0, err
}
