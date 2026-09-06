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
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/util"
)

// DefaultPrefixesDir returns the base directory where isolated per-game Wine prefixes are stored for Player 1.
func DefaultPrefixesDir() string {
	return DefaultPrefixesDirForSeat("p1")
}

// DefaultPrefixesDirForSeat returns the base directory where isolated per-game Wine prefixes are stored
// for a specific seat ("p1", "p2", etc.). Default ("p1" or empty) is ~/Games/umu; "p2" is ~/Games/umu-p2.
func DefaultPrefixesDirForSeat(seatID string) string {
	home, err := os.UserHomeDir()
	baseName := "umu"
	if seatID != "" && seatID != "p1" {
		baseName = fmt.Sprintf("umu-%s", seatID)
	}
	if err != nil {
		return filepath.Join("Games", baseName)
	}
	return filepath.Join(home, "Games", baseName)
}

// DefaultPrefixDirs returns known default prefix base directories (e.g. ~/Games/umu, ~/Games/umu-p2).
func DefaultPrefixDirs() []string {
	return []string{
		DefaultPrefixesDirForSeat("p1"),
		DefaultPrefixesDirForSeat("p2"),
	}
}

// IsKnownPrefixDir returns true if the specified path matches a known default prefix root directory.
func IsKnownPrefixDir(path string) bool {
	cleanPath := util.CanonicalPath(path)
	if cleanPath == "" {
		return false
	}
	for _, dir := range DefaultPrefixDirs() {
		if cleanPath == util.CanonicalPath(dir) {
			return true
		}
	}
	return false
}

// DefaultCentralUserDataDir returns the shared central directory for Syncthing user data for Player 1.
func DefaultCentralUserDataDir(cfg *config.Config) string {
	return DefaultCentralUserDataDirForSeat(cfg, "p1")
}

