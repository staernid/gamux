package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/saves"
	"github.com/staernid/gamux/util"
)

// resolveGameParams detects game metadata and launch profile overrides for save resolution.
func (eng *Engine) resolveGameParams(ctx context.Context, gameDir string) (saves.ResolveOptions, error) {
	absGameDir := util.CanonicalPath(gameDir)
	if absGameDir == "" {
		return saves.ResolveOptions{}, fmt.Errorf("game directory cannot be empty")
	}

	info, err := detector.Detect(ctx, absGameDir)
	if err != nil {
		info = domain.NewGame(filepath.Base(absGameDir), absGameDir, "", "win64")
	}

	opts := saves.ResolveOptions{
		AppID:    info.AppID,
		GameName: info.Name,
		GameDir:  info.GameDir,
		Platform: info.Platform,
	}

	// Check if launch profile overrides wine prefix or platform
	if profile, err := compat.LoadProfile(absGameDir); err == nil && profile != nil {
		if profile.WinePrefix != "" {
			opts.WinePrefix = profile.WinePrefix
		}
		if profile.Platform != "" {
			opts.Platform = profile.Platform
		}
		if profile.AppID != "" {
			opts.AppID = profile.AppID
		}
	}

	// Global config wine prefix fallback
	if opts.WinePrefix == "" && eng.Config != nil {
		opts.WinePrefix = eng.Config.WinePrefix
	}

	return opts, nil
}

// InspectSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (eng *Engine) InspectSaveLocations(ctx context.Context, gameDir string) (*saves.SaveLocationsResult, error) {
	opts, err := eng.resolveGameParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}

	if eng.savesResolver == nil {
		return nil, fmt.Errorf("save resolver not initialized")
	}

	res, err := eng.savesResolver.Resolve(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("resolve save locations: %w", err)
	}

	return res, nil
}

// CreateSaveBackup creates a timestamped .zip backup snapshot of all save and config files for a game.
func (eng *Engine) CreateSaveBackup(ctx context.Context, gameDir, note, createdBy string) (*saves.BackupInfo, error) {
	opts, err := eng.resolveGameParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}

	if createdBy == "" {
		createdBy = "manual"
	}

	if eng.savesBackup == nil {
		return nil, fmt.Errorf("save backup manager not initialized")
	}

	info, err := eng.savesBackup.CreateBackup(ctx, opts, note, createdBy)
	if err != nil {
		return nil, fmt.Errorf("create save backup: %w", err)
	}

	return info, nil
}

// ListSaveBackups returns all available backup snapshots for a game.
func (eng *Engine) ListSaveBackups(ctx context.Context, gameDir string) ([]saves.BackupInfo, error) {
	opts, err := eng.resolveGameParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}

	if eng.savesBackup == nil {
		return []saves.BackupInfo{}, nil
	}

	return eng.savesBackup.ListBackups(opts.GameName, opts.AppID)
}

// RestoreSaveBackup restores a backup snapshot archive to the game's active save locations.
func (eng *Engine) RestoreSaveBackup(ctx context.Context, gameDir, backupID string) error {
	opts, err := eng.resolveGameParams(ctx, gameDir)
	if err != nil {
		return err
	}

	folderName := saves.SanitizeFolderName(opts.GameName)
	if folderName == "" || folderName == "Unknown_Game" {
		if opts.AppID != "" {
			folderName = "App_" + opts.AppID
		}
	}

	backupDir := ""
	if eng.Config != nil {
		backupDir = eng.Config.BackupDir
	}
	if backupDir == "" {
		home, _ := os.UserHomeDir()
		backupDir = filepath.Join(home, ".local", "share", "gamux", "backups")
	}

	cleanBackupID := strings.TrimSuffix(backupID, ".zip")
	zipPath := filepath.Join(util.CanonicalPath(backupDir), folderName, cleanBackupID+".zip")

	if !util.FileExists(zipPath) {
		return fmt.Errorf("backup archive not found: %s", zipPath)
	}

	if eng.savesBackup == nil {
		return fmt.Errorf("save backup manager not initialized")
	}

	return eng.savesBackup.RestoreBackup(ctx, zipPath, opts)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (eng *Engine) DeleteSaveBackup(ctx context.Context, gameDir, backupID string) error {
	opts, err := eng.resolveGameParams(ctx, gameDir)
	if err != nil {
		return err
	}

	folderName := saves.SanitizeFolderName(opts.GameName)
	if folderName == "" || folderName == "Unknown_Game" {
		if opts.AppID != "" {
			folderName = "App_" + opts.AppID
		}
	}

	backupDir := ""
	if eng.Config != nil {
		backupDir = eng.Config.BackupDir
	}
	if backupDir == "" {
		home, _ := os.UserHomeDir()
		backupDir = filepath.Join(home, ".local", "share", "gamux", "backups")
	}

	cleanBackupID := strings.TrimSuffix(backupID, ".zip")
	zipPath := filepath.Join(util.CanonicalPath(backupDir), folderName, cleanBackupID+".zip")

	if eng.savesBackup == nil {
		return fmt.Errorf("save backup manager not initialized")
	}

	return eng.savesBackup.DeleteBackup(zipPath)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (eng *Engine) UpdateSaveManifest(ctx context.Context) error {
	if eng.savesManifest == nil {
		return fmt.Errorf("save manifest manager not initialized")
	}
	return eng.savesManifest.LoadOrFetch(ctx, true)
}
