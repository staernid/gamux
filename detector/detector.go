package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)


// GameInfo represents auto-detected metadata and paths for a game directory.
type GameInfo struct {
	AppID            string
	Name             string
	InstallDir       string
	Platform         string // "linux", "win64", "win32"
	GameDir          string // Primary root directory of the game
	ExePath          string // Path to the main executable
	ExeArgs          string // Command line arguments for launch
	TargetLibPath    string // Path to steam_api library
	ManifestPath     string // Path to appmanifest_<appid>.acf
	RawDepotLayout   bool   // True if raw steamapps/common/... structure
	Store            string // "Steam", "GOG", "Epic", "Itch", "Custom"
	OfficialFileCount int
	UntrackedFiles   []string
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

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrGameNotFound, absPath)
	}

	info := &GameInfo{
		GameDir: absPath,
	}

	// 1. Search for ACF Manifests across subdirectories
	acfPath, acfData := findACF(absPath)
	if acfData != nil {
		info.AppID = acfData.AppID
		info.Name = acfData.Name
		info.InstallDir = acfData.InstallDir
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

	// 4. Detect store origin FIRST (Steam, GOG, Epic, Itch, Standalone)
	detectStore(info)

	// 5. Fetch Game Name if AppID is known but Name is missing or matches raw AppID
	if info.AppID != "" && (info.Name == "" || info.Name == info.AppID) {
		if name, err := steam.FetchAppName(ctx, info.AppID); err == nil && name != "" {
			info.Name = name
		}
	}

	// 6. Fallback Name to directory basename if still empty
	if info.Name == "" {
		info.Name = filepath.Base(info.GameDir)
	}

	// 7. Try searching Steam Store API ONLY if store is Steam and AppID is missing
	if info.Store == "Steam" && info.AppID == "" && info.Name != "" {
		if appID, searchName, err := steam.SearchAppID(ctx, info.Name); err == nil && appID != "" {
			info.AppID = appID
			if searchName != "" {
				info.Name = searchName
			}
			slog.Debug("Auto-discovered AppID via Steam Store search", "title", info.Name, "appID", info.AppID)
		}
	}

	// 8. Resolve InstallDir if missing
	if info.InstallDir == "" && info.Name != "" {
		info.InstallDir = util.SanitizeInstallDir(info.Name)
	}

	// 9. Detect main executable
	detectExecutable(info)

	// 10. Check if a saved Lutris config exists for this game and use its target executable if valid
	if info.Name != "" {
		slug := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(info.Name, "-"))
		slug = strings.Trim(slug, "-")
		if home, err := os.UserHomeDir(); err == nil {
			lutrisYamlPath := filepath.Join(home, ".config", "lutris", "games", slug+".yml")
			if data, err := os.ReadFile(lutrisYamlPath); err == nil {
				reExe := regexp.MustCompile(`(?m)^\s*exe:\s*["']?([^"'\r\n]+)`)
				if matches := reExe.FindStringSubmatch(string(data)); len(matches) > 1 {
					savedExe := strings.TrimSpace(matches[1])
					if savedExe != "" && util.FileExists(savedExe) {
						info.ExePath = savedExe
					}
				}
			}
		}
	}

	return info, nil
}

type GOGInfo struct {
	GameID string `json:"gameId"`
	Name   string `json:"name"`
	Title  string `json:"title"`
}

func parseGOGInfo(gameDir string) (string, string) {
	entries, err := os.ReadDir(gameDir)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		lname := strings.ToLower(e.Name())
		if strings.HasPrefix(lname, "goggame-") && strings.HasSuffix(lname, ".info") {
			data, err := os.ReadFile(filepath.Join(gameDir, e.Name()))
			if err != nil {
				continue
			}
			var gog GOGInfo
			if err := json.Unmarshal(data, &gog); err == nil {
				name := gog.Name
				if name == "" {
					name = gog.Title
				}
				return name, gog.GameID
			}
		}
	}
	return "", ""
}

