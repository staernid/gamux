package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

// InspectLocal performs fast static local inspection (executables, manifests, backups, configuration profile).
func (eng *Engine) InspectLocal(ctx context.Context, targetPath string) (*GameStatus, error) {
	ctx = eng.WrapContext(ctx)
	status, err := detector.Inspect(ctx, targetPath)
	if err != nil {
		return nil, err
	}

	if status.CurrentLanguage == "" && eng.Config != nil && eng.Config.Language != "" {
		status.CurrentLanguage = eng.Config.Language
	}

	if status.LaunchProfile == nil {
		status.LaunchProfile = &domain.LaunchProfile{
			GameDir:       status.GameDir,
			OperatingMode: runner.ModeStandalone,
		}
		if eng.Config != nil {
			if eng.Config.OperatingMode != "" {
				status.LaunchProfile.OperatingMode = eng.Config.OperatingMode
			}
			if eng.Config.DefaultRunner != "" {
				status.LaunchProfile.Runner = eng.Config.DefaultRunner
			}
			if eng.Config.WinePrefix != "" {
				status.LaunchProfile.WinePrefix = eng.Config.WinePrefix
			}
		}
	} else {
		if status.LaunchProfile.OperatingMode == "" && eng.Config != nil && eng.Config.OperatingMode != "" {
			status.LaunchProfile.OperatingMode = eng.Config.OperatingMode
		}
		if status.LaunchProfile.Runner == "" && eng.Config != nil && eng.Config.DefaultRunner != "" {
			status.LaunchProfile.Runner = eng.Config.DefaultRunner
		}
		if status.LaunchProfile.WinePrefix == "" && eng.Config != nil && eng.Config.WinePrefix != "" {
			status.LaunchProfile.WinePrefix = eng.Config.WinePrefix
		}
	}

	status.RecomputeState()

	numAppID, _ := util.ParseAppID(status.AppID)
	if numAppID > 0 {
		if depots, err := manifest.InspectGameDepots(status.GameDir, numAppID); err == nil && len(depots) > 0 {
			status.Depots = depots
			var platforms []string
			platSeen := make(map[string]bool)
			for _, d := range depots {
				var p string
				switch d.Category {
				case "windows":
					p = "Windows"
				case "linux":
					p = "Linux Native"
				case "macos":
					p = "macOS"
				}
				if p != "" && !platSeen[p] {
					platSeen[p] = true
					platforms = append(platforms, p)
				}
			}
			status.AvailablePlatforms = platforms
		}

		if manifestCandidates := manifest.DiscoverManifestExecutables(status.GameDir, numAppID, status.Platform); len(manifestCandidates) > 0 {
			status.LaunchCandidates = manifestCandidates
		}
	}

	if eng.ArtworkService != nil {
		_, _ = eng.ArtworkService.Resolve(ctx, status)
	}

	return status, nil
}


// InspectStatus inspects a game directory to determine its patch and configuration state, enriched with remote metadata.
func (eng *Engine) InspectStatus(ctx context.Context, targetPath string) (*GameStatus, error) {
	ctx = eng.WrapContext(ctx)
	status, err := eng.InspectLocal(ctx, targetPath)
	if err != nil {
		return nil, err
	}

	// Fetch remote Steam news and languages if AppID is present
	if status.AppID != "" && status.AppID != "0" {
		if news, err := steam.FetchAppNews(ctx, status.AppID); err == nil {
			status.RecentPatchNote = news
		}
		if langs, err := steam.FetchSupportedLanguages(ctx, status.AppID); err == nil {
			status.SupportedLanguages = langs
		}
	}

	return status, nil
}

