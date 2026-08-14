package detector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// GameInfo represents auto-detected metadata and paths for a game directory.
type GameInfo struct {
	AppID            string
	Name             string
	Platform         string // "linux", "win64", "win32"
	GameDir          string // Primary root directory of the game
	ExePath          string // Path to the main executable
	ExeArgs          string // Command line arguments for launch
	TargetLibPath    string // Path to steam_api library
	ManifestPath     string // Path to appmanifest_<appid>.acf
	RawDepotLayout   bool   // True if raw steamapps/common/... structure
	LaunchCandidates []LaunchCandidate
}

var (
	appmanifestRegex  = regexp.MustCompile(`(?i)^appmanifest_(\d+)\.acf$`)
	numericAppIDRegex = regexp.MustCompile(`^\d+$`)
)

// Detect scans a target path and automatically identifies its AppID, Title, Platform, and Executable.
func Detect(ctx context.Context, path string) (*GameInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve abs path: %w", err)
	}

	info := &GameInfo{
		GameDir: absPath,
	}

	// 1. Search for ACF Manifests across subdirectories
	acfPath, acfData := findACF(absPath)
	if acfData != nil {
		info.AppID = acfData.AppID
		info.Name = acfData.Name
		info.ManifestPath = acfPath

		// Check for raw Steam depot layout: steamapps/common/<installdir>
		commonPath := filepath.Join(absPath, "steamapps", "common")
		if entries, err := os.ReadDir(commonPath); err == nil && len(entries) > 0 {
			info.RawDepotLayout = true
			for _, e := range entries {
				if e.IsDir() {
					if acfData.InstallDir != "" && strings.EqualFold(e.Name(), acfData.InstallDir) {
						info.GameDir = filepath.Join(commonPath, e.Name())
						break
					}
					info.GameDir = filepath.Join(commonPath, e.Name())
				}
			}
		}
	}

	// 2. Search for steam_appid.txt if AppID still missing
	if info.AppID == "" {
		if appID, err := findAppIDTxt(info.GameDir); err == nil && appID != "" {
			info.AppID = appID
		} else {
			baseName := filepath.Base(info.GameDir)
			if numericAppIDRegex.MatchString(baseName) {
				info.AppID = baseName
			}
		}
	}

	// 3. Detect Platform & Target Library
	detectPlatformAndLib(info)

	// 4. Fetch Game Name if AppID is known but Name is missing or matches raw AppID
	if info.AppID != "" && (info.Name == "" || info.Name == info.AppID) {
		if name, err := steam.FetchAppName(ctx, info.AppID); err == nil && name != "" {
			info.Name = name
		}
	}

	// 5. Fallback Name to directory basename if still empty
	if info.Name == "" {
		info.Name = filepath.Base(info.GameDir)
	}

	// 6. Try searching Steam Store API if AppID is still missing
	if info.AppID == "" && info.Name != "" {
		if appID, searchName, err := steam.SearchAppID(ctx, info.Name); err == nil && appID != "" {
			info.AppID = appID
			if searchName != "" {
				info.Name = searchName
			}
			slog.Info("Auto-discovered AppID via Steam Store search", "title", info.Name, "appID", info.AppID)
		}
	}

	// 7. Detect main executable
	detectExecutable(info)

	return info, nil
}

// ConsolidateManifests copies/moves appmanifest and .manifest files into <GameDir>/[Manifests]/.
// If a legacy [Steam] folder exists, its contents are migrated into [Manifests] and [Steam] is removed.
// If promoteFolder is true and the game is inside a raw steamapps/common layout,
// it moves the game folder up to the main parent directory and cleans up steamapps/depotcache.
func ConsolidateManifests(info *GameInfo, promoteFolder bool) error {
	manifestsDir := filepath.Join(info.GameDir, "[Manifests]")
	legacySteamDir := filepath.Join(info.GameDir, "[Steam]")

	// Migrate legacy [Steam] folder to [Manifests] if present
	if fileExists(legacySteamDir) && legacySteamDir != manifestsDir {
		if err := os.MkdirAll(manifestsDir, 0755); err == nil {
			entries, err := os.ReadDir(legacySteamDir)
			if err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						oldFile := filepath.Join(legacySteamDir, e.Name())
						newFile := filepath.Join(manifestsDir, e.Name())
						if err := os.Rename(oldFile, newFile); err != nil {
							_ = copyFile(oldFile, newFile)
							_ = os.Remove(oldFile)
						}
					}
				}
			}
			_ = os.RemoveAll(legacySteamDir)
			slog.Info("Migrated legacy [Steam] folder to [Manifests]", "dir", manifestsDir)
		}
	}

	// Move/copy ACF file if found
	if info.ManifestPath != "" {
		if err := os.MkdirAll(manifestsDir, 0755); err != nil {
			return fmt.Errorf("create [Manifests] dir: %w", err)
		}

		srcDir := filepath.Dir(info.ManifestPath)
		destACF := filepath.Join(manifestsDir, filepath.Base(info.ManifestPath))
		if info.ManifestPath != destACF && fileExists(info.ManifestPath) {
			if err := copyFile(info.ManifestPath, destACF); err == nil {
				slog.Info("Consolidated ACF manifest", "dest", destACF)
				if strings.HasSuffix(srcDir, "[Steam]") {
					_ = os.Remove(info.ManifestPath)
				}
			}
		}
		info.ManifestPath = destACF

		// Also look for matching .manifest files in srcDir or depotcache
		scanDirs := []string{srcDir}
		if info.RawDepotLayout {
			rootParent := filepath.Dir(filepath.Dir(srcDir))
			scanDirs = append(scanDirs, filepath.Join(rootParent, "depotcache"))
		}

		for _, d := range scanDirs {
			entries, err := os.ReadDir(d)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".manifest") {
					srcMan := filepath.Join(d, e.Name())
					destMan := filepath.Join(manifestsDir, e.Name())
					if srcMan != destMan {
						if err := os.Rename(srcMan, destMan); err != nil {
							_ = copyFile(srcMan, destMan)
							_ = os.Remove(srcMan)
						}
					}
				}
			}
			if strings.HasSuffix(d, "[Steam]") {
				_ = os.RemoveAll(d)
			}
		}
	}

	// Handle promotion if raw depot layout
	if promoteFolder && info.RawDepotLayout {
		steamappsParent := filepath.Dir(filepath.Dir(info.GameDir)) // root directory containing steamapps/
		targetParentDir := filepath.Dir(steamappsParent)

		promotedDir := filepath.Join(targetParentDir, filepath.Base(info.GameDir))
		if promotedDir != info.GameDir && !fileExists(promotedDir) {
			if err := os.Rename(info.GameDir, promotedDir); err == nil {
				slog.Info("Promoted game directory to top level", "from", info.GameDir, "to", promotedDir)
				info.GameDir = promotedDir

				// Clean up empty steamapps/depotcache structure if possible
				_ = os.RemoveAll(steamappsParent)
			}
		}
	}

	return nil
}

