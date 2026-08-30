package gui

import (
	"fmt"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/util"
)

// -----------------------------------------------------------------------------
// Library & Game Inspection
// -----------------------------------------------------------------------------

// InspectLibraries scans one or more library parent directories concurrently and consolidates results.
func (a *App) InspectLibraries(dirs []string) ([]*engine.GameStatus, error) {
	if len(dirs) == 0 {
		return []*engine.GameStatus{}, nil
	}
	cleanDirs := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if c := util.CanonicalPath(d); c != "" {
			cleanDirs = append(cleanDirs, c)
		}
	}
	if len(cleanDirs) == 0 {
		return nil, fmt.Errorf("library directories cannot be empty")
	}
	return a.Engine.BatchInspectLibraries(a.getContext(), cleanDirs)
}

// InspectGame inspects a game directory and returns its full GameStatus, optionally fetching news/patch notes.
func (a *App) InspectGame(gamePath string, newsCount int) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if newsCount > 0 {
		return a.Engine.InspectStatusEx(a.getContext(), gamePath, newsCount)
	}
	return a.Engine.InspectStatus(a.getContext(), gamePath)
}

// DetectGame inspects a given game directory and returns detected game metadata.
func (a *App) DetectGame(gamePath string) (*detector.GameInfo, error) {
	if a.Engine != nil {
		return a.Engine.DetectGame(a.getContext(), gamePath)
	}
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return detector.Detect(a.getContext(), gamePath)
}

// VerifyIntegrity runs full depot integrity verification for a game directory on-demand.
func (a *App) VerifyIntegrity(gamePath string) (*manifest.IntegrityReport, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.VerifyIntegrity(a.getContext(), gamePath)
}

// -----------------------------------------------------------------------------
// Game Execution & Lifecycle
// -----------------------------------------------------------------------------

// LaunchOptions alias for runner.LaunchOptions.
type LaunchOptions = runner.LaunchOptions

// LaunchGame starts the game process asynchronously according to the supplied launch options or detected game profile.
func (a *App) LaunchGame(opts LaunchOptions) error {
	_, err := a.Engine.LaunchGame(a.getContext(), opts)
	return err
}

// QuickPlay executes one-click setup and launch for a game.
func (a *App) QuickPlay(opts LaunchOptions) error {
	_, err := a.Engine.EnsureAndLaunchGame(a.getContext(), opts)
	return err
}

// IsGameRunning checks if a game process in the given game directory is currently active.
func (a *App) IsGameRunning(gamePath string) bool {
	if a.Engine != nil && a.Engine.RunnerMgr != nil {
		return a.Engine.RunnerMgr.IsRunning(gamePath)
	}
	return runner.DefaultManager().IsRunning(gamePath)
}

// StopGame terminates an active game process and its process tree.
func (a *App) StopGame(gamePath string) error {
	if a.Engine != nil && a.Engine.RunnerMgr != nil {
		return a.Engine.RunnerMgr.Stop(gamePath)
	}
	return runner.DefaultManager().Stop(gamePath)
}

// GetAvailableRunners returns list of detected execution runners on the host.
func (a *App) GetAvailableRunners() []string {
	if a.Engine != nil {
		return a.Engine.GetAvailableRunners()
	}
	return runner.GetAvailableRunners()
}

// GetGameLog returns the stdout/stderr log content from the current or last execution of a game.
func (a *App) GetGameLog(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.GetGameLog(a.getContext(), gamePath)
}

// GetGameAchievements returns list of achievements and unlocked status for a game.
func (a *App) GetGameAchievements(gamePath string) ([]domain.AchievementItem, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.GetGameAchievements(a.getContext(), gamePath)
}

// -----------------------------------------------------------------------------
// Configuration & Launch Profiles
// -----------------------------------------------------------------------------

// GetLaunchProfile retrieves the custom launch configuration for a game directory.
func (a *App) GetLaunchProfile(gamePath string) (*runner.LaunchOptions, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.GetLaunchProfile(a.getContext(), gamePath)
}

// SaveLaunchProfile persists custom launch configuration for a game directory and returns updated GameStatus.
func (a *App) SaveLaunchProfile(gamePath string, opts runner.LaunchOptions) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if err := a.Engine.SaveLaunchProfile(a.getContext(), gamePath, opts); err != nil {
		return nil, err
	}
	return a.Engine.InspectStatus(a.getContext(), gamePath)
}

// ResetLaunchProfile removes custom launch configuration and returns updated GameStatus.
func (a *App) ResetLaunchProfile(gamePath string) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if err := a.Engine.ResetLaunchProfile(a.getContext(), gamePath); err != nil {
		return nil, err
	}
	return a.Engine.InspectStatus(a.getContext(), gamePath)
}

// GetGameDepots retrieves structured metadata and installation state for all depots of a game.
func (a *App) GetGameDepots(gamePath string) ([]domain.DepotSummary, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.GetGameDepots(a.getContext(), gamePath)
}

// SwitchGamePlatform switches the active execution platform and runner for a game directory.
func (a *App) SwitchGamePlatform(gamePath, targetPlatform string) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if err := a.Engine.SwitchGamePlatform(a.getContext(), gamePath, targetPlatform); err != nil {
		return nil, err
	}
	return a.Engine.InspectStatus(a.getContext(), gamePath)
}

// UninstallDepot removes on-disk files belonging to a specific depot ID while preserving shared assets and manifests.
func (a *App) UninstallDepot(gamePath string, depotID uint32) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if err := a.Engine.UninstallDepot(a.getContext(), gamePath, depotID); err != nil {
		return nil, err
	}
	return a.Engine.InspectStatus(a.getContext(), gamePath)
}

