package saves

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
)

// Resolver resolves game save and configuration locations on disk across Linux native, Wine/Proton, and Goldberg emulator.
type Resolver struct {
	manifestMgr *ManifestManager
}

// NewResolver creates an initialized save location Resolver.
func NewResolver(manifestMgr *ManifestManager) *Resolver {
	return &Resolver{
		manifestMgr: manifestMgr,
	}
}

// ResolveOptions encapsulates game and environment parameters for save resolution.
type ResolveOptions struct {
	AppID      string
	GameName   string
	GameDir    string
	Platform   string
	WinePrefix string
	GbeBaseDir string
}

// Resolve identifies all on-disk save, config, and cloud storage directories for a game.
func (r *Resolver) Resolve(ctx context.Context, opts ResolveOptions) (*SaveLocationsResult, error) {
	absGameDir := util.CanonicalPath(opts.GameDir)
	appIDNum, _ := util.ParseAppID(opts.AppID)
	platform := strings.ToLower(opts.Platform)
	if platform == "" {
		platform = "win64"
	}

	result := &SaveLocationsResult{
		GameDir:   absGameDir,
		GameName:  opts.GameName,
		AppID:     opts.AppID,
		Platform:  platform,
		Locations: make([]SaveLocation, 0),
	}

	seenPaths := make(map[string]bool)

	addLocation := func(loc SaveLocation) {
		clean := filepath.Clean(loc.Path)
		if clean == "" || seenPaths[clean] {
			return
		}
		seenPaths[clean] = true
		loc.Path = clean
		loc.DisplayPath = util.DisplayPath(clean)

		// Calculate size and item counts if existing
		if util.FileExists(clean) {
			loc.Exists = true
			fi, err := os.Stat(clean)
			if err == nil {
				loc.IsDir = fi.IsDir()
				if loc.IsDir {
					loc.FileCount, loc.SizeBytes = calculateDirMetrics(clean)
				} else {
					loc.FileCount = 1
					loc.SizeBytes = fi.Size()
				}
			}
		}

		result.Locations = append(result.Locations, loc)
	}

	// 1. Goldberg Steamworks Cloud Storage (~/.local/share/GSE Saves/<appid>/remote)
	if appIDNum > 0 {
		homeDir, _ := os.UserHomeDir()
		gseRemoteDir := filepath.Join(homeDir, ".local", "share", "GSE Saves", fmt.Sprintf("%d", appIDNum), "remote")
		gseStatsDir := filepath.Join(homeDir, ".local", "share", "GSE Saves", fmt.Sprintf("%d", appIDNum), "stats")

		if util.FileExists(gseRemoteDir) {
			addLocation(SaveLocation{
				Path:     gseRemoteDir,
				Category: CategoryCloud,
				Tags:     []string{"save", "cloud"},
				Source:   "gbe",
			})
			result.HasCloudSaves = true
		}
		if util.FileExists(gseStatsDir) {
			addLocation(SaveLocation{
				Path:     gseStatsDir,
				Category: CategoryConfig,
				Tags:     []string{"stats", "config"},
				Source:   "gbe",
			})
		}
	}

	// 2. Portable Saves in Game Directory (<base>/saves, <base>/save, <base>/profile)
	if absGameDir != "" {
		for _, cand := range []string{"saves", "save", "profiles", "profile", "UserData"} {
			p := filepath.Join(absGameDir, cand)
			if util.FileExists(p) {
				addLocation(SaveLocation{
					Path:     p,
					Category: CategoryPortable,
					Tags:     []string{"save", "portable"},
					Source:   "portable",
				})
			}
		}
	}

	// 3. Ludusavi Manifest Rules Resolution
	if r.manifestMgr != nil {
		if gameRule, found := r.manifestMgr.FindGame(appIDNum, opts.GameName, 0); found && gameRule != nil {
			winePrefix := r.resolveEffectiveWinePrefix(opts.WinePrefix, opts.AppID, opts.GameName)
			wineUser := r.detectWineUser(winePrefix)

			for pathTemplate, fileRule := range gameRule.Files {
				if !r.isRuleApplicable(fileRule, platform) {
					continue
				}

				category := CategorySave
				if len(fileRule.Tags) > 0 {
					if containsTag(fileRule.Tags, "config") && !containsTag(fileRule.Tags, "save") {
						category = CategoryConfig
					}
				}

				expandedPaths := r.expandPlaceholder(pathTemplate, absGameDir, opts.AppID, winePrefix, wineUser, platform)
				for _, expPath := range expandedPaths {
					if expPath == "" {
						continue
					}
					// Handle wildcards if template contains '*'
					if strings.Contains(expPath, "*") {
						matches, _ := filepath.Glob(expPath)
						for _, m := range matches {
							addLocation(SaveLocation{
								Path:     m,
								Category: category,
								Tags:     fileRule.Tags,
								Source:   "ludusavi",
							})
						}
					} else {
						// Case-insensitive traversal on disk if path doesn't directly exist
						resolvedPath := expPath
						if !util.FileExists(resolvedPath) {
							resolvedPath = caseInsensitiveResolve(expPath)
						}
						addLocation(SaveLocation{
							Path:     resolvedPath,
							Category: category,
							Tags:     fileRule.Tags,
							Source:   "ludusavi",
						})
					}
				}
			}
		}
	}

	// 4. Heuristic Fallbacks for Native Linux / Wine if no manifest matches were found
	if len(result.Locations) == 0 && opts.GameName != "" {
		homeDir, _ := os.UserHomeDir()
		slug := util.Slugify(opts.GameName)

		if platform == "linux" {
			for _, base := range []string{
				filepath.Join(homeDir, ".config", opts.GameName),
				filepath.Join(homeDir, ".config", slug),
				filepath.Join(homeDir, ".local", "share", opts.GameName),
				filepath.Join(homeDir, ".local", "share", slug),
			} {
				if util.FileExists(base) {
					addLocation(SaveLocation{
						Path:     base,
						Category: CategorySave,
						Source:   "heuristic",
					})
				}
			}
		} else {
			winePrefix := r.resolveEffectiveWinePrefix(opts.WinePrefix, opts.AppID, opts.GameName)
			wineUser := r.detectWineUser(winePrefix)
			userDir := filepath.Join(winePrefix, "drive_c", "users", wineUser)

			for _, base := range []string{
				filepath.Join(userDir, "AppData", "Roaming", opts.GameName),
				filepath.Join(userDir, "AppData", "Local", opts.GameName),
				filepath.Join(userDir, "AppData", "LocalLow", opts.GameName),
				filepath.Join(userDir, "Saved Games", opts.GameName),
				filepath.Join(userDir, "Documents", "My Games", opts.GameName),
				filepath.Join(userDir, "Documents", opts.GameName),
			} {
				if resolved := caseInsensitiveResolve(base); util.FileExists(resolved) {
					addLocation(SaveLocation{
						Path:     resolved,
						Category: CategorySave,
						Source:   "wine_prefix",
					})
				}
			}
		}
	}

	// Compute summary totals & primary paths
	for _, loc := range result.Locations {
		if loc.Exists {
			if loc.Category == CategorySave || loc.Category == CategoryPortable || loc.Category == CategoryCloud {
				result.TotalSaveBytes += loc.SizeBytes
				if result.PrimarySavePath == "" {
					result.PrimarySavePath = loc.Path
				}
			} else if loc.Category == CategoryConfig {
				result.TotalConfigFiles += loc.FileCount
				if result.PrimaryConfigPath == "" {
					result.PrimaryConfigPath = loc.Path
				}
			}
		}
	}

	return result, nil
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, target) {
			return true
		}
	}
	return false
}

