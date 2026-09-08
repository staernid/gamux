package gui

import (
	"fmt"

	"github.com/staernid/gamux/saves"
)

// -----------------------------------------------------------------------------
// Save Game & Configuration Data
// -----------------------------------------------------------------------------

func (a *App) requireSaveService() (*saves.Service, error) {
	if a.Engine == nil || a.Engine.Saves == nil {
		return nil, fmt.Errorf("save service not initialized")
	}
	return a.Engine.Saves, nil
}

// GetGameSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (a *App) GetGameSaveLocations(gamePath string) (*saves.SaveLocationsResult, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	svc, err := a.requireSaveService()
	if err != nil {
		return nil, err
	}
	return svc.InspectLocations(a.getContext(), path)
}

// GetGameBackups lists available backup snapshots for a game.
func (a *App) GetGameBackups(gamePath string) ([]saves.BackupInfo, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return []saves.BackupInfo{}, nil
	}
	svc, err := a.requireSaveService()
	if err != nil {
		return []saves.BackupInfo{}, nil
	}
	return svc.ListBackups(a.getContext(), path)
}

// CreateSaveBackup creates a new timestamped snapshot archive for a game.
func (a *App) CreateSaveBackup(gamePath, note string) (*saves.BackupInfo, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	svc, err := a.requireSaveService()
	if err != nil {
		return nil, err
	}
	return svc.CreateBackup(a.getContext(), path, note, "manual")
}

// RestoreSaveBackup restores a snapshot archive to the game's active save locations.
func (a *App) RestoreSaveBackup(gamePath, backupID string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}
	svc, err := a.requireSaveService()
	if err != nil {
		return err
	}
	return svc.RestoreBackup(a.getContext(), path, backupID)
}

// RestoreGameBackup restores a snapshot archive to the game's active save locations.
func (a *App) RestoreGameBackup(gamePath, backupID string) error {
	return a.RestoreSaveBackup(gamePath, backupID)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (a *App) DeleteSaveBackup(gamePath, backupID string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}
	svc, err := a.requireSaveService()
	if err != nil {
		return err
	}
	return svc.DeleteBackup(a.getContext(), path, backupID)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (a *App) UpdateSaveManifest() error {
	svc, err := a.requireSaveService()
	if err != nil {
		return err
	}
	return svc.UpdateManifest(a.getContext())
}