func detectStore(info *GameInfo) {
	// 1. Check for concrete Steam artifacts on disk FIRST (manifests, appid.txt, or steam_api DLL/SO)
	hasSteamArtifacts := info.ManifestPath != "" || info.TargetLibPath != ""
	if !hasSteamArtifacts {
		manifestsDir := filepath.Join(info.GameDir, "[Manifests]")
		if entries, err := os.ReadDir(manifestsDir); err == nil && len(entries) > 0 {
			hasSteamArtifacts = true
		}
	}
	if !hasSteamArtifacts {
		if appidFile := filepath.Join(info.GameDir, "steam_appid.txt"); util.FileExists(appidFile) {
			hasSteamArtifacts = true
		}
	}

	if hasSteamArtifacts {
		info.Store = "Steam"
		return
	}

	// 2. Check for GOG signatures & parse GOG metadata
	if gogName, gogID := parseGOGInfo(info.GameDir); gogName != "" || gogID != "" {
		info.Store = "GOG"
		if gogName != "" {
			info.Name = gogName
		}
		return
	}

	if entries, err := os.ReadDir(info.GameDir); err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if strings.HasPrefix(name, "goggame-") || strings.HasPrefix(name, "goglog") || name == "webcache.zip" || strings.HasPrefix(name, "gog") {
				info.Store = "GOG"
				return
			}
			if name == ".egstore" {
				info.Store = "Epic"
				return
			}
			if name == ".itch" || name == ".itch.toml" {
				info.Store = "Itch"
				return
			}
		}
	}

	// 3. Standalone DRM-free download (Factorio.com, Humble Bundle, direct site download)
	info.Store = "Standalone"
}

// ConsolidateManifests copies/moves appmanifest and .manifest files into <GameDir>/[Manifests]/.
func ConsolidateManifests(info *GameInfo) error {
	if info == nil || info.GameDir == "" {
		return nil
	}

	manifestsDir := filepath.Join(info.GameDir, "[Manifests]")

	if info.ManifestPath != "" {
		if strings.HasPrefix(info.ManifestPath, manifestsDir) {
			return nil
		}
		_ = os.MkdirAll(manifestsDir, 0755)

		srcDir := filepath.Dir(info.ManifestPath)
		destACF := filepath.Join(manifestsDir, filepath.Base(info.ManifestPath))
		if info.ManifestPath != destACF && util.FileExists(info.ManifestPath) {
			if err := util.CopyFile(info.ManifestPath, destACF); err == nil {
				slog.Info("Consolidated ACF manifest", "dest", destACF)
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
							_ = util.CopyFile(srcMan, destMan)
							_ = os.Remove(srcMan)
						}
					}
				}
			}
		}
	}

	return nil
}

// EnsureACFManifest guarantees that a 1:1 appmanifest_<appid>.acf file exists inside <GameDir>/[Manifests]/.
// If missing or incomplete, it generates a standard Steam KeyValues manifest with canonical name & installdir.
func EnsureACFManifest(info *GameInfo, dryRun bool) error {
	if info == nil || info.AppID == "" || info.AppID == "0" {
		return nil
	}

	manifestsDir := filepath.Join(info.GameDir, "[Manifests]")
	acfName := fmt.Sprintf("appmanifest_%s.acf", info.AppID)
	targetACF := filepath.Join(manifestsDir, acfName)

	if util.FileExists(targetACF) {
		return nil
	}

	if info.InstallDir == "" {
		info.InstallDir = util.SanitizeInstallDir(info.Name)
	}

	var launchOpts []LaunchCandidate
	if info.ExePath != "" {
		relExe, err := filepath.Rel(info.GameDir, info.ExePath)
		if err == nil && !strings.HasPrefix(relExe, "..") {
			launchOpts = append(launchOpts, LaunchCandidate{
				Executable:  relExe,
				Arguments:   info.ExeArgs,
				Description: info.Name,
			})
		} else {
			launchOpts = append(launchOpts, LaunchCandidate{
				Executable:  info.ExePath,
				Arguments:   info.ExeArgs,
				Description: info.Name,
			})
		}
	}

	acfData := &ACFData{
		AppID:         info.AppID,
		Name:          info.Name,
		InstallDir:    info.InstallDir,
		BuildID:       "0",
		LaunchOptions: launchOpts,
	}

	content := GenerateACFContent(acfData)
	if dryRun {
		slog.Info("[DRY RUN] Would synthesize 1:1 ACF manifest", "path", targetACF, "appID", info.AppID)
		return nil
	}

	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return fmt.Errorf("create [Manifests] dir: %w", err)
	}

	if err := os.WriteFile(targetACF, []byte(content), 0644); err != nil {
		return fmt.Errorf("write ACF manifest: %w", err)
	}

	info.ManifestPath = targetACF
	slog.Info("Synthesized 1:1 ACF manifest", "path", targetACF, "appID", info.AppID, "installdir", info.InstallDir)
	return nil
}

