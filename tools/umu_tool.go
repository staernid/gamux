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

// UMUTool is the tool adapter for Open-Wine-Components umu-launcher.
type UMUTool struct {
	Config *config.Config
}

// NewUMUTool creates a new umu-launcher tool adapter.
func NewUMUTool(cfg *config.Config) *UMUTool {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &UMUTool{Config: cfg}
}

// Name returns the human-readable name of the tool.
func (u *UMUTool) Name() string {
	return "umu-launcher"
}

// Key returns the unique key identifier for the tool.
func (u *UMUTool) Key() string {
	return "umu"
}

// Status queries installed version vs latest GitHub release for umu-launcher.
func (u *UMUTool) Status(ctx context.Context) (*ToolStatus, error) {
	info, err := github.FetchUMURelease(ctx, u.Config)
	if err != nil {
		return nil, fmt.Errorf("fetch umu release status: %w", err)
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

// EnsureAvailable ensures that umu-run exists locally, downloading it from GitHub if missing.
func (u *UMUTool) EnsureAvailable(ctx context.Context) (string, error) {
	home, _ := os.UserHomeDir()
	umuRun := filepath.Join(home, ".local", "share", "umu", "umu-run")
	if util.FileExists(umuRun) {
		return umuRun, nil
	}

	if err := u.Update(ctx); err != nil {
		return "", fmt.Errorf("ensure umu available: %w", err)
	}
	return umuRun, nil
}

// Update downloads the latest standalone umu-run release asset from GitHub.
func (u *UMUTool) Update(ctx context.Context) error {
	return github.UpdateUMU(ctx, u.Config)
}
