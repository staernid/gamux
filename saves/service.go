package saves

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/util"
)

// ServiceConfig encapsulates configuration parameters for the save management service.
type ServiceConfig struct {
	CacheDir         string
	BackupDir        string
	RetentionCount   int
	ManifestURL      string
	HTTPClient       *http.Client
	GlobalPrefix     string
	DisableAutoFetch bool
}

// Service is the unified deep module for save location discovery,
// snapshot archive creation, rolling retention management, and atomic restore operations.
type Service struct {
	cfg         ServiceConfig
	manifestMgr *ManifestManager
	resolver    *Resolver
	backupMgr   *BackupManager
}

// NewService creates an initialized Service and begins non-blocking background manifest caching if enabled.
func NewService(cfg ServiceConfig) *Service {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	if cfg.BackupDir == "" {
		home, _ := os.UserHomeDir()
		cfg.BackupDir = filepath.Join(home, ".local", "share", "gamux", "backups")
	}
	retention := 5
	if cfg.RetentionCount > 0 {
		retention = cfg.RetentionCount
	}

	manifestMgr := NewManifestManager(cfg.CacheDir, cfg.ManifestURL, cfg.HTTPClient)
	resolver := NewResolver(manifestMgr)
	backupMgr := NewBackupManager(cfg.BackupDir, retention, resolver)

	svc := &Service{
		cfg:         cfg,
		manifestMgr: manifestMgr,
		resolver:    resolver,
		backupMgr:   backupMgr,
	}

	if !cfg.DisableAutoFetch {
		go func() {
			_ = manifestMgr.LoadOrFetch(context.Background(), false)
		}()
	}

	return svc
}

// ResolveParams inspects a game directory and returns fully resolved ResolveOptions.
func (s *Service) ResolveParams(ctx context.Context, gameDir string) (ResolveOptions, error) {
	absGameDir := util.CanonicalPath(gameDir)
	if absGameDir == "" {
		return ResolveOptions{}, fmt.Errorf("game directory cannot be empty")
	}

	info, err := detector.Detect(ctx, absGameDir)
	if err != nil {
		info = domain.NewGame(filepath.Base(absGameDir), absGameDir, "", "win64")
	}

	opts := ResolveOptions{
		AppID:    info.AppID,
		GameName: info.Name,
		GameDir:  info.GameDir,
		Platform: info.Platform,
	}

	// Check if launch profile overrides wine prefix, platform, or AppID
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

	// Fallback to configured global Wine prefix
	if opts.WinePrefix == "" && s.cfg.GlobalPrefix != "" {
		opts.WinePrefix = s.cfg.GlobalPrefix
	}

	return opts, nil
}

// InspectLocations discovers on-disk save, config, and cloud storage directories for a game directory.
func (s *Service) InspectLocations(ctx context.Context, gameDir string) (*SaveLocationsResult, error) {
	opts, err := s.ResolveParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return s.InspectLocationsWithOptions(ctx, opts)
}

// InspectLocationsWithOptions discovers on-disk save directories for explicitly provided options.
func (s *Service) InspectLocationsWithOptions(ctx context.Context, opts ResolveOptions) (*SaveLocationsResult, error) {
	if s.resolver == nil {
		return nil, fmt.Errorf("save resolver not initialized")
	}
	return s.resolver.Resolve(ctx, opts)
}

// CreateBackup creates a timestamped .zip snapshot archive of all save and config files for a game directory.
func (s *Service) CreateBackup(ctx context.Context, gameDir, note, createdBy string) (*BackupInfo, error) {
	opts, err := s.ResolveParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return s.CreateBackupWithOptions(ctx, opts, note, createdBy)
}

// CreateBackupWithOptions creates a timestamped .zip snapshot archive using explicitly provided options.
func (s *Service) CreateBackupWithOptions(ctx context.Context, opts ResolveOptions, note, createdBy string) (*BackupInfo, error) {
	if createdBy == "" {
		createdBy = "manual"
	}
	if s.backupMgr == nil {
		return nil, fmt.Errorf("save backup manager not initialized")
	}
	return s.backupMgr.CreateBackup(ctx, opts, note, createdBy)
}

// ListBackups returns all available backup snapshots for a game directory.
func (s *Service) ListBackups(ctx context.Context, gameDir string) ([]BackupInfo, error) {
	opts, err := s.ResolveParams(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return s.ListBackupsForGame(opts.GameName, opts.AppID)
}

// ListBackupsForGame returns all available backup snapshots for a given game name and AppID.
func (s *Service) ListBackupsForGame(gameName, appID string) ([]BackupInfo, error) {
	if s.backupMgr == nil {
		return []BackupInfo{}, nil
	}
	return s.backupMgr.ListBackups(gameName, appID)
}

// RestoreBackup restores a backup snapshot archive to the game's active save locations.
func (s *Service) RestoreBackup(ctx context.Context, gameDir, backupID string) error {
	opts, err := s.ResolveParams(ctx, gameDir)
	if err != nil {
		return err
	}

	folderName := SanitizeFolderName(opts.GameName)
	if folderName == "" || folderName == "Unknown_Game" {
		if opts.AppID != "" {
			folderName = "App_" + opts.AppID
		}
	}

	cleanBackupID := strings.TrimSuffix(backupID, ".zip")
	zipPath := filepath.Join(util.CanonicalPath(s.cfg.BackupDir), folderName, cleanBackupID+".zip")

	if !util.FileExists(zipPath) {
		return fmt.Errorf("backup archive not found: %s", zipPath)
	}

	if s.backupMgr == nil {
		return fmt.Errorf("save backup manager not initialized")
	}

	return s.backupMgr.RestoreBackup(ctx, zipPath, opts)
}

// DeleteBackup removes a snapshot archive from disk.
func (s *Service) DeleteBackup(ctx context.Context, gameDir, backupID string) error {
	opts, err := s.ResolveParams(ctx, gameDir)
	if err != nil {
		return err
	}

	folderName := SanitizeFolderName(opts.GameName)
	if folderName == "" || folderName == "Unknown_Game" {
		if opts.AppID != "" {
			folderName = "App_" + opts.AppID
		}
	}

	cleanBackupID := strings.TrimSuffix(backupID, ".zip")
	zipPath := filepath.Join(util.CanonicalPath(s.cfg.BackupDir), folderName, cleanBackupID+".zip")

	if s.backupMgr == nil {
		return fmt.Errorf("save backup manager not initialized")
	}

	return s.backupMgr.DeleteBackup(zipPath)
}

// UpdateManifest forces a re-download and re-indexing of the Ludusavi save database.
func (s *Service) UpdateManifest(ctx context.Context) error {
	if s.manifestMgr == nil {
		return fmt.Errorf("save manifest manager not initialized")
	}
	return s.manifestMgr.LoadOrFetch(ctx, true)
}
