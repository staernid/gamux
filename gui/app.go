package gui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/steamless"
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

// Startup is called when the Wails application starts up.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	// Bind engine desktop/toast notifications to Wails frontend event emitter
	a.Engine.Notifier = func(nCtx context.Context, title, message string) error {
		if a.ctx != nil && a.ctx.Value("events") != nil {
			wailsRuntime.EventsEmit(a.ctx, "gamux:notification", map[string]string{
				"title":   title,
				"message": message,
			})
		}
		return nil
	}

	// Bind step progress updates to Wails frontend event emitter
	a.Engine.StepReporter = func(sCtx context.Context, prog engine.StepProgress) {
		if a.ctx != nil && a.ctx.Value("events") != nil {
			wailsRuntime.EventsEmit(a.ctx, "gamux:step-progress", map[string]any{
				"step":    prog.Step,
				"total":   prog.Total,
				"title":   prog.Title,
				"details": prog.Details,
				"percent": float64(prog.Step) / float64(prog.Total) * 100,
			})
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
	a.Engine = engine.New(newCfg)
	slog.Info("Configuration updated and saved via GUI")
	return nil
}

// DetectGame inspects a given game directory and returns detected game metadata.
func (a *App) DetectGame(gamePath string) (*detector.GameInfo, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return detector.Detect(a.getContext(), gamePath)
}

// InspectGame inspects a game directory and returns its full GameStatus.
func (a *App) InspectGame(gamePath string) (*engine.GameStatus, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.InspectStatus(a.getContext(), gamePath)
}

// InspectGameEx inspects a game directory and fetches up to newsCount patch notes.
func (a *App) InspectGameEx(gamePath string, newsCount int) (*engine.GameStatus, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.InspectStatusEx(a.getContext(), gamePath, newsCount)
}

// BatchInspect scans a parent folder containing multiple game directories.
func (a *App) BatchInspect(parentDir string) ([]*engine.GameStatus, error) {
	if parentDir == "" {
		return nil, fmt.Errorf("parent directory cannot be empty")
	}
	return a.Engine.BatchInspect(a.getContext(), parentDir)
}

// ProcessGame executes Goldberg emulator setup & Lutris registration for a game directory.
func (a *App) ProcessGame(opts engine.ProcessOptions) (*engine.ProcessResult, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.ProcessGame(a.getContext(), opts)
}

// ApplyGame executes the complete one-shot lifecycle: inspect, normalize, synth ACF, GBE patch, Lutris add.
func (a *App) ApplyGame(opts engine.ProcessOptions) (*engine.ProcessResult, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.ApplyGame(a.getContext(), opts)
}

// Rollback restores original backup files and cleans up generated Goldberg/Lutris configs.
func (a *App) Rollback(gamePath string, dryRun bool) error {
	if gamePath == "" {
		return fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.Rollback(a.getContext(), gamePath, dryRun)
}

// VerifyIntegrity runs full depot integrity verification for a game directory on-demand.
func (a *App) VerifyIntegrity(gamePath string) (*manifest.IntegrityReport, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.VerifyIntegrity(a.getContext(), gamePath)
}

// GetPatchNote fetches a specific patch note for a game.
func (a *App) GetPatchNote(gamePath string, itemIndex int) (*steam.NewsItem, string, error) {
	if gamePath == "" {
		return nil, "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.GetPatchNote(a.getContext(), gamePath, itemIndex)
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

// SearchSteamGames searches for Steam games by title.
func (a *App) SearchSteamGames(query string) ([]steam.AppSearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	return steam.SearchAppIDCandidates(a.getContext(), query)
}

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

// CheckLibraryUpdates scans a parent directory for available updates on Steam.
func (a *App) CheckLibraryUpdates(parentDir string) ([]engine.LibraryUpdateStatus, error) {
	if parentDir == "" {
		return nil, fmt.Errorf("parent directory cannot be empty")
	}
	return a.Engine.CheckLibraryUpdates(a.getContext(), parentDir)
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

// DownloadGameOptions defines parameters for acquiring or updating a game depot.
type DownloadGameOptions struct {
	AppID        uint32                 `json:"app_id"`
	TargetDir    string                 `json:"target_dir"`
	Platform     string                 `json:"platform"`
	LuaPath      string                 `json:"lua_path"`
	DryRun       bool                   `json:"dry_run"`
	AutoApply    bool                   `json:"auto_apply"`
	ApplyOptions *engine.ProcessOptions `json:"apply_options"`
}

// DownloadGame downloads game files from SteamPipe CDNs with progress streaming.
func (a *App) DownloadGame(opts DownloadGameOptions) (*downloader.Result, error) {
	if opts.AppID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID")
	}
	if opts.TargetDir == "" {
		return nil, fmt.Errorf("target directory cannot be empty")
	}

	dlOpts := downloader.DownloadOptions{
		TargetDir: opts.TargetDir,
		AppID:     opts.AppID,
		Platform:  opts.Platform,
		LuaPath:   opts.LuaPath,
		DryRun:    opts.DryRun,
		ProgressCallback: func(current, total int, item string) {
			if a.ctx != nil && a.ctx.Value("events") != nil {
				wailsRuntime.EventsEmit(a.ctx, "gamux:download-progress", map[string]any{
					"app_id":  opts.AppID,
					"current": current,
					"total":   total,
					"item":    item,
					"percent": float64(current) / float64(total) * 100,
				})
			}
		},
	}

	res, err := downloader.DownloadGame(a.getContext(), a.Config, dlOpts)
	if err != nil {
		return nil, fmt.Errorf("download game: %w", err)
	}

	if opts.AutoApply && !opts.DryRun {
		applyOpts := engine.ProcessOptions{
			Path:              opts.TargetDir,
			ApplyGBE:          true,
			AddLutris:         true,
			NormalizeDir:      true,
			FetchAchievements: true,
		}
		if opts.ApplyOptions != nil {
			applyOpts = *opts.ApplyOptions
			applyOpts.Path = opts.TargetDir
		}
		if _, applyErr := a.ApplyGame(applyOpts); applyErr != nil {
			slog.Warn("Post-download auto-apply warning", "error", applyErr)
		}
	}

	return res, nil
}

// UpdateGame checks for available Steam depot updates for an existing game directory, downloads changed files, and re-applies setup.
func (a *App) UpdateGame(gamePath string) (*engine.GameStatus, error) {
	if gamePath == "" {
		return nil, fmt.Errorf("game path cannot be empty")
	}

	info, err := detector.Detect(a.getContext(), gamePath)
	if err != nil {
		return nil, fmt.Errorf("detect game for update: %w", err)
	}

	appIDVal, err := strconv.ParseUint(info.AppID, 10, 32)
	if err != nil || appIDVal == 0 {
		return nil, fmt.Errorf("invalid AppID %q for update", info.AppID)
	}

	dlOpts := DownloadGameOptions{
		AppID:     uint32(appIDVal),
		TargetDir: info.GameDir,
		Platform:  info.Platform,
		AutoApply: true,
	}

	if _, dlErr := a.DownloadGame(dlOpts); dlErr != nil {
		return nil, fmt.Errorf("update depot download: %w", dlErr)
	}

	return a.InspectGameEx(info.GameDir, 5)
}

// UpdateGBE checks GitHub for the latest Goldberg Steam Emulator release and updates local binaries.
func (a *App) UpdateGBE() (string, error) {
	if err := github.UpdateGBE(a.getContext(), a.Config); err != nil {
		return "", fmt.Errorf("update GBE: %w", err)
	}
	return "Goldberg Emulator updated to latest release", nil
}

// UpdateSteamless checks and ensures local steamless-rs binary is up to date.
func (a *App) UpdateSteamless() (string, error) {
	if _, err := steamless.EnsureBinary(a.getContext(), a.Config); err != nil {
		return "", fmt.Errorf("update Steamless: %w", err)
	}
	return "Steamless-rs binary verified and up-to-date", nil
}

// OpenFolder opens the specified directory in the system default file manager.
func (a *App) OpenFolder(dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}
	expanded := util.ExpandPath(dirPath)
	if !util.FileExists(expanded) {
		return fmt.Errorf("directory does not exist: %s", expanded)
	}
	cmd := exec.Command("xdg-open", expanded)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open file manager: %w", err)
	}
	return nil
}

// LaunchLutrisGame starts the game via Lutris by slug or AppID.
func (a *App) LaunchLutrisGame(slug string) error {
	if slug == "" {
		return fmt.Errorf("game slug cannot be empty")
	}
	cmd := exec.Command("lutris", fmt.Sprintf("lutris:rungame/%s", slug))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch game via Lutris: %w", err)
	}
	return nil
}

// GetToolsStatus returns version, release notes, and update status for GBE fork and steamless-rs.
func (a *App) GetToolsStatus() ([]github.ToolInfo, error) {
	return github.GetToolsStatus(a.getContext(), a.Config)
}

// UpdateTool updates the specified tool ("gbe" or "steamless").
func (a *App) UpdateTool(toolKey string) error {
	if toolKey == "gbe" {
		return github.UpdateGBE(a.getContext(), a.Config)
	} else if toolKey == "steamless" {
		destDir := util.ExpandPath(a.Config.SteamlessDir)
		return github.UpdateSteamlessAssets(a.getContext(), a.Config, destDir)
	}
	return fmt.Errorf("unknown tool key: %s", toolKey)
}
