package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/util"
)

// Proton flavor keys
const (
	FlavorGE      = "proton-ge"
	FlavorCachyOS = "proton-cachyos"
)

// ProtonTool is the tool adapter for managing external Proton distributions (GE-Proton, Proton-CachyOS).
type ProtonTool struct {
	Config    *config.Config
	FlavorKey string
}

// NewProtonTool creates a new Proton tool adapter for the specified flavor.
func NewProtonTool(cfg *config.Config, flavorKey string) *ProtonTool {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &ProtonTool{
		Config:    cfg,
		FlavorKey: flavorKey,
	}
}

// Name returns the display name of the tool.
func (p *ProtonTool) Name() string {
	spec, err := github.GetProtonFlavor(p.FlavorKey)
	if err == nil && spec != nil {
		return spec.Name
	}
	return p.FlavorKey
}

// Key returns the unique key identifier for the tool.
func (p *ProtonTool) Key() string {
	return p.FlavorKey
}

// Status queries installed version vs latest GitHub release for this Proton flavor.
func (p *ProtonTool) Status(ctx context.Context) (*ToolStatus, error) {
	info, err := github.FetchProtonRelease(ctx, p.FlavorKey)
	if err != nil {
		return nil, fmt.Errorf("fetch %s release status: %w", p.FlavorKey, err)
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

// EnsureAvailable ensures that the Proton distribution is installed, downloading it if missing.
func (p *ProtonTool) EnsureAvailable(ctx context.Context) (string, error) {
	spec, err := github.GetProtonFlavor(p.FlavorKey)
	if err != nil {
		return "", err
	}

	compatDir := github.GetProtonCompatDir()
	protonDir := filepath.Join(compatDir, spec.DirName)
	if util.DirExists(protonDir) {
		return protonDir, nil
	}

	if err := p.Update(ctx); err != nil {
		return "", fmt.Errorf("ensure %s available: %w", p.FlavorKey, err)
	}
	return protonDir, nil
}

// Update downloads and installs the latest Proton distribution from GitHub.
func (p *ProtonTool) Update(ctx context.Context) error {
	return github.UpdateProton(ctx, p.FlavorKey)
}
