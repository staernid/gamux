package manifest

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/staernid/gamux/util"
)

// ScanUntrackedFiles walks gameDir and compares every file on disk against official manifest entries.
// It returns official file count and a slice of relative paths for untracked/mod files.
// IntegrityReport details file matching, modifications, missing files, and untracked mods.
type IntegrityReport struct {
	OfficialCount  int      `json:"official_count"`
	IntactCount    int      `json:"intact_count"`
	ModifiedFiles  []string `json:"modified_files,omitempty"`
	MissingFiles   []string `json:"missing_files,omitempty"`
	UntrackedFiles []string `json:"untracked_files,omitempty"`
	RedistModified []string `json:"redist_modified,omitempty"`
	RedistMissing  []string `json:"redist_missing,omitempty"`
}

// IsRedistFile checks if a relative path belongs to shared DirectX/VC++ runtime installers.
func IsRedistFile(relPath string) bool {
	norm := strings.ToLower(NormalizePath(relPath))
	return strings.HasPrefix(norm, "_commonredist") ||
		strings.HasPrefix(norm, "directx") ||
		strings.HasPrefix(norm, "vcredist") ||
		strings.HasPrefix(norm, "redist") ||
		strings.HasPrefix(norm, "installers")
}

// IsChunkValidOnDisk checks if the chunk at the specified offset in f matches the expected ChunkID.
func IsChunkValidOnDisk(f io.ReaderAt, chunk ChunkInfo, buf []byte) bool {
	if chunk.UncompressedSize == 0 {
		return true
	}
	if len(chunk.ChunkID) != 40 {
		return false
	}
	slice := buf
	if len(slice) < int(chunk.UncompressedSize) {
		slice = make([]byte, chunk.UncompressedSize)
	} else {
		slice = slice[:chunk.UncompressedSize]
	}
	if _, err := f.ReadAt(slice, int64(chunk.Offset)); err != nil {
		return false
	}
	sum := sha1.Sum(slice)
	return strings.EqualFold(hex.EncodeToString(sum[:]), chunk.ChunkID)
}

// VerifyFileChunksOnDisk checks if all chunks of a file on disk match their expected chunk hashes.
func VerifyFileChunksOnDisk(filePath string, chunks []ChunkInfo) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 1024*1024)
	for _, ch := range chunks {
		if !IsChunkValidOnDisk(f, ch, buf) {
			return false
		}
	}
	return true
}

// IntegrityProgress reports quantitative telemetry during depot integrity verification.
type IntegrityProgress struct {
	CurrentFile  string  `json:"current_file"`
	CheckedFiles int     `json:"checked_files"`
	TotalFiles   int     `json:"total_files"`
	CheckedBytes int64   `json:"checked_bytes"`
	TotalBytes   int64   `json:"total_bytes"`
	SpeedBps     float64 `json:"speed_bps"`
	ETASeconds   int     `json:"eta_seconds"`
	Percent      float64 `json:"percent"`
}

// IntegrityProgressFunc is a callback invoked with progress updates during integrity scanning.
type IntegrityProgressFunc func(IntegrityProgress)

// ScanDepotIntegrity compares files on disk against official manifest entries for size/SHA1/chunk mismatches.
func ScanDepotIntegrity(gameDir string, appID uint32) (*IntegrityReport, error) {
	return ScanDepotIntegrityWithPlatform(gameDir, appID, "")
}

// ScanDepotIntegrityWithPlatform compares files on disk against official manifest entries matching a target platform.
func ScanDepotIntegrityWithPlatform(gameDir string, appID uint32, platform string) (*IntegrityReport, error) {
	return ScanDepotIntegrityWithProgress(context.Background(), gameDir, appID, platform, nil)
}

