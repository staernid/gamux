package manifest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/staernid/gamux/domain"
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

// DepotCategory classifies a Steam content depot's platform or payload type.
type DepotCategory string

const (
	DepotCategoryCommon     DepotCategory = "common"
	DepotCategoryWindows    DepotCategory = "windows"
	DepotCategoryLinux      DepotCategory = "linux"
	DepotCategoryMacOS      DepotCategory = "macos"
	DepotCategorySoundtrack DepotCategory = "soundtrack"
)

// ClassifyDepot inspects manifest file entries to determine the target OS or payload category.
func ClassifyDepot(files []ManifestFileEntry) DepotCategory {
	if len(files) == 0 {
		return DepotCategoryCommon
	}

	var hasWin, hasMac, hasLinux, hasExecutables, hasMediaOnly bool
	mediaExts := map[string]bool{
		".mp3": true, ".flac": true, ".wav": true, ".ogg": true, ".m4a": true,
		".aac": true, ".opus": true, ".cue": true, ".log": true, ".jpg": true,
		".jpeg": true, ".png": true, ".pdf": true,
	}

	mediaCount := 0
	nonDirCount := 0

	for _, f := range files {
		if (f.Flags & 0x40) != 0 {
			continue // directory
		}
		nonDirCount++
		p := strings.ToLower(NormalizePath(f.Path))

		// 1. macOS checks
		if strings.Contains(p, ".app/") || strings.Contains(p, "contents/macos/") || strings.HasPrefix(p, "macos/") || strings.HasPrefix(p, "osx/") || strings.HasSuffix(p, ".dylib") {
			hasMac = true
		}

		// 2. Linux checks
		if strings.HasSuffix(p, ".x86_64") || strings.HasSuffix(p, ".x86") || strings.HasSuffix(p, ".bin.x86") || strings.HasSuffix(p, ".bin.x86_64") || strings.Contains(p, "linux_client") || strings.HasPrefix(p, "linux/") {
			hasLinux = true
		}

		// 3. Windows checks
		if strings.HasSuffix(p, ".exe") || strings.HasSuffix(p, ".dll") || strings.HasSuffix(p, ".bat") || strings.HasSuffix(p, ".cmd") || strings.HasPrefix(p, "win32/") || strings.HasPrefix(p, "win64/") || strings.HasPrefix(p, "windows/") || strings.Contains(p, "_commonredist") {
			hasWin = true
		}

		// 4. Executable / binary check
		if strings.HasSuffix(p, ".exe") || strings.HasSuffix(p, ".dll") || strings.HasSuffix(p, ".so") || strings.HasSuffix(p, ".dylib") || strings.HasSuffix(p, ".app") || strings.HasSuffix(p, ".x86_64") || strings.HasSuffix(p, ".x86") {
			hasExecutables = true
		}

		ext := filepath.Ext(p)
		if mediaExts[ext] {
			mediaCount++
		}
	}

	if nonDirCount > 0 && mediaCount == nonDirCount && !hasExecutables {
		hasMediaOnly = true
	}

	if hasMediaOnly {
		return DepotCategorySoundtrack
	}
	if hasMac && !hasWin && !hasLinux {
		return DepotCategoryMacOS
	}
	if hasLinux && !hasWin && !hasMac {
		return DepotCategoryLinux
	}
	if hasWin && !hasMac && !hasLinux {
		return DepotCategoryWindows
	}
	if hasMac {
		return DepotCategoryMacOS
	}
	if hasLinux {
		return DepotCategoryLinux
	}
	if hasWin {
		return DepotCategoryWindows
	}

	return DepotCategoryCommon
}

// MatchesPlatform tests whether a depot category matches the requested platform target.
func MatchesPlatform(cat DepotCategory, targetPlat string) bool {
	target := strings.ToLower(strings.TrimSpace(targetPlat))
	if target == "" || target == "all" {
		return true
	}

	switch cat {
	case DepotCategoryCommon:
		return true
	case DepotCategoryWindows:
		return target == "win64" || target == "win32" || target == "windows" || target == "win"
	case DepotCategoryLinux:
		return target == "linux" || target == "linuxnative"
	case DepotCategoryMacOS:
		return target == "macos" || target == "osx" || target == "darwin" || target == "mac"
	case DepotCategorySoundtrack:
		return target == "soundtrack" || target == "music"
	default:
		return true
	}
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
	return ScanDepotIntegrityWithPlatform(gameDir, appID, "")
}

