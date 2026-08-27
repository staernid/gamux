package gui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/saves"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Go backend adapter exposed to the Wails frontend.
type App struct {
	ctx     context.Context
	Config  *config.Config
	SC      *engine.ServiceContext
	Library *engine.LibraryService
	Game    *engine.GameService
	Task    *engine.TaskService
	Saves   *engine.SaveService
}

// NewApp creates a new App instance with configuration and services initialized.
func NewApp(cfg *config.Config) *App {
	if cfg == nil {
		var err error
		cfg, err = config.LoadConfig("")
		if err != nil {
			slog.Warn("Failed to load user config, using defaults", "error", err)
			cfg = config.DefaultConfig()
		}
	}
	sc, lib, game, task := engine.NewServices(cfg)
	return &App{
		Config:  cfg,
		SC:      sc,
		Library: lib,
		Game:    game,
		Task:    task,
		Saves:   sc.Saves,
	}
}

func safeEmit(ctx context.Context, eventName string, data ...any) {
	if ctx == nil || ctx.Value("events") == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("Ignoring panic during safeEmit (context likely closed)", "event", eventName, "recover", r)
		}
	}()
	wailsRuntime.EventsEmit(ctx, eventName, data...)
}

// Startup is called when the Wails application starts up.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Bind central EventBus to Wails frontend event emitter
	if a.SC != nil && a.SC.EventBus != nil {
		a.SC.EventBus.SubscribeFunc(func(ev events.Event) {
			if strings.HasPrefix(string(ev.Type), "transfer:") {
				safeEmit(a.ctx, "gamux:transfer-event", ev)
			} else {
				safeEmit(a.ctx, "gamux:task-event", ev)
			}
			if ev.Type == events.EventNotification {
				title := ev.Stage
				if t, ok := ev.Metadata["title"]; ok && t != "" {
					title = t
				}
				msg := ev.Details
				if m, ok := ev.Metadata["message"]; ok && m != "" {
					msg = m
				}
				safeEmit(a.ctx, "gamux:notification", map[string]string{
					"title":   title,
					"message": msg,
				})
			}
			if ev.Type == events.EventGameLaunched {
				pid := 0
				if pStr, ok := ev.Metadata["pid"]; ok {
					pid, _ = strconv.Atoi(pStr)
				}
				safeEmit(a.ctx, "gamux:game-status", map[string]any{
					"gameDir":   ev.GameDir,
					"isRunning": true,
					"pid":       pid,
				})
			}
			if ev.Type == events.EventGameExited {
				safeEmit(a.ctx, "gamux:game-status", map[string]any{
					"gameDir":   ev.GameDir,
					"isRunning": false,
					"pid":       0,
				})
			}
			if ev.Type == events.EventTaskProgress && ev.Metadata != nil && ev.Metadata["step"] != "" {
				step, _ := strconv.Atoi(ev.Metadata["step"])
				total, _ := strconv.Atoi(ev.Metadata["total"])
				safeEmit(a.ctx, "gamux:step-progress", map[string]any{
					"step":    step,
					"total":   total,
					"title":   ev.Stage,
					"details": ev.Details,
					"percent": ev.Progress.Percent,
				})
			}
		})
	}

	// Bind desktop/toast notifications to EventBus
	a.SC.Notifier = func(nCtx context.Context, title, message string) error {
		if a.SC != nil && a.SC.EventBus != nil {
			a.SC.EventBus.Publish(events.NewNotification(title, message))
		}
		return nil
	}

	// Bind step progress updates to EventBus
	a.SC.StepReporter = func(sCtx context.Context, prog engine.StepProgress) {
		if a.SC != nil && a.SC.EventBus != nil {
			a.SC.EventBus.Publish(events.NewStepProgress(prog.Step, prog.Total, prog.Title, prog.Details))
		}
	}

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
	a.SC, a.Library, a.Game, a.Task = engine.NewServices(newCfg)
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
	return a.Library.BatchInspectLibraries(a.getContext(), cleanDirs)
}

