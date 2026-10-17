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

// SeatIDFromProfileID maps a profile ID ("main", "p1", "p2", etc.) to a canonical seat identifier ("p1", "p2", etc.).
func SeatIDFromProfileID(profileID string) string {
	if profileID == "" || profileID == "main" || profileID == "p1" {
		return "p1"
	}
	return profileID
}

// ResolveSeatHome returns the filesystem home directory for a seat ID ("p1", "p2", etc.) or profile ID.
// For "main", "p1", or empty string, it returns the host user's actual home directory.
// For secondary profiles ("p2", etc.), it returns $XDG_DATA_HOME/gamux/profiles/<seatID>/home.
func ResolveSeatHome(seatID string) string {
	canonSeat := SeatIDFromProfileID(seatID)
	if canonSeat == "p1" {
		home, _ := os.UserHomeDir()
		return home
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			dataHome = filepath.Join(home, ".local", "share")
		} else {
			dataHome = filepath.Join(os.TempDir(), ".local", "share")
		}
	}
	return filepath.Join(dataHome, "gamux", "profiles", canonSeat, "home")
}

// DefaultPrefixesDir returns the base directory where isolated per-game Wine prefixes are stored for Player 1.
func DefaultPrefixesDir() string {
	return DefaultPrefixesDirForSeat("p1")
}

// DefaultPrefixesDirForSeat returns the base directory where isolated per-game Wine prefixes are stored
// for a specific seat ("p1", "p2", etc.).
// All profiles store prefixes relative to their home directory under ~/Games/umu.
func DefaultPrefixesDirForSeat(seatID string) string {
	home := ResolveSeatHome(seatID)
	return filepath.Join(home, "Games", "umu")
}

// DefaultPrefixDirs returns known default prefix base directories.
func DefaultPrefixDirs() []string {
	return []string{
		DefaultPrefixesDirForSeat("p1"),
	}
}

