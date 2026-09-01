package gui

import (
	"fmt"

	"github.com/staernid/gamux/saves"
)

// -----------------------------------------------------------------------------
// Save Game & Configuration Data
// -----------------------------------------------------------------------------

func (a *App) requireSaveService() error {
	if a.Engine == nil {
		return fmt.Errorf("save service not initialized")
	}
	return nil
}

// GetGameSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (a *App) GetGameSaveLocations(gamePath string) (*saves.SaveLocationsResult, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	if err := a.requireSaveService(); err != nil {
		return nil, err
	}
	return a.Engine.InspectSaveLocations(a.getContext(), path)
}

// GetGameBackups lists available backup snapshots for a game.
func (a *App) GetGameBackups(gamePath string) ([]saves.BackupInfo, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return []saves.BackupInfo{}, nil
	}
	if err := a.requireSaveService(); err != nil {
		return []saves.BackupInfo{}, nil
	}
	return a.Engine.ListSaveBackups(a.getContext(), path)
}

// CreateSaveBackup creates a new timestamped snapshot archive for a game.
func (a *App) CreateSaveBackup(gamePath, note string) (*saves.BackupInfo, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	if err := a.requireSaveService(); err != nil {
		return nil, err
	}
	return a.Engine.CreateSaveBackup(a.getContext(), path, note, "manual")
}

// RestoreSaveBackup restores a snapshot archive to the game's active save locations.
func (a *App) RestoreSaveBackup(gamePath, backupID string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}
	if err := a.requireSaveService(); err != nil {
		return err
	}
	return a.Engine.RestoreSaveBackup(a.getContext(), path, backupID)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (a *App) DeleteSaveBackup(gamePath, backupID string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}
	if err := a.requireSaveService(); err != nil {
		return err
	}
	return a.Engine.DeleteSaveBackup(a.getContext(), path, backupID)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (a *App) UpdateSaveManifest() error {
	if err := a.requireSaveService(); err != nil {
		return err
	}
	return a.Engine.UpdateSaveManifest(a.getContext())
}