// InspectGame inspects a game directory and returns its full GameStatus, optionally fetching news/patch notes.
func (a *App) InspectGame(gamePath string, newsCount int) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if newsCount > 0 {
		return a.Library.InspectStatusEx(a.getContext(), gamePath, newsCount)
	}
	return a.Library.InspectStatus(a.getContext(), gamePath)
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
	return a.Library.VerifyIntegrity(a.getContext(), gamePath)
}

// -----------------------------------------------------------------------------
// Asynchronous Task Pipeline Submissions
// -----------------------------------------------------------------------------

// SubmitApply enqueues an asynchronous game post-processing setup task.
func (a *App) SubmitApply(opts engine.ProcessOptions) (string, error) {
	opts.Path = util.CanonicalPath(opts.Path)
	if opts.Path == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Task.SubmitApply(opts)
}

// SubmitNormalize enqueues an asynchronous directory normalization task.
func (a *App) SubmitNormalize(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Task.SubmitNormalize(gamePath)
}

// SubmitRollback enqueues an asynchronous game rollback task.
func (a *App) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Task.SubmitRollback(gamePath, dryRun)
}

// SubmitVerify enqueues an asynchronous depot integrity verification task.
func (a *App) SubmitVerify(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Task.SubmitVerify(gamePath)
}

// SubmitUpdate enqueues an asynchronous game update task.
func (a *App) SubmitUpdate(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Task.SubmitUpdate(gamePath)
}

// StartDownloadTransfer enqueues a new resumable file/depot download in the unified tasks engine.
func (a *App) StartDownloadTransfer(opts engine.DownloadGameOptions) (string, error) {
	return a.Task.SubmitDownload(opts)
}

// UpdateTool enqueues an asynchronous tool update download in the unified Tasks engine.
func (a *App) UpdateTool(toolKey string) (string, error) {
	return a.Task.SubmitToolUpdate(toolKey)
}

// -----------------------------------------------------------------------------
// Unified Task Management
// -----------------------------------------------------------------------------

// ListTasks returns all background tasks tracked by the unified tasks manager.
func (a *App) ListTasks() []*tasks.Task {
	if a.SC == nil || a.SC.Tasks == nil {
		return []*tasks.Task{}
	}
	return a.SC.Tasks.List()
}

// GetTask retrieves details for a single task.
func (a *App) GetTask(id string) (*tasks.Task, bool) {
	if a.SC == nil || a.SC.Tasks == nil {
		return nil, false
	}
	return a.SC.Tasks.Get(id)
}

// PauseTask suspends an active or queued task.
func (a *App) PauseTask(id string) error {
	if a.SC == nil || a.SC.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.SC.Tasks.Pause(id)
}

// ResumeTask continues a paused task.
func (a *App) ResumeTask(id string) error {
	if a.SC == nil || a.SC.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.SC.Tasks.Resume(id)
}

// CancelTask terminates a running, queued, or paused task.
func (a *App) CancelTask(id string) error {
	if a.SC == nil || a.SC.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.SC.Tasks.Cancel(id)
}

// RemoveTask removes a finished task from the history list.
func (a *App) RemoveTask(id string) error {
	if a.SC == nil || a.SC.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.SC.Tasks.Remove(id)
}

// ReorderTasks rearranges queued tasks by ID priority.
func (a *App) ReorderTasks(ids []string) error {
	if a.SC == nil || a.SC.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.SC.Tasks.Reorder(ids)
}

// ClearFinishedTasks purges finished tasks from manager memory.
func (a *App) ClearFinishedTasks() error {
	if a.SC == nil || a.SC.Tasks == nil {
		return nil
	}
	return a.SC.Tasks.ClearFinished()
}

// SetTransferSpeedLimit configures the download bandwidth limiter (bytes/sec, 0 for unlimited) and persists it.
func (a *App) SetTransferSpeedLimit(bps int64) {
	if a.SC != nil && a.SC.Tasks != nil {
		a.SC.Tasks.SetSpeedLimit(bps)
	}
	if a.Config != nil {
		a.Config.TransferSpeedLimit = bps
		_ = config.SaveConfig(a.Config, "")
	}
}

