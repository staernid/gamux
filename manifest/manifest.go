package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
)


// ChunkInfo represents a single file chunk in a Steam manifest.
type ChunkInfo struct {
	ChunkID      string `json:"chunk_id"`
	Offset       uint64 `json:"offset"`
	CompressedSize uint32 `json:"compressed_size"`
	UncompressedSize uint32 `json:"uncompressed_size"`
	Checksum     string `json:"checksum"`
}

// ManifestFileEntry represents a file entry in a Steam manifest.
type ManifestFileEntry struct {
	Path        string      `json:"path"`
	Size        uint64      `json:"size"`
	Flags       uint32      `json:"flags"`
	SHA1        string      `json:"sha1"`
	DepotID     uint32      `json:"depot_id,omitempty"`
	Chunks      []ChunkInfo `json:"chunks"`
}

// Manifest represents parsed manifest file details for a Steam depot.
type Manifest struct {
	AppID         uint32              `json:"app_id"`
	DepotID       uint32              `json:"depot_id"`
	ManifestID    uint64              `json:"manifest_id"`
	CreationTime  uint32              `json:"creation_time"`
	DecryptionKey string              `json:"decryption_key,omitempty"`
	DepotKeys     map[uint32]string   `json:"depot_keys,omitempty"`
	Files         []ManifestFileEntry `json:"files"`
}

// GetDecryptionKey returns the decryption key for a specific depot ID or falls back to the default key.
func (m *Manifest) GetDecryptionKey(depotID uint32) string {
	if m.DepotKeys != nil {
		if key, ok := m.DepotKeys[depotID]; ok && key != "" {
			return key
		}
	}
	return m.DecryptionKey
}

// NormalizePath cleans and normalizes relative file paths for Linux/Windows cross-platform safety.
func NormalizePath(p string) string {
	cleaned := filepath.Clean(strings.ReplaceAll(p, "\\", "/"))
	return strings.TrimPrefix(cleaned, "/")
}

// TotalSize returns the aggregate uncompressed size of all files in the manifest.
func (m *Manifest) TotalSize() uint64 {
	var total uint64
	for _, f := range m.Files {
		total += f.Size
	}
	return total
}

// FindFile returns the file entry for a given relative path.
func (m *Manifest) FindFile(relativePath string) (*ManifestFileEntry, error) {
	norm := NormalizePath(relativePath)
	for i := range m.Files {
		if NormalizePath(m.Files[i].Path) == norm {
			return &m.Files[i], nil
		}
	}
	return nil, fmt.Errorf("file %q not found in manifest", relativePath)
}


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

// ScanDepotIntegrity compares files on disk against official manifest entries for size/SHA1 mismatches.
func ScanDepotIntegrity(gameDir string, appID uint32) (*IntegrityReport, error) {
	manifest, err := LoadManifestFromDir(gameDir, appID)
	if err != nil || manifest == nil || len(manifest.Files) == 0 {
		return nil, nil
	}

	// Check if manifest files are encrypted
	for _, f := range manifest.Files {
		if util.IsEncryptedBase64Path(f.Path) {
			return nil, nil
		}
	}

	report := &IntegrityReport{
		OfficialCount: len(manifest.Files),
	}

	officialMap := make(map[string]ManifestFileEntry, len(manifest.Files))
	foundMap := make(map[string]bool, len(manifest.Files))

	for _, f := range manifest.Files {
		norm := strings.ToLower(NormalizePath(f.Path))
		officialMap[norm] = f
	}

	err = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
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
				if fi, err := d.Info(); err == nil {
					if uint64(fi.Size()) != entry.Size {
						lowerRel := strings.ToLower(rel)
						if (strings.HasSuffix(lowerRel, "steam_api64.dll") || strings.HasSuffix(lowerRel, "steam_api.dll") || strings.HasSuffix(lowerRel, "libsteam_api.so")) && hasOriginalBackup(gameDir, rel) {
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

	return report, nil
}


// ScanUntrackedFiles walks gameDir and compares every file on disk against official manifest entries.
func ScanUntrackedFiles(gameDir string, appID uint32) (int, []string, error) {
	rep, err := ScanDepotIntegrity(gameDir, appID)
	if err != nil || rep == nil {
		return 0, nil, err
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


