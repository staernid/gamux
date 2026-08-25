package gui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Go backend adapter exposed to the Wails frontend.
type App struct {
	ctx    context.Context
	Config *config.Config
	Engine *engine.Engine
}

// NewApp creates a new App instance with configuration and engine initialized.
func NewApp(cfg *config.Config) *App {
	if cfg == nil {
		var err error
		cfg, err = config.LoadConfig("")
		if err != nil {
			slog.Warn("Failed to load user config, using defaults", "error", err)
			cfg = config.DefaultConfig()
		}
	}
	eng := engine.New(cfg)
	return &App{
		Config: cfg,
		Engine: eng,
	}
}

func safeEmit(ctx context.Context, eventName string, data ...any) {
	if ctx == nil || ctx.Value("events") == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	wailsRuntime.EventsEmit(ctx, eventName, data...)
}

// Startup is called when the Wails application starts up.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Bind central EventBus to Wails frontend event emitter
	if a.Engine != nil && a.Engine.EventBus != nil {
		a.Engine.EventBus.SubscribeFunc(func(ev events.Event) {
			safeEmit(a.ctx, "gamux:task-event", ev)
			if strings.HasPrefix(string(ev.Type), "transfer:") {
				safeEmit(a.ctx, "gamux:transfer-event", ev)
			}
		})
	}

	// Bind engine desktop/toast notifications to Wails frontend event emitter
	a.Engine.Notifier = func(nCtx context.Context, title, message string) error {
		safeEmit(a.ctx, "gamux:notification", map[string]string{
			"title":   title,
			"message": message,
		})
		return nil
	}

	// Bind step progress updates to Wails frontend event emitter
	a.Engine.StepReporter = func(sCtx context.Context, prog engine.StepProgress) {
		safeEmit(a.ctx, "gamux:step-progress", map[string]any{
			"step":    prog.Step,
			"total":   prog.Total,
			"title":   prog.Title,
			"details": prog.Details,
			"percent": float64(prog.Step) / float64(prog.Total) * 100,
		})
	}

	// Bind runner process supervisor events to Wails frontend event emitter
	runner.DefaultManager().Subscribe(func(ev runner.ProcessEvent) {
		safeEmit(a.ctx, "gamux:game-status", map[string]any{
			"gameDir":   ev.GameDir,
			"isRunning": ev.IsRunning,
			"pid":       ev.PID,
		})
	})

	slog.Info("Gamux GUI backend initialized")
}

// getContext returns the current context (useful for fallback and testing).
func (a *App) getContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// GetConfig returns the current active configuration.
func (a *App) GetConfig() (*config.Config, error) {
	if a.Config == nil {
		a.Config = config.DefaultConfig()
	}
	return a.Config, nil
}

// SaveConfig validates and saves new configuration settings.
func (a *App) SaveConfig(newCfg *config.Config) error {
	if newCfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if err := config.SaveConfig(newCfg, ""); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	a.Config = newCfg
	a.Engine = engine.New(newCfg)
	slog.Info("Configuration updated and saved via GUI")
	return nil
}

// GetGSEUserConfig returns the global Goldberg user persona settings from configs.user.ini.
func (a *App) GetGSEUserConfig() (*gbe.GSEUserConfig, error) {
	return gbe.LoadGSEUserConfig()
}

// SaveGSEUserConfig persists the global Goldberg user persona settings to configs.user.ini.
func (a *App) SaveGSEUserConfig(cfg *gbe.GSEUserConfig) error {
	if cfg == nil {
		return fmt.Errorf("gse user config cannot be nil")
	}
	return gbe.SaveGSEUserConfig(cfg)
}

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

// CheckLibraryUpdates scans a parent directory for available updates on Steam.
func (a *App) CheckLibraryUpdates(parentDir string) ([]engine.LibraryUpdateStatus, error) {
	parentDir = util.CanonicalPath(parentDir)
	if parentDir == "" {
		return nil, fmt.Errorf("parent directory cannot be empty")
	}
	return a.Engine.CheckLibraryUpdates(a.getContext(), parentDir)
}