// ScanDepotIntegrityWithPlatform compares files on disk against official manifest entries matching a target platform.
func ScanDepotIntegrityWithPlatform(gameDir string, appID uint32, platform string) (*IntegrityReport, error) {
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
		res, err := chain.Resolve(context.Background(), appID)
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

	report := &IntegrityReport{
		OfficialCount: len(manifest.Files),
	}

	officialMap := make(map[string]ManifestFileEntry, len(manifest.Files))
	foundMap := make(map[string]bool, len(manifest.Files))

	for _, f := range manifest.Files {
		norm := strings.ToLower(NormalizePath(f.Path))
		officialMap[norm] = f
	}

	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
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

// InspectGameDepots inspects local manifests to produce a structured inventory of all game depots.
func InspectGameDepots(gameDir string, appID uint32) ([]domain.DepotSummary, error) {
	manifestsDir := filepath.Join(gameDir, "[Manifests]")
	var summaries []domain.DepotSummary

	if entries, err := os.ReadDir(manifestsDir); err == nil && len(entries) > 0 {
		depotLatest := make(map[uint32]string)
		depotLatestManifestID := make(map[uint32]uint64)

		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
				parts := strings.Split(e.Name(), "_")
				if len(parts) >= 2 {
					var did uint32
					var mid uint64
					if d, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
						did = uint32(d)
					}
					midStr := strings.TrimSuffix(parts[1], ".manifest")
					if m, err := strconv.ParseUint(midStr, 10, 64); err == nil {
						mid = m
					}
					if mid > depotLatestManifestID[did] {
						depotLatestManifestID[did] = mid
						depotLatest[did] = filepath.Join(manifestsDir, e.Name())
					}
				}
			}
		}

		// Read depot keys from lua if present
		depotKeys := make(map[uint32]string)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
				if content, err := os.ReadFile(filepath.Join(manifestsDir, e.Name())); err == nil {
					if parsed, err := ParseLua(string(content)); err == nil {
						for _, pair := range parsed.Depots {
							depotKeys[pair.DepotID] = pair.DecryptionKey
						}
					}
				}
			}
		}

		for dID, manifestPath := range depotLatest {
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			files, err := ParseBinaryManifestWithKey(data, depotKeys[dID])
			if err != nil || len(files) == 0 {
				continue
			}

			cat := ClassifyDepot(files)
			var totalBytes uint64
			fileCount := 0
			installedCount := 0

			for _, f := range files {
				if (f.Flags & 0x40) != 0 {
					continue
				}
				totalBytes += f.Size
				fileCount++

				norm := NormalizePath(f.Path)
				if norm != "" {
					if fi, statErr := os.Stat(filepath.Join(gameDir, norm)); statErr == nil && !fi.IsDir() {
						installedCount++
					}
				}
			}

			isInstalled := fileCount > 0 && installedCount > 0 && float64(installedCount)/float64(fileCount) >= 0.5
			depotName := fmt.Sprintf("Depot %d", dID)
			switch cat {
			case DepotCategoryCommon:
				depotName = fmt.Sprintf("Base Content (%d)", dID)
			case DepotCategoryWindows:
				depotName = fmt.Sprintf("Windows Binaries (%d)", dID)
			case DepotCategoryLinux:
				depotName = fmt.Sprintf("Linux Native Binaries (%d)", dID)
			case DepotCategoryMacOS:
				depotName = fmt.Sprintf("macOS Bundle (%d)", dID)
			case DepotCategorySoundtrack:
				depotName = fmt.Sprintf("Official Soundtrack (%d)", dID)
			}

			summaries = append(summaries, domain.DepotSummary{
				DepotID:    dID,
				Name:       depotName,
				Category:   string(cat),
				Installed:  isInstalled,
				Size:       totalBytes,
				FileCount:  fileCount,
				ManifestID: depotLatestManifestID[dID],
			})
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].DepotID < summaries[j].DepotID
	})

	return summaries, nil
}

// DiscoverManifestExecutables discovers candidate launch targets directly from developer depot manifests.
func DiscoverManifestExecutables(gameDir string, appID uint32, platform string) []domain.LaunchCandidate {
	m, err := LoadManifestFromDirWithPlatform(gameDir, appID, nil, platform)
	if err != nil || m == nil || len(m.Files) == 0 {
		return nil
	}

	var candidates []domain.LaunchCandidate
	seen := make(map[string]bool)

	for _, f := range m.Files {
		if (f.Flags & 0x40) != 0 {
			continue
		}
		p := NormalizePath(f.Path)
		pLower := strings.ToLower(p)

		isExe := false
		if strings.HasSuffix(pLower, ".exe") || strings.HasSuffix(pLower, ".bin.x86_64") || strings.HasSuffix(pLower, ".bin.x86") || strings.HasSuffix(pLower, ".x86_64") || strings.HasSuffix(pLower, ".x86") || strings.HasSuffix(pLower, ".app") || strings.Contains(pLower, ".app/contents/macos/") {
			isExe = true
		}

		if !isExe {
			continue
		}

		// Filter out obvious installers, crash dumpers, and redistributables
		if strings.Contains(pLower, "_commonredist") || strings.Contains(pLower, "directx") || strings.Contains(pLower, "vcredist") || strings.Contains(pLower, "crashreport") || strings.Contains(pLower, "unins000") || strings.Contains(pLower, "uninstall") || strings.Contains(pLower, "cleanup") {
			continue
		}

		if seen[p] {
			continue
		}
		seen[p] = true

		baseName := filepath.Base(p)
		cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		desc := "Discovered Executable"
		if strings.Contains(pLower, "shipping") {
			desc = "Main Shipping Binary"
		} else if strings.Contains(pLower, "dx11") || strings.Contains(pLower, "d3d11") {
			desc = "DirectX 11 Mode"
		} else if strings.Contains(pLower, "dx12") || strings.Contains(pLower, "d3d12") {
			desc = "DirectX 12 Mode"
		} else if strings.Contains(pLower, "vulkan") || strings.Contains(pLower, "vk") {
			desc = "Vulkan Mode"
		} else if strings.Contains(pLower, "server") {
			desc = "Dedicated Server"
		} else if strings.Contains(pLower, "config") || strings.Contains(pLower, "setup") || strings.Contains(pLower, "launcher") {
			desc = "Launcher / Config Utility"
		}

		candidates = append(candidates, domain.LaunchCandidate{
			Name:        cleanName,
			Executable:  p,
			Description: desc,
		})
	}

	// Sort candidates: main game executables first, launcher/config utilities second
	sort.SliceStable(candidates, func(i, j int) bool {
		iLower := strings.ToLower(candidates[i].Executable)
		jLower := strings.ToLower(candidates[j].Executable)
		iIsHelper := strings.Contains(iLower, "config") || strings.Contains(iLower, "setup") || strings.Contains(iLower, "server")
		jIsHelper := strings.Contains(jLower, "config") || strings.Contains(jLower, "setup") || strings.Contains(jLower, "server")
		if iIsHelper != jIsHelper {
			return !iIsHelper
		}
		return len(candidates[i].Executable) < len(candidates[j].Executable)
	})

	return candidates
}

