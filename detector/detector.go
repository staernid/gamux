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
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// GameInfo is an alias for the canonical domain.Game entity.
type GameInfo = domain.Game

// DetectGame scans a target directory and returns a canonical domain.Game entity.
// DetectGame scans a target directory and returns a canonical domain.Game entity.
func DetectGame(ctx context.Context, path string) (*domain.Game, error) {
	return Inspect(ctx, path)
}

// Detect scans a target path and automatically identifies its metadata and launch configuration.
// Detect is strictly read-only and does not mutate files on disk (alias for Inspect).
func Detect(ctx context.Context, path string) (*GameInfo, error) {
	return Inspect(ctx, path)
}

var (
	appmanifestRegex  = regexp.MustCompile(`(?i)^appmanifest_(\d+)\.acf$`)
	numericAppIDRegex = regexp.MustCompile(`^\d+$`)
)

type scanState struct {
	fileCount              int
	diskSizeBytes          int64
	originalBackups        []string
	steamAppIDTxtFound     bool
	appIDFromTxt           string
	settingsDirExists      bool
	currentLanguage        string
	targetLibPath          string
	detectedPlatform       string
	manifestPath           string
	manifestACFData        *ACFData
	manifestID             string
	rawDepotLayout         bool
	coldClientLoaderExists bool
	coldClientLoaderExe    string
	gogInfoFound           bool
	gogName                string
	gogID                  string
	epicFound              bool
	epicApp                string
	itchFound              bool
	candidateExes          []string
}

// GOGInfo holds parsed metadata from GOG installation descriptors.
type GOGInfo struct {
	GameID string `json:"gameId"`
	Name   string `json:"name"`
	Title  string `json:"title"`
}

