package gui

import (
	"fmt"

	"github.com/staernid/gamux/saves"
	"github.com/staernid/gamux/util"
)

// -----------------------------------------------------------------------------
// Save Game & Configuration Data
// -----------------------------------------------------------------------------

// GetGameSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (a *App) GetGameSaveLocations(gamePath string) (*saves.SaveLocationsResult, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if a.Engine == nil {
		return nil, fmt.Errorf("save service not initialized")
	}
	return a.Engine.InspectSaveLocations(a.getContext(), gamePath)
}

// GetGameBackups lists available backup snapshots for a game.
func (a *App) GetGameBackups(gamePath string) ([]saves.BackupInfo, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return []saves.BackupInfo{}, nil
	}
	if a.Engine == nil {
		return []saves.BackupInfo{}, nil
	}
	return a.Engine.ListSaveBackups(a.getContext(), gamePath)
}

// CreateSaveBackup creates a new timestamped snapshot archive for a game.
func (a *App) CreateSaveBackup(gamePath, note string) (*saves.BackupInfo, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if a.Engine == nil {
		return nil, fmt.Errorf("save service not initialized")
	}
	return a.Engine.CreateSaveBackup(a.getContext(), gamePath, note, "manual")
}

// RestoreSaveBackup restores a snapshot archive to the game's active save locations.
func (a *App) RestoreSaveBackup(gamePath, backupID string) error {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	if a.Engine == nil {
		return fmt.Errorf("save service not initialized")
	}
	return a.Engine.RestoreSaveBackup(a.getContext(), gamePath, backupID)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (a *App) DeleteSaveBackup(gamePath, backupID string) error {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	if a.Engine == nil {
		return fmt.Errorf("save service not initialized")
	}
	return a.Engine.DeleteSaveBackup(a.getContext(), gamePath, backupID)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (a *App) UpdateSaveManifest() error {
	if a.Engine == nil {
		return fmt.Errorf("save service not initialized")
	}
	return a.Engine.UpdateSaveManifest(a.getContext())
}