// NormalizeDirectory renames the game directory to match its official Steam 1:1 InstallDir if different.
func NormalizeDirectory(info *GameInfo, dryRun bool) error {
	if info == nil || info.GameDir == "" || info.InstallDir == "" {
		return nil
	}

	currentBase := filepath.Base(info.GameDir)
	if currentBase == info.InstallDir {
		return nil
	}

	parentDir := filepath.Dir(info.GameDir)
	targetDir := filepath.Join(parentDir, info.InstallDir)

	if targetDir == info.GameDir || util.FileExists(targetDir) {
		return nil
	}

	if dryRun {
		slog.Info("[DRY RUN] Would normalize game directory name to 1:1 Steam installdir", "from", info.GameDir, "to", targetDir)
		return nil
	}

	oldGameDir := info.GameDir
	if err := os.Rename(oldGameDir, targetDir); err != nil {
		return fmt.Errorf("normalize game directory from %s to %s: %w", oldGameDir, targetDir, err)
	}

	info.GameDir = targetDir
	if info.ManifestPath != "" && strings.HasPrefix(info.ManifestPath, oldGameDir) {
		rel, err := filepath.Rel(oldGameDir, info.ManifestPath)
		if err == nil {
			info.ManifestPath = filepath.Join(targetDir, rel)
		}
	}
	if info.ExePath != "" && strings.HasPrefix(info.ExePath, oldGameDir) {
		rel, err := filepath.Rel(oldGameDir, info.ExePath)
		if err == nil {
			info.ExePath = filepath.Join(targetDir, rel)
		}
	}

	slog.Info("Normalized game directory name to 1:1 Steam installdir", "from", oldGameDir, "to", targetDir)
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

func scoreExecutable(exePath string, gameDir string, gameName string) int {
	base := filepath.Base(exePath)
	lowerBase := strings.ToLower(base)
	baseNoExt := strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))

	cleanDirName := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(filepath.Base(gameDir), ""))
	cleanGameName := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(gameName, ""))
	cleanExeName := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(baseNoExt, ""))

	score := 0

	// Exclude / demote crash report tools and secondary helpers
	if strings.Contains(lowerBase, "bssndrpt") ||
		strings.Contains(lowerBase, "crashdumper") ||
		strings.Contains(lowerBase, "bugreport") ||
		strings.Contains(lowerBase, "errorreport") ||
		strings.Contains(lowerBase, "unitycrashhandler") {
		score -= 2000
	}

	// Demote common secondary tools/editors
	if strings.Contains(lowerBase, "contentcompiler") ||
		strings.Contains(lowerBase, "particleeditor") ||
		strings.Contains(lowerBase, "tileeditor") ||
		strings.Contains(lowerBase, "worldbuilder") ||
		strings.Contains(lowerBase, "mapeditor") ||
		strings.Contains(lowerBase, "dedic") {
		score -= 500
	}

	// Boost exact matches with clean directory name or clean game title
	if cleanExeName != "" {
		if cleanDirName != "" && cleanExeName == cleanDirName {
			score += 1000
		} else if cleanGameName != "" && cleanExeName == cleanGameName {
			score += 1000
		} else if cleanDirName != "" && (strings.HasPrefix(cleanExeName, cleanDirName) || strings.HasPrefix(cleanDirName, cleanExeName)) {
			score += 500
		} else if cleanGameName != "" && (strings.HasPrefix(cleanExeName, cleanGameName) || strings.HasPrefix(cleanGameName, cleanExeName)) {
			score += 500
		}
	}

	// Boost binaries inside Release/bin/x64 directories
	dirLower := strings.ToLower(filepath.Dir(exePath))
	if strings.Contains(dirLower, "release") || strings.Contains(dirLower, "bin") || strings.Contains(dirLower, "x64") {
		score += 100
	}

	return score
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
			strings.Contains(name, "bssndrpt") ||
			strings.Contains(name, "bugreport") ||
			strings.Contains(name, "errorreport") ||
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

	// Sort candidate executables by smart score so main game executables rank #1
	sort.SliceStable(candidateExes, func(i, j int) bool {
		scoreI := scoreExecutable(candidateExes[i], info.GameDir, info.Name)
		scoreJ := scoreExecutable(candidateExes[j], info.GameDir, info.Name)
		return scoreI > scoreJ
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

	// Add sorted detected executables to LaunchCandidates
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