func (r *Resolver) isRuleApplicable(rule ManifestFileRule, platform string) bool {
	if len(rule.When) == 0 {
		return true
	}
	for _, constraint := range rule.When {
		if constraint.OS != "" {
			reqOS := strings.ToLower(constraint.OS)
			if platform == "linux" && reqOS == "linux" {
				return true
			}
			if (platform == "win64" || platform == "win32") && reqOS == "windows" {
				return true
			}
		}
		if constraint.Store != "" && strings.EqualFold(constraint.Store, "steam") {
			return true
		}
	}
	return false
}

// resolveEffectiveWinePrefix discovers the candidate Wine/Proton prefix on disk in priority order.
func (r *Resolver) resolveEffectiveWinePrefix(customPrefix, appID, gameName string) string {
	home, _ := os.UserHomeDir()

	if customPrefix != "" {
		exp := util.ExpandPath(customPrefix)
		if util.FileExists(exp) {
			return exp
		}
	}

	// 1. Shared standard UMU prefix
	umuDefault := filepath.Join(home, "Games", "umu", "umu-default")
	if util.FileExists(umuDefault) {
		return umuDefault
	}

	// 2. Per-game UMU prefix
	if appID != "" && appID != "0" {
		umuApp := filepath.Join(home, "Games", "umu", appID)
		if util.FileExists(umuApp) {
			return umuApp
		}
	}

	// 3. Standard Wine prefixes directory
	if gameName != "" {
		slug := util.Slugify(gameName)
		wineName := filepath.Join(home, ".local", "share", "wineprefixes", slug)
		if util.FileExists(wineName) {
			return wineName
		}
	}

	// 4. Default Wine prefix
	wineDefault := filepath.Join(home, ".local", "share", "wineprefixes", "default")
	if util.FileExists(wineDefault) {
		return wineDefault
	}

	// Fallback to shared UMU default path
	return umuDefault
}