// IsKnownPrefixDir returns true if the specified path matches a known default prefix root directory.
func IsKnownPrefixDir(path string) bool {
	cleanPath := util.CanonicalPath(path)
	if cleanPath == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err == nil {
		gamesDir := util.CanonicalPath(filepath.Join(home, "Games"))
		if filepath.Dir(cleanPath) == gamesDir {
			base := filepath.Base(cleanPath)
			if base == "umu" || strings.HasPrefix(base, "umu-p") {
				return true
			}
		}
	}
	if filepath.Base(cleanPath) == "umu" && filepath.Base(filepath.Dir(cleanPath)) == "Games" {
		return true
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
	canonSeat := SeatIDFromProfileID(seatID)
	if canonSeat == "p1" && cfg != nil && cfg.SteamUserdata != "" && strings.Contains(cfg.SteamUserdata, "drive_c") {
		return util.ExpandPath(cfg.SteamUserdata)
	}

	home := ResolveSeatHome(canonSeat)
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

// EnsurePrefix resolves the effective prefix for Player 1, creating an isolated per-game Wine prefix
// and symlinking its user profile (drive_c/users/steamuser) to the central Syncthing user data directory.
func EnsurePrefix(ctx context.Context, cfg *config.Config, game *domain.Game, customPrefix string) (string, error) {
	return EnsureSeatPrefix(ctx, cfg, game, "p1", customPrefix)
}

// EnsureSeatPrefix resolves the effective prefix for a player seat ("p1", "p2", etc.), creating an isolated
// per-game Wine prefix and symlinking its user profile (drive_c/users/steamuser) to the central user data directory.
func EnsureSeatPrefix(ctx context.Context, cfg *config.Config, game *domain.Game, seatID string, customPrefix string) (string, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	canonSeat := SeatIDFromProfileID(seatID)

	var prefixDir string
	if customPrefix != "" {
		expanded := util.CanonicalPath(util.ExpandPath(customPrefix))
		if IsKnownPrefixDir(expanded) {
			prefixID := ResolvePrefixIdentifier(game)
			prefixDir = filepath.Join(expanded, prefixID)
		} else {
			prefixDir = expanded
		}
	} else if canonSeat == "p1" && cfg.WinePrefix != "" {
		prefixDir = util.CanonicalPath(util.ExpandPath(cfg.WinePrefix))
	} else {
		prefixID := ResolvePrefixIdentifier(game)
		prefixBase := DefaultPrefixesDirForSeat(canonSeat)
		prefixDir = filepath.Join(prefixBase, prefixID)
	}

	usersDir := filepath.Join(prefixDir, "drive_c", "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return "", fmt.Errorf("create prefix users directory %s: %w", usersDir, err)
	}

	// Ensure central user data directory exists (e.g. ~/Games/umu/umu-default/drive_c/users/steamuser or ~/Games/umu-p2/...)
	centralUserDir := DefaultCentralUserDataDirForSeat(cfg, canonSeat)
	if err := os.MkdirAll(centralUserDir, 0755); err != nil {
		slog.Warn("Failed to create central user data directory", "path", centralUserDir, "error", err)
	}
	centralRoaming := filepath.Join(centralUserDir, "AppData", "Roaming")
	if err := os.MkdirAll(centralRoaming, 0755); err != nil {
		slog.Debug("Failed to pre-create central user AppData/Roaming", "path", centralRoaming, "error", err)
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

	// Ensure Goldberg GSE Saves symlinks to the profile's isolated GSE directory
	seatHome := ResolveSeatHome(canonSeat)
	gseDir := filepath.Join(seatHome, ".local", "share", "GSE Saves")
	_ = EnsurePrefixGSESymlink(prefixDir, gseDir)
	_ = EnsurePrefixGSESymlink(filepath.Join(DefaultPrefixesDirForSeat(canonSeat), "umu-default"), gseDir)

	canonicalPrefix := util.CanonicalPath(prefixDir)
	opMode := domain.ModeLoader
	if game != nil && game.EffectiveOperatingMode() != "" {
		opMode = domain.NormalizeOperatingMode(game.EffectiveOperatingMode())
	} else if cfg != nil && cfg.OperatingMode != "" {
		opMode = domain.NormalizeOperatingMode(cfg.OperatingMode)
	}

	if opMode == domain.ModeLoader && game != nil {
		if ShouldEmulate(game) {
			gbeBase := cfg.GbeDir
			if gbeBase == "" {
				gbeBase = config.DefaultConfig().GbeDir
			}
			settingsDir := filepath.Join(game.GameDir, "steam_settings")
			_ = gbe.SetupWinePrefixSteamDir(canonicalPrefix, gbeBase, settingsDir, game.AppID, false)
		} else {
			_ = ScrubPrefixEmulation(canonicalPrefix)
		}
	} else if game != nil && (opMode == domain.ModeNative || opMode == domain.ModeNone || opMode == domain.ModeStandalone) {
		_ = ScrubPrefixEmulation(canonicalPrefix)
	}

	return canonicalPrefix, nil
}

// ResolveGamePrefix returns the effective Wine/Proton prefix path for Player 1 without mutating the filesystem.
func ResolveGamePrefix(cfg *config.Config, game *domain.Game, customPrefix string) string {
	return ResolveSeatGamePrefix(cfg, game, "p1", customPrefix)
}

// ResolveSeatGamePrefix returns the effective Wine/Proton prefix path for a seat without mutating the filesystem.
func ResolveSeatGamePrefix(cfg *config.Config, game *domain.Game, seatID string, customPrefix string) string {
	canonSeat := SeatIDFromProfileID(seatID)
	if customPrefix != "" {
		expanded := util.CanonicalPath(util.ExpandPath(customPrefix))
		if IsKnownPrefixDir(expanded) {
			prefixID := ResolvePrefixIdentifier(game)
			return filepath.Join(expanded, prefixID)
		}
		return expanded
	}
	if canonSeat == "p1" && cfg != nil && cfg.WinePrefix != "" {
		return util.CanonicalPath(util.ExpandPath(cfg.WinePrefix))
	}

	prefixID := ResolvePrefixIdentifier(game)
	prefixBase := DefaultPrefixesDirForSeat(canonSeat)
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

// EnsurePrefixGSESymlink ensures that the GSE Saves directory inside a Wine prefix
// (e.g. <prefix>/drive_c/users/<user>/AppData/Roaming/GSE Saves) is a symlink pointing
// directly to targetGSEDir, ensuring a single source of truth for Goldberg configurations and saves.
func EnsurePrefixGSESymlink(prefixDir, targetGSEDir string) error {
	if prefixDir == "" || targetGSEDir == "" {
		return nil
	}

	// Candidates for AppData/Roaming within this prefix or its umu-default subfolder
	candidates := []string{
		filepath.Join(prefixDir, "drive_c", "users"),
		filepath.Join(prefixDir, "umu-default", "drive_c", "users"),
	}

	var roamingDir string
	for _, usersDir := range candidates {
		if util.FileExists(usersDir) {
			wineUser := DetectWineUser(filepath.Dir(filepath.Dir(usersDir)))
			targetRoaming := filepath.Join(usersDir, wineUser, "AppData", "Roaming")
			if util.FileExists(targetRoaming) {
				roamingDir = targetRoaming
				break
			}
		}
	}

	if roamingDir == "" {
		return nil
	}

	linkPath := filepath.Join(roamingDir, "GSE Saves")
	fi, err := os.Lstat(linkPath)
	if err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(linkPath)
			if err == nil && target == targetGSEDir {
				return nil // Already points to correct target
			}
			_ = os.Remove(linkPath)
		} else {
			// Existing directory: remove stale directory to replace with symlink
			_ = os.RemoveAll(linkPath)
		}
	}

	slog.Info("Symlinking prefix GSE Saves to profile directory", "prefix", prefixDir, "target", targetGSEDir)
	return os.Symlink(targetGSEDir, linkPath)
}

// ShouldEmulate evaluates whether Goldberg Steam Emulator should be injected or active for a game.
// It returns true if ForceEmulation is explicitly enabled, or if the game has a valid Steam AppID
// and is not marked as unlinked/standalone without a Steam AppID.
func ShouldEmulate(game *domain.Game) bool {
	if game == nil {
		return false
	}
	if game.LaunchProfile != nil && game.LaunchProfile.ForceEmulation != nil {
		return *game.LaunchProfile.ForceEmulation
	}
	// Title lacks a valid Steam AppID or is unlinked standalone
	if game.AppID == "" || game.AppID == "0" {
		return false
	}
	if game.NumericAppID == 0 {
		return false
	}
	return game.IsSteam()
}

// ScrubPrefixEmulation idempotently removes Goldberg emulation artifacts from a Wine prefix:
//   - Removes Goldberg DLLs, steam_settings symlinks, and steam_appid.txt from
//     <prefix>/drive_c/Program Files (x86)/Steam
//   - Leaves genuine user files untouched.
func ScrubPrefixEmulation(prefixDir string) error {
	if prefixDir == "" {
		return nil
	}
	steamDir := filepath.Join(prefixDir, "drive_c", "Program Files (x86)", "Steam")
	if !util.DirExists(steamDir) {
		return nil
	}

	targets := []string{
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
		"steam_settings",
		"steam_appid.txt",
	}

	for _, name := range targets {
		p := filepath.Join(steamDir, name)
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		// If it's a symlink or regular file, remove it
		if fi.Mode()&os.ModeSymlink != 0 || fi.Mode().IsRegular() {
			_ = os.Remove(p)
		} else if fi.IsDir() && name == "steam_settings" {
			_ = os.RemoveAll(p)
		}
	}

	return nil
}