// Inspect performs a comprehensive single-pass filesystem inspection of a game directory,
// extracting identity, platform, executables, backups, manifest integrity markers, and launch configuration.
func Inspect(ctx context.Context, path string) (*GameInfo, error) {
	absPath := util.CanonicalPath(path)
	if absPath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrGameNotFound, absPath)
	}

	// Guard against scanning runner prefix roots or system folders as games
	if isRunnerPrefixDir(absPath) && !util.FileExists(filepath.Join(absPath, "steam_settings", "launch_profile.json")) {
		return nil, fmt.Errorf("path %s is a runner/wine prefix directory, not a game", absPath)
	}

	var state scanState

	// Single-pass filesystem traversal
	walkErr := filepath.WalkDir(absPath, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		name := d.Name()
		lowerName := strings.ToLower(name)

		if d.IsDir() {
			if p != absPath && (lowerName == "drive_c" || lowerName == "dosdevices" || lowerName == "pfx" || lowerName == ".git" || lowerName == ".gamux_backup" || lowerName == "node_modules") {
				return filepath.SkipDir
			}
			if lowerName == "steam_settings" {
				state.settingsDirExists = true
			}
			if lowerName == ".egstore" {
				state.epicFound = true
			}
			if lowerName == ".itch" {
				state.itchFound = true
			}
			// Check for raw Steam depot layout: steamapps/common/<installdir>
			if lowerName == "common" && strings.HasSuffix(filepath.ToSlash(p), "steamapps/common") {
				if entries, err := os.ReadDir(p); err == nil && len(entries) > 0 {
					state.rawDepotLayout = true
				}
			}
			return nil
		}

		// File metrics
		state.fileCount++
		if fi, err := d.Info(); err == nil {
			state.diskSizeBytes += fi.Size()
		}

		// Backup libraries
		if strings.Contains(name, ".ORIGINAL") {
			state.originalBackups = append(state.originalBackups, p)
		}

		// steam_appid.txt
		if lowerName == "steam_appid.txt" {
			state.steamAppIDTxtFound = true
			if state.appIDFromTxt == "" {
				if data, err := os.ReadFile(p); err == nil {
					state.appIDFromTxt = strings.TrimSpace(string(data))
				}
			}
		}

		// steam_settings/force_language.txt
		if lowerName == "force_language.txt" && strings.Contains(p, "steam_settings") {
			if data, err := os.ReadFile(p); err == nil {
				state.currentLanguage = strings.TrimSpace(string(data))
			}
		}

		// ColdClientLoader.ini
		if lowerName == "coldclientloader.ini" {
			state.coldClientLoaderExists = true
			if data, err := os.ReadFile(p); err == nil {
				reIniExe := regexp.MustCompile(`(?i)^\s*Exe\s*=\s*(.+)`)
				for _, line := range strings.Split(string(data), "\n") {
					if m := reIniExe.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
						realExe := strings.TrimSpace(m[1])
						if !filepath.IsAbs(realExe) {
							realExe = util.JoinPaths(absPath, realExe)
						} else {
							realExe = util.CanonicalPath(realExe)
						}
						state.coldClientLoaderExe = realExe
						break
					}
				}
			}
		}

		// ACF Manifests: appmanifest_(\d+).acf
		if appmanifestRegex.MatchString(name) {
			isHigherPriority := state.manifestPath == "" ||
				strings.Contains(p, "[Manifests]") ||
				(strings.Contains(p, "[Steam]") && !strings.Contains(state.manifestPath, "[Manifests]")) ||
				(strings.Contains(p, "steamapps") && !strings.Contains(state.manifestPath, "[Manifests]") && !strings.Contains(state.manifestPath, "[Steam]"))

			if isHigherPriority {
				if data, err := ParseACFFile(p); err == nil {
					state.manifestPath = p
					state.manifestACFData = data
				}
			}
		}

		// .manifest files inside [Manifests]
		if strings.HasSuffix(lowerName, ".manifest") && strings.Contains(p, "[Manifests]") {
			if state.manifestID == "" {
				parts := strings.Split(name, "_")
				if len(parts) >= 2 {
					state.manifestID = strings.TrimSuffix(parts[1], ".manifest")
				} else {
					state.manifestID = name
				}
			}
		}

		// GBE Target Libs (for platform identification)
		if state.targetLibPath == "" {
			for plat, cfg := range gbe.PlatformConfig {
				if name == cfg.Target {
					state.detectedPlatform = plat
					state.targetLibPath = p
					break
				}
			}
		}

		// GOG store files
		if strings.HasPrefix(lowerName, "goggame-") && strings.HasSuffix(lowerName, ".info") {
			state.gogInfoFound = true
			if data, err := os.ReadFile(p); err == nil {
				var gog GOGInfo
				if err := json.Unmarshal(data, &gog); err == nil {
					state.gogID = gog.GameID
					state.gogName = gog.Name
					if state.gogName == "" {
						state.gogName = gog.Title
					}
				}
			}
		} else if strings.HasPrefix(lowerName, "goggame-") || strings.HasPrefix(lowerName, "goglog") || lowerName == "webcache.zip" || strings.HasPrefix(lowerName, "gog") {
			state.gogInfoFound = true
		}

		// Epic store files
		if strings.HasSuffix(lowerName, ".mancpn") && strings.Contains(p, ".egstore") {
			state.epicFound = true
			state.epicApp = strings.TrimSuffix(name, filepath.Ext(name))
		}

		// Itch store files
		if lowerName == ".itch.toml" {
			state.itchFound = true
		}

		// Executable candidates
		if strings.Contains(lowerName, "unitycrashhandler") ||
			strings.Contains(lowerName, "unins") ||
			strings.Contains(lowerName, "setup") ||
			strings.Contains(lowerName, "redist") ||
			strings.Contains(lowerName, "dxsetup") ||
			strings.Contains(lowerName, "vcredist") ||
			strings.Contains(lowerName, "crashdumper") ||
			strings.Contains(lowerName, "bssndrpt") ||
			strings.Contains(lowerName, "bugreport") ||
			strings.Contains(lowerName, "errorreport") ||
			strings.Contains(lowerName, "steamclient_loader") ||
			strings.Contains(lowerName, "coldclientloader") ||
			strings.Contains(lowerName, "steamless") ||
			strings.HasSuffix(lowerName, ".console.exe") {
			return nil
		}

		if strings.HasSuffix(lowerName, ".exe") || strings.HasSuffix(lowerName, ".sh") || strings.HasSuffix(lowerName, ".x86_64") || !strings.Contains(name, ".") {
			fileInfo, err := d.Info()
			if err == nil && (strings.HasSuffix(lowerName, ".exe") || fileInfo.Mode()&0111 != 0) {
				state.candidateExes = append(state.candidateExes, p)
			}
		}

		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("scan directory %s: %w", absPath, walkErr)
	}

	info := domain.NewGame("", absPath, "", "")

	// 1. Populate file metrics & backup/config markers
	info.FileCount = state.fileCount
	info.DiskSizeBytes = state.diskSizeBytes
	info.OriginalBackups = state.originalBackups
	info.SteamAppIDTxtFound = state.steamAppIDTxtFound
	info.SettingsDirExists = state.settingsDirExists
	info.CurrentLanguage = state.currentLanguage
	info.TargetLibPath = state.targetLibPath
	info.ManifestPath = state.manifestPath
	info.ManifestID = state.manifestID
	info.RawDepotLayout = state.rawDepotLayout

	// 2. Process ACF Data
	if state.manifestACFData != nil {
		info.AppID = state.manifestACFData.AppID
		info.Name = state.manifestACFData.Name
		info.InstallDir = state.manifestACFData.InstallDir

		if len(state.manifestACFData.LaunchOptions) > 0 {
			for _, option := range state.manifestACFData.LaunchOptions {
				if option.Executable != "" {
					baseLower := strings.ToLower(filepath.Base(option.Executable))
					if strings.Contains(baseLower, "steamclient_loader") || strings.Contains(baseLower, "coldclientloader") || strings.Contains(baseLower, "unitycrashhandler") {
						continue
					}
					fullPath := util.JoinPaths(info.GameDir, option.Executable)
					if _, err := os.Stat(fullPath); err == nil {
						candName := option.Description
						if candName == "" {
							candName = filepath.Base(fullPath)
						}
						info.LaunchCandidates = append(info.LaunchCandidates, domain.LaunchCandidate{
							Name:        candName,
							Executable:  fullPath,
							Arguments:   option.Arguments,
							Description: option.Description,
						})
					}
				}
			}
		}
	}

	// 3. Resolve AppID
	if info.AppID == "" {
		if state.appIDFromTxt != "" {
			info.AppID = state.appIDFromTxt
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

	// 4. Resolve Platform
	if state.detectedPlatform != "" {
		info.Platform = state.detectedPlatform
	} else {
		info.Platform = "linux"
	}

	// 5. Detect Store Origin
	if state.gogInfoFound || state.gogID != "" || state.gogName != "" {
		info.Store = domain.StoreGOG
		info.Source = domain.SourceGOG
		info.GOGID = state.gogID
		if state.gogName != "" && info.Name == "" {
			info.Name = state.gogName
		}
	} else if state.epicFound || state.epicApp != "" {
		info.Store = domain.StoreEpic
		info.Source = domain.SourceEpic
		info.EpicApp = state.epicApp
	} else if state.itchFound {
		info.Store = domain.StoreItch
		info.Source = domain.SourceItch
	} else if info.ManifestPath != "" || info.TargetLibPath != "" || info.ManifestID != "" || info.SteamAppIDTxtFound || info.NumericAppID > 0 {
		info.Store = domain.StoreSteam
		info.Source = domain.SourceSteam
	} else {
		info.Store = domain.StoreStandalone
		info.Source = domain.SourceStandalone
	}

	// 6. Resolve Title
	if info.AppID != "" && (info.Name == "" || info.Name == info.AppID) {
		if name, err := steam.FetchAppName(ctx, info.AppID); err == nil && name != "" {
			info.Name = name
		}
	}
	if info.Name == "" {
		info.Name = filepath.Base(info.GameDir)
	}
	if (info.Store == domain.StoreSteam || info.Source == domain.SourceSteam) && info.AppID == "" && info.Name != "" {
		if appID, searchName, err := steam.SearchAppID(ctx, info.Name); err == nil && appID != "" {
			if util.IsTitleMatch(searchName, info.Name) {
				info.AppID = appID
				if num, ok := util.ParseAppID(appID); ok && num > 0 {
					info.NumericAppID = num
					info.ExternalIDs.SteamAppID = num
				}
				if searchName != "" {
					info.Name = searchName
				}
				slog.Debug("Auto-discovered AppID via Steam Store search", "title", info.Name, "appID", info.AppID)
			} else {
				slog.Debug("Steam search candidate rejected due to title mismatch", "query", info.Name, "candidate", searchName, "candidateAppID", appID)
			}
		}
	}

	// 7. Resolve InstallDir
	if info.InstallDir == "" && info.Name != "" {
		info.InstallDir = util.SanitizeInstallDir(info.Name)
	}

	// 8. Executable detection & scoring
	sort.SliceStable(state.candidateExes, func(i, j int) bool {
		scoreI := scoreExecutable(state.candidateExes[i], info.GameDir, info.Name)
		scoreJ := scoreExecutable(state.candidateExes[j], info.GameDir, info.Name)
		return scoreI > scoreJ
	})

	seenExes := make(map[string]bool)
	for _, cand := range info.LaunchCandidates {
		seenExes[cand.Executable] = true
	}

	dirName := filepath.Base(info.GameDir)
	for _, exe := range state.candidateExes {
		canExe := util.CanonicalPath(exe)
		if !seenExes[canExe] {
			baseNoExt := strings.TrimSuffix(filepath.Base(canExe), filepath.Ext(canExe))
			candName := baseNoExt
			if strings.EqualFold(baseNoExt, dirName) || (info.Name != "" && strings.EqualFold(baseNoExt, info.Name)) {
				candName = info.Name + " (Main Executable)"
			}
			info.LaunchCandidates = append(info.LaunchCandidates, domain.LaunchCandidate{
				Name:       candName,
				Executable: canExe,
			})
			seenExes[canExe] = true
		}
	}

	// Use ColdClientLoader.ini Exe if present
	if state.coldClientLoaderExe != "" && util.FileExists(state.coldClientLoaderExe) {
		realLower := strings.ToLower(filepath.Base(state.coldClientLoaderExe))
		if !strings.Contains(realLower, "steamclient_loader") && !strings.Contains(realLower, "coldclientloader") {
			info.ExePath = state.coldClientLoaderExe
		}
	}

	// 9. Load custom LaunchProfile if present
	launchProfilePath := filepath.Join(info.GameDir, "steam_settings", "launch_profile.json")
	launchProfileExists := util.FileExists(launchProfilePath)
	if profile, err := runner.LoadLaunchProfile(info.GameDir); err == nil && profile != nil {
		info.LaunchProfile = profile
		if profile.SourceOverride != "" {
			info.Source = profile.SourceOverride
		}
		if profile.MetadataProvider != "" {
			info.Store = profile.MetadataProvider
		} else if launchProfileExists && profile.Store != "" && !strings.EqualFold(profile.Store, string(info.Source)) {
			info.Store = profile.Store
		}
		if profile.MetadataID != "" {
			if strings.EqualFold(profile.MetadataProvider, "steam") {
				info.AppID = profile.MetadataID
				if num, ok := util.ParseAppID(profile.MetadataID); ok && num > 0 {
					info.NumericAppID = num
					info.ExternalIDs.SteamAppID = num
				}
			} else if strings.EqualFold(profile.MetadataProvider, "gog") {
				info.ExternalIDs.GOGID = profile.MetadataID
			} else if strings.EqualFold(profile.MetadataProvider, "rawg") {
				info.ExternalIDs.RAWGID = profile.MetadataID
			}
		}
		if launchProfileExists && profile.StoreAppID != "" {
			info.AppID = profile.StoreAppID
			info.NumericAppID, _ = util.ParseAppID(profile.StoreAppID)
			info.ExternalIDs.SteamAppID = info.NumericAppID
		}
		if profile.GOGID != "" {
			info.ExternalIDs.GOGID = profile.GOGID
		}
		if profile.SGDBID != "" {
			info.ExternalIDs.SGDBID = profile.SGDBID
		}
		if profile.CustomCoverURL != "" {
			info.CustomCoverURL = profile.CustomCoverURL
		}
		if profile.CustomBannerURL != "" {
			info.CustomBannerURL = profile.CustomBannerURL
		}
		if profile.CustomHeroURL != "" {
			info.CustomHeroURL = profile.CustomHeroURL
		}
		if profile.CustomLogoURL != "" {
			info.CustomLogoURL = profile.CustomLogoURL
		}
		if profile.CustomIconURL != "" {
			info.CustomIconURL = profile.CustomIconURL
		}
		if profile.LockedArtwork {
			info.LockedArtwork = true
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

	// 10. Recompute state
	info.RecomputeState()

	return info, nil
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

func isRunnerPrefixDir(path string) bool {
	baseLower := strings.ToLower(filepath.Base(path))
	if baseLower == "umu" || baseLower == "umu-default" || baseLower == "wine" || baseLower == "proton" || baseLower == "pfx" || baseLower == "prefix" || baseLower == "prefixes" {
		return true
	}
	// Prefix root check: contains drive_c and (system.reg or user.reg or dosdevices or tracked_files)
	if util.FileExists(filepath.Join(path, "drive_c")) && (util.FileExists(filepath.Join(path, "system.reg")) || util.FileExists(filepath.Join(path, "user.reg")) || util.FileExists(filepath.Join(path, "dosdevices"))) {
		return true
	}
	return false
}