// ── Internal Helpers ────────────────────────────────────────────────

func findACF(root string) (string, *ACFData) {
	searchLocations := []string{
		filepath.Join(root, "[Manifests]"),
		filepath.Join(root, "[Steam]"),
		filepath.Join(root, "steamapps"),
		root,
	}

	for _, loc := range searchLocations {
		entries, err := os.ReadDir(loc)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && appmanifestRegex.MatchString(e.Name()) {
				path := filepath.Join(loc, e.Name())
				if data, err := ParseACFFile(path); err == nil {
					return path, data
				}
			}
		}
	}

	return "", nil
}

func findAppIDTxt(root string) (string, error) {
	var appID string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "steam_appid.txt" {
			data, err := os.ReadFile(path)
			if err == nil {
				appID = strings.TrimSpace(string(data))
				return filepath.SkipAll
			}
		}
		return nil
	})
	return appID, err
}

func detectPlatformAndLib(info *GameInfo) {
	for p, cfg := range config.PlatformConfig {
		_ = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == cfg.Target {
				info.Platform = p
				info.TargetLibPath = path
				return filepath.SkipAll
			}
			return nil
		})
		if info.Platform != "" {
			break
		}
	}

	// Default fallback platform if steam_api not found
	if info.Platform == "" {
		info.Platform = "linux"
	}
}

func detectExecutable(info *GameInfo) {
	var candidateExes []string

	_ = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		// Skip installers, crash handlers, redistributables, setup files, console wrappers
		if strings.Contains(name, "unitycrashhandler") ||
			strings.Contains(name, "unins") ||
			strings.Contains(name, "setup") ||
			strings.Contains(name, "redist") ||
			strings.Contains(name, "dxsetup") ||
			strings.Contains(name, "vcredist") ||
			strings.Contains(name, "crashdumper") ||
			strings.HasSuffix(name, ".console.exe") {
			return nil
		}

		if strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".x86_64") || !strings.Contains(d.Name(), ".") {
			fileInfo, err := d.Info()
			if err == nil && (strings.HasSuffix(name, ".exe") || fileInfo.Mode()&0111 != 0) {
				candidateExes = append(candidateExes, path)
			}
		}
		return nil
	})

	// Build LaunchCandidates list
	seenExes := make(map[string]bool)

	// Check if ACF ACFData parsed launch options
	if _, acfData := findACF(info.GameDir); acfData != nil && len(acfData.LaunchOptions) > 0 {
		for _, option := range acfData.LaunchOptions {
			if option.Executable != "" {
				fullPath := filepath.Join(info.GameDir, option.Executable)
				if _, err := os.Stat(fullPath); err == nil {
					name := option.Description
					if name == "" {
						name = filepath.Base(fullPath)
					}
					info.LaunchCandidates = append(info.LaunchCandidates, LaunchCandidate{
						Name:        name,
						Executable:  fullPath,
						Arguments:   option.Arguments,
						Description: option.Description,
					})
					seenExes[fullPath] = true
				}
			}
		}
	}

	// Add any remaining detected executables to LaunchCandidates
	dirName := filepath.Base(info.GameDir)
	for _, exe := range candidateExes {
		if !seenExes[exe] {
			baseNoExt := strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
			name := baseNoExt
			if strings.EqualFold(baseNoExt, dirName) || (info.Name != "" && strings.EqualFold(baseNoExt, info.Name)) {
				name = info.Name + " (Main Executable)"
			}
			info.LaunchCandidates = append(info.LaunchCandidates, LaunchCandidate{
				Name:       name,
				Executable: exe,
			})
			seenExes[exe] = true
		}
	}

	if len(info.LaunchCandidates) > 0 {
		info.ExePath = info.LaunchCandidates[0].Executable
		info.ExeArgs = info.LaunchCandidates[0].Arguments
	}

	// Update platform if executable is a Windows .exe and platform was defaulted to linux
	if info.ExePath != "" && strings.HasSuffix(strings.ToLower(info.ExePath), ".exe") && info.Platform == "linux" {
		info.Platform = "win64"
	}
}

func copyFile(src, dest string) error {
	return util.CopyFile(src, dest)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
