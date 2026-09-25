package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/pipeline"
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
			GameDir: status.GameDir,
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

		if len(status.LaunchCandidates) == 0 {
			if manifestCandidates := manifest.DiscoverManifestExecutables(status.GameDir, numAppID, status.Platform); len(manifestCandidates) > 0 {
				status.LaunchCandidates = manifestCandidates
			}
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

	// Fetch remote title, metadata, news, and languages if AppID or metadata provider is present
	if (status.AppID != "" && status.AppID != "0") || status.HasCustomMetadata() {
		if eng.StoreRegistry != nil {
			providerName := status.EffectiveMetadataProvider()
			if providerName == "" || providerName == "standalone" {
				providerName = "steam"
			}
			if p, ok := eng.StoreRegistry.Get(providerName); ok {
				metaID := status.EffectiveMetadataID()
				if metaID == "" {
					metaID = status.AppID
				}
				if meta, err := p.GetMetadata(ctx, metaID); err == nil && meta != nil {
					if (status.Name == "" || status.Name == status.AppID || status.Name == filepath.Base(status.GameDir) || util.IsPlaceholderTitle(status.Name, status.AppID)) && meta.Name != "" {
						status.Name = meta.Name
						if status.LaunchProfile != nil && (status.LaunchProfile.Name == "" || util.IsPlaceholderTitle(status.LaunchProfile.Name, status.AppID)) {
							status.LaunchProfile.Name = status.Name
						}
						if status.InstallDir == "" || util.IsPlaceholderTitle(status.InstallDir, status.AppID) {
							status.InstallDir = util.SanitizeInstallDir(status.Name)
						}
					}
					if len(status.SupportedLanguages) <= 1 && len(meta.SupportedLanguages) > 0 {
						status.SupportedLanguages = meta.SupportedLanguages
					}
					if meta.Summary != "" {
						status.Summary = meta.Summary
					}
					if meta.ReleaseDate != "" {
						status.ReleaseDate = meta.ReleaseDate
					}
					if len(meta.Developers) > 0 {
						status.Developers = meta.Developers
					}
					if len(meta.Publishers) > 0 {
						status.Publishers = meta.Publishers
					}
					if len(meta.Genres) > 0 {
						status.Genres = meta.Genres
					}
				}
				if np, ok := p.(store.NewsProvider); ok {
					if news, err := np.GetNews(ctx, metaID, 1); err == nil && len(news) > 0 {
						status.RecentPatchNote = news[0].Title + ": " + news[0].Contents
					}
				}
			}
		}
	}

	// Enrich launch options from Steam if not already populated
	if len(status.LaunchCandidates) == 0 && status.NumericAppID > 0 {
		if steamOpts, err := steam.FetchAppLaunchOptions(ctx, status.NumericAppID, status.Platform, status.Branch); err == nil && len(steamOpts) > 0 {
			for _, opt := range steamOpts {
				cleanRel := util.NormalizeRelativePath(opt.Executable)
				if cleanRel == "" {
					continue
				}
				if _, err := os.Stat(filepath.Join(status.GameDir, cleanRel)); err == nil {
					status.LaunchCandidates = append(status.LaunchCandidates, domain.LaunchCandidate{
						Name:        opt.Name,
						Executable:  cleanRel,
						Arguments:   opt.Arguments,
						Description: opt.Description,
					})
				}
			}
			if len(status.LaunchCandidates) > 0 && status.LaunchProfile != nil && status.LaunchProfile.ExePath == "" && status.ExePath == "" {
				status.ExePath = status.LaunchCandidates[0].Executable
				status.ExeArgs = status.LaunchCandidates[0].Arguments
			}
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
	if newsCount > 1 && eng.StoreRegistry != nil {
		providerName := status.EffectiveMetadataProvider()
		if providerName == "" || providerName == "standalone" {
			providerName = "steam"
		}
		if p, ok := eng.StoreRegistry.Get(providerName); ok {
			if np, ok := p.(store.NewsProvider); ok {
				metaID := status.EffectiveMetadataID()
				if metaID == "" {
					metaID = status.AppID
				}
				if items, err := np.GetNews(ctx, metaID, newsCount); err == nil {
					status.NewsItems = items
				}
			}
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
		if !entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(absParent, entry.Name())
		if detector.IsIgnoredLibraryDir(fullPath) {
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

// VerifyIntegrity runs an on-demand full depot integrity comparison for a game directory via the Verify pipeline recipe.
func (eng *Engine) VerifyIntegrity(ctx context.Context, targetPath string) (*manifest.IntegrityReport, error) {
	ctx = eng.WrapContext(ctx)
	targetPath = util.CanonicalPath(targetPath)
	if targetPath == "" {
		targetPath = "."
	}

	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path: targetPath,
		},
		Game:     domain.NewGame("", targetPath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}

	pipe := pipeline.NewVerify()
	if err := pipe.Execute(ctx, pctx); err != nil {
		return nil, fmt.Errorf("verify pipeline failed: %w", err)
	}

	return pctx.Results.IntegrityReport, nil
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
		if detector.IsIgnoredLibraryDir(gameDir) {
			continue
		}
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

// SearchStoreGames searches across registered game store & metadata providers (Steam, GOG, RAWG).
func (eng *Engine) SearchStoreGames(ctx context.Context, query string) ([]store.SearchResult, error) {
	if eng.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return eng.StoreRegistry.SearchGameProviders(eng.WrapContext(ctx), query)
}


// GetGameMetadata fetches store metadata for a specific provider (steam, gog, rawg, etc.) and ID.
func (eng *Engine) GetGameMetadata(ctx context.Context, provider string, id string) (*store.Metadata, error) {
	cleanID := strings.TrimSpace(id)
	cleanProvider := strings.ToLower(strings.TrimSpace(provider))
	if cleanID == "" || cleanID == "0" || cleanProvider == "none" || cleanProvider == "standalone" {
		return nil, fmt.Errorf("no metadata for unlinked game")
	}

	if eng.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}

	if cleanProvider == "" {
		cleanProvider = "steam"
	}

	p, ok := eng.StoreRegistry.Get(cleanProvider)
	if !ok {
		return nil, fmt.Errorf("provider %s not found", cleanProvider)
	}

	meta, err := p.GetMetadata(eng.WrapContext(ctx), cleanID)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata for %s/%s: %w", cleanProvider, cleanID, err)
	}
	if meta == nil || meta.Name == "" {
		return nil, fmt.Errorf("metadata not found for provider %s and id %s", cleanProvider, cleanID)
	}

	return meta, nil
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
	profile, err := compat.LoadProfile(gameDir)
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
	_ = compat.SaveProfile(profile)

	return art, nil
}

// LinkGameMetadata links a game installation to a specific metadata provider and ID.
func (eng *Engine) LinkGameMetadata(ctx context.Context, gameDir, storeName, storeID string) (*domain.Game, error) {
	ctx = eng.WrapContext(ctx)
	profile, err := compat.LoadProfile(gameDir)
	if err != nil {
		profile = &domain.LaunchProfile{
			GameDir: gameDir,
		}
	}

	storeNameLower := strings.ToLower(strings.TrimSpace(storeName))
	profile.MetadataProvider = storeNameLower
	profile.MetadataID = strings.TrimSpace(storeID)

	switch storeNameLower {
	case "none", "standalone", "custom", "":
		profile.MetadataProvider = ""
		profile.MetadataID = ""
	case "steam":
		if eng.StoreRegistry != nil {
			if p, ok := eng.StoreRegistry.Get("steam"); ok {
				if meta, err := p.GetMetadata(ctx, profile.MetadataID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	case "gog":
		if eng.StoreRegistry != nil {
			if p, ok := eng.StoreRegistry.Get("gog"); ok {
				if meta, err := p.GetMetadata(ctx, profile.MetadataID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	default:
		// Generic metadata provider (e.g. RAWG, etc.)
		if eng.StoreRegistry != nil {
			if p, ok := eng.StoreRegistry.Get(storeNameLower); ok {
				if meta, err := p.GetMetadata(ctx, profile.MetadataID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	}

	if err := compat.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("save launch profile: %w", err)
	}
	_ = compat.SyncSettings(ctx, profile, eng.Config)

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
	if err := compat.UnlinkMetadata(gameDir); err != nil {
		return nil, fmt.Errorf("unlink metadata: %w", err)
	}

	status, err := eng.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return status, nil
}

// GetPatchNote fetches and retrieves a single patch note at itemIndex (1-indexed) for a game path or AppID.
func (eng *Engine) GetPatchNote(ctx context.Context, targetPath string, itemIndex int) (*domain.NewsItem, string, error) {
	ctx = eng.WrapContext(ctx)
	appID := targetPath
	gameName := targetPath
	providerName := "steam"

	if info, err := detector.Detect(ctx, targetPath); err == nil && info != nil {
		if info.AppID != "" && info.AppID != "0" {
			appID = info.AppID
		}
		if info.Name != "" {
			gameName = info.Name
		}
		if info.EffectiveMetadataProvider() != "" {
			providerName = info.EffectiveMetadataProvider()
		}
	}

	var items []domain.NewsItem
	if eng.StoreRegistry != nil {
		if p, ok := eng.StoreRegistry.Get(providerName); ok {
			if np, ok := p.(store.NewsProvider); ok {
				if fetched, err := np.GetNews(ctx, appID, itemIndex+5); err == nil {
					items = fetched
				}
			}
		}
	}

	if len(items) == 0 {
		return nil, gameName, fmt.Errorf("no news items found for %s", appID)
	}

	if itemIndex <= 0 || itemIndex > len(items) {
		return nil, gameName, fmt.Errorf("news item index %d out of range (1-%d available)", itemIndex, len(items))
	}

	return &items[itemIndex-1], gameName, nil
}

// DetectGame inspects a given game directory and returns detected game metadata.
func (eng *Engine) DetectGame(ctx context.Context, gamePath string) (*detector.GameInfo, error) {
	ctx = eng.WrapContext(ctx)
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return detector.Detect(ctx, gamePath)
}

// FetchNews retrieves news and patch note items for a Steam AppID.
func (eng *Engine) FetchNews(ctx context.Context, appID string, count int) ([]steam.NewsItem, error) {
	if appID == "" || appID == "0" {
		return nil, fmt.Errorf("invalid steam app id")
	}
	if count <= 0 {
		count = 5
	}
	return steam.FetchAppNewsItems(eng.WrapContext(ctx), appID, count)
}

// GetDiskSpace returns filesystem disk usage metrics (total, free, used bytes) for the given path.
func (eng *Engine) GetDiskSpace(path string) (*util.DiskSpaceInfo, error) {
	return util.GetDiskSpace(path)
}