// UninstallDepot removes all files on disk belonging to a specific depot ID, preserving shared files and manifests.
// Returns the count of removed files and freed bytes.
func UninstallDepot(gameDir string, appID uint32, depotID uint32) (int, uint64, error) {
	manifestsDir := filepath.Join(gameDir, "[Manifests]")
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		return 0, 0, fmt.Errorf("read manifests dir: %w", err)
	}

	var targetManifest string
	var targetManifestID uint64

	// Read depot keys from lua if present
	depotKeys := make(map[uint32]string)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
			if content, err := os.ReadFile(filepath.Join(manifestsDir, e.Name())); err == nil {
				if parsed, err := ParseLua(string(content)); err == nil {
					for _, pair := range parsed.Depots {
						depotKeys[pair.DepotID] = pair.DecryptionKey
					}
				}
			}
		}
	}

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
			parts := strings.Split(e.Name(), "_")
			if len(parts) >= 2 {
				var did uint32
				var mid uint64
				if d, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
					did = uint32(d)
				}
				midStr := strings.TrimSuffix(parts[1], ".manifest")
				if m, err := strconv.ParseUint(midStr, 10, 64); err == nil {
					mid = m
				}
				if did == depotID && mid > targetManifestID {
					targetManifestID = mid
					targetManifest = filepath.Join(manifestsDir, e.Name())
				}
			}
		}
	}

	if targetManifest == "" {
		return 0, 0, fmt.Errorf("no manifest found for depot %d", depotID)
	}

	data, err := os.ReadFile(targetManifest)
	if err != nil {
		return 0, 0, fmt.Errorf("read manifest %s: %w", targetManifest, err)
	}

	targetFiles, err := ParseBinaryManifestWithKey(data, depotKeys[depotID])
	if err != nil {
		return 0, 0, fmt.Errorf("parse manifest: %w", err)
	}

	// Collect files from other installed depots to prevent deleting shared files
	otherDepotFiles := make(map[string]bool)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".manifest") {
			parts := strings.Split(e.Name(), "_")
			if len(parts) >= 2 {
				var did uint32
				if d, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
					did = uint32(d)
				}
				if did != depotID {
					otherData, err := os.ReadFile(filepath.Join(manifestsDir, e.Name()))
					if err == nil {
						if oFiles, err := ParseBinaryManifestWithKey(otherData, depotKeys[did]); err == nil {
							for _, of := range oFiles {
								norm := NormalizePath(of.Path)
								if norm != "" {
									otherDepotFiles[norm] = true
								}
							}
						}
					}
				}
			}
		}
	}

	deletedCount := 0
	var freedBytes uint64
	var parentDirs []string

	for _, f := range targetFiles {
		if (f.Flags & 0x40) != 0 {
			continue
		}
		norm := NormalizePath(f.Path)
		if norm == "" || otherDepotFiles[norm] {
			continue
		}

		absPath := filepath.Join(gameDir, norm)
		fi, statErr := os.Stat(absPath)
		if statErr == nil && !fi.IsDir() {
			freedBytes += uint64(fi.Size())
			if removeErr := os.Remove(absPath); removeErr == nil {
				deletedCount++
				parentDirs = append(parentDirs, filepath.Dir(absPath))
			}
		}
	}

	// Clean up empty directories
	for _, pDir := range parentDirs {
		curr := pDir
		for curr != gameDir && curr != "." && curr != "/" && curr != "" {
			_ = os.Remove(curr) // Only succeeds if directory is empty
			curr = filepath.Dir(curr)
		}
	}

	return deletedCount, freedBytes, nil
}




