package runner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"
)

// PrepareLinuxSteamEnvironment configures ~/.steam for Linux native games running in Goldberg loader mode.
// It backs up existing steam files, symlinks Goldberg's steamclient.so into sdk64 and sdk32,
// symlinks the game's steam_settings, and writes the current process PID to ~/.steam/steam.pid.
// It returns a cleanup function that restores original state upon game termination.
func PrepareLinuxSteamEnvironment(cfg *config.Config, resolved LaunchOptions) (func(), error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home dir: %w", err)
	}

	steamBase := filepath.Join(home, ".steam")
	sdk64Dir := filepath.Join(steamBase, "sdk64")
	sdk32Dir := filepath.Join(steamBase, "sdk32")

	if err := os.MkdirAll(sdk64Dir, 0755); err != nil {
		return nil, fmt.Errorf("create ~/.steam/sdk64: %w", err)
	}
	if err := os.MkdirAll(sdk32Dir, 0755); err != nil {
		return nil, fmt.Errorf("create ~/.steam/sdk32: %w", err)
	}

	gbeBase := cfg.GbeDir
	if gbeBase == "" {
		gbeBase = config.DefaultConfig().GbeDir
	}
	gbeBase = util.CanonicalPath(gbeBase)

	so64 := filepath.Join(gbeBase, "linux_release", "experimental", "x64", "steamclient.so")
	so32 := filepath.Join(gbeBase, "linux_release", "experimental", "x86", "steamclient.so")
	if !util.FileExists(so32) {
		fallback32 := filepath.Join(gbeBase, "linux_release", "experimental", "x32", "steamclient.so")
		if util.FileExists(fallback32) {
			so32 = fallback32
		}
	}

	pidFile := filepath.Join(steamBase, "steam.pid")
	pidBackup := filepath.Join(steamBase, "steam.pid.orig")
	var pidHadOrig bool
	if util.FileExists(pidFile) {
		if pidBytes, rErr := os.ReadFile(pidFile); rErr == nil {
			if pidVal, pErr := strconv.Atoi(strings.TrimSpace(string(pidBytes))); pErr == nil && pidVal > 0 {
				if syscall.Kill(pidVal, 0) == nil {
					slog.Warn("Existing Steam PID is currently active; GBE loader may conflict with running Steam", "pid", pidVal)
				}
			}
		}
		if !util.FileExists(pidBackup) {
			_ = os.Rename(pidFile, pidBackup)
			pidHadOrig = true
		} else {
			_ = os.Remove(pidFile)
		}
	}

	// Write our current PID so SteamAPI_IsSteamRunning() passes
	currentPid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", currentPid)), 0644); err != nil {
		slog.Warn("Failed to write ~/.steam/steam.pid", "error", err)
	}

	// Setup sdk64 steamclient.so
	sdk64Client := filepath.Join(sdk64Dir, "steamclient.so")
	sdk64Backup := filepath.Join(sdk64Dir, "steamclient.so.orig")
	var sdk64HadOrig bool
	if util.FileExists(sdk64Client) {
		if !util.FileExists(sdk64Backup) {
			_ = os.Rename(sdk64Client, sdk64Backup)
			sdk64HadOrig = true
		} else {
			_ = os.Remove(sdk64Client)
		}
	}
	if util.FileExists(so64) {
		if err := os.Symlink(so64, sdk64Client); err != nil {
			_ = util.CopyFile(so64, sdk64Client)
		}
	}

	// Setup sdk32 steamclient.so
	sdk32Client := filepath.Join(sdk32Dir, "steamclient.so")
	sdk32Backup := filepath.Join(sdk32Dir, "steamclient.so.orig")
	var sdk32HadOrig bool
	if util.FileExists(sdk32Client) {
		if !util.FileExists(sdk32Backup) {
			_ = os.Rename(sdk32Client, sdk32Backup)
			sdk32HadOrig = true
		} else {
			_ = os.Remove(sdk32Client)
		}
	}
	if util.FileExists(so32) {
		if err := os.Symlink(so32, sdk32Client); err != nil {
			_ = util.CopyFile(so32, sdk32Client)
		}
	}

	// Setup steam_settings symlinks in sdk64 and sdk32
	gameSettings := filepath.Join(resolved.GameDir, "steam_settings")
	sdk64Settings := filepath.Join(sdk64Dir, "steam_settings")
	sdk32Settings := filepath.Join(sdk32Dir, "steam_settings")
	if util.DirExists(gameSettings) {
		_ = os.Remove(sdk64Settings)
		_ = os.Symlink(gameSettings, sdk64Settings)
		_ = os.Remove(sdk32Settings)
		_ = os.Symlink(gameSettings, sdk32Settings)
	}

	cleanup := func() {
		slog.Info("Cleaning up Goldberg Linux Steam environment")
		_ = os.Remove(sdk64Settings)
		_ = os.Remove(sdk32Settings)

		_ = os.Remove(sdk64Client)
		if sdk64HadOrig && util.FileExists(sdk64Backup) {
			_ = os.Rename(sdk64Backup, sdk64Client)
		}

		_ = os.Remove(sdk32Client)
		if sdk32HadOrig && util.FileExists(sdk32Backup) {
			_ = os.Rename(sdk32Backup, sdk32Client)
		}

		_ = os.Remove(pidFile)
		if pidHadOrig && util.FileExists(pidBackup) {
			_ = os.Rename(pidBackup, pidFile)
		}
	}

	return cleanup, nil
}
