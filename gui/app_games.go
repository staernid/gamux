package gui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/util"
	"github.com/staernid/gamux/util/trash"
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
	res, err := a.Engine.BatchInspectLibraries(a.getContext(), cleanDirs)
	if err == nil && len(res) > 0 {
		a.UpdateCachedGames(res)
	}
	return res, err
}

// InspectGame inspects a game directory and returns its full GameStatus, optionally fetching news/patch notes.
func (a *App) InspectGame(gamePath string, newsCount int) (*engine.GameStatus, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	var st *engine.GameStatus
	if newsCount > 0 {
		st, err = a.Engine.InspectStatusEx(a.getContext(), path, newsCount)
	} else {
		st, err = a.Engine.InspectStatus(a.getContext(), path)
	}
	if err == nil && st != nil {
		a.UpdateCachedGame(st)
	}
	return st, err
}

// DetectGame inspects a given game directory and returns detected game metadata.
func (a *App) DetectGame(gamePath string) (*detector.GameInfo, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.DetectGame(a.getContext(), gamePath)
}

// VerifyIntegrity runs full depot integrity verification for a game directory on-demand.
func (a *App) VerifyIntegrity(gamePath string) (*manifest.IntegrityReport, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	return a.Engine.VerifyIntegrity(a.getContext(), path)
}

// VerifyGameIntegrity alias for VerifyIntegrity to support deep deck operations.
func (a *App) VerifyGameIntegrity(gamePath string) (*manifest.IntegrityReport, error) {
	return a.VerifyIntegrity(gamePath)
}

// StartVerifyGameTask initiates background depot integrity verification and returns the task ID.
func (a *App) StartVerifyGameTask(gamePath string) (string, error) {
	return a.SubmitVerify(gamePath)
}

// KillAllWine forces termination of any running wine processes or wineserver instances.
func (a *App) KillAllWine() error {
	_ = exec.Command("wineserver", "-k").Run()
	_ = exec.Command("pkill", "-9", "-f", "wineserver|wine|proton").Run()
	return nil
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

// IsGameRunning checks if a game process in the given game directory is currently active
// across single-player runner and multi-seat supervisor.
func (a *App) IsGameRunning(gamePath string) bool {
	cPath := util.CanonicalPath(gamePath)
	if cPath == "" {
		return false
	}
	if a.Engine != nil && a.Engine.RunnerMgr != nil && a.Engine.RunnerMgr.IsRunning(cPath) {
		return true
	}
	if a.multiseatSupervisor().IsSessionRunning(cPath) {
		return true
	}
	return false
}

// StopGame terminates an active game process or multi-seat session.
func (a *App) StopGame(gamePath string) error {
	cPath := util.CanonicalPath(gamePath)
	if cPath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	if a.multiseatSupervisor().IsSessionRunning(cPath) {
		return a.multiseatSupervisor().StopSession(cPath)
	}
	if a.Engine == nil || a.Engine.RunnerMgr == nil {
		return fmt.Errorf("engine not initialized")
	}
	return a.Engine.RunnerMgr.Stop(cPath)
}

// StopGameInstance terminates an individual running instance (single-player or a specific multi-seat seat).
func (a *App) StopGameInstance(instanceID string) error {
	if instanceID == "" {
		return fmt.Errorf("instanceID cannot be empty")
	}
	gameDir, profileID := domain.ParseInstanceID(instanceID)
	cDir := util.CanonicalPath(gameDir)

	// Check if this instance is running in a multi-seat session
	msSup := a.multiseatSupervisor()
	if msSup.IsSessionRunning(cDir) {
		status := msSup.GetStatus(cDir)
		if status != nil && status.Seats != nil {
			for seatID, st := range status.Seats {
				if st.ProfileID == profileID || seatID == profileID {
					return msSup.StopSeat(cDir, seatID)
				}
			}
		}
		// If profile match wasn't found but seatID matched instance directly
		if err := msSup.StopSeat(cDir, profileID); err == nil {
			return nil
		}
		return msSup.StopSession(cDir)
	}

	// Single-player instance
	if a.Engine != nil && a.Engine.RunnerMgr != nil {
		return a.Engine.RunnerMgr.StopInstance(instanceID)
	}
	return fmt.Errorf("engine not initialized")
}

// GetRunningInstances returns all active execution instances (solo sessions and multi-seat seats).
func (a *App) GetRunningInstances() []domain.RunningInstance {
	var result []domain.RunningInstance
	seen := make(map[string]bool)

	// Solo instances from RunnerMgr
	if a.Engine != nil && a.Engine.RunnerMgr != nil {
		for _, inst := range a.Engine.RunnerMgr.RunningInstances() {
			if !seen[inst.InstanceID] {
				seen[inst.InstanceID] = true
				result = append(result, inst)
			}
		}
	}

	// Multiseat seat instances
	for _, inst := range a.multiseatSupervisor().GetRunningInstances() {
		if !seen[inst.InstanceID] {
			seen[inst.InstanceID] = true
			result = append(result, inst)
		}
	}

	return result
}

// GetRunningGames returns merged, deduplicated canonical paths of all active single-player and multi-seat games.
func (a *App) GetRunningGames() []string {
	seen := make(map[string]bool)
	var result []string

	if a.Engine != nil && a.Engine.RunnerMgr != nil {
		for _, dir := range a.Engine.RunnerMgr.RunningGames() {
			c := util.CanonicalPath(dir)
			if c != "" && !seen[c] {
				seen[c] = true
				result = append(result, c)
			}
		}
	}

	for _, dir := range a.multiseatSupervisor().RunningSessions() {
		c := util.CanonicalPath(dir)
		if c != "" && !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}

	return result
}

// GetAvailableRunners returns list of detected execution runners on the host.
func (a *App) GetAvailableRunners() []string {
	if a.Engine != nil {
		return a.Engine.GetAvailableRunners()
	}
	return nil
}

// GetGameLog returns the stdout/stderr log content from the current or last execution of a game.
func (a *App) GetGameLog(gamePath string) (string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return "", err
	}
	return a.Engine.GetGameLog(a.getContext(), path)
}

