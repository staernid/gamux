package detector

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
)

// GameInfo represents auto-detected metadata and paths for a game directory.
type GameInfo struct {
	AppID          string
	Name           string
	Platform       string // "linux", "win64", "win32"
	GameDir        string // Primary root directory of the game
	ExePath        string // Path to the main executable
	TargetLibPath  string // Path to steam_api library
	ManifestPath   string // Path to appmanifest_<appid>.acf
	RawDepotLayout bool   // True if raw steamapps/common/... structure
}

var appmanifestRegex = regexp.MustCompile(`(?i)^appmanifest_(\d+)\.acf$`)

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
		}
	}

	// 3. Detect Platform & Target Library
	detectPlatformAndLib(info)

	// 4. Fetch Game Name if AppID is known but Name is missing
	if info.AppID != "" && info.Name == "" {
		if name, err := steam.FetchAppName(ctx, info.AppID); err == nil && name != "" {
			info.Name = name
		}
	}

	// 5. Fallback Name to directory basename if still empty
	if info.Name == "" {
		info.Name = filepath.Base(info.GameDir)
	}

	// 6. Detect main executable
	detectExecutable(info)

	return info, nil
}

// ConsolidateManifests copies/moves appmanifest and .manifest files into <GameDir>/[Steam]/.
// If promoteFolder is true and the game is inside a raw steamapps/common layout,
// it moves the game folder up to the main parent directory and cleans up steamapps/depotcache.
func ConsolidateManifests(info *GameInfo, promoteFolder bool) error {
	// Move/copy ACF file if found
	if info.ManifestPath != "" {
		steamDir := filepath.Join(info.GameDir, "[Steam]")
		if err := os.MkdirAll(steamDir, 0755); err != nil {
			return fmt.Errorf("create [Steam] dir: %w", err)
		}

		srcDir := filepath.Dir(info.ManifestPath)
		destACF := filepath.Join(steamDir, filepath.Base(info.ManifestPath))
		if info.ManifestPath != destACF {
			if err := copyFile(info.ManifestPath, destACF); err == nil {
				slog.Info("Consolidated ACF manifest", "dest", destACF)
			}
		}

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
					destMan := filepath.Join(steamDir, e.Name())
					if srcMan != destMan {
						_ = copyFile(srcMan, destMan)
					}
				}
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
		filepath.Join(root, "[Steam]"),
		filepath.Join(root, "[Manifests]"),
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
	var bestExe string
	_ = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		// Skip installers, crash handlers, redistributables
		if strings.Contains(name, "unitycrashhandler") ||
			strings.Contains(name, "unins") ||
			strings.Contains(name, "setup") ||
			strings.Contains(name, "redist") {
			return nil
		}

		if info.Platform == "win64" || info.Platform == "win32" {
			if strings.HasSuffix(name, ".exe") {
				if bestExe == "" || strings.EqualFold(strings.TrimSuffix(d.Name(), ".exe"), filepath.Base(info.GameDir)) {
					bestExe = path
				}
			}
		} else { // linux
			if strings.HasSuffix(name, ".sh") || strings.HasSuffix(name, ".x86_64") || !strings.Contains(d.Name(), ".") {
				info, err := d.Info()
				if err == nil && (info.Mode()&0111 != 0) { // Executable bit set
					if bestExe == "" || strings.EqualFold(d.Name(), filepath.Base(info.Name())) {
						bestExe = path
					}
				}
			}
		}
		return nil
	})

	info.ExePath = bestExe
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