// GetTransferSpeedLimit returns the active download bandwidth limit.
func (a *App) GetTransferSpeedLimit() int64 {
	if a.SC == nil || a.SC.Tasks == nil {
		return 0
	}
	return a.SC.Tasks.SpeedLimit()
}

// -----------------------------------------------------------------------------
// Game Execution & Lifecycle
// -----------------------------------------------------------------------------

// LaunchOptions alias for runner.LaunchOptions.
type LaunchOptions = runner.LaunchOptions

// LaunchGame starts the game process asynchronously according to the supplied launch options or detected game profile.
func (a *App) LaunchGame(opts LaunchOptions) error {
	_, err := a.Game.LaunchGame(a.getContext(), opts)
	return err
}

// QuickPlay executes one-click setup and launch for a game.
func (a *App) QuickPlay(opts LaunchOptions) error {
	_, err := a.Game.EnsureAndLaunchGame(a.getContext(), opts)
	return err
}

// IsGameRunning checks if a game process in the given game directory is currently active.
func (a *App) IsGameRunning(gamePath string) bool {
	if a.SC != nil && a.SC.RunnerMgr != nil {
		return a.SC.RunnerMgr.IsRunning(gamePath)
	}
	return runner.DefaultManager().IsRunning(gamePath)
}

// StopGame terminates an active game process and its process tree.
func (a *App) StopGame(gamePath string) error {
	if a.SC != nil && a.SC.RunnerMgr != nil {
		return a.SC.RunnerMgr.Stop(gamePath)
	}
	return runner.DefaultManager().Stop(gamePath)
}

// GetAvailableRunners returns list of detected execution runners on the host.
func (a *App) GetAvailableRunners() []string {
	return runner.GetAvailableRunners()
}

// GetGameLog returns the stdout/stderr log content from the current or last execution of a game.
func (a *App) GetGameLog(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Game.GetGameLog(a.getContext(), gamePath)
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
	return a.Game.GetLaunchProfile(a.getContext(), gamePath)
}

// SaveLaunchProfile persists custom launch configuration for a game directory and returns updated GameStatus.
func (a *App) SaveLaunchProfile(gamePath string, opts runner.LaunchOptions) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if err := a.Game.SaveLaunchProfile(a.getContext(), gamePath, opts); err != nil {
		return nil, err
	}
	return a.Library.InspectStatus(a.getContext(), gamePath)
}

// ResetLaunchProfile removes custom launch configuration and returns updated GameStatus.
func (a *App) ResetLaunchProfile(gamePath string) (*engine.GameStatus, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if err := a.Game.ResetLaunchProfile(a.getContext(), gamePath); err != nil {
		return nil, err
	}
	return a.Library.InspectStatus(a.getContext(), gamePath)
}

// GetToolsStatus returns version, release notes, and update status for GBE fork and steamless-rs.
func (a *App) GetToolsStatus() ([]tools.ToolStatus, error) {
	return a.SC.GetToolsStatus(a.getContext())
}

// -----------------------------------------------------------------------------
// Store, Metadata, Artwork & News
// -----------------------------------------------------------------------------

// SearchStoreGames queries all configured store providers (Steam, GOG, SteamGridDB) for game titles.
func (a *App) SearchStoreGames(query string) ([]store.SearchResult, error) {
	return a.Library.SearchStoreGames(a.getContext(), query)
}

// ResolveArtwork resolves covers, heroes, and banners for a game directory.
func (a *App) ResolveArtwork(gameDir string) (*domain.GameArtwork, error) {
	gameDir = util.CanonicalPath(gameDir)
	info, err := detector.Detect(a.getContext(), gameDir)
	if err != nil {
		return nil, err
	}
	return a.Library.ResolveArtwork(a.getContext(), info)
}

// QueryArtworkOptions queries candidate providers for specific asset options (covers, heroes, banners, logos).
func (a *App) QueryArtworkOptions(query domain.ArtworkQuery) ([]domain.ArtworkOption, error) {
	query.GameDir = util.CanonicalPath(query.GameDir)
	return a.Library.QueryArtworkOptions(a.getContext(), query)
}

