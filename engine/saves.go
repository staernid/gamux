package engine

import (
	"context"
	"fmt"

	"github.com/staernid/gamux/saves"
)

// InspectSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (eng *Engine) InspectSaveLocations(ctx context.Context, gameDir string) (*saves.SaveLocationsResult, error) {
	if eng.Saves == nil {
		return nil, fmt.Errorf("saves service not initialized")
	}
	return eng.Saves.InspectLocations(ctx, gameDir)
}

// CreateSaveBackup creates a timestamped .zip backup snapshot of all save and config files for a game.
func (eng *Engine) CreateSaveBackup(ctx context.Context, gameDir, note, createdBy string) (*saves.BackupInfo, error) {
	if eng.Saves == nil {
		return nil, fmt.Errorf("saves service not initialized")
	}
	return eng.Saves.CreateBackup(ctx, gameDir, note, createdBy)
}

// ListSaveBackups returns all available backup snapshots for a game.
func (eng *Engine) ListSaveBackups(ctx context.Context, gameDir string) ([]saves.BackupInfo, error) {
	if eng.Saves == nil {
		return []saves.BackupInfo{}, nil
	}
	return eng.Saves.ListBackups(ctx, gameDir)
}

// RestoreSaveBackup restores a backup snapshot archive to the game's active save locations.
func (eng *Engine) RestoreSaveBackup(ctx context.Context, gameDir, backupID string) error {
	if eng.Saves == nil {
		return fmt.Errorf("saves service not initialized")
	}
	return eng.Saves.RestoreBackup(ctx, gameDir, backupID)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (eng *Engine) DeleteSaveBackup(ctx context.Context, gameDir, backupID string) error {
	if eng.Saves == nil {
		return fmt.Errorf("saves service not initialized")
	}
	return eng.Saves.DeleteBackup(ctx, gameDir, backupID)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (eng *Engine) UpdateSaveManifest(ctx context.Context) error {
	if eng.Saves == nil {
		return fmt.Errorf("saves service not initialized")
	}
	return eng.Saves.UpdateManifest(ctx)
}
