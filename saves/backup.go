package saves

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/staernid/gamux/util"
)

// BackupManager orchestrates save snapshot creation, portable cross-device restoration, and rolling retention.
type BackupManager struct {
	backupBaseDir  string
	retentionCount int
	resolver       *Resolver
}

// NewBackupManager creates an initialized BackupManager.
func NewBackupManager(backupBaseDir string, retentionCount int, resolver *Resolver) *BackupManager {
	if backupBaseDir == "" {
		home, _ := os.UserHomeDir()
		backupBaseDir = filepath.Join(home, ".local", "share", "gamux", "backups")
	}
	if retentionCount <= 0 {
		retentionCount = 5
	}
	return &BackupManager{
		backupBaseDir:  util.CanonicalPath(backupBaseDir),
		retentionCount: retentionCount,
		resolver:       resolver,
	}
}

// SanitizeFolderName generates a clean filesystem directory name for a game.
func SanitizeFolderName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "Unknown_Game"
	}
	clean := util.SanitizeInstallDir(s)
	if clean == "" {
		return "Game"
	}
	return clean
}

// CreateBackup archives all active save, config, and cloud files for a game into a timestamped portable .zip archive.
func (bm *BackupManager) CreateBackup(ctx context.Context, opts ResolveOptions, note, createdBy string) (*BackupInfo, error) {
	if bm.resolver == nil {
		return nil, fmt.Errorf("save resolver not initialized")
	}

	locationsResult, err := bm.resolver.Resolve(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("resolve save locations for backup: %w", err)
	}

	var existingLocations []SaveLocation
	for _, loc := range locationsResult.Locations {
		if loc.Exists && loc.SizeBytes > 0 {
			existingLocations = append(existingLocations, loc)
		}
	}

	if len(existingLocations) == 0 {
		return nil, fmt.Errorf("no existing save or config files found on disk to backup for %s", opts.GameName)
	}

	folderName := SanitizeFolderName(opts.GameName)
	if folderName == "" || folderName == "Unknown_Game" {
		if opts.AppID != "" {
			folderName = "App_" + opts.AppID
		}
	}

	gameBackupDir := filepath.Join(bm.backupBaseDir, folderName)
	if err := os.MkdirAll(gameBackupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	timestamp := time.Now()
	backupID := timestamp.Format("2006-01-02_15-04-05")
	zipFileName := fmt.Sprintf("%s.zip", backupID)
	zipFilePath := filepath.Join(gameBackupDir, zipFileName)
	tmpZipPath := zipFilePath + ".tmp"

	zipFile, err := os.Create(tmpZipPath)
	if err != nil {
		return nil, fmt.Errorf("create backup zip file: %w", err)
	}
	defer func() {
		_ = zipFile.Close()
		_ = os.Remove(tmpZipPath)
	}()

	zipWriter := zip.NewWriter(zipFile)

	meta := BackupMetadata{
		ID:             backupID,
		AppID:          opts.AppID,
		GameTitle:      opts.GameName,
		GameDir:        opts.GameDir,
		Platform:       opts.Platform,
		CreatedAt:      timestamp.Unix(),
		CreatedBy:      createdBy,
		Note:           note,
		Files:          make([]BackupFileEntry, 0),
		TotalSizeBytes: 0,
	}

	for _, loc := range existingLocations {
		locPath := loc.Path
		locInfo, err := os.Stat(locPath)
		if err != nil {
			continue
		}

		semanticPrefix := deriveSemanticPrefix(locPath, opts.GameDir, opts.AppID, opts.Platform)

		if !locInfo.IsDir() {
			relZipPath := filepath.Join("data", filepath.Base(locPath))
			if entry, err := addFileToZip(zipWriter, locPath, relZipPath, semanticPrefix); err == nil {
				meta.Files = append(meta.Files, *entry)
				meta.TotalSizeBytes += entry.SizeBytes
			}
		} else {
			_ = filepath.Walk(locPath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info == nil || info.IsDir() {
					return nil
				}
				relSub, err := filepath.Rel(locPath, path)
				if err != nil {
					relSub = filepath.Base(path)
				}
				relZipPath := filepath.Join("data", filepath.Base(locPath), relSub)
				semanticPath := filepath.Join(semanticPrefix, relSub)

				if entry, err := addFileToZip(zipWriter, path, relZipPath, semanticPath); err == nil {
					meta.Files = append(meta.Files, *entry)
					meta.TotalSizeBytes += entry.SizeBytes
				}
				return nil
			})
		}
	}

	if len(meta.Files) == 0 {
		return nil, fmt.Errorf("no files could be read into backup archive")
	}

	// Write backup_meta.json inside the root of the archive
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal backup metadata: %w", err)
	}

	metaHeader := &zip.FileHeader{
		Name:     "backup_meta.json",
		Method:   zip.Deflate,
		Modified: timestamp,
	}
	metaWriter, err := zipWriter.CreateHeader(metaHeader)
	if err != nil {
		return nil, fmt.Errorf("create metadata entry in zip: %w", err)
	}
	if _, err := metaWriter.Write(metaBytes); err != nil {
		return nil, fmt.Errorf("write metadata into zip: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close backup zip: %w", err)
	}
	if err := zipFile.Close(); err != nil {
		return nil, fmt.Errorf("flush backup zip: %w", err)
	}

	if err := os.Rename(tmpZipPath, zipFilePath); err != nil {
		return nil, fmt.Errorf("finalize backup file: %w", err)
	}

	slog.Info("Created game save backup snapshot",
		"game", opts.GameName,
		"backupID", backupID,
		"files", len(meta.Files),
		"size", meta.TotalSizeBytes,
		"path", zipFilePath,
	)

	// Enforce rolling retention limit
	bm.pruneOldBackups(folderName)

	return &BackupInfo{
		ID:        backupID,
		ZipPath:   zipFilePath,
		AppID:     opts.AppID,
		GameTitle: opts.GameName,
		CreatedAt: meta.CreatedAt,
		CreatedBy: meta.CreatedBy,
		Note:      meta.Note,
		FileCount: len(meta.Files),
		SizeBytes: meta.TotalSizeBytes,
	}, nil
}

