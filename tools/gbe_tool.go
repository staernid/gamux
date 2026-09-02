package tools

import (
	"context"
	"fmt"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/github"
)

// GBETool is the tool adapter for the Goldberg Steam Emulator fork.
type GBETool struct {
	Config *config.Config
}

// NewGBETool creates a new Goldberg Emulator tool adapter.
func NewGBETool(cfg *config.Config) *GBETool {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &GBETool{Config: cfg}
}

// Name returns the descriptive name of the tool.
func (g *GBETool) Name() string {
	return "gbe_fork"
}

// Key returns the unique key identifier for the tool.
func (g *GBETool) Key() string {
	return "gbe"
}

// Status queries installed version vs latest GitHub release for GBE.
func (g *GBETool) Status(ctx context.Context) (*ToolStatus, error) {
	info, err := github.FetchGBERelease(ctx, g.Config)
	if err != nil {
		return nil, fmt.Errorf("fetch GBE release status: %w", err)
	}
	return &ToolStatus{
		Name:          info.Name,
		Key:           info.Key,
		InstalledPath: info.InstalledPath,
		InstalledTag:  info.InstalledTag,
		LatestTag:     info.LatestTag,
		ReleaseName:   info.ReleaseName,
		ReleaseNotes:  info.ReleaseNotes,
		ReleaseURL:    info.ReleaseURL,
		PublishedAt:   info.PublishedAt,
		HasUpdate:     info.HasUpdate,
		IsInstalled:   info.IsInstalled,
	}, nil
}

// EnsureAvailable ensures that GBE assets exist locally, updating them from GitHub if missing.
func (g *GBETool) EnsureAvailable(ctx context.Context) (string, error) {
	return gbe.EnsureAvailable(ctx, g.Config)
}

// Update downloads and extracts the latest Goldberg Emulator release assets from GitHub.
func (g *GBETool) Update(ctx context.Context) error {
	return github.UpdateGBE(ctx, g.Config)
}