// detectWineUser finds the active user profile name inside a Wine prefix.
func (r *Resolver) detectWineUser(winePrefix string) string {
	usersDir := filepath.Join(winePrefix, "drive_c", "users")
	if !util.FileExists(usersDir) {
		if u := os.Getenv("USER"); u != "" {
			return u
		}
		return "steamuser"
	}

	entries, err := os.ReadDir(usersDir)
	if err != nil {
		if u := os.Getenv("USER"); u != "" {
			return u
		}
		return "steamuser"
	}

	var candidates []string
	systemUsers := map[string]bool{
		"public": true, "default": true, "default user": true, "all users": true,
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		nameLower := strings.ToLower(entry.Name())
		if !systemUsers[nameLower] {
			candidates = append(candidates, entry.Name())
		}
	}

	currentLinuxUser := os.Getenv("USER")
	for _, c := range candidates {
		if currentLinuxUser != "" && strings.EqualFold(c, currentLinuxUser) {
			return c
		}
	}
	for _, c := range candidates {
		if strings.EqualFold(c, "steamuser") {
			return c
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	if currentLinuxUser != "" {
		return currentLinuxUser
	}
	return "steamuser"
}

// expandPlaceholder replaces Ludusavi 16 semantic placeholders with concrete on-disk absolute paths.
func (r *Resolver) expandPlaceholder(template, gameDir, appID, winePrefix, wineUser, platform string) []string {
	home, _ := os.UserHomeDir()
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	// Direct Steam Cloud / Userdata mapping for Goldberg
	if strings.Contains(template, "<root>/userdata") || strings.Contains(template, "<root>") {
		var results []string
		if appID != "" && appID != "0" {
			gseRemote := filepath.Join(dataHome, "GSE Saves", appID, "remote")
			results = append(results, gseRemote)
		}
		// Also expand Steam userdata standard path if present
		steamUserdata := filepath.Join(dataHome, "Steam", "userdata")
		if util.FileExists(steamUserdata) {
			p := strings.ReplaceAll(template, "<root>", filepath.Join(dataHome, "Steam"))
			p = strings.ReplaceAll(p, "<storeUserId>", "*")
			p = strings.ReplaceAll(p, "<storeGameId>", appID)
			results = append(results, p)
		}
		if len(results) > 0 {
			return results
		}
	}

	p := template

	// Common Base & Identity Placeholders
	p = strings.ReplaceAll(p, "<base>", gameDir)
	p = strings.ReplaceAll(p, "<game>", filepath.Base(gameDir))
	p = strings.ReplaceAll(p, "<osUserName>", wineUser)
	p = strings.ReplaceAll(p, "<storeGameId>", appID)
	p = strings.ReplaceAll(p, "<storeUserId>", "*")

	if platform == "linux" {
		p = strings.ReplaceAll(p, "<home>", home)
		p = strings.ReplaceAll(p, "<xdgConfig>", configHome)
		p = strings.ReplaceAll(p, "<xdgData>", dataHome)
		p = strings.ReplaceAll(p, "<winAppData>", configHome)
		p = strings.ReplaceAll(p, "<winLocalAppData>", dataHome)
		p = strings.ReplaceAll(p, "<winLocalAppDataLow>", filepath.Join(configHome, "unity3d"))
		p = strings.ReplaceAll(p, "<winDocuments>", filepath.Join(home, "Documents"))
		p = strings.ReplaceAll(p, "<winSavedGames>", dataHome)
		p = strings.ReplaceAll(p, "<winProgramData>", filepath.Join(dataHome, "programdata"))
		p = strings.ReplaceAll(p, "<winDir>", "/tmp")
		p = strings.ReplaceAll(p, "<winPublic>", filepath.Join(home, "Public"))
	} else {
		// Windows PE via Wine/Proton Prefix
		userProfile := filepath.Join(winePrefix, "drive_c", "users", wineUser)
		p = strings.ReplaceAll(p, "<home>", userProfile)
		p = strings.ReplaceAll(p, "<winAppData>", filepath.Join(userProfile, "AppData", "Roaming"))
		p = strings.ReplaceAll(p, "<winLocalAppData>", filepath.Join(userProfile, "AppData", "Local"))
		p = strings.ReplaceAll(p, "<winLocalAppDataLow>", filepath.Join(userProfile, "AppData", "LocalLow"))
		p = strings.ReplaceAll(p, "<winDocuments>", filepath.Join(userProfile, "Documents"))
		p = strings.ReplaceAll(p, "<winSavedGames>", filepath.Join(userProfile, "Saved Games"))
		p = strings.ReplaceAll(p, "<winProgramData>", filepath.Join(winePrefix, "drive_c", "ProgramData"))
		p = strings.ReplaceAll(p, "<winDir>", filepath.Join(winePrefix, "drive_c", "windows"))
		p = strings.ReplaceAll(p, "<winPublic>", filepath.Join(winePrefix, "drive_c", "users", "Public"))
		p = strings.ReplaceAll(p, "<xdgConfig>", filepath.Join(userProfile, "AppData", "Roaming"))
		p = strings.ReplaceAll(p, "<xdgData>", filepath.Join(userProfile, "AppData", "Local"))
	}

	p = filepath.Clean(p)
	return []string{p}
}

// caseInsensitiveResolve steps through a path on disk segment-by-segment matching directory entries case-insensitively.
func caseInsensitiveResolve(targetPath string) string {
	if util.FileExists(targetPath) {
		return targetPath
	}

	parts := strings.Split(filepath.Clean(targetPath), string(os.PathSeparator))
	if len(parts) == 0 {
		return targetPath
	}

	curr := "/"
	if !filepath.IsAbs(targetPath) {
		curr = "."
	}

	for _, part := range parts {
		if part == "" || part == "/" {
			continue
		}
		if strings.Contains(part, "*") {
			curr = filepath.Join(curr, part)
			continue
		}

		candidate := filepath.Join(curr, part)
		if util.FileExists(candidate) {
			curr = candidate
			continue
		}

		entries, err := os.ReadDir(curr)
		if err != nil {
			return targetPath
		}

		matched := false
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				curr = filepath.Join(curr, entry.Name())
				matched = true
				break
			}
		}

		if !matched {
			curr = filepath.Join(curr, part)
		}
	}

	return curr
}

// calculateDirMetrics computes recursive file count and total byte size for a directory.
func calculateDirMetrics(path string) (int, int64) {
	var count int
	var size int64

	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() {
			count++
			size += info.Size()
		}
		return nil
	})

	return count, size
}
