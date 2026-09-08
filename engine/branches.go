package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// GetSteamBranches retrieves available release and beta branches for a Steam AppID.
func (eng *Engine) GetSteamBranches(ctx context.Context, appID uint32) ([]domain.SteamBranch, error) {
	ctx = eng.WrapContext(ctx)
	if appID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID 0")
	}
	return steam.FetchAppBranches(ctx, appID)
}

// SetGameBranch updates the configured beta/branch for a game installation or stub,
// saving the preference in the launch profile and updating any local Steam ACF manifest.
func (eng *Engine) SetGameBranch(ctx context.Context, gameDir string, branch string, password string) (*domain.Game, error) {
	ctx = eng.WrapContext(ctx)

	canonicalDir := util.CanonicalPath(gameDir)
	if canonicalDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	profile, err := compat.LoadProfile(canonicalDir)
	if err != nil {
		profile = &domain.LaunchProfile{
			GameDir: canonicalDir,
		}
	}

	cleanBranch := strings.TrimSpace(branch)
	if cleanBranch == "" {
		cleanBranch = "public"
	}

	profile.Branch = cleanBranch
	profile.BranchPassword = strings.TrimSpace(password)

	if err := compat.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("save profile with branch %q: %w", cleanBranch, err)
	}

	// Update local ACF manifest if present in game directory
	appID := profile.AppID
	if appID != "" && appID != "0" {
		candidates := []string{
			filepath.Join(canonicalDir, "[Manifests]", fmt.Sprintf("appmanifest_%s.acf", appID)),
			filepath.Join(canonicalDir, "[Steam]", fmt.Sprintf("appmanifest_%s.acf", appID)),
			filepath.Join(canonicalDir, "steamapps", fmt.Sprintf("appmanifest_%s.acf", appID)),
			filepath.Join(canonicalDir, fmt.Sprintf("appmanifest_%s.acf", appID)),
		}
		for _, acfPath := range candidates {
			if util.FileExists(acfPath) {
				if acfData, pErr := detector.ParseACFFile(acfPath); pErr == nil && acfData != nil {
					acfData.BetaKey = cleanBranch
					newContent := detector.GenerateACFContent(acfData)
					if wErr := os.WriteFile(acfPath, []byte(newContent), 0644); wErr != nil {
						slog.Warn("Failed to update BetaKey in local ACF file", "path", acfPath, "error", wErr)
					} else {
						slog.Info("Updated BetaKey in local ACF file", "path", acfPath, "branch", cleanBranch)
					}
				}
			}
		}
	}

	slog.Info("Set game branch successfully", "gameDir", canonicalDir, "branch", cleanBranch)
	return eng.InspectStatus(ctx, canonicalDir)
}
