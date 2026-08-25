package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/util"
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
	}, nil
}

// EnsureAvailable ensures that GBE assets exist locally, updating them from GitHub if missing.
func (g *GBETool) EnsureAvailable(ctx context.Context) (string, error) {
	gbeDir := util.ExpandPath(g.Config.GbeDir)
	if _, err := os.Stat(filepath.Join(gbeDir, "linux_release")); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(gbeDir, "win_release")); os.IsNotExist(err) {
			if err := g.Update(ctx); err != nil {
				return "", fmt.Errorf("ensure GBE available: %w", err)
			}
		}
	}
	return gbeDir, nil
}

// Update downloads and extracts the latest Goldberg Emulator release assets from GitHub.
func (g *GBETool) Update(ctx context.Context) error {
	return github.UpdateGBE(ctx, g.Config)
}