// GetGameAchievements returns list of achievements and unlocked status for a game.
func (a *App) GetGameAchievements(gamePath string) ([]domain.AchievementItem, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	return a.Engine.GetGameAchievements(a.getContext(), path)
}

// GetGameCurrentPlayers returns the real-time concurrent player count for a Steam AppID.
func (a *App) GetGameCurrentPlayers(appID string) (int, error) {
	parsed, ok := util.ParseAppID(appID)
	if !ok || parsed == 0 {
		return 0, fmt.Errorf("invalid appID %q", appID)
	}
	return steam.FetchCurrentPlayers(a.getContext(), parsed)
}

// LibraryUpdateReport summarizes the results of a library-wide update inspection.
type LibraryUpdateReport struct {
	TotalChecked int                  `json:"total_checked"`
	UpdatesFound int                  `json:"updates_found"`
	UpdatedGames []*engine.GameStatus `json:"updated_games"`
}

// CheckGameUpdates inspects a game's update status using hybrid version checking.
func (a *App) CheckGameUpdates(gamePath string) (*domain.VersionStatus, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	vs, err := a.Engine.CheckGameUpdates(a.getContext(), path)
	if err == nil && vs != nil {
		if st, sErr := a.Engine.InspectLocal(a.getContext(), path); sErr == nil && st != nil {
			st.VersionStatus = vs
			st.HasUpdate = vs.Checked && !vs.IsUpToDate
			if vs.RemoteManifestID != "" {
				st.RemoteManifestID = vs.RemoteManifestID
			}
			st.RecomputeState()
			a.UpdateCachedGame(st)
			safeEmit(a.getContext(), "gamux:game-updated", map[string]any{"game": st})
		}
	}
	return vs, err
}

// CheckLibraryUpdates inspects all installed games in the library for updates.
func (a *App) CheckLibraryUpdates() (*LibraryUpdateReport, error) {
	updated, err := a.Engine.CheckLibraryUpdates(a.getContext())
	if err != nil {
		return nil, err
	}
	for _, g := range updated {
		if g != nil {
			a.UpdateCachedGame(g)
			safeEmit(a.getContext(), "gamux:game-updated", map[string]any{"game": g})
		}
	}
	total := len(updated)
	a.gamesMu.RLock()
	if len(a.cachedGames) > total {
		total = len(a.cachedGames)
	}
	a.gamesMu.RUnlock()

	return &LibraryUpdateReport{
		TotalChecked: total,
		UpdatesFound: len(updated),
		UpdatedGames: updated,
	}, nil
}

// GetGameDLCs retrieves configured DLCs or discovers available DLCs for a game.
func (a *App) GetGameDLCs(gamePath string) ([]domain.DLCItem, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	return a.Engine.GetGameDLCs(a.getContext(), path)
}