func addFileToZip(zw *zip.Writer, physicalPath, relZipPath, semanticPath string) (*BackupFileEntry, error) {
	file, err := os.Open(physicalPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	header, err := zip.FileInfoHeader(fi)
	if err != nil {
		return nil, err
	}
	header.Name = relZipPath
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	multiWriter := io.MultiWriter(writer, hasher)

	written, err := io.Copy(multiWriter, file)
	if err != nil {
		return nil, err
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	return &BackupFileEntry{
		SemanticPath:         semanticPath,
		RelativeZipPath:      relZipPath,
		OriginalPhysicalPath: physicalPath,
		SizeBytes:            written,
		Hash:                 hashStr,
		ModifiedTime:         fi.ModTime().Unix(),
	}, nil
}

func deriveSemanticPrefix(physicalPath, gameDir, appID, platform string) string {
	home, _ := os.UserHomeDir()

	// Goldberg Cloud Saves
	if strings.Contains(physicalPath, "GSE Saves") {
		return "<gbeRemote>"
	}

	// Portable game dir
	if gameDir != "" && strings.HasPrefix(physicalPath, gameDir) {
		rel, err := filepath.Rel(gameDir, physicalPath)
		if err == nil {
			return filepath.Join("<base>", rel)
		}
	}

	// Wine prefix mappings
	if strings.Contains(physicalPath, "drive_c") {
		lower := strings.ToLower(physicalPath)
		userPatternIdx := strings.Index(lower, "drive_c/users/")
		searchScope := lower
		offset := 0
		if userPatternIdx != -1 {
			afterUser := lower[userPatternIdx+len("drive_c/users/"):]
			if slashIdx := strings.Index(afterUser, "/"); slashIdx != -1 {
				offset = userPatternIdx + len("drive_c/users/") + slashIdx + 1
				searchScope = lower[offset:]
			}
		}

		rules := []struct {
			tag    string
			prefix string
		}{
			{"appdata/locallow", "<winLocalAppDataLow>"},
			{"appdata/roaming", "<winAppData>"},
			{"appdata/local", "<winLocalAppData>"},
			{"saved games", "<winSavedGames>"},
			{"documents", "<winDocuments>"},
		}

		for _, r := range rules {
			if idx := strings.Index(searchScope, r.tag); idx != -1 {
				absIdx := offset + idx + len(r.tag)
				subPath := ""
				if absIdx < len(physicalPath) {
					subPath = strings.TrimPrefix(physicalPath[absIdx:], "/")
				}
				if subPath == "" {
					return r.prefix
				}
				return filepath.Join(r.prefix, subPath)
			}
		}
	}

	// Linux native home
	if home != "" && strings.HasPrefix(physicalPath, home) {
		rel, err := filepath.Rel(home, physicalPath)
		if err == nil {
			return filepath.Join("<home>", rel)
		}
	}

	return physicalPath
}

// ListBackups returns all available snapshots for a game sorted newest first.
func (bm *BackupManager) ListBackups(gameName, appID string) ([]BackupInfo, error) {
	folderName := SanitizeFolderName(gameName)
	if folderName == "" || folderName == "Unknown_Game" {
		if appID != "" {
			folderName = "App_" + appID
		}
	}

	gameBackupDir := filepath.Join(bm.backupBaseDir, folderName)
	if !util.FileExists(gameBackupDir) {
		return []BackupInfo{}, nil
	}

	entries, err := os.ReadDir(gameBackupDir)
	if err != nil {
		return nil, fmt.Errorf("read backup dir %s: %w", gameBackupDir, err)
	}

	var results []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}

		zipPath := filepath.Join(gameBackupDir, entry.Name())
		info, err := readBackupZipInfo(zipPath)
		if err == nil && info != nil {
			results = append(results, *info)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt > results[j].CreatedAt
	})

	return results, nil
}

