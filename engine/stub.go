package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// CreateGameStub creates a skeleton game directory with launch_profile.json
// in the target library folder without downloading binaries, allowing uninstalled
// catalog games to be tracked in the library with branch preferences.
func (eng *Engine) CreateGameStub(ctx context.Context, libraryPath, appID, name, storeName, branch string) (*domain.Game, error) {
	ctx = eng.WrapContext(ctx)

	libCanonical := util.CanonicalPath(libraryPath)
	if libCanonical == "" {
		return nil, fmt.Errorf("library path cannot be empty")
	}

	cleanName := strings.TrimSpace(name)
	var folderName string
	storeLower := strings.ToLower(strings.TrimSpace(storeName))
	if storeLower == "" {
		storeLower = "steam"
	}

	if cleanName == "" || util.IsPlaceholderTitle(cleanName, appID) {
		// Attempt to resolve official title from cache or remote store
		if storeLower == "steam" && appID != "" && appID != "0" {
			if cachedName, ok := cache.GetAppName(appID); ok && cachedName != "" && !util.IsPlaceholderTitle(cachedName, appID) {
				cleanName = cachedName
			} else if rName, _, rInstallDir, err := steam.FetchAppInfo(ctx, appID); err == nil {
				if rName != "" && !util.IsPlaceholderTitle(rName, appID) {
					cleanName = rName
				}
				if rInstallDir != "" {
					folderName = rInstallDir
				}
			}
		}
	}

	if cleanName == "" {
		cleanName = fmt.Sprintf("Game %s", appID)
	}

	if folderName == "" {
		folderName = util.SanitizeInstallDir(cleanName)
	}
	if folderName == "" {
		folderName = cleanName
	}

	gameDir := filepath.Join(libCanonical, folderName)
	settingsDir := filepath.Join(gameDir, "steam_settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return nil, fmt.Errorf("create stub directory: %w", err)
	}

	cleanBranch := strings.TrimSpace(branch)
	if cleanBranch == "" {
		cleanBranch = "public"
	}

	profile := &domain.LaunchProfile{
		GameDir:          gameDir,
		AppID:            strings.TrimSpace(appID),
		StoreAppID:       strings.TrimSpace(appID),
		Branch:           cleanBranch,
		Name:             cleanName,
		Store:            storeLower,
		MetadataProvider: storeLower,
		MetadataID:       strings.TrimSpace(appID),
	}

	if err := compat.SaveProfileDirect(profile); err != nil {
		return nil, fmt.Errorf("save stub profile: %w", err)
	}

	return eng.InspectStatus(ctx, gameDir)
}
