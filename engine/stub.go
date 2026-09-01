package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

// CreateGameStub creates a skeleton game directory with launch_profile.json
// in the target library folder without downloading binaries, allowing uninstalled
// catalog games to be tracked in the library.
func (eng *Engine) CreateGameStub(ctx context.Context, libraryPath, appID, name, storeName string) (*domain.Game, error) {
	ctx = eng.WrapContext(ctx)

	libCanonical := util.CanonicalPath(libraryPath)
	if libCanonical == "" {
		return nil, fmt.Errorf("library path cannot be empty")
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = fmt.Sprintf("Game %s", appID)
	}

	folderName := util.SanitizeInstallDir(cleanName)
	if folderName == "" {
		folderName = cleanName
	}

	gameDir := filepath.Join(libCanonical, folderName)
	settingsDir := filepath.Join(gameDir, "steam_settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return nil, fmt.Errorf("create stub directory: %w", err)
	}

	storeLower := strings.ToLower(strings.TrimSpace(storeName))
	if storeLower == "" {
		storeLower = "steam"
	}

	profile := &domain.LaunchProfile{
		GameDir:          gameDir,
		AppID:            strings.TrimSpace(appID),
		Name:             cleanName,
		Store:            storeLower,
		MetadataProvider: storeLower,
		MetadataID:       strings.TrimSpace(appID),
	}

	if err := compat.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("save stub profile: %w", err)
	}

	return eng.InspectStatus(ctx, gameDir)
}
