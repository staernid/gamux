package manifest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/staernid/gamux/domain"
)

// InspectGameDepots inspects local manifests to produce a structured inventory of all game depots.
func InspectGameDepots(gameDir string, appID uint32) ([]domain.DepotSummary, error) {
	return InspectGameDepotsWithMetadata(gameDir, appID, nil)
}

// InspectGameDepotsWithMetadata inspects local manifests enriched with optional remote metadata to produce a structured inventory of game depots.
func InspectGameDepotsWithMetadata(gameDir string, appID uint32, metaMap map[uint32]DepotMeta) ([]domain.DepotSummary, error) {
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

		// If no depot keys were found locally and appID is known, resolve from default providers
		if len(depotKeys) == 0 && appID > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			chain := NewDefaultProviderChain("", "")
			if res, err := chain.Resolve(ctx, appID); err == nil && res != nil && len(res.Depots) > 0 {
				for _, pair := range res.Depots {
					depotKeys[pair.DepotID] = pair.DecryptionKey
				}
				luaPath := filepath.Join(manifestsDir, fmt.Sprintf("%d.lua", appID))
				_ = SaveLuaFile(luaPath, appID, res.Depots)
			}
			cancel()
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
			var meta *DepotMeta
			if metaMap != nil {
				if m, ok := metaMap[dID]; ok {
					meta = &m
				}
			}

			// Authoritative platform override from remote metadata
			if meta != nil && meta.OSList != "" {
				switch strings.ToLower(meta.OSList) {
				case "windows", "win":
					cat = DepotCategoryWindows
				case "linux":
					cat = DepotCategoryLinux
				case "macos", "osx", "darwin":
					cat = DepotCategoryMacOS
				}
			}

			contentCat, lang := ClassifyDepotContent(dID, "", files, meta)

			var totalBytes uint64
			fileCount := 0
			installedCount := 0
			execFileCount := 0
			installedExecCount := 0

			for _, f := range files {
				if (f.Flags & 0x40) != 0 {
					continue
				}
				totalBytes += f.Size
				fileCount++

				norm := NormalizePath(f.Path)
				isExec := isBinaryOrExecutable(norm)
				if isExec {
					execFileCount++
				}

				if norm != "" {
					if fi, statErr := os.Stat(filepath.Join(gameDir, norm)); statErr == nil && !fi.IsDir() {
						installedCount++
						if isExec {
							installedExecCount++
						}
					}
				}
			}

			var isInstalled bool
			if fileCount > 0 {
				if execFileCount > 0 {
					// Depots with platform binaries MUST have at least one binary present on disk
					isInstalled = installedExecCount > 0 && float64(installedCount)/float64(fileCount) >= 0.5
				} else {
					// Content / asset depots without binaries require high matching ratio
					isInstalled = installedCount > 0 && float64(installedCount)/float64(fileCount) >= 0.8
				}
			}
			depotName := fmt.Sprintf("Depot %d", dID)
			if meta != nil && meta.Name != "" {
				depotName = meta.Name
			} else if strings.HasPrefix(contentCat, ContentCategoryLocalePrefix) {
				depotName = fmt.Sprintf("%s Language Pack (%d)", strings.Title(lang), dID)
			} else if contentCat == ContentCategoryOptionalHD {
				depotName = fmt.Sprintf("High-Resolution Assets (%d)", dID)
			} else if contentCat == ContentCategorySoundtrack {
				depotName = fmt.Sprintf("Official Soundtrack (%d)", dID)
			} else {
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
			}

			var downloadSize uint64
			var osArch string
			var dlcAppID uint32
			var isDLC bool
			var dlcName string
			if meta != nil {
				downloadSize = meta.DownloadSize
				osArch = meta.OSArch
				dlcAppID = meta.DLCAppID
				isDLC = meta.IsDLC
				dlcName = meta.DLCName
				if meta.Size > 0 && totalBytes == 0 {
					totalBytes = meta.Size
				}
			}

			summaries = append(summaries, domain.DepotSummary{
				DepotID:         dID,
				Name:            depotName,
				Category:        string(cat),
				ContentCategory: contentCat,
				Language:        lang,
				Installed:       isInstalled,
				Size:            totalBytes,
				DownloadSize:    downloadSize,
				FileCount:       fileCount,
				ManifestID:      depotLatestManifestID[dID],
				OSArch:          osArch,
				DLCAppID:        dlcAppID,
				IsDLC:           isDLC,
				DLCName:         dlcName,
			})
		}
	}

	// If no local manifests found on disk, but remote metadata exists (e.g. preview or uninstalled stub)
	if len(summaries) == 0 && len(metaMap) > 0 {
		for dID, meta := range metaMap {
			m := meta
			contentCat, lang := ClassifyDepotContent(dID, m.Name, nil, &m)
			cat := DepotCategoryCommon
			if m.OSList == "windows" {
				cat = DepotCategoryWindows
			} else if m.OSList == "linux" {
				cat = DepotCategoryLinux
			} else if m.OSList == "macos" {
				cat = DepotCategoryMacOS
			}

			depotName := m.Name
			if depotName == "" {
				if strings.HasPrefix(contentCat, ContentCategoryLocalePrefix) {
					depotName = fmt.Sprintf("%s Language Pack (%d)", strings.Title(lang), dID)
				} else if contentCat == ContentCategoryOptionalHD {
					depotName = fmt.Sprintf("High-Resolution Assets (%d)", dID)
				} else if contentCat == ContentCategorySoundtrack {
					depotName = fmt.Sprintf("Official Soundtrack (%d)", dID)
				} else {
					archStr := ""
					if m.OSArch == "64" {
						archStr = " (64-bit)"
					} else if m.OSArch == "32" {
						archStr = " (32-bit)"
					}
					prefix := ""
					if m.IsDLC && m.DLCAppID > 0 {
						prefix = fmt.Sprintf("DLC #%d ", m.DLCAppID)
					}
					switch cat {
					case DepotCategoryCommon:
						if m.IsDLC {
							depotName = fmt.Sprintf("%sContent (%d)", prefix, dID)
						} else {
							depotName = fmt.Sprintf("Base Content (%d)", dID)
						}
					case DepotCategoryWindows:
						depotName = fmt.Sprintf("%sWindows Binaries%s (%d)", prefix, archStr, dID)
					case DepotCategoryLinux:
						depotName = fmt.Sprintf("%sLinux Native Binaries%s (%d)", prefix, archStr, dID)
					case DepotCategoryMacOS:
						depotName = fmt.Sprintf("%smacOS Bundle%s (%d)", prefix, archStr, dID)
					default:
						depotName = fmt.Sprintf("Depot %d", dID)
					}
				}
			}

			summaries = append(summaries, domain.DepotSummary{
				DepotID:         dID,
				Name:            depotName,
				Category:        string(cat),
				ContentCategory: contentCat,
				Language:        lang,
				Installed:       false,
				Size:            m.Size,
				DownloadSize:    m.DownloadSize,
				FileCount:       0,
				ManifestID:      0,
				OSArch:          m.OSArch,
				DLCAppID:        m.DLCAppID,
				IsDLC:           m.IsDLC,
				DLCName:         m.DLCName,
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

	// If no depot keys were found locally and appID is known, resolve from default providers
	if len(depotKeys) == 0 && appID > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		chain := NewDefaultProviderChain("", "")
		if res, err := chain.Resolve(ctx, appID); err == nil && res != nil && len(res.Depots) > 0 {
			for _, pair := range res.Depots {
				depotKeys[pair.DepotID] = pair.DecryptionKey
			}
			luaPath := filepath.Join(manifestsDir, fmt.Sprintf("%d.lua", appID))
			_ = SaveLuaFile(luaPath, appID, res.Depots)
		}
		cancel()
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