// InspectStatusEx extends InspectStatus with customizable news history depth.
func (eng *Engine) InspectStatusEx(ctx context.Context, targetPath string, newsCount int) (*GameStatus, error) {
	ctx = eng.WrapContext(ctx)
	status, err := eng.InspectStatus(ctx, targetPath)
	if err != nil {
		return nil, err
	}
	if newsCount > 1 && status.AppID != "" && status.AppID != "0" {
		if items, err := steam.FetchAppNewsItems(ctx, status.AppID, newsCount); err == nil {
			status.NewsItems = items
		}
	}
	return status, nil
}

// BatchInspect scans all candidate game subdirectories in parentDir and returns their GameStatus.
func (eng *Engine) BatchInspect(ctx context.Context, parentDir string) ([]*GameStatus, error) {
	ctx = eng.WrapContext(ctx)
	absParent, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve parent directory path: %w", err)
	}

	entries, err := os.ReadDir(absParent)
	if err != nil {
		return nil, fmt.Errorf("read parent directory %s: %w", absParent, err)
	}

	var validPaths []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dirLower := strings.ToLower(entry.Name())
		if dirLower == "umu" || dirLower == "umu-default" || dirLower == "wine" || dirLower == "proton" || dirLower == "pfx" || dirLower == "prefix" || dirLower == "prefixes" || dirLower == "art" || dirLower == "backups" {
			continue
		}
		fullPath := filepath.Join(absParent, entry.Name())
		if util.FileExists(filepath.Join(fullPath, "drive_c")) && (util.FileExists(filepath.Join(fullPath, "system.reg")) || util.FileExists(filepath.Join(fullPath, "user.reg")) || util.FileExists(filepath.Join(fullPath, "dosdevices"))) {
			continue
		}
		validPaths = append(validPaths, fullPath)
	}

	results := make([]*GameStatus, len(validPaths))
	var g errgroup.Group
	g.SetLimit(16)

	for i, p := range validPaths {
		idx := i
		path := p
		g.Go(func() error {
			status, err := eng.InspectLocal(ctx, path)
			if err == nil && status != nil {
				results[idx] = status
			}
			return nil
		})
	}

	_ = g.Wait()

	var finalResults []*GameStatus
	for _, res := range results {
		if res != nil {
			finalResults = append(finalResults, res)
		}
	}
	return finalResults, nil
}

// BatchInspectLibraries scans multiple library parent directories concurrently and consolidates results.
func (eng *Engine) BatchInspectLibraries(ctx context.Context, libDirs []string) ([]*GameStatus, error) {
	ctx = eng.WrapContext(ctx)
	seen := make(map[string]bool)
	var consolidated []*GameStatus

	for _, dir := range libDirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		expanded := util.ExpandPath(trimmed)
		if !util.FileExists(expanded) {
			continue
		}
		statuses, err := eng.BatchInspect(ctx, expanded)
		if err != nil {
			slog.Warn("Failed to inspect library directory", "path", expanded, "error", err)
			continue
		}
		for _, s := range statuses {
			if s != nil && s.GameDir != "" && !seen[s.GameDir] {
				seen[s.GameDir] = true
				consolidated = append(consolidated, s)
			}
		}
	}

	return consolidated, nil
}

// VerifyIntegrity runs an on-demand full depot integrity comparison for a game directory.
func (eng *Engine) VerifyIntegrity(ctx context.Context, targetPath string) (*manifest.IntegrityReport, error) {
	ctx = eng.WrapContext(ctx)
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("detect game for integrity check: %w", err)
	}

	appIDVal, ok := util.ParseAppID(info.AppID)
	if !ok {
		return nil, fmt.Errorf("invalid AppID %q for integrity verification", info.AppID)
	}

	return manifest.ScanDepotIntegrityWithPlatform(info.GameDir, appIDVal, info.Platform)
}

