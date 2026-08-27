package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/saves"
	"github.com/staernid/gamux/util"
)

// SaveService coordinates game save and config location resolution, Ludusavi manifest updates, and backup snapshots.
type SaveService struct {
	sc          *ServiceContext
	manifestMgr *saves.ManifestManager
	resolver    *saves.Resolver
	backupMgr   *saves.BackupManager
}

// NewSaveService creates an initialized SaveService.
func NewSaveService(sc *ServiceContext) *SaveService {
	customURL := ""
	backupDir := ""
	retention := 5
	if sc != nil && sc.Config != nil {
		customURL = sc.Config.SaveManifestURL
		backupDir = sc.Config.BackupDir
		if sc.Config.BackupRetentionCount > 0 {
			retention = sc.Config.BackupRetentionCount
		}
	}

	manifestMgr := saves.NewManifestManager("", customURL, sc.HTTPClient)
	resolver := saves.NewResolver(manifestMgr)
	backupMgr := saves.NewBackupManager(backupDir, retention, resolver)

	svc := &SaveService{
		sc:          sc,
		manifestMgr: manifestMgr,
		resolver:    resolver,
		backupMgr:   backupMgr,
	}

	// Auto-backup on game process termination if configured
	if sc != nil && sc.EventBus != nil {
		sc.EventBus.SubscribeFunc(func(ev events.Event) {
			if ev.Type == events.EventGameExited && sc.Config != nil && sc.Config.AutoBackupSaves && ev.GameDir != "" {
				go func(gDir, gName, aID string) {
					_, err := svc.CreateSaveBackup(context.Background(), gDir, "Automatic post-game snapshot", "auto_exit")
					if err != nil {
						slog.Debug("Auto-backup on exit notice", "game", gName, "notice", err)
					}
				}(ev.GameDir, ev.GameName, ev.AppID)
			}
		})
	}

	// Non-blocking background manifest load on startup
	go func() {
		if err := manifestMgr.LoadOrFetch(context.Background(), false); err != nil {
			slog.Debug("Initial save manifest load completed with notice", "notice", err)
		}
	}()

	return svc
}

// resolveGameParams detects game metadata and launch profile overrides for save resolution.
func (ss *SaveService) resolveGameParams(ctx context.Context, gameDir string) (saves.ResolveOptions, error) {
	absGameDir := util.CanonicalPath(gameDir)
	if absGameDir == "" {
		return saves.ResolveOptions{}, fmt.Errorf("game directory cannot be empty")
	}

	info, err := detector.Detect(ctx, absGameDir)
	if err != nil {
		info = &detector.GameInfo{
			Name:     filepath.Base(absGameDir),
			GameDir:  absGameDir,
			Platform: "win64",
		}
	}

	opts := saves.ResolveOptions{
		AppID:    info.AppID,
		GameName: info.Name,
		GameDir:  info.GameDir,
		Platform: info.Platform,
	}

	// Check if launch profile overrides wine prefix or platform
	if profile, err := runner.LoadLaunchProfile(absGameDir); err == nil && profile != nil {
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
	if opts.WinePrefix == "" && ss.sc != nil && ss.sc.Config != nil {
		opts.WinePrefix = ss.sc.Config.WinePrefix
	}

	return opts, nil
}

// InspectSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (ss *SaveService) InspectSaveLocations(ctx context.Context, gameDir string) (*saves.SaveLocationsResult, error) {
	opts, err := ss.resolveGameParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}

	res, err := ss.resolver.Resolve(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("resolve save locations: %w", err)
	}

	return res, nil
}

// CreateSaveBackup creates a timestamped .zip backup snapshot of all save and config files for a game.
func (ss *SaveService) CreateSaveBackup(ctx context.Context, gameDir, note, createdBy string) (*saves.BackupInfo, error) {
	opts, err := ss.resolveGameParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}

	if createdBy == "" {
		createdBy = "manual"
	}

	info, err := ss.backupMgr.CreateBackup(ctx, opts, note, createdBy)
	if err != nil {
		return nil, fmt.Errorf("create save backup: %w", err)
	}

	return info, nil
}

// ListSaveBackups returns all available backup snapshots for a game.
func (ss *SaveService) ListSaveBackups(ctx context.Context, gameDir string) ([]saves.BackupInfo, error) {
	opts, err := ss.resolveGameParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}

	return ss.backupMgr.ListBackups(opts.GameName, opts.AppID)
}

// RestoreSaveBackup restores a backup snapshot archive to the game's active save locations.
func (ss *SaveService) RestoreSaveBackup(ctx context.Context, gameDir, backupID string) error {
	opts, err := ss.resolveGameParams(ctx, gameDir)
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
	if ss.sc != nil && ss.sc.Config != nil {
		backupDir = ss.sc.Config.BackupDir
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

	return ss.backupMgr.RestoreBackup(ctx, zipPath, opts)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (ss *SaveService) DeleteSaveBackup(ctx context.Context, gameDir, backupID string) error {
	opts, err := ss.resolveGameParams(ctx, gameDir)
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
	if ss.sc != nil && ss.sc.Config != nil {
		backupDir = ss.sc.Config.BackupDir
	}
	if backupDir == "" {
		home, _ := os.UserHomeDir()
		backupDir = filepath.Join(home, ".local", "share", "gamux", "backups")
	}

	cleanBackupID := strings.TrimSuffix(backupID, ".zip")
	zipPath := filepath.Join(util.CanonicalPath(backupDir), folderName, cleanBackupID+".zip")

	return ss.backupMgr.DeleteBackup(zipPath)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (ss *SaveService) UpdateSaveManifest(ctx context.Context) error {
	return ss.manifestMgr.LoadOrFetch(ctx, true)
}