// ScanDepotIntegrityWithProgress compares files on disk against official manifest entries with streaming progress telemetry.
func ScanDepotIntegrityWithProgress(ctx context.Context, gameDir string, appID uint32, platform string, onProgress IntegrityProgressFunc) (*IntegrityReport, error) {
	manifest, _ := LoadManifestFromDirWithPlatform(gameDir, appID, nil, platform)

	// Check if manifest files are missing or encrypted
	needsKeyOrManifest := manifest == nil || len(manifest.Files) == 0
	if !needsKeyOrManifest {
		for _, f := range manifest.Files {
			if util.IsEncryptedBase64Path(f.Path) {
				needsKeyOrManifest = true
				break
			}
		}
	}

	if needsKeyOrManifest {
		chain := NewDefaultProviderChain("", "")
		res, err := chain.Resolve(ctx, appID)
		if err == nil && res != nil && len(res.Depots) > 0 {
			depotKeys := make(map[uint32]string)
			for _, pair := range res.Depots {
				depotKeys[pair.DepotID] = pair.DecryptionKey
			}

			// Try loading local files with resolved keys and platform filtering
			if loaded, err := LoadManifestFromDirWithPlatform(gameDir, appID, depotKeys, platform); err == nil && loaded != nil && len(loaded.Files) > 0 {
				manifest = loaded
			} else if len(res.ManifestFiles) > 0 {
				// Parse remote/provider manifests directly with platform filtering
				var allFiles []ManifestFileEntry
				var resolvedDepotID uint32
				for filename, data := range res.ManifestFiles {
					parts := strings.Split(filename, "_")
					var fileDepotID uint32
					if len(parts) >= 1 {
						if did, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
							fileDepotID = uint32(did)
						}
					}
					hexKey := depotKeys[fileDepotID]
					files, err := ParseBinaryManifestWithKey(data, hexKey)
					if err == nil && len(files) > 0 {
						if platform != "" && platform != "all" {
							cat := ClassifyDepot(files)
							if !MatchesPlatform(cat, platform) {
								continue
							}
						}
						if resolvedDepotID == 0 {
							resolvedDepotID = fileDepotID
						}
						for idx := range files {
							files[idx].DepotID = fileDepotID
						}
						allFiles = append(allFiles, files...)
					}
				}
				if len(allFiles) > 0 {
					manifest = &Manifest{
						AppID:     appID,
						DepotID:   resolvedDepotID,
						DepotKeys: depotKeys,
						Files:     allFiles,
					}
				}
			}
		}
	}

	if manifest == nil || len(manifest.Files) == 0 {
		return nil, fmt.Errorf("no SteamPipe depot manifests or keys found for AppID %d", appID)
	}

	// Verify decrypted paths
	for _, f := range manifest.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, fmt.Errorf("depot manifest files for AppID %d could not be decrypted (missing decryption key)", appID)
		}
	}

	officialMap := make(map[string]ManifestFileEntry, len(manifest.Files))
	foundMap := make(map[string]bool, len(manifest.Files))
	officialCount := 0
	var totalBytes int64

	for _, f := range manifest.Files {
		// Ignore directory manifest nodes from official file count & verification
		if (f.Flags&0x40 != 0) || (len(f.Chunks) == 0 && f.Size == 0) {
			continue
		}
		norm := strings.ToLower(NormalizePath(f.Path))
		officialMap[norm] = f
		officialCount++
		totalBytes += int64(f.Size)
	}

	report := &IntegrityReport{
		OfficialCount: officialCount,
	}

	var checkedFiles int
	var checkedBytes int64
	startTime := time.Now()
	var lastReport time.Time

	reportProgress := func(currentFile string, force bool) {
		if onProgress == nil {
			return
		}
		now := time.Now()
		if !force && now.Sub(lastReport) < 100*time.Millisecond {
			return
		}
		lastReport = now

		elapsed := now.Sub(startTime).Seconds()
		speed := float64(0)
		eta := 0
		if elapsed > 0.5 {
			speed = float64(checkedBytes) / elapsed
			if speed > 0 && totalBytes > checkedBytes {
				eta = int(float64(totalBytes-checkedBytes) / speed)
			}
		}
		pct := float64(0)
		if totalBytes > 0 {
			pct = float64(checkedBytes) / float64(totalBytes) * 100.0
			if pct > 100 {
				pct = 100
			}
		} else if officialCount > 0 {
			pct = float64(checkedFiles) / float64(officialCount) * 100.0
			if pct > 100 {
				pct = 100
			}
		}

		onProgress(IntegrityProgress{
			CurrentFile:  currentFile,
			CheckedFiles: checkedFiles,
			TotalFiles:   officialCount,
			CheckedBytes: checkedBytes,
			TotalBytes:   totalBytes,
			SpeedBps:     speed,
			ETASeconds:   eta,
			Percent:      pct,
		})
	}

	walkErr := filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rel, relErr := filepath.Rel(gameDir, path)
		if relErr != nil {
			return nil
		}

		normRel := strings.ToLower(NormalizePath(rel))
		baseName := d.Name()

		// Skip internal gamux folders, GPU caches, and Wine log files from untracked reporting
		if strings.HasPrefix(rel, "[Manifests]") ||
			strings.HasPrefix(rel, "[Steam]") ||
			strings.HasPrefix(rel, "steam_settings") ||
			strings.HasPrefix(rel, "GLCache") ||
			strings.HasPrefix(rel, "ShaderCache") ||
			strings.HasPrefix(rel, ".dxvk-cache") ||
			strings.HasSuffix(strings.ToLower(baseName), ".log") ||
			strings.Contains(baseName, ".ORIGINAL") {
			return nil
		}

		entry, isOfficial := officialMap[normRel]
		if isOfficial {
			foundMap[normRel] = true
			if !d.IsDir() {
				checkedFiles++
				checkedBytes += int64(entry.Size)
				reportProgress(rel, false)

				if fi, err := d.Info(); err == nil {
					lowerRel := strings.ToLower(rel)
					isDrmWrapped := (strings.HasSuffix(lowerRel, "steam_api64.dll") || strings.HasSuffix(lowerRel, "steam_api.dll") || strings.HasSuffix(lowerRel, "libsteam_api.so")) && hasOriginalBackup(gameDir, rel)

					corrupted := false
					if uint64(fi.Size()) != entry.Size {
						corrupted = true
					} else if !isDrmWrapped {
						if entry.SHA1 != "" {
							localHash, err := util.GetSHA1Hash(path)
							if err != nil || !strings.EqualFold(localHash, entry.SHA1) {
								corrupted = true
							}
						} else if len(entry.Chunks) > 0 {
							if !VerifyFileChunksOnDisk(path, entry.Chunks) {
								corrupted = true
							}
						}
					}

					if corrupted {
						if isDrmWrapped {
							report.IntactCount++
						} else if IsRedistFile(rel) {
							report.RedistModified = append(report.RedistModified, rel)
						} else {
							report.ModifiedFiles = append(report.ModifiedFiles, rel)
						}
					} else {
						report.IntactCount++
					}
				} else {
					report.IntactCount++
				}
			}
		} else if !d.IsDir() {
			// Skip emulator/gamux generated files from untracked reporting
			if baseName != "steam_appid.txt" &&
				baseName != "ColdClientLoader.ini" &&
				baseName != "steamclient_loader_x64.exe" &&
				baseName != "steamclient_loader_x86.exe" &&
				baseName != "steamclient64.dll" &&
				baseName != "steamclient.dll" &&
				baseName != "GameOverlayRenderer64.dll" &&
				baseName != "GameOverlayRenderer.dll" &&
				baseName != "steam_interfaces.txt" {
				report.UntrackedFiles = append(report.UntrackedFiles, rel)
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	for normRel, entry := range officialMap {
		if !foundMap[normRel] {
			baseName := filepath.Base(entry.Path)
			if baseName == "steam_appid.txt" ||
				baseName == "steam_interfaces.txt" ||
				baseName == "ColdClientLoader.ini" ||
				baseName == "steamclient_loader_x64.exe" ||
				baseName == "steamclient_loader_x86.exe" ||
				baseName == "steamclient64.dll" ||
				baseName == "steamclient.dll" ||
				baseName == "GameOverlayRenderer64.dll" ||
				baseName == "GameOverlayRenderer.dll" {
				continue
			}
			if IsRedistFile(entry.Path) {
				report.RedistMissing = append(report.RedistMissing, entry.Path)
			} else {
				// Ignore empty directory manifest nodes or platform-mismatched files (e.g. .so/.dylib for Windows games)
				if (entry.Flags&0x40 != 0) || (len(entry.Chunks) == 0 && entry.Size == 0) {
					continue
				}
				lowerPath := strings.ToLower(entry.Path)
				if strings.HasSuffix(lowerPath, ".so") || strings.HasSuffix(lowerPath, ".dylib") || strings.Contains(lowerPath, ".app/contents/") {
					continue
				}
				report.MissingFiles = append(report.MissingFiles, entry.Path)
			}
		}
	}

	reportProgress("Completed", true)
	return report, nil
}

// ScanUntrackedFiles walks gameDir and compares every file on disk against official manifest entries.
func ScanUntrackedFiles(gameDir string, appID uint32) (int, []string, error) {
	rep, err := ScanDepotIntegrity(gameDir, appID)
	if err != nil || rep == nil {
		return 0, nil, nil
	}
	return rep.OfficialCount, rep.UntrackedFiles, nil
}

func hasOriginalBackup(gameDir, rel string) bool {
	absPath := filepath.Join(gameDir, rel)
	if matches, err := filepath.Glob(absPath + "*ORIGINAL*"); err == nil && len(matches) > 0 {
		return true
	}
	return false
}