// DefaultCentralUserDataDirForSeat returns the shared central directory for user data for a seat.
func DefaultCentralUserDataDirForSeat(cfg *config.Config, seatID string) string {
	if (seatID == "" || seatID == "p1") && cfg != nil && cfg.SteamUserdata != "" && strings.Contains(cfg.SteamUserdata, "drive_c") {
		return util.ExpandPath(cfg.SteamUserdata)
	}

	home, err := os.UserHomeDir()
	baseName := "umu"
	if seatID != "" && seatID != "p1" {
		baseName = fmt.Sprintf("umu-%s", seatID)
	}
	if err != nil {
		return filepath.Join("Games", baseName, "umu-default", "drive_c", "users", "steamuser")
	}
	return filepath.Join(home, "Games", baseName, "umu-default", "drive_c", "users", "steamuser")
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

// EnsurePrefix resolves the effective prefix for Player 1, creating an isolated per-game Wine prefix
// and symlinking its user profile (drive_c/users/steamuser) to the central Syncthing user data directory.
func EnsurePrefix(ctx context.Context, cfg *config.Config, game *domain.Game, customPrefix string) (string, error) {
	return EnsureSeatPrefix(ctx, cfg, game, "p1", customPrefix)
}

// EnsureSeatPrefix resolves the effective prefix for a player seat ("p1" or "p2"), creating an isolated
// per-game Wine prefix and symlinking its user profile (drive_c/users/steamuser) to the central user data directory.
func EnsureSeatPrefix(ctx context.Context, cfg *config.Config, game *domain.Game, seatID string, customPrefix string) (string, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	var prefixDir string
	if customPrefix != "" {
		prefixDir = util.CanonicalPath(util.ExpandPath(customPrefix))
	} else if seatID == "p2" && cfg.Multiseat.Player2WinePrefix != "" {
		prefixDir = util.CanonicalPath(util.ExpandPath(cfg.Multiseat.Player2WinePrefix))
	} else if seatID != "p2" && cfg.WinePrefix != "" {
		prefixDir = util.CanonicalPath(util.ExpandPath(cfg.WinePrefix))
	} else {
		prefixID := ResolvePrefixIdentifier(game)
		prefixBase := DefaultPrefixesDirForSeat(seatID)
		prefixDir = filepath.Join(prefixBase, prefixID)
	}

	usersDir := filepath.Join(prefixDir, "drive_c", "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return "", fmt.Errorf("create prefix users directory %s: %w", usersDir, err)
	}

	// Ensure central user data directory exists (e.g. ~/Games/umu/umu-default/drive_c/users/steamuser or ~/Games/umu-p2/...)
	centralUserDir := DefaultCentralUserDataDirForSeat(cfg, seatID)
	if err := os.MkdirAll(centralUserDir, 0755); err != nil {
		slog.Warn("Failed to create central user data directory", "path", centralUserDir, "error", err)
	}

	// Symlink drive_c/users/steamuser -> centralUserDir
	steamuserSymlink := filepath.Join(usersDir, "steamuser")
	fi, err := os.Lstat(steamuserSymlink)
	if err != nil {
		if os.IsNotExist(err) {
			if symErr := os.Symlink(centralUserDir, steamuserSymlink); symErr != nil {
				slog.Warn("Failed to symlink prefix user profile to central directory", "target", steamuserSymlink, "dest", centralUserDir, "error", symErr)
			} else {
				slog.Debug("Linked prefix user profile to central user directory", "prefix", prefixDir, "central", centralUserDir)
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

	canonicalPrefix := util.CanonicalPath(prefixDir)
	if game != nil && domain.NormalizeOperatingMode(game.EffectiveOperatingMode()) == domain.ModeLoader {
		gbeBase := cfg.GbeDir
		if gbeBase == "" {
			gbeBase = config.DefaultConfig().GbeDir
		}
		settingsDir := filepath.Join(game.GameDir, "steam_settings")
		_ = gbe.SetupWinePrefixSteamDir(canonicalPrefix, gbeBase, settingsDir, game.AppID, false)
	}

	return canonicalPrefix, nil
}

// ResolveGamePrefix returns the effective Wine/Proton prefix path for Player 1 without mutating the filesystem.
func ResolveGamePrefix(cfg *config.Config, game *domain.Game, customPrefix string) string {
	return ResolveSeatGamePrefix(cfg, game, "p1", customPrefix)
}

// ResolveSeatGamePrefix returns the effective Wine/Proton prefix path for a seat without mutating the filesystem.
func ResolveSeatGamePrefix(cfg *config.Config, game *domain.Game, seatID string, customPrefix string) string {
	if customPrefix != "" {
		return util.CanonicalPath(util.ExpandPath(customPrefix))
	}
	if seatID == "p2" && cfg != nil && cfg.Multiseat.Player2WinePrefix != "" {
		return util.CanonicalPath(util.ExpandPath(cfg.Multiseat.Player2WinePrefix))
	} else if seatID != "p2" && cfg != nil && cfg.WinePrefix != "" {
		return util.CanonicalPath(util.ExpandPath(cfg.WinePrefix))
	}

	prefixID := ResolvePrefixIdentifier(game)
	prefixBase := DefaultPrefixesDirForSeat(seatID)
	gamePrefix := filepath.Join(prefixBase, prefixID)

	if util.FileExists(gamePrefix) {
		return util.CanonicalPath(gamePrefix)
	}

	// If game prefix doesn't exist yet, check shared umu-default
	umuDefault := filepath.Join(prefixBase, "umu-default")
	if util.FileExists(umuDefault) {
		return util.CanonicalPath(umuDefault)
	}

	return util.CanonicalPath(gamePrefix)
}

// DetectWineUser discovers the active Windows user profile name inside a Wine prefix.
func DetectWineUser(winePrefix string) string {
	if winePrefix == "" {
		return "steamuser"
	}

	usersDir := filepath.Join(winePrefix, "drive_c", "users")
	if !util.FileExists(usersDir) {
		return "steamuser"
	}

	entries, err := os.ReadDir(usersDir)
	if err != nil {
		return "steamuser"
	}

	systemUsers := map[string]bool{
		"public": true, "default": true, "default user": true, "all users": true,
	}

	var candidates []string
	for _, entry := range entries {
		nameLower := strings.ToLower(entry.Name())
		if !systemUsers[nameLower] {
			candidates = append(candidates, entry.Name())
		}
	}

	for _, c := range candidates {
		if strings.EqualFold(c, "steamuser") {
			return c
		}
	}

	currentLinuxUser := os.Getenv("USER")
	for _, c := range candidates {
		if currentLinuxUser != "" && strings.EqualFold(c, currentLinuxUser) {
			return c
		}
	}

	if len(candidates) > 0 {
		return candidates[0]
	}

	return "steamuser"
}