// FetchDLCs fetches DLC definitions from Steam and writes them into configs.app.ini.
func (a *App) FetchDLCs(gamePath string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}
	return a.Engine.FetchGameDLCs(a.getContext(), path)
}

// -----------------------------------------------------------------------------
// Configuration & Launch Profiles
// -----------------------------------------------------------------------------

func (a *App) mutateAndInspect(gamePath string, mutate func(cleanPath string) error) (*engine.GameStatus, error) {
	cleanPath, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	if err := mutate(cleanPath); err != nil {
		return nil, err
	}
	st, err := a.Engine.InspectStatus(a.getContext(), cleanPath)
	if err == nil && st != nil {
		a.UpdateCachedGame(st)
	}
	return st, err
}

// PreviewGameLaunch resolves and returns the dry-run execution specification and active overrides for a game.
func (a *App) PreviewGameLaunch(gamePath string) (*runner.LaunchPreview, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	return a.Engine.PreviewLaunch(a.getContext(), path)
}

// GetLaunchProfile retrieves the custom launch configuration for a game directory.
func (a *App) GetLaunchProfile(gamePath string) (*runner.LaunchOptions, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	return a.Engine.GetLaunchProfile(a.getContext(), path)
}

// SaveLaunchProfile persists custom launch configuration for a game directory and returns updated GameStatus.
func (a *App) SaveLaunchProfile(gamePath string, opts runner.LaunchOptions) (*engine.GameStatus, error) {
	return a.mutateAndInspect(gamePath, func(p string) error {
		return a.Engine.SaveLaunchProfile(a.getContext(), p, opts)
	})
}

// ResetLaunchProfile removes custom launch configuration and returns updated GameStatus.
func (a *App) ResetLaunchProfile(gamePath string) (*engine.GameStatus, error) {
	return a.mutateAndInspect(gamePath, func(p string) error {
		return a.Engine.ResetLaunchProfile(a.getContext(), p)
	})
}

// GetGameDepots retrieves structured metadata and installation state for all depots of a game.
func (a *App) GetGameDepots(gamePath string) ([]domain.DepotSummary, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if parsed, ok := util.ParseAppID(gamePath); ok && parsed > 0 && !util.DirExists(gamePath) {
		return a.Engine.GetAppDepots(a.getContext(), parsed)
	}
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	return a.Engine.GetGameDepots(a.getContext(), path)
}

// GetAppDepots retrieves structured depot metadata for a Steam AppID from remote metadata.
func (a *App) GetAppDepots(appID any) ([]domain.DepotSummary, error) {
	str := strings.TrimSpace(fmt.Sprintf("%v", appID))
	parsed, ok := util.ParseAppID(str)
	if !ok || parsed == 0 {
		return nil, fmt.Errorf("invalid appID %v", appID)
	}
	return a.Engine.GetAppDepots(a.getContext(), parsed)
}

// SwitchGamePlatform switches the active execution platform and runner for a game directory.
func (a *App) SwitchGamePlatform(gamePath, targetPlatform string) (*engine.GameStatus, error) {
	return a.mutateAndInspect(gamePath, func(p string) error {
		return a.Engine.SwitchGamePlatform(a.getContext(), p, targetPlatform)
	})
}

// UninstallDepot removes on-disk files belonging to a specific depot ID while preserving shared assets and manifests.
func (a *App) UninstallDepot(gamePath string, depotID uint32) (*engine.GameStatus, error) {
	return a.mutateAndInspect(gamePath, func(p string) error {
		return a.Engine.UninstallDepot(a.getContext(), p, depotID)
	})
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
	art, err := a.Engine.ApplyCustomArtwork(a.getContext(), gameDir, assetType, imageSource)
	if err == nil {
		if st, sErr := a.Engine.InspectStatus(a.getContext(), gameDir); sErr == nil && st != nil {
			a.UpdateCachedGame(st)
		}
	}
	return art, err
}

// LinkGameMetadata links a game directory to a specific store provider and ID.
func (a *App) LinkGameMetadata(gameDir string, storeName string, storeID string) (*domain.Game, error) {
	gameDir = util.CanonicalPath(gameDir)
	res, err := a.Engine.LinkGameMetadata(a.getContext(), gameDir, storeName, storeID)
	if err == nil && res != nil {
		a.UpdateCachedGame(res)
	}
	return res, err
}

