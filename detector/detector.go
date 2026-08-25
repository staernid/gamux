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

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// GameInfo is an alias for the canonical domain.Game entity.
type GameInfo = domain.Game

// DetectGame scans a target directory and returns a canonical domain.Game entity.
func DetectGame(ctx context.Context, path string) (*domain.Game, error) {
	return Detect(ctx, path)
}

var (
	appmanifestRegex  = regexp.MustCompile(`(?i)^appmanifest_(\d+)\.acf$`)
	numericAppIDRegex = regexp.MustCompile(`^\d+$`)
)

// Detect scans a target path and automatically identifies its AppID, Title, Platform, and Executable.
// Detect is strictly read-only and does not mutate files on disk.
func Detect(ctx context.Context, path string) (*GameInfo, error) {
	absPath := util.CanonicalPath(path)
	if absPath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrGameNotFound, absPath)
	}

	info := domain.NewGame("", absPath, "", "")

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

	if num, ok := util.ParseAppID(info.AppID); ok && num > 0 {
		info.NumericAppID = num
		info.ExternalIDs.SteamAppID = num
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
	if (info.Store == "Steam" || info.Source == "Steam") && info.AppID == "" && info.Name != "" {
		if appID, searchName, err := steam.SearchAppID(ctx, info.Name); err == nil && appID != "" {
			info.AppID = appID
			if num, ok := util.ParseAppID(appID); ok && num > 0 {
				info.NumericAppID = num
				info.ExternalIDs.SteamAppID = num
			}
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

	return info, nil
}

// GOGInfo holds parsed metadata from GOG installation descriptors.
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

func parseEpicInfo(gameDir string) (string, string) {
	egPath := filepath.Join(gameDir, ".egstore")
	entries, err := os.ReadDir(egPath)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".mancpn") {
			appName := strings.TrimSuffix(name, filepath.Ext(name))
			return appName, appName
		}
	}
	return "", ""
}

