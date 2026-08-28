package compat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

// DefaultPrefixesDir returns the base directory where isolated per-game Wine prefixes are stored.
func DefaultPrefixesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "Games/umu"
	}
	return filepath.Join(home, "Games", "umu")
}

// DefaultCentralUserDataDir returns the shared central directory for Syncthing user data.
func DefaultCentralUserDataDir(cfg *config.Config) string {
	if cfg != nil && cfg.SteamUserdata != "" && strings.Contains(cfg.SteamUserdata, "drive_c") {
		return util.ExpandPath(cfg.SteamUserdata)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "Games/umu/umu-default/drive_c/users/steamuser"
	}
	return filepath.Join(home, "Games", "umu", "umu-default", "drive_c", "users", "steamuser")
}

// ResolvePrefixIdentifier generates a deterministic, filesystem-safe identifier for a game's prefix.
func ResolvePrefixIdentifier(game *domain.Game) string {
	if game == nil {
		return "default"
	}
	if game.NumericAppID > 0 {
		return fmt.Sprintf("%d", game.NumericAppID)
	}
	if game.AppID != "" && game.AppID != "0" {
		return game.AppID
	}
	if game.GOGID != "" {
		return "gog_" + game.GOGID
	}
	if game.Name != "" {
		sanitized := util.SanitizeInstallDir(game.Name)
		if sanitized != "" {
			return sanitized
		}
	}
	if game.GameDir != "" {
		return filepath.Base(game.GameDir)
	}
	return "default"
}

// EnsurePrefix resolves the effective prefix for a game, creating an isolated per-game Wine prefix
// and symlinking its user profile (drive_c/users/steamuser) to the central Syncthing user data directory.
func EnsurePrefix(ctx context.Context, cfg *config.Config, game *domain.Game, customPrefix string) (string, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// 1. If explicit custom prefix is supplied, respect it directly
	if customPrefix != "" {
		expanded := util.ExpandPath(customPrefix)
		return util.CanonicalPath(expanded), nil
	}

	// 2. If global config has an explicit custom wine prefix configured (and not default), use it
	if cfg.WinePrefix != "" {
		expanded := util.ExpandPath(cfg.WinePrefix)
		return util.CanonicalPath(expanded), nil
	}

	// 3. Construct isolated per-game prefix path
	prefixID := ResolvePrefixIdentifier(game)
	prefixBase := DefaultPrefixesDir()
	prefixDir := filepath.Join(prefixBase, prefixID)

	usersDir := filepath.Join(prefixDir, "drive_c", "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return "", fmt.Errorf("create prefix users directory %s: %w", usersDir, err)
	}

	// 4. Ensure central user data directory exists (e.g. ~/Games/umu/umu-default/drive_c/users/steamuser)
	centralUserDir := DefaultCentralUserDataDir(cfg)
	if err := os.MkdirAll(centralUserDir, 0755); err != nil {
		slog.Warn("Failed to create central user data directory", "path", centralUserDir, "error", err)
	}

	// 5. Symlink drive_c/users/steamuser -> centralUserDir
	steamuserSymlink := filepath.Join(usersDir, "steamuser")
	fi, err := os.Lstat(steamuserSymlink)
	if err != nil {
		if os.IsNotExist(err) {
			if symErr := os.Symlink(centralUserDir, steamuserSymlink); symErr != nil {
				slog.Warn("Failed to symlink prefix user profile to central directory", "target", steamuserSymlink, "dest", centralUserDir, "error", symErr)
			} else {
				slog.Debug("Linked prefix user profile to central Syncthing directory", "prefix", prefixDir, "central", centralUserDir)
			}
		}
	} else if fi.Mode()&os.ModeSymlink == 0 {
		// Existing physical directory: if empty, replace with symlink
		entries, rErr := os.ReadDir(steamuserSymlink)
		if rErr == nil && len(entries) == 0 {
			_ = os.Remove(steamuserSymlink)
			_ = os.Symlink(centralUserDir, steamuserSymlink)
		}
	}

	return util.CanonicalPath(prefixDir), nil
}
