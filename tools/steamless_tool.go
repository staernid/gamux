package tools

import (
	"context"
	"fmt"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/steamless"
	"github.com/staernid/gamux/util"
)

// SteamlessTool is the tool adapter for Steamless-rs SteamStub DRM unpacker.
type SteamlessTool struct {
	Config *config.Config
}

// NewSteamlessTool creates a new Steamless tool adapter.
func NewSteamlessTool(cfg *config.Config) *SteamlessTool {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &SteamlessTool{Config: cfg}
}

// Name returns the descriptive name of the tool.
func (s *SteamlessTool) Name() string {
	return "steamless-rs"
}

// Key returns the unique key identifier for the tool.
func (s *SteamlessTool) Key() string {
	return "steamless"
}

// Status queries installed version vs latest GitHub release for steamless-rs.
func (s *SteamlessTool) Status(ctx context.Context) (*ToolStatus, error) {
	info, err := github.FetchSteamlessRelease(ctx, s.Config)
	if err != nil {
		return nil, fmt.Errorf("fetch steamless release status: %w", err)
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

// EnsureAvailable ensures that the steamless binary is available either via embedded extraction or system PATH.
func (s *SteamlessTool) EnsureAvailable(ctx context.Context) (string, error) {
	return steamless.EnsureBinary(ctx, s.Config)
}

// Update downloads the latest steamless release assets from GitHub into the configured steamless directory.
func (s *SteamlessTool) Update(ctx context.Context) error {
	destDir := util.ExpandPath(s.Config.SteamlessDir)
	return github.UpdateSteamlessAssets(ctx, s.Config, destDir)
}