// -----------------------------------------------------------------------------
// Asynchronous Task & Job Submissions
// -----------------------------------------------------------------------------

// SubmitApply enqueues an asynchronous game post-processing setup task.
func (a *App) SubmitApply(opts engine.ProcessOptions) (string, error) {
	opts.Path = util.CanonicalPath(opts.Path)
	if opts.Path == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitApply(opts)
}

// SubmitNormalize enqueues an asynchronous directory normalization task.
func (a *App) SubmitNormalize(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitNormalize(gamePath)
}

// SubmitRollback enqueues an asynchronous game rollback task.
func (a *App) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitRollback(gamePath, dryRun)
}

// SubmitMigrateMode enqueues an asynchronous operating mode migration task.
func (a *App) SubmitMigrateMode(gamePath string, targetMode string, unregisterLutris bool) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitMigrateMode(gamePath, targetMode, unregisterLutris)
}

// SubmitVerify enqueues an asynchronous depot integrity verification task.
func (a *App) SubmitVerify(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitVerify(gamePath)
}

// SubmitUpdate enqueues an asynchronous game update task.
func (a *App) SubmitUpdate(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitUpdate(gamePath)
}

// StartDownloadTransfer enqueues a new resumable file/depot download in the unified tasks engine.
func (a *App) StartDownloadTransfer(opts engine.DownloadGameOptions) (string, error) {
	if a.Engine == nil {
		return "", fmt.Errorf("engine not initialized")
	}
	return a.Engine.SubmitDownload(opts)
}

// UpdateTool enqueues an asynchronous tool update download in the unified Tasks engine.
func (a *App) UpdateTool(toolKey string) (string, error) {
	if a.Engine == nil {
		return "", fmt.Errorf("engine not initialized")
	}
	return a.Engine.SubmitToolUpdate(toolKey)
}

// -----------------------------------------------------------------------------
// Unified Task Management
// -----------------------------------------------------------------------------

// ListTasks returns all background tasks tracked by the unified tasks manager.
func (a *App) ListTasks() []*tasks.Task {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return []*tasks.Task{}
	}
	return a.Engine.Tasks.List()
}

// GetTask retrieves details for a single task.
func (a *App) GetTask(id string) (*tasks.Task, bool) {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return nil, false
	}
	return a.Engine.Tasks.Get(id)
}

// PauseTask suspends an active or queued task.
func (a *App) PauseTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Pause(id)
}

// ResumeTask continues a paused task.
func (a *App) ResumeTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Resume(id)
}

// CancelTask terminates a running, queued, or paused task.
func (a *App) CancelTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Cancel(id)
}

// RemoveTask removes a finished task from the history list.
func (a *App) RemoveTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Remove(id)
}

// ReorderTasks rearranges queued tasks by ID priority.
func (a *App) ReorderTasks(ids []string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Reorder(ids)
}

// ClearFinishedTasks purges finished tasks from manager memory.
func (a *App) ClearFinishedTasks() error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return nil
	}
	return a.Engine.Tasks.ClearFinished()
}

// SetTransferSpeedLimit configures the download bandwidth limiter (bytes/sec, 0 for unlimited) and persists it.
func (a *App) SetTransferSpeedLimit(bps int64) {
	if a.Engine != nil && a.Engine.Tasks != nil {
		a.Engine.Tasks.SetSpeedLimit(bps)
	}
	if a.Config != nil {
		a.Config.TransferSpeedLimit = bps
		_ = config.SaveConfig(a.Config, "")
	}
}

// GetTransferSpeedLimit returns the active download bandwidth limit.
func (a *App) GetTransferSpeedLimit() int64 {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return 0
	}
	return a.Engine.Tasks.SpeedLimit()
}

// -----------------------------------------------------------------------------
// Game Execution & Lifecycle
// -----------------------------------------------------------------------------

