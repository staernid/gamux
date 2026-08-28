package runner

import (
	"context"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
)

// LoadLaunchProfile retrieves the active launch profile, delegating to compat.LoadProfile.
func LoadLaunchProfile(gameDir string) (*LaunchProfile, error) {
	return compat.LoadProfile(gameDir)
}

// GetLaunchProfile reads custom launch configurations, delegating to compat.LoadProfile.
func GetLaunchProfile(gameDir string) (*LaunchProfile, error) {
	return compat.LoadProfile(gameDir)
}

// SaveLaunchProfile writes custom launch configurations to launch_profile.json atomically.
func SaveLaunchProfile(gameDir string, opts LaunchProfile) error {
	opts.GameDir = gameDir
	return compat.SaveProfile(&opts)
}

// SaveProfile persists the profile to launch_profile.json atomically, delegating to compat.SaveProfile.
func SaveProfile(p *LaunchProfile) error {
	return compat.SaveProfile(p)
}

// ResetLaunchProfile removes custom launch configurations, delegating to compat.ResetProfile.
func ResetLaunchProfile(gameDir string) error {
	return compat.ResetProfile(gameDir)
}

// UnlinkMetadata removes custom metadata bindings, delegating to compat.UnlinkMetadata.
func UnlinkMetadata(gameDir string) error {
	return compat.UnlinkMetadata(gameDir)
}

// SyncLaunchProfile updates on-disk Goldberg settings to match the LaunchProfile, delegating to compat.SyncSettings.
func SyncLaunchProfile(ctx context.Context, p *LaunchProfile, cfg *config.Config) error {
	return compat.SyncSettings(ctx, p, cfg)
}

// ApplyCompatibility orchestrates profile persistence, setting synchronization, and DRM layer application, delegating to compat.Apply.
func ApplyCompatibility(ctx context.Context, cfg *config.Config, game *domain.Game, profile *LaunchProfile, dryRun bool) error {
	return compat.Apply(ctx, cfg, game, profile, dryRun)
}

// RevertCompatibility restores original backups and removes all Goldberg emulator artifacts, delegating to compat.Revert.
func RevertCompatibility(ctx context.Context, gameDir string, dryRun bool) (int, error) {
	return compat.Revert(ctx, gameDir, dryRun)
}