// CheckLibraryUpdates scans a directory containing game folders and checks for available Steam updates.
func (eng *Engine) CheckLibraryUpdates(ctx context.Context, parentDir string) ([]LibraryUpdateStatus, error) {
	ctx = eng.WrapContext(ctx)
	if parentDir == "" {
		parentDir = "."
	}

	absPath, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve abs path: %w", err)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("read library directory %s: %w", absPath, err)
	}

	var results []LibraryUpdateStatus
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		gameDir := filepath.Join(absPath, entry.Name())
		manifestsDir := filepath.Join(gameDir, "[Manifests]")
		if _, err := os.Stat(manifestsDir); err != nil {
			continue
		}

		info, err := detector.Detect(ctx, gameDir)
		if err != nil || info.AppID == "" || info.AppID == "0" {
			continue
		}

		status := LibraryUpdateStatus{
			GameTitle: info.Name,
			AppID:     info.AppID,
			GameDir:   gameDir,
		}

		manifestEntries, _ := os.ReadDir(manifestsDir)
		for _, mFile := range manifestEntries {
			if strings.HasSuffix(mFile.Name(), ".manifest") {
				status.LocalManifestID = mFile.Name()
				break
			}
		}

		results = append(results, status)
	}

	return results, nil
}

func (eng *Engine) resolveGameName(ctx context.Context, path string) string {
	if info, err := detector.Detect(ctx, path); err == nil && info != nil && info.Name != "" {
		return info.Name
	}
	return filepath.Base(path)
}

// SearchStoreGames searches across registered game store providers (Steam, GOG).
func (eng *Engine) SearchStoreGames(ctx context.Context, query string) ([]store.SearchResult, error) {
	if eng.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return eng.StoreRegistry.SearchStores(eng.WrapContext(ctx), query)
}

// ResolveArtwork resolves covers, banners, heroes, and logos for a game.
func (eng *Engine) ResolveArtwork(ctx context.Context, game *domain.Game) (*domain.GameArtwork, error) {
	if eng.ArtworkService == nil {
		return nil, fmt.Errorf("artwork service not initialized")
	}
	return eng.ArtworkService.Resolve(eng.WrapContext(ctx), game)
}

// SearchArtworkCandidates queries registered providers (SteamGridDB, Steam, GOG) for artwork matching a title.
func (eng *Engine) SearchArtworkCandidates(ctx context.Context, query string) ([]store.SearchResult, error) {
	if eng.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return eng.StoreRegistry.SearchAll(eng.WrapContext(ctx), query)
}

// QueryArtworkOptions queries candidate providers for specific asset options (covers, heroes, banners, logos).
func (eng *Engine) QueryArtworkOptions(ctx context.Context, query domain.ArtworkQuery) ([]domain.ArtworkOption, error) {
	if eng.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return eng.StoreRegistry.QueryArtworkOptions(eng.WrapContext(ctx), query)
}

// ApplyCustomArtwork sets custom artwork on a game and persists the choice in the launch profile.
func (eng *Engine) ApplyCustomArtwork(ctx context.Context, gameDir string, assetType string, imageSource string) (*domain.GameArtwork, error) {
	ctx = eng.WrapContext(ctx)
	if eng.ArtworkService == nil {
		return nil, fmt.Errorf("artwork service not initialized")
	}

	status, err := eng.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, fmt.Errorf("inspect game: %w", err)
	}

	art, err := eng.ArtworkService.ApplyCustomArtwork(ctx, status, assetType, imageSource)
	if err != nil {
		return nil, err
	}

	// Persist to launch profile
	profile, err := runner.LoadLaunchProfile(gameDir)
	if err != nil {
		profile = &domain.LaunchProfile{GameDir: gameDir}
	}
	switch assetType {
	case "cover":
		profile.CustomCoverURL = imageSource
	case "banner":
		profile.CustomBannerURL = imageSource
	case "hero":
		profile.CustomHeroURL = imageSource
	case "logo":
		profile.CustomLogoURL = imageSource
	case "icon":
		profile.CustomIconURL = imageSource
	}
	profile.LockedArtwork = true
	_ = runner.SaveProfile(profile)

	return art, nil
}