// LaunchOptions alias for runner.LaunchOptions.
type LaunchOptions = runner.LaunchOptions

// LaunchGame starts the game process asynchronously according to the supplied launch options or detected game profile.
func (a *App) LaunchGame(opts LaunchOptions) error {
	if opts.GameDir == "" && opts.ExePath == "" {
		return fmt.Errorf("game directory or executable path cannot be empty")
	}

	// If ExePath is missing, inspect status to resolve target executable
	if opts.ExePath == "" && opts.GameDir != "" {
		status, err := a.Engine.InspectStatus(a.getContext(), opts.GameDir)
		if err == nil && status != nil {
			if opts.Name == "" {
				opts.Name = status.Name
			}
			if opts.AppID == "" {
				opts.AppID = status.AppID
			}
			if opts.Platform == "" {
				opts.Platform = status.Platform
			}
			if opts.ExePath == "" {
				opts.ExePath = status.ExePath
			}
			if opts.ExeArgs == "" {
				opts.ExeArgs = status.ExeArgs
			}
			if opts.OperatingMode == "" {
				opts.OperatingMode = status.OperatingMode
			}
			if opts.Runner == "" {
				opts.Runner = status.Runner
			}
			if opts.WinePrefix == "" {
				opts.WinePrefix = status.WinePrefix
			}
		}
	}

	// Merge launch profile if not overridden
	if opts.GameDir != "" {
		if profile, err := a.Engine.GetLaunchProfile(a.getContext(), opts.GameDir); err == nil && profile != nil {
			if opts.OperatingMode == "" {
				opts.OperatingMode = profile.OperatingMode
			}
			if opts.Runner == "" {
				opts.Runner = profile.Runner
			}
			if opts.WinePrefix == "" {
				opts.WinePrefix = profile.WinePrefix
			}
			if opts.MangoHud == nil && profile.MangoHud != nil {
				opts.MangoHud = profile.MangoHud
			}
			if opts.WineFullscreenFSR == nil && profile.WineFullscreenFSR != nil {
				opts.WineFullscreenFSR = profile.WineFullscreenFSR
			}
			if opts.ProtonNVAPI == nil && profile.ProtonNVAPI != nil {
				opts.ProtonNVAPI = profile.ProtonNVAPI
			}
			if opts.DXVKHud == "" {
				opts.DXVKHud = profile.DXVKHud
			}
		}
	}

	_, err := runner.DefaultManager().Start(a.getContext(), a.Config, opts)
	return err
}

// IsGameRunning checks if a game process in the given game directory is currently active.
func (a *App) IsGameRunning(gamePath string) bool {
	return runner.DefaultManager().IsRunning(gamePath)
}

// StopGame terminates an active game process and its process tree.
func (a *App) StopGame(gamePath string) error {
	return runner.DefaultManager().Stop(gamePath)
}