// -----------------------------------------------------------------------------
// Store, Metadata, Artwork & News
// -----------------------------------------------------------------------------

// SearchStoreGames queries all configured store providers (Steam, GOG, SteamGridDB) for game titles.
func (a *App) SearchStoreGames(query string) ([]store.SearchResult, error) {
	return a.Engine.SearchStoreGames(a.getContext(), query)
}

// ResolveArtwork resolves covers, heroes, and banners for a game directory.
func (a *App) ResolveArtwork(gameDir string) (*domain.GameArtwork, error) {
	gameDir = util.CanonicalPath(gameDir)
	info, err := a.DetectGame(gameDir)
	if err != nil {
		return nil, err
	}
	return a.Engine.ResolveArtwork(a.getContext(), info)
}

// QueryArtworkOptions queries candidate providers for specific asset options (covers, heroes, banners, logos).
func (a *App) QueryArtworkOptions(query domain.ArtworkQuery) ([]domain.ArtworkOption, error) {
	query.GameDir = util.CanonicalPath(query.GameDir)
	return a.Engine.QueryArtworkOptions(a.getContext(), query)
}

// ApplyCustomArtwork sets custom artwork on a game directory and updates metadata.
func (a *App) ApplyCustomArtwork(gameDir string, assetType string, imageSource string) (*domain.GameArtwork, error) {
	gameDir = util.CanonicalPath(gameDir)
	return a.Engine.ApplyCustomArtwork(a.getContext(), gameDir, assetType, imageSource)
}

// LinkGameMetadata links a game directory to a specific store provider and ID.
func (a *App) LinkGameMetadata(gameDir string, storeName string, storeID string) (*domain.Game, error) {
	gameDir = util.CanonicalPath(gameDir)
	return a.Engine.LinkGameMetadata(a.getContext(), gameDir, storeName, storeID)
}

// UnlinkGameMetadata removes custom metadata overrides and restores auto-detected identity.
func (a *App) UnlinkGameMetadata(gameDir string) (*domain.Game, error) {
	gameDir = util.CanonicalPath(gameDir)
	return a.Engine.UnlinkGameMetadata(a.getContext(), gameDir)
}

// ListMetadataProviders returns registered metadata and artwork providers with their capabilities.
func (a *App) ListMetadataProviders() ([]store.ProviderDescriptor, error) {
	return a.Engine.ListMetadataProviders(a.getContext()), nil
}

// GameMetadataDetails holds rich game synopsis and metadata for the About view.
type GameMetadataDetails struct {
	Name                string   `json:"name"`
	ShortDescription    string   `json:"short_description"`
	DetailedDescription string   `json:"detailed_description"`
	AboutTheGame        string   `json:"about_the_game"`
	HeaderImage         string   `json:"header_image"`
	Developers          []string `json:"developers"`
	Publishers          []string `json:"publishers"`
	Genres              []string `json:"genres"`
	ReleaseDate         string   `json:"release_date"`
}

// GetGameDetails fetches store metadata and game description for a given AppID or store.
func (a *App) GetGameDetails(appID string) (*GameMetadataDetails, error) {
	if appID == "" || appID == "0" {
		return nil, fmt.Errorf("invalid app id")
	}

	if a.Engine == nil || a.Engine.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}

	// 1. Try Steam provider first for numeric or Steam AppIDs
	if p, ok := a.Engine.StoreRegistry.Get("steam"); ok {
		if meta, err := p.GetMetadata(a.getContext(), appID); err == nil && meta != nil && meta.Name != "" {
			return &GameMetadataDetails{
				Name:                meta.Name,
				ShortDescription:    meta.Summary,
				DetailedDescription: meta.Summary,
				AboutTheGame:        meta.Summary,
				HeaderImage:         meta.HeaderImage,
				Developers:          meta.Developers,
				Publishers:          meta.Publishers,
				Genres:              meta.Genres,
				ReleaseDate:         meta.ReleaseDate,
			}, nil
		}
	}

	// 2. Try remaining registered store/metadata providers (GOG, RAWG, etc.)
	for _, p := range a.Engine.StoreRegistry.List() {
		if p.ID() == "steam" || p.ID() == "steamgriddb" {
			continue
		}
		if meta, err := p.GetMetadata(a.getContext(), appID); err == nil && meta != nil && meta.Name != "" {
			return &GameMetadataDetails{
				Name:                meta.Name,
				ShortDescription:    meta.Summary,
				DetailedDescription: meta.Summary,
				AboutTheGame:        meta.Summary,
				HeaderImage:         meta.HeaderImage,
				Developers:          meta.Developers,
				Publishers:          meta.Publishers,
				Genres:              meta.Genres,
				ReleaseDate:         meta.ReleaseDate,
			}, nil
		}
	}

	return nil, fmt.Errorf("metadata not found for id %s", appID)
}

// FetchNews retrieves news and patch note items for a Steam AppID.
func (a *App) FetchNews(appID string, count int) ([]steam.NewsItem, error) {
	if a.Engine != nil {
		return a.Engine.FetchNews(a.getContext(), appID, count)
	}
	if appID == "" || appID == "0" {
		return nil, fmt.Errorf("invalid steam app id")
	}
	if count <= 0 {
		count = 5
	}
	return steam.FetchAppNewsItems(a.getContext(), appID, count)
}

// GetPatchNote fetches a specific patch note for a game.
func (a *App) GetPatchNote(gamePath string, itemIndex int) (*steam.NewsItem, string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.GetPatchNote(a.getContext(), gamePath, itemIndex)
}
