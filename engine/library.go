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

type LibraryService struct {
	sc *ServiceContext
}

func NewLibraryService(sc *ServiceContext) *LibraryService {
	return &LibraryService{sc: sc}
}

// InspectLocal performs fast static local inspection (executables, manifests, backups, configuration profile).
func (ls *LibraryService) InspectLocal(ctx context.Context, targetPath string) (*GameStatus, error) {
	ctx = ls.sc.WrapContext(ctx)
	status, err := detector.Inspect(ctx, targetPath)
	if err != nil {
		return nil, err
	}

	if status.CurrentLanguage == "" && ls.sc.Config != nil && ls.sc.Config.Language != "" {
		status.CurrentLanguage = ls.sc.Config.Language
	}

	if status.OperatingMode == "" {
		if ls.sc.Config != nil && ls.sc.Config.OperatingMode != "" {
			status.OperatingMode = ls.sc.Config.OperatingMode
		} else {
			status.OperatingMode = runner.ModeStandalone
		}
	}
	if status.Runner == "" {
		if ls.sc.Config != nil && ls.sc.Config.DefaultRunner != "" {
			status.Runner = ls.sc.Config.DefaultRunner
		}
	}
	if status.WinePrefix == "" {
		if ls.sc.Config != nil {
			status.WinePrefix = ls.sc.Config.WinePrefix
		}
	}

	status.RecomputeState()

	if ls.sc.ArtworkService != nil {
		_, _ = ls.sc.ArtworkService.Resolve(ctx, status)
	}

	return status, nil
}

// InspectStatus inspects a game directory to determine its patch and configuration state, enriched with remote metadata.
func (ls *LibraryService) InspectStatus(ctx context.Context, targetPath string) (*GameStatus, error) {
	ctx = ls.sc.WrapContext(ctx)
	status, err := ls.InspectLocal(ctx, targetPath)
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
func (ls *LibraryService) InspectStatusEx(ctx context.Context, targetPath string, newsCount int) (*GameStatus, error) {
	ctx = ls.sc.WrapContext(ctx)
	status, err := ls.InspectStatus(ctx, targetPath)
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
func (ls *LibraryService) BatchInspect(ctx context.Context, parentDir string) ([]*GameStatus, error) {
	ctx = ls.sc.WrapContext(ctx)
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
		validPaths = append(validPaths, filepath.Join(absParent, entry.Name()))
	}

	results := make([]*GameStatus, len(validPaths))
	var g errgroup.Group
	g.SetLimit(16)

	for i, p := range validPaths {
		idx := i
		path := p
		g.Go(func() error {
			status, err := ls.InspectLocal(ctx, path)
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
func (ls *LibraryService) BatchInspectLibraries(ctx context.Context, libDirs []string) ([]*GameStatus, error) {
	ctx = ls.sc.WrapContext(ctx)
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
		statuses, err := ls.BatchInspect(ctx, expanded)
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
func (ls *LibraryService) VerifyIntegrity(ctx context.Context, targetPath string) (*manifest.IntegrityReport, error) {
	ctx = ls.sc.WrapContext(ctx)
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("detect game for integrity check: %w", err)
	}

	appIDVal, ok := util.ParseAppID(info.AppID)
	if !ok {
		return nil, fmt.Errorf("invalid AppID %q for integrity verification", info.AppID)
	}

	return manifest.ScanDepotIntegrity(info.GameDir, appIDVal)
}

// CheckLibraryUpdates scans a directory containing game folders and checks for available Steam updates.
func (ls *LibraryService) CheckLibraryUpdates(ctx context.Context, parentDir string) ([]LibraryUpdateStatus, error) {
	ctx = ls.sc.WrapContext(ctx)
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

func (ls *LibraryService) resolveGameName(ctx context.Context, path string) string {
	if info, err := detector.Detect(ctx, path); err == nil && info != nil && info.Name != "" {
		return info.Name
	}
	return filepath.Base(path)
}

// SearchStoreGames searches across registered game store providers (Steam, GOG).
func (ls *LibraryService) SearchStoreGames(ctx context.Context, query string) ([]store.SearchResult, error) {
	if ls.sc.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return ls.sc.StoreRegistry.SearchStores(ls.sc.WrapContext(ctx), query)
}

// ResolveArtwork resolves covers, banners, heroes, and logos for a game.
func (ls *LibraryService) ResolveArtwork(ctx context.Context, game *domain.Game) (*domain.GameArtwork, error) {
	if ls.sc.ArtworkService == nil {
		return nil, fmt.Errorf("artwork service not initialized")
	}
	return ls.sc.ArtworkService.Resolve(ls.sc.WrapContext(ctx), game)
}

// SearchArtworkCandidates queries registered providers (SteamGridDB, Steam, GOG) for artwork matching a title.
func (ls *LibraryService) SearchArtworkCandidates(ctx context.Context, query string) ([]store.SearchResult, error) {
	if ls.sc.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return ls.sc.StoreRegistry.SearchAll(ls.sc.WrapContext(ctx), query)
}

// QueryArtworkOptions queries candidate providers for specific asset options (covers, heroes, banners, logos).
func (ls *LibraryService) QueryArtworkOptions(ctx context.Context, query domain.ArtworkQuery) ([]domain.ArtworkOption, error) {
	if ls.sc.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return ls.sc.StoreRegistry.QueryArtworkOptions(ls.sc.WrapContext(ctx), query)
}

// ApplyCustomArtwork sets custom artwork on a game and persists the choice in the launch profile.
func (ls *LibraryService) ApplyCustomArtwork(ctx context.Context, gameDir string, assetType string, imageSource string) (*domain.GameArtwork, error) {
	ctx = ls.sc.WrapContext(ctx)
	if ls.sc.ArtworkService == nil {
		return nil, fmt.Errorf("artwork service not initialized")
	}

	status, err := ls.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, fmt.Errorf("inspect game: %w", err)
	}

	art, err := ls.sc.ArtworkService.ApplyCustomArtwork(ctx, status, assetType, imageSource)
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
func (ls *LibraryService) LinkGameMetadata(ctx context.Context, gameDir, storeName, storeID string) (*domain.Game, error) {
	ctx = ls.sc.WrapContext(ctx)
	profile, err := runner.LoadLaunchProfile(gameDir)
	if err != nil {
		profile = &domain.LaunchProfile{
			GameDir: gameDir,
		}
	}

	profile.Store = storeName
	storeNameLower := strings.ToLower(storeName)
	switch storeNameLower {
	case "steam":
		profile.AppID = storeID
		profile.StoreAppID = storeID
		// Try fetching Steam game name if empty
		if name, err := steam.FetchAppName(ctx, storeID); err == nil && name != "" {
			profile.Name = name
		}
	case "gog":
		profile.GOGID = storeID
		if ls.sc.StoreRegistry != nil {
			if p, ok := ls.sc.StoreRegistry.Get("gog"); ok {
				if meta, err := p.GetMetadata(ctx, storeID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	case "standalone", "custom":
		profile.AppID = "0"
		profile.StoreAppID = ""
		profile.GOGID = ""
		profile.Name = filepath.Base(gameDir)
	}

	if err := runner.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("save launch profile: %w", err)
	}
	_ = runner.SyncLaunchProfile(ctx, profile, ls.sc.Config)

	status, err := ls.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return status, nil
}

// GetPatchNote fetches and retrieves a single patch note at itemIndex (1-indexed) for a game path or AppID.
func (ls *LibraryService) GetPatchNote(ctx context.Context, targetPath string, itemIndex int) (*steam.NewsItem, string, error) {
	ctx = ls.sc.WrapContext(ctx)
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