// GetAvailableRunners returns list of detected execution runners on the host.
func (a *App) GetAvailableRunners() []string {
	return runner.GetAvailableRunners()
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

// MigrateOperatingMode updates the launch configuration and Lutris integration for a game directory.
func (a *App) MigrateOperatingMode(gamePath string, targetMode string, unregisterLutris bool) error {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.MigrateOperatingMode(a.getContext(), gamePath, targetMode, unregisterLutris)
}

// GetToolsStatus returns version, release notes, and update status for GBE fork and steamless-rs.
func (a *App) GetToolsStatus() ([]tools.ToolStatus, error) {
	return a.Engine.GetToolsStatus(a.getContext())
}

// -----------------------------------------------------------------------------
// Store, Metadata, Artwork & News
// -----------------------------------------------------------------------------

// SearchStoreGames queries all configured store providers (Steam, GOG, SteamGridDB) for game titles.
func (a *App) SearchStoreGames(query string) ([]store.SearchResult, error) {
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.SearchStoreGames(a.getContext(), query)
}

// ResolveArtwork resolves covers, heroes, and banners for a game directory.
func (a *App) ResolveArtwork(gameDir string) (*domain.GameArtwork, error) {
	gameDir = util.CanonicalPath(gameDir)
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	info, err := detector.Detect(a.getContext(), gameDir)
	if err != nil {
		return nil, err
	}
	return a.Engine.ResolveArtwork(a.getContext(), info)
}

// QueryArtworkOptions queries candidate providers for specific asset options (covers, heroes, banners, logos).
func (a *App) QueryArtworkOptions(query domain.ArtworkQuery) ([]domain.ArtworkOption, error) {
	query.GameDir = util.CanonicalPath(query.GameDir)
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.QueryArtworkOptions(a.getContext(), query)
}

// ApplyCustomArtwork sets custom artwork on a game directory and updates metadata.
func (a *App) ApplyCustomArtwork(gameDir string, assetType string, imageSource string) (*domain.GameArtwork, error) {
	gameDir = util.CanonicalPath(gameDir)
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.ApplyCustomArtwork(a.getContext(), gameDir, assetType, imageSource)
}

// LinkGameMetadata links a game directory to a specific store provider and ID.
func (a *App) LinkGameMetadata(gameDir string, storeName string, storeID string) (*domain.Game, error) {
	gameDir = util.CanonicalPath(gameDir)
	if a.Engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	return a.Engine.LinkGameMetadata(a.getContext(), gameDir, storeName, storeID)
}

// FetchNews retrieves news and patch note items for a Steam AppID.
func (a *App) FetchNews(appID string, count int) ([]steam.NewsItem, error) {
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

// -----------------------------------------------------------------------------
// System & OS Integration
// -----------------------------------------------------------------------------

// SelectDirectory opens a native OS directory picker dialog.
func (a *App) SelectDirectory(title, defaultDir string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("GUI runtime context not ready")
	}
	if defaultDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultDir = home
		}
	}
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: defaultDir,
	})
}

// SelectFile opens a native OS file picker dialog.
func (a *App) SelectFile(title, filterName, filterPattern string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("GUI runtime context not ready")
	}
	var filters []wailsRuntime.FileFilter
	if filterName != "" && filterPattern != "" {
		filters = append(filters, wailsRuntime.FileFilter{
			DisplayName: filterName,
			Pattern:     filterPattern,
		})
	}
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
}

// OpenDirectory opens the specified directory in the system default file manager.
func (a *App) OpenDirectory(dirPath string) error {
	dirPath = util.CanonicalPath(dirPath)
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	if !util.FileExists(dirPath) {
		return fmt.Errorf("directory does not exist: %s", dirPath)
	}
	cmd := exec.Command("xdg-open", dirPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open file manager: %w", err)
	}
	return nil
}

// EnsureDirectoryExists ensures that a given directory path exists or creates it.
func (a *App) EnsureDirectoryExists(dirPath string) (bool, error) {
	if dirPath == "" {
		return false, fmt.Errorf("directory path cannot be empty")
	}
	abs, err := filepath.Abs(dirPath)
	if err != nil {
		return false, fmt.Errorf("resolve directory path: %w", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return false, fmt.Errorf("create directory: %w", err)
	}
	return true, nil
}

// AssetHandler returns an http.Handler that serves cached artwork from disk under /art/.
func (a *App) AssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/art/") {
			rel := strings.TrimPrefix(r.URL.Path, "/art/")
			rel = filepath.Clean(rel)
			if strings.HasPrefix(rel, "..") {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			artDir := ""
			if a.Config != nil && a.Config.ArtworkDir != "" {
				artDir = util.ExpandPath(a.Config.ArtworkDir)
			} else {
				home, _ := os.UserHomeDir()
				artDir = filepath.Join(home, ".local", "share", "gamux", "art")
			}
			filePath := filepath.Join(artDir, rel)
			if util.FileExists(filePath) {
				w.Header().Set("Cache-Control", "no-cache, must-revalidate")
				http.ServeFile(w, r, filePath)
				return
			}
		}
		http.NotFound(w, r)
	})
}