func readBackupZipInfo(zipPath string) (*BackupInfo, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var meta BackupMetadata
	for _, f := range r.File {
		if f.Name == "backup_meta.json" {
			rc, err := f.Open()
			if err != nil {
				break
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			if err == nil {
				_ = json.Unmarshal(data, &meta)
			}
			break
		}
	}

	fi, _ := os.Stat(zipPath)
	size := int64(0)
	if fi != nil {
		size = fi.Size()
	}

	backupID := strings.TrimSuffix(filepath.Base(zipPath), ".zip")
	if meta.ID != "" {
		backupID = meta.ID
	}

	return &BackupInfo{
		ID:        backupID,
		ZipPath:   zipPath,
		AppID:     meta.AppID,
		GameTitle: meta.GameTitle,
		CreatedAt: meta.CreatedAt,
		CreatedBy: meta.CreatedBy,
		Note:      meta.Note,
		FileCount: len(meta.Files),
		SizeBytes: size,
	}, nil
}

// RestoreBackup extracts a snapshot archive back into the active physical save paths on the current device.
func (bm *BackupManager) RestoreBackup(ctx context.Context, zipPath string, opts ResolveOptions) error {
	if !util.FileExists(zipPath) {
		return fmt.Errorf("backup archive not found: %s", zipPath)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open backup zip: %w", err)
	}
	defer r.Close()

	var meta BackupMetadata
	for _, f := range r.File {
		if f.Name == "backup_meta.json" {
			rc, err := f.Open()
			if err == nil {
				data, _ := io.ReadAll(rc)
				_ = rc.Close()
				_ = json.Unmarshal(data, &meta)
			}
			break
		}
	}

	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	home, _ := os.UserHomeDir()
	effectivePrefix := bm.resolver.resolveEffectiveWinePrefix(opts.WinePrefix, opts.AppID, opts.GameName)
	wineUser := bm.resolver.detectWineUser(effectivePrefix)
	userProfile := filepath.Join(effectivePrefix, "drive_c", "users", wineUser)

	for _, entry := range meta.Files {
		zipFile, exists := fileMap[entry.RelativeZipPath]
		if !exists {
			continue
		}

		// Re-resolve target physical path from semantic path
		targetPath := entry.OriginalPhysicalPath
		if entry.SemanticPath != "" {
			p := entry.SemanticPath
			p = strings.ReplaceAll(p, "<base>", opts.GameDir)
			p = strings.ReplaceAll(p, "<home>", home)
			p = strings.ReplaceAll(p, "<gbeRemote>", filepath.Join(home, ".local", "share", "GSE Saves", opts.AppID, "remote"))
			p = strings.ReplaceAll(p, "<winAppData>", filepath.Join(userProfile, "AppData", "Roaming"))
			p = strings.ReplaceAll(p, "<winLocalAppData>", filepath.Join(userProfile, "AppData", "Local"))
			p = strings.ReplaceAll(p, "<winLocalAppDataLow>", filepath.Join(userProfile, "AppData", "LocalLow"))
			p = strings.ReplaceAll(p, "<winDocuments>", filepath.Join(userProfile, "Documents"))
			p = strings.ReplaceAll(p, "<winSavedGames>", filepath.Join(userProfile, "Saved Games"))
			targetPath = filepath.Clean(p)
		}

		if err := extractZipEntry(zipFile, targetPath); err != nil {
			slog.Warn("Failed extracting save file during restore", "target", targetPath, "error", err)
		}
	}

	slog.Info("Successfully restored save backup snapshot", "game", opts.GameName, "zip", zipPath)
	return nil
}

func extractZipEntry(zf *zip.File, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	tmpPath := targetPath + ".restoring"
	outFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, zf.Mode())
	if err != nil {
		return err
	}

	_, err = io.Copy(outFile, rc)
	_ = outFile.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, targetPath)
}

// DeleteBackup removes a backup archive from disk.
func (bm *BackupManager) DeleteBackup(zipPath string) error {
	if !util.FileExists(zipPath) {
		return nil
	}
	return os.Remove(zipPath)
}

func (bm *BackupManager) pruneOldBackups(folderName string) {
	if bm.retentionCount <= 0 {
		return
	}

	dir := filepath.Join(bm.backupBaseDir, folderName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var zips []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".zip") {
			zips = append(zips, filepath.Join(dir, e.Name()))
		}
	}

	if len(zips) <= bm.retentionCount {
		return
	}

	// Sort oldest first
	sort.Slice(zips, func(i, j int) bool {
		fi1, _ := os.Stat(zips[i])
		fi2, _ := os.Stat(zips[j])
		if fi1 != nil && fi2 != nil {
			return fi1.ModTime().Before(fi2.ModTime())
		}
		return zips[i] < zips[j]
	})

	toDelete := len(zips) - bm.retentionCount
	for i := 0; i < toDelete; i++ {
		_ = os.Remove(zips[i])
		slog.Info("Pruned old backup snapshot exceeding retention limit", "path", zips[i])
	}
}