// UnlinkGameMetadata removes custom metadata overrides and restores auto-detected identity.
func (a *App) UnlinkGameMetadata(gameDir string) (*domain.Game, error) {
	gameDir = util.CanonicalPath(gameDir)
	res, err := a.Engine.UnlinkGameMetadata(a.getContext(), gameDir)
	if err == nil && res != nil {
		a.UpdateCachedGame(res)
	}
	return res, err
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

// GetGameMetadata fetches store metadata and game description for a specific provider (steam, gog, rawg) and ID.
func (a *App) GetGameMetadata(provider string, id string) (*GameMetadataDetails, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}

	meta, err := a.Engine.GetGameMetadata(a.getContext(), provider, id)
	if err != nil {
		return nil, err
	}

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

// FetchNews retrieves news and patch note items for a Steam AppID.
func (a *App) FetchNews(appID string, count int) ([]steam.NewsItem, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.FetchNews(a.getContext(), appID, count)
}

// GetPatchNote fetches a specific patch note for a game.
func (a *App) GetPatchNote(gamePath string, itemIndex int) (*steam.NewsItem, string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, "", err
	}
	return a.Engine.GetPatchNote(a.getContext(), path, itemIndex)
}

// CreateGameStub creates a skeleton game directory in the library path without downloading game files.
func (a *App) CreateGameStub(libraryPath, appID, name, storeName, branch string) (*domain.Game, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.CreateGameStub(a.getContext(), libraryPath, appID, name, storeName, branch)
}

// GetSteamBranches returns available official and beta branches for a Steam AppID.
func (a *App) GetSteamBranches(appID uint32) ([]domain.SteamBranch, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.GetSteamBranches(a.getContext(), appID)
}

// SetGameBranch updates the configured branch for a game directory.
func (a *App) SetGameBranch(gameDir, branch, password string) (*domain.Game, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	res, err := a.Engine.SetGameBranch(a.getContext(), gameDir, branch, password)
	if err == nil && res != nil {
		a.UpdateCachedGame(res)
	}
	return res, err
}

// TrashGame safely moves a game directory to the user's system trash.
// It returns an error if the game is currently running or if trashing fails.
func (a *App) TrashGame(gamePath string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}

	if a.IsGameRunning(path) {
		return fmt.Errorf("cannot trash game: process is currently running")
	}

	if err := trash.TrashWithContext(a.getContext(), path); err != nil {
		return fmt.Errorf("failed to trash game: %w", err)
	}

	a.RemoveCachedGame(path)
	return nil
}

// -----------------------------------------------------------------------------
// Mod Management
// -----------------------------------------------------------------------------

// GetGameMods returns the mod topology and installed mods for a game directory.
func (a *App) GetGameMods(gamePath string) (*domain.GameModStatus, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.GetGameMods(a.getContext(), path)
}

// ToggleGameMod enables or disables an installed mod by renaming.
func (a *App) ToggleGameMod(gamePath, modID string, enable bool) (*domain.InstalledMod, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	mod, err := a.Engine.ToggleGameMod(path, modID, enable)
	if err == nil {
		if st, sErr := a.Engine.InspectLocal(a.getContext(), path); sErr == nil && st != nil {
			a.UpdateCachedGame(st)
		}
	}
	return mod, err
}

// DeleteGameMod deletes an installed mod and cleans up its registry entry.
func (a *App) DeleteGameMod(gamePath, modID string) error {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return err
	}
	if a.Engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	err = a.Engine.DeleteGameMod(path, modID)
	if err == nil {
		if st, sErr := a.Engine.InspectLocal(a.getContext(), path); sErr == nil && st != nil {
			a.UpdateCachedGame(st)
		}
	}
	return err
}

// CheckGameModUpdates checks for newer versions of installed workshop mods via Steam Web API.
func (a *App) CheckGameModUpdates(gamePath string) (*domain.GameModStatus, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return nil, err
	}
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	res, err := a.Engine.CheckGameModUpdates(a.getContext(), path)
	if err == nil {
		if st, sErr := a.Engine.InspectLocal(a.getContext(), path); sErr == nil && st != nil {
			a.UpdateCachedGame(st)
		}
	}
	return res, err
}

// InstallWorkshopMod initiates an asynchronous workshop mod installation task and returns the task ID.
func (a *App) InstallWorkshopMod(gamePath, itemURLOrID string) (string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return "", err
	}
	if a.Engine == nil {
		return "", fmt.Errorf("engine not initialized")
	}
	return a.Engine.SubmitWorkshop(itemURLOrID, path, false)
}