// LinkGameMetadata links a game installation to a specific metadata provider and ID.
func (eng *Engine) LinkGameMetadata(ctx context.Context, gameDir, storeName, storeID string) (*domain.Game, error) {
	ctx = eng.WrapContext(ctx)
	profile, err := runner.LoadLaunchProfile(gameDir)
	if err != nil {
		profile = &domain.LaunchProfile{
			GameDir: gameDir,
		}
	}

	storeNameLower := strings.ToLower(strings.TrimSpace(storeName))
	profile.MetadataProvider = storeNameLower
	profile.MetadataID = strings.TrimSpace(storeID)
	profile.Store = storeName

	switch storeNameLower {
	case "standalone", "custom", "none", "":
		profile.MetadataProvider = "standalone"
		profile.MetadataID = ""
		profile.AppID = "0"
		profile.StoreAppID = ""
		profile.GOGID = ""
		profile.Store = domain.StoreStandalone
		profile.Name = filepath.Base(gameDir)
	case "steam":
		profile.AppID = profile.MetadataID
		profile.StoreAppID = profile.MetadataID
		profile.GOGID = ""
		// Try fetching Steam game name if empty
		if name, err := steam.FetchAppName(ctx, profile.MetadataID); err == nil && name != "" {
			profile.Name = name
		}
	case "gog":
		profile.GOGID = profile.MetadataID
		profile.StoreAppID = ""
		profile.AppID = "0"
		if eng.StoreRegistry != nil {
			if p, ok := eng.StoreRegistry.Get("gog"); ok {
				if meta, err := p.GetMetadata(ctx, profile.MetadataID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	default:
		// Generic metadata provider (e.g. RAWG, PCGW, etc.)
		profile.AppID = "0"
		profile.StoreAppID = ""
		profile.GOGID = ""
		if eng.StoreRegistry != nil {
			if p, ok := eng.StoreRegistry.Get(storeNameLower); ok {
				if meta, err := p.GetMetadata(ctx, profile.MetadataID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	}

	if err := runner.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("save launch profile: %w", err)
	}
	_ = runner.SyncLaunchProfile(ctx, profile, eng.Config)

	status, err := eng.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return status, nil
}

// ListMetadataProviders returns descriptors for all registered metadata and artwork providers.
func (eng *Engine) ListMetadataProviders(ctx context.Context) []store.ProviderDescriptor {
	if eng.StoreRegistry == nil {
		return []store.ProviderDescriptor{}
	}
	return eng.StoreRegistry.ListDescriptors()
}

// UnlinkGameMetadata removes custom metadata bindings and restores auto-detected game identity.
func (eng *Engine) UnlinkGameMetadata(ctx context.Context, gameDir string) (*domain.Game, error) {
	ctx = eng.WrapContext(ctx)
	if err := runner.UnlinkMetadata(gameDir); err != nil {
		return nil, fmt.Errorf("unlink metadata: %w", err)
	}

	status, err := eng.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return status, nil
}

// GetPatchNote fetches and retrieves a single patch note at itemIndex (1-indexed) for a game path or AppID.
func (eng *Engine) GetPatchNote(ctx context.Context, targetPath string, itemIndex int) (*steam.NewsItem, string, error) {
	ctx = eng.WrapContext(ctx)
	appID := targetPath
	gameName := targetPath

	if info, err := detector.Detect(ctx, targetPath); err == nil && info.AppID != "" && info.AppID != "0" {
		appID = info.AppID
		gameName = info.Name
	}

	items, err := steam.FetchAppNewsItems(ctx, appID, itemIndex+5)
	if err != nil {
		return nil, gameName, fmt.Errorf("fetch news: %w", err)
	}

	if itemIndex <= 0 || itemIndex > len(items) {
		return nil, gameName, fmt.Errorf("news item index %d out of range (1-%d available)", itemIndex, len(items))
	}

	return &items[itemIndex-1], gameName, nil
}