func detectStore(info *GameInfo) {
	// 1. Check for GOG signatures & parse GOG metadata FIRST
	if gogName, gogID := parseGOGInfo(info.GameDir); gogName != "" || gogID != "" {
		info.Store = "GOG"
		info.Source = domain.SourceGOG
		info.GOGID = gogID
		if gogName != "" && info.Name == "" {
			info.Name = gogName
		}
		return
	}

	if entries, err := os.ReadDir(info.GameDir); err == nil {
		for _, e := range entries {
			name := strings.ToLower(e.Name())
			if strings.HasPrefix(name, "goggame-") || strings.HasPrefix(name, "goglog") || name == "webcache.zip" || strings.HasPrefix(name, "gog") {
				info.Store = "GOG"
				info.Source = domain.SourceGOG
				return
			}
			if name == ".egstore" {
				info.Store = "Epic"
				info.Source = domain.SourceEpic
				if appName, _ := parseEpicInfo(info.GameDir); appName != "" {
					info.EpicApp = appName
				}
				return
			}
			if name == ".itch" || name == ".itch.toml" {
				info.Store = "Itch"
				info.Source = domain.SourceItch
				return
			}
		}
	}

	// 2. Check for concrete Steam artifacts on disk (manifests, appid.txt, or steam_api DLL/SO)
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
		info.Source = domain.SourceSteam
		return
	}

	// 3. Standalone DRM-free download (Factorio.com, Humble Bundle, direct site download)
	info.Store = "Standalone"
	info.Source = domain.SourceStandalone
}

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
	for p, cfg := range gbe.PlatformConfig {
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
		// Skip installers, crash handlers, redistributables, setup files, loader wrappers, console wrappers
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
			strings.Contains(name, "steamclient_loader") ||
			strings.Contains(name, "coldclientloader") ||
			strings.Contains(name, "steamless") ||
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
				baseLower := strings.ToLower(filepath.Base(option.Executable))
				if strings.Contains(baseLower, "steamclient_loader") || strings.Contains(baseLower, "coldclientloader") || strings.Contains(baseLower, "unitycrashhandler") {
					continue
				}
				fullPath := util.JoinPaths(info.GameDir, option.Executable)
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
		canExe := util.CanonicalPath(exe)
		if !seenExes[canExe] {
			baseNoExt := strings.TrimSuffix(filepath.Base(canExe), filepath.Ext(canExe))
			name := baseNoExt
			if strings.EqualFold(baseNoExt, dirName) || (info.Name != "" && strings.EqualFold(baseNoExt, info.Name)) {
				name = info.Name + " (Main Executable)"
			}
			info.LaunchCandidates = append(info.LaunchCandidates, LaunchCandidate{
				Name:       name,
				Executable: canExe,
			})
			seenExes[canExe] = true
		}
	}

	// Check steam_settings/launch_profile.json for configured target exe if present
	lpPath := filepath.Join(info.GameDir, "steam_settings", "launch_profile.json")
	if lpData, err := os.ReadFile(lpPath); err == nil {
		var lp struct {
			ExePath string `json:"exe_path"`
			ExeArgs string `json:"exe_args"`
		}
		if err := json.Unmarshal(lpData, &lp); err == nil && lp.ExePath != "" {
			realExe := lp.ExePath
			if !filepath.IsAbs(realExe) {
				realExe = util.JoinPaths(info.GameDir, realExe)
			} else {
				realExe = util.CanonicalPath(realExe)
			}
			realLower := strings.ToLower(filepath.Base(realExe))
			if util.FileExists(realExe) && !strings.Contains(realLower, "steamclient_loader") && !strings.Contains(realLower, "coldclientloader") {
				info.ExePath = realExe
				if lp.ExeArgs != "" {
					info.ExeArgs = lp.ExeArgs
				}
			}
		}
	}

	// Check ColdClientLoader.ini for configured target exe if present
	if info.ExePath == "" {
		iniPath := filepath.Join(info.GameDir, "ColdClientLoader.ini")
		if iniData, err := os.ReadFile(iniPath); err == nil {
			reIniExe := regexp.MustCompile(`(?i)^\s*Exe\s*=\s*(.+)`)
			for _, line := range strings.Split(string(iniData), "\n") {
				if m := reIniExe.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
					realExe := strings.TrimSpace(m[1])
					if !filepath.IsAbs(realExe) {
						realExe = util.JoinPaths(info.GameDir, realExe)
					} else {
						realExe = util.CanonicalPath(realExe)
					}
					realLower := strings.ToLower(filepath.Base(realExe))
					if util.FileExists(realExe) && !strings.Contains(realLower, "steamclient_loader") && !strings.Contains(realLower, "coldclientloader") {
						info.ExePath = realExe
						break
					}
				}
			}
		}
	}

	if info.ExePath == "" && len(info.LaunchCandidates) > 0 {
		info.ExePath = info.LaunchCandidates[0].Executable
		info.ExeArgs = info.LaunchCandidates[0].Arguments
	}

	// Final safeguard: never expose loader wrapper as main game ExePath
	if strings.Contains(strings.ToLower(info.ExePath), "steamclient_loader") || strings.Contains(strings.ToLower(info.ExePath), "coldclientloader") {
		info.ExePath = ""
		for _, cand := range info.LaunchCandidates {
			candBase := strings.ToLower(filepath.Base(cand.Executable))
			if !strings.Contains(candBase, "steamclient_loader") && !strings.Contains(candBase, "coldclientloader") {
				info.ExePath = cand.Executable
				break
			}
		}
	}

	// Update platform if executable is a Windows .exe and platform was defaulted to linux
	if info.ExePath != "" && strings.HasSuffix(strings.ToLower(info.ExePath), ".exe") && info.Platform == "linux" {
		info.Platform = "win64"
	}
}
