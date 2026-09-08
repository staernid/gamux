package compat

import (
	"context"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
)

// ModeParams encapsulates context and arguments required for mode strategy execution.
type ModeParams struct {
	Config        *config.Config
	GameDir       string
	Platform      string
	AppID         string
	Language      string
	WinePrefix    string
	ExePath       string
	CandidateExes []string
	DryRun        bool
}

// ModeStrategy represents a polymorphic operating mode strategy for applying and reverting DRM/emulation layers.
type ModeStrategy interface {
	Mode() string
	Apply(ctx context.Context, p ModeParams, profile *domain.LaunchProfile) error
	Revert(ctx context.Context, gameDir string, dryRun bool) (int, error)
}

// LoaderModeStrategy manages Goldberg emulator execution in loader mode (official binaries kept 100% untouched).
type LoaderModeStrategy struct{}

// Mode returns canonical mode identifier.
func (s *LoaderModeStrategy) Mode() string {
	return domain.ModeLoader
}

// Apply configures Goldberg Steam Emulator in loader mode.
func (s *LoaderModeStrategy) Apply(ctx context.Context, p ModeParams, profile *domain.LaunchProfile) error {
	return gbe.Apply(ctx, gbe.Options{
		Config:        p.Config,
		TargetDir:     p.GameDir,
		Platform:      p.Platform,
		AppID:         p.AppID,
		Language:      p.Language,
		Portable:      false,
		WinePrefix:    p.WinePrefix,
		ExePath:       p.ExePath,
		CandidateExes: p.CandidateExes,
		DryRun:        p.DryRun,
	})
}

// Revert removes emulator artifacts.
func (s *LoaderModeStrategy) Revert(ctx context.Context, gameDir string, dryRun bool) (int, error) {
	return gbe.Revert(gameDir, dryRun)
}

// PortableModeStrategy manages Goldberg emulator execution in portable mode (in-place library replacement with .ORIGINAL backups).
type PortableModeStrategy struct{}

// Mode returns canonical mode identifier.
func (s *PortableModeStrategy) Mode() string {
	return domain.ModePortable
}

// Apply injects Goldberg emulator libraries directly into the game tree.
func (s *PortableModeStrategy) Apply(ctx context.Context, p ModeParams, profile *domain.LaunchProfile) error {
	return gbe.Apply(ctx, gbe.Options{
		Config:        p.Config,
		TargetDir:     p.GameDir,
		Platform:      p.Platform,
		AppID:         p.AppID,
		Language:      p.Language,
		Portable:      true,
		WinePrefix:    p.WinePrefix,
		ExePath:       p.ExePath,
		CandidateExes: p.CandidateExes,
		DryRun:        p.DryRun,
	})
}

// Revert restores backed up original libraries and removes emulator artifacts.
func (s *PortableModeStrategy) Revert(ctx context.Context, gameDir string, dryRun bool) (int, error) {
	return gbe.Revert(gameDir, dryRun)
}

// StandaloneModeStrategy handles games running cleanly without emulation or DRM layers.
type StandaloneModeStrategy struct {
	mode string
}

// Mode returns canonical mode identifier.
func (s *StandaloneModeStrategy) Mode() string {
	if s.mode != "" {
		return s.mode
	}
	return domain.ModeNone
}

// Apply reverts emulator artifacts and saves the unmodified launch profile.
func (s *StandaloneModeStrategy) Apply(ctx context.Context, p ModeParams, profile *domain.LaunchProfile) error {
	if _, err := gbe.Revert(p.GameDir, p.DryRun); err != nil {
		return err
	}
	if profile != nil && !p.DryRun {
		_ = SaveProfileDirect(profile)
	}
	return nil
}

// Revert cleans up any remaining emulator artifacts.
func (s *StandaloneModeStrategy) Revert(ctx context.Context, gameDir string, dryRun bool) (int, error) {
	return gbe.Revert(gameDir, dryRun)
}

// ResolveStrategy returns the polymorphic ModeStrategy for a given operating mode.
func ResolveStrategy(mode string) ModeStrategy {
	normalized := domain.NormalizeOperatingMode(mode)
	switch normalized {
	case domain.ModePortable:
		return &PortableModeStrategy{}
	case domain.ModeLoader:
		return &LoaderModeStrategy{}
	case domain.ModeNone, domain.ModeStandalone, domain.ModeNative:
		return &StandaloneModeStrategy{mode: normalized}
	default:
		return &LoaderModeStrategy{}
	}
}