// ApplyCustomArtwork sets custom artwork on a game directory and updates metadata.
func (a *App) ApplyCustomArtwork(gameDir string, assetType string, imageSource string) (*domain.GameArtwork, error) {
	gameDir = util.CanonicalPath(gameDir)
	return a.Library.ApplyCustomArtwork(a.getContext(), gameDir, assetType, imageSource)
}

// LinkGameMetadata links a game directory to a specific store provider and ID.
func (a *App) LinkGameMetadata(gameDir string, storeName string, storeID string) (*domain.Game, error) {
	gameDir = util.CanonicalPath(gameDir)
	return a.Library.LinkGameMetadata(a.getContext(), gameDir, storeName, storeID)
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
	return a.Library.GetPatchNote(a.getContext(), gamePath, itemIndex)
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

// OpenDirectory opens the specified directory or containing folder in the system default file manager.
func (a *App) OpenDirectory(dirPath string) error {
	dirPath = util.CanonicalPath(dirPath)
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	if !util.FileExists(dirPath) {
		return fmt.Errorf("directory does not exist: %s", dirPath)
	}

	target := dirPath
	if fi, err := os.Stat(dirPath); err == nil && !fi.IsDir() {
		target = filepath.Dir(dirPath)
	}

	cmd := exec.Command("xdg-open", target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open file manager: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Save Game & Configuration Data
// -----------------------------------------------------------------------------

// GetGameSaveLocations discovers on-disk save, config, and cloud storage directories for a game.
func (a *App) GetGameSaveLocations(gamePath string) (*saves.SaveLocationsResult, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if a.SC == nil || a.SC.Saves == nil {
		return nil, fmt.Errorf("save service not initialized")
	}
	return a.SC.Saves.InspectSaveLocations(a.getContext(), gamePath)
}

// GetGameBackups lists available backup snapshots for a game.
func (a *App) GetGameBackups(gamePath string) ([]saves.BackupInfo, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return []saves.BackupInfo{}, nil
	}
	if a.SC == nil || a.SC.Saves == nil {
		return []saves.BackupInfo{}, nil
	}
	return a.SC.Saves.ListSaveBackups(a.getContext(), gamePath)
}

// CreateSaveBackup creates a new timestamped snapshot archive for a game.
func (a *App) CreateSaveBackup(gamePath, note string) (*saves.BackupInfo, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	if a.SC == nil || a.SC.Saves == nil {
		return nil, fmt.Errorf("save service not initialized")
	}
	return a.SC.Saves.CreateSaveBackup(a.getContext(), gamePath, note, "manual")
}

// RestoreSaveBackup restores a snapshot archive to the game's active save locations.
func (a *App) RestoreSaveBackup(gamePath, backupID string) error {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	if a.SC == nil || a.SC.Saves == nil {
		return fmt.Errorf("save service not initialized")
	}
	return a.SC.Saves.RestoreSaveBackup(a.getContext(), gamePath, backupID)
}

// DeleteSaveBackup removes a snapshot archive from disk.
func (a *App) DeleteSaveBackup(gamePath, backupID string) error {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	if a.SC == nil || a.SC.Saves == nil {
		return fmt.Errorf("save service not initialized")
	}
	return a.SC.Saves.DeleteSaveBackup(a.getContext(), gamePath, backupID)
}

// UpdateSaveManifest forces a re-download and re-indexing of the Ludusavi save database.
func (a *App) UpdateSaveManifest() error {
	if a.SC == nil || a.SC.Saves == nil {
		return fmt.Errorf("save service not initialized")
	}
	return a.SC.Saves.UpdateSaveManifest(a.getContext())
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

// GetDiskSpace returns filesystem disk usage metrics (total, free, used bytes) for the given path.
func (a *App) GetDiskSpace(path string) (*util.DiskSpaceInfo, error) {
	return util.GetDiskSpace(path)
}
