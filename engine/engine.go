package engine

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/gamecfg"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/pipeline"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/store"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
	"golang.org/x/sync/errgroup"
)

// ProcessOptions is an alias for pipeline.Options — the single source of truth
// for game processing execution flags shared across CLI, GUI, and pipeline stages.
type ProcessOptions = pipeline.Options

// ProcessResult summarizes the outcome of processing a game.
type ProcessResult struct {
	Info                *detector.GameInfo `json:"info,omitempty"`
	LutrisAdded         bool               `json:"lutris_added"`
	Patched             bool               `json:"patched"`
	SteamlessUnpacked   bool               `json:"steamless_unpacked"`
	ManifestsMoved      bool               `json:"manifests_moved"`
	DirectoryNormalized bool               `json:"directory_normalized"`
	ACFSynthesized      bool               `json:"acf_synthesized"`
	Errors              []string           `json:"errors,omitempty"`
}

// GameStatus describes the current patch & integration state of a game.
type GameStatus = domain.Game

// StepProgress defines a progress stage update for UI/CLI consumers.
type StepProgress struct {
	Step    int    `json:"step"`
	Total   int    `json:"total"`
	Title   string `json:"title"`
	Details string `json:"details"`
}

// DownloadGameOptions defines parameters for downloading or updating game depots.
type DownloadGameOptions struct {
	AppID            uint32                 `json:"app_id"`
	TargetDir        string                 `json:"target_dir"`
	Platform         string                 `json:"platform"`
	LuaPath          string                 `json:"lua_path"`
	DryRun           bool                   `json:"dry_run"`
	WorkerLimit      int                    `json:"worker_limit,omitempty"`
	AutoApply        bool                   `json:"auto_apply"`
	ApplyOptions         *ProcessOptions                                                                    `json:"apply_options,omitempty"`
	ProgressCallback     func(current, total int, item string)                                              `json:"-"`
	ByteProgressCallback func(currBytes, totalBytes int64, speed float64, eta int, pct float64, item string) `json:"-"`
}

// Engine is the core domain service orchestrating game scanning, patching, and launcher integrations.
type Engine struct {
	Config         *config.Config
	HTTPClient     *http.Client
	EventBus       *events.Bus
	Tasks          *tasks.Manager
	StoreRegistry  *store.Registry
	ArtworkService *store.ArtworkService
	Notifier       func(ctx context.Context, title, message string) error
	StepReporter   func(ctx context.Context, prog StepProgress)
	toolManager    *tools.Manager
}

// New creates a new Engine instance.
func New(cfg *config.Config) *Engine {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	bus := events.NewBus()
	tasksFile := filepath.Join(filepath.Dir(config.GetConfigPath("")), "tasks.json")
	storeAdapter := tasks.NewFileStore(tasksFile)
	eventSink := tasks.NewBusEventSink(bus)
	limiter := tasks.NewTokenBucketLimiter(cfg.TransferSpeedLimit)

	reg := store.NewRegistry()
	reg.Register(store.NewSteamProvider())
	reg.Register(store.NewGOGProvider())
	reg.Register(store.NewSGDBProvider(cfg.SteamGridDBAPIKey))

	artSvc := store.NewArtworkService(cfg.ArtworkDir, reg)

	taskMgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads:  2,
		MaxConcurrentOperations: 4,
		Store:                   storeAdapter,
		EventSink:               eventSink,
		Limiter:                 limiter,
	})

	eng := &Engine{
		Config:         cfg,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
		EventBus:       bus,
		Tasks:          taskMgr,
		StoreRegistry:  reg,
		ArtworkService: artSvc,
		toolManager:    tools.NewManager(cfg),
	}

	eng.registerDefaultTaskRunners()
	return eng
}

func (e *Engine) registerDefaultTaskRunners() {
	if e.Tasks == nil {
		return
	}

	e.Tasks.RegisterRunner(tasks.TypeDownload, func(ctx context.Context, t *tasks.Task, ctrl *tasks.Controller) error {
		appIDVal, ok := util.ParseAppID(t.AppID)
		if !ok || appIDVal == 0 {
			return fmt.Errorf("invalid AppID for download task %s: %q", t.ID, t.AppID)
		}
		if t.GameDir == "" {
			return fmt.Errorf("target directory cannot be empty for download task %s", t.ID)
		}

		wrappedCtx := e.wrapContext(ctx)
		ctrl.ReportStage("Resolving Keys & Manifests", "Contacting depot database...")

		dlOpts := DownloadGameOptions{
			AppID:     appIDVal,
			TargetDir: t.GameDir,
			AutoApply: true,
			ByteProgressCallback: func(currBytes, totalBytes int64, speed float64, eta int, pct float64, item string) {
				if err := ctrl.CheckPauseOrCancel(); err != nil {
					return
				}
				_ = ctrl.Throttle(int(speed / 10))
				ctrl.ReportProgress(currBytes, totalBytes, 0, 0, speed, eta, pct)
				ctrl.ReportStage("Downloading Depots", item)
			},
		}

		_, err := e.DownloadGame(wrappedCtx, dlOpts)
		return err
	})

	e.Tasks.RegisterRunner(tasks.TypeToolUpdate, func(ctx context.Context, t *tasks.Task, ctrl *tasks.Controller) error {
		toolKey := t.Metadata["tool_key"]
		if toolKey == "" {
			toolKey = "gbe"
		}
		tool, err := e.Tools().GetTool(toolKey)
		if err != nil {
			return err
		}
		ctrl.ReportStage("Updating Tool", fmt.Sprintf("Downloading %s...", tool.Name()))
		return tool.Update(e.wrapContext(ctx))
	})
}

func (e *Engine) wrapContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if e.HTTPClient != nil {
		ctx = steam.WithHTTPClient(ctx, e.HTTPClient)
		ctx = github.WithHTTPClient(ctx, e.HTTPClient)
		ctx = manifest.WithHTTPClient(ctx, e.HTTPClient)
	}
	return ctx
}

// Tools returns the Tool Manager instance.
func (e *Engine) Tools() *tools.Manager {
	if e.toolManager == nil {
		e.toolManager = tools.NewManager(e.Config)
	}
	return e.toolManager
}

// GetToolsStatus retrieves current statuses for all registered post-processing tools.
func (e *Engine) GetToolsStatus(ctx context.Context) ([]tools.ToolStatus, error) {
	return e.Tools().GetToolsStatus(ctx)
}

// ProcessGame executes the end-to-end post-download setup for a single game directory.
func ProcessGame(ctx context.Context, cfg *config.Config, opts ProcessOptions) (*ProcessResult, error) {
	eng := New(cfg)
	return eng.ProcessGame(ctx, opts)
}

func (e *Engine) reportStep(ctx context.Context, step, total int, title, details string) {
	if e.StepReporter != nil {
		e.StepReporter(ctx, StepProgress{
			Step:    step,
			Total:   total,
			Title:   title,
			Details: details,
		})
	}
}

// DownloadGame orchestrates key resolution, manifest persistence, depot chunk downloading, and optional post-download setup.
func (e *Engine) DownloadGame(ctx context.Context, opts DownloadGameOptions) (*downloader.Result, error) {
	ctx = e.wrapContext(ctx)
	if opts.AppID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID")
	}
	if opts.TargetDir == "" {
		opts.TargetDir = "."
	}

	e.reportStep(ctx, 1, 3, "Resolving Depot Keys & Manifests", fmt.Sprintf("AppID: %d", opts.AppID))

	hubcapKey := ""
	if e.Config != nil {
		hubcapKey = e.Config.HubcapAPIKey
	}

	parsedLua, err := manifest.ResolveKeys(ctx, opts.AppID, opts.LuaPath, hubcapKey)
	if err != nil {
		return nil, fmt.Errorf("resolve keys and manifests for AppID %d: %w", opts.AppID, err)
	}

	opts.AppID = parsedLua.AppID

	if !opts.DryRun && len(parsedLua.ManifestFiles) > 0 {
		manifestsDir := filepath.Join(opts.TargetDir, "[Manifests]")
		_ = os.MkdirAll(manifestsDir, 0755)
		for fname, fcontent := range parsedLua.ManifestFiles {
			parts := strings.Split(fname, "_")
			if len(parts) >= 2 {
				prefix := parts[0] + "_"
				if existingEntries, err := os.ReadDir(manifestsDir); err == nil {
					for _, ee := range existingEntries {
						if !ee.IsDir() && strings.HasPrefix(ee.Name(), prefix) && strings.HasSuffix(ee.Name(), ".manifest") && ee.Name() != fname {
							_ = os.Remove(filepath.Join(manifestsDir, ee.Name()))
						}
					}
				}
			}
			outPath := filepath.Join(manifestsDir, fname)
			_ = os.WriteFile(outPath, fcontent, 0644)
			slog.Info("Saved manifest file", "path", outPath)
		}
	}

	dummyManifest := &manifest.Manifest{
		AppID:     opts.AppID,
		DepotKeys: make(map[uint32]string),
	}
	for _, d := range parsedLua.Depots {
		dummyManifest.DepotKeys[d.DepotID] = d.DecryptionKey
	}
	if len(parsedLua.Depots) > 0 {
		dummyManifest.DepotID = parsedLua.Depots[0].DepotID
		dummyManifest.DecryptionKey = parsedLua.Depots[0].DecryptionKey
	}

	e.reportStep(ctx, 2, 3, "Streaming Depot Chunks", fmt.Sprintf("Target: %s", opts.TargetDir))

	dlOpts := downloader.DownloadOptions{
		TargetDir:            opts.TargetDir,
		AppID:                opts.AppID,
		LuaPath:              opts.LuaPath,
		Platform:             opts.Platform,
		DryRun:               opts.DryRun,
		WorkerLimit:          opts.WorkerLimit,
		AuditTrace:           parsedLua.AuditTrace,
		ProgressCallback:     opts.ProgressCallback,
		ByteProgressCallback: opts.ByteProgressCallback,
		HTTPClient:           e.HTTPClient,
	}

	res, err := downloader.DownloadOrUpdateGame(ctx, dummyManifest, dlOpts)
	if err != nil {
		return nil, fmt.Errorf("download game depot: %w", err)
	}

	if opts.AutoApply {
		e.reportStep(ctx, 3, 3, "Running Post-Download Setup", fmt.Sprintf("Target: %s", opts.TargetDir))
		applyOpts := ProcessOptions{
			Path:              opts.TargetDir,
			ApplyGBE:          true,
			AddLutris:         true,
			NormalizeDir:      false,
			DryRun:            opts.DryRun,
			AutoYes:           true,
			FetchAchievements: true,
		}
		if opts.ApplyOptions != nil {
			applyOpts = *opts.ApplyOptions
			applyOpts.Path = opts.TargetDir
		}
		if _, err := e.ProcessGame(ctx, applyOpts); err != nil {
			slog.Warn("Auto-apply setup failed following download", "targetDir", opts.TargetDir, "error", err)
		}
	}

	return res, nil
}

// UpdateGame checks for available Steam depot updates for an existing game directory, downloads changed files, and re-applies setup.
func (e *Engine) UpdateGame(ctx context.Context, gamePath string) (*GameStatus, error) {
	ctx = e.wrapContext(ctx)
	targetPath := gamePath
	if targetPath == "" {
		targetPath = "."
	}

	slog.Info("Updating game installation", "target", targetPath)

	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("detect game info: %w", err)
	}

	appIDVal, ok := util.ParseAppID(info.AppID)
	if !ok || appIDVal == 0 {
		return nil, fmt.Errorf("could not resolve valid numeric AppID for %s (detected %q)", info.Name, info.AppID)
	}

	dlOpts := DownloadGameOptions{
		AppID:     appIDVal,
		TargetDir: info.GameDir,
		Platform:  info.Platform,
		AutoApply: true,
	}

	if _, dlErr := e.DownloadGame(ctx, dlOpts); dlErr != nil {
		return nil, fmt.Errorf("update download failed: %w", dlErr)
	}

	return e.InspectStatus(ctx, info.GameDir)
}

// ProcessGame processes a game directory according to the supplied options.
func (e *Engine) ProcessGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	ctx = e.wrapContext(ctx)
	targetPath := opts.Path
	if targetPath == "" {
		targetPath = "."
	}

	game := domain.NewGame("", targetPath, "", "")
	pctx := &pipeline.Context{
		Options:  opts,
		Game:     game,
		Config:   e.Config,
		EventBus: e.EventBus,
	}

	pipe := pipeline.ForGame(game)
	pipeErr := pipe.Execute(ctx, pctx)

	// Re-inspect or reconstruct GameInfo
	info, _ := detector.Detect(ctx, pctx.Game.GameDir)
	if info == nil {
		info = pctx.Game
	}

	res := &ProcessResult{
		Info:                info,
		LutrisAdded:         pctx.LutrisAdded,
		Patched:             pctx.Patched,
		SteamlessUnpacked:   pctx.SteamlessUnpacked,
		ManifestsMoved:      true,
		DirectoryNormalized: pctx.DirectoryNormalized,
		ACFSynthesized:      pctx.ACFSynthesized,
		Errors:              pctx.Errors,
	}

	if pipeErr != nil {
		return res, pipeErr
	}

	return res, nil
}



// BatchProcess scans a parent directory containing multiple game folders and processes each discovered game.
func (e *Engine) BatchProcess(ctx context.Context, parentDir string, opts ProcessOptions) ([]*ProcessResult, error) {
	absParent, err := filepath.Abs(parentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve batch directory: %w", err)
	}

	entries, err := os.ReadDir(absParent)
	if err != nil {
		return nil, fmt.Errorf("read batch directory %s: %w", absParent, err)
	}

	var results []*ProcessResult
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gamePath := filepath.Join(absParent, entry.Name())

		// Skip hidden folders
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		slog.Info("Batch scanning candidate game directory", "path", gamePath)
		subOpts := opts
		subOpts.Path = gamePath

		res, err := e.ProcessGame(ctx, subOpts)
		if err != nil {
			slog.Warn("Failed to batch process game directory", "path", gamePath, "error", err)
			continue
		}
		results = append(results, res)
	}

	return results, nil
}

// BatchInspect scans all candidate game subdirectories in parentDir and returns their GameStatus.
func (e *Engine) BatchInspect(ctx context.Context, parentDir string) ([]*GameStatus, error) {
	ctx = e.wrapContext(ctx)
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
			status, _, err := e.InspectLocal(ctx, path)
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
func (e *Engine) BatchInspectLibraries(ctx context.Context, libDirs []string) ([]*GameStatus, error) {
	ctx = e.wrapContext(ctx)
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
		statuses, err := e.BatchInspect(ctx, expanded)
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

// InspectLocal performs fast static local inspection (executables, manifests, backups, configuration profile).
func (e *Engine) InspectLocal(ctx context.Context, targetPath string) (*GameStatus, *detector.GameInfo, error) {
	ctx = e.wrapContext(ctx)
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("detect game: %w", err)
	}

	status := info

	// Walk game directory to check for backup files, size, and file count
	_ = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			status.FileCount++
			if fi, err := d.Info(); err == nil {
				status.DiskSizeBytes += fi.Size()
			}
		}
		name := d.Name()
		if strings.Contains(name, ".ORIGINAL") {
			status.OriginalBackups = append(status.OriginalBackups, path)
		}
		if name == "steam_appid.txt" {
			status.SteamAppIDTxtFound = true
		}
		return nil
	})

	// Check manifests
	manifestsDir := filepath.Join(info.GameDir, "[Manifests]")
	if entries, err := os.ReadDir(manifestsDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".manifest") {
				parts := strings.Split(e.Name(), "_")
				if len(parts) >= 2 {
					status.ManifestID = strings.TrimSuffix(parts[1], ".manifest")
				} else {
					status.ManifestID = e.Name()
				}
				break
			}
		}
	}

	// Check Lutris registration
	if e.Config != nil {
		status.LutrisRegistered = lutris.IsGameRegistered(info.Name, e.Config.LutrisDir)
	}

	steamSettingsDir := filepath.Join(info.GameDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		status.SettingsDirExists = true
		langFile := filepath.Join(steamSettingsDir, "force_language.txt")
		if data, err := os.ReadFile(langFile); err == nil {
			status.CurrentLanguage = strings.TrimSpace(string(data))
		}
	}

	if status.CurrentLanguage == "" && e.Config != nil && e.Config.Language != "" {
		status.CurrentLanguage = e.Config.Language
	}

	coldClientIni := filepath.Join(info.GameDir, "ColdClientLoader.ini")
	if len(status.OriginalBackups) > 0 {
		status.State = "Portable-Patched"
	} else if util.FileExists(coldClientIni) || status.SettingsDirExists || status.SteamAppIDTxtFound {
		status.State = "Loader-Configured"
	}

	// Load custom LaunchProfile if present
	if profile, err := runner.GetLaunchProfile(info.GameDir); err == nil && profile != nil {
		status.LaunchProfile = profile
		if profile.ExePath != "" {
			exePath := profile.ExePath
			if !filepath.IsAbs(exePath) {
				exePath = filepath.Join(info.GameDir, exePath)
			}
			realLower := strings.ToLower(filepath.Base(exePath))
			if util.FileExists(exePath) && !strings.Contains(realLower, "steamclient_loader") && !strings.Contains(realLower, "coldclientloader") {
				status.ExePath = exePath
				info.ExePath = exePath
			}
		}
		if profile.ExeArgs != "" {
			status.ExeArgs = profile.ExeArgs
			info.ExeArgs = profile.ExeArgs
		}
		if profile.OperatingMode != "" {
			status.OperatingMode = profile.OperatingMode
		} else if e.Config != nil && e.Config.OperatingMode != "" {
			status.OperatingMode = e.Config.OperatingMode
		}
		if profile.Runner != "" {
			status.Runner = profile.Runner
		} else if e.Config != nil && e.Config.DefaultRunner != "" {
			status.Runner = e.Config.DefaultRunner
		}
		if profile.WinePrefix != "" {
			status.WinePrefix = profile.WinePrefix
		} else if e.Config != nil {
			status.WinePrefix = e.Config.WinePrefix
		}
		if profile.Store != "" {
			status.Store = profile.Store
			status.Source = profile.Store
		}
		if profile.StoreAppID != "" {
			status.AppID = profile.StoreAppID
			status.NumericAppID, _ = util.ParseAppID(profile.StoreAppID)
			status.ExternalIDs.SteamAppID = status.NumericAppID
		}
		if profile.GOGID != "" {
			status.GOGID = profile.GOGID
		}
		if profile.SGDBID != "" {
			status.SGDBID = profile.SGDBID
		}
		if profile.CustomCoverURL != "" {
			status.CustomCoverURL = profile.CustomCoverURL
		}
		if profile.CustomBannerURL != "" {
			status.CustomBannerURL = profile.CustomBannerURL
		}
		if profile.CustomHeroURL != "" {
			status.CustomHeroURL = profile.CustomHeroURL
		}
		if profile.CustomLogoURL != "" {
			status.CustomLogoURL = profile.CustomLogoURL
		}
		if profile.CustomIconURL != "" {
			status.CustomIconURL = profile.CustomIconURL
		}
		if profile.LockedArtwork {
			status.LockedArtwork = true
		}
	} else if e.Config != nil {
		status.OperatingMode = e.Config.OperatingMode
		status.Runner = e.Config.DefaultRunner
		status.WinePrefix = e.Config.WinePrefix
	}

	if status.OperatingMode == "" {
		status.OperatingMode = runner.ModeStandalone
	}

	status.IsPatched = status.State == "Portable-Patched" || status.State == "Loader-Configured" || status.SettingsDirExists
	status.IsPortable = status.State == "Portable-Patched"
	status.HasUpdatesOrIssues = status.Store == "Steam" && (status.HasUpdate || len(status.ModifiedFiles) > 0 || len(status.MissingFiles) > 0)

	// Compute overall ready state
	if status.OperatingMode == runner.ModeLutris {
		status.IsReady = status.LutrisRegistered
	} else {
		// Standalone mode is ready if binaries/loader exist or patched
		status.IsReady = status.State == "Portable-Patched" || status.State == "Loader-Configured" || status.ExePath != ""
	}

	if e.ArtworkService != nil {
		_, _ = e.ArtworkService.Resolve(ctx, status)
	}

	return status, info, nil
}

// InspectStatus inspects a game directory to determine its patch and configuration state, enriched with remote metadata.
func (e *Engine) InspectStatus(ctx context.Context, targetPath string) (*GameStatus, error) {
	ctx = e.wrapContext(ctx)
	status, info, err := e.InspectLocal(ctx, targetPath)
	if err != nil {
		return nil, err
	}

	// Fetch remote Steam news and languages if AppID is present
	if info.AppID != "" && info.AppID != "0" {
		if news, err := steam.FetchAppNews(ctx, info.AppID); err == nil {
			status.RecentPatchNote = news
		}
		if langs, err := steam.FetchSupportedLanguages(ctx, info.AppID); err == nil {
			status.SupportedLanguages = langs
		}
	}

	return status, nil
}

// LaunchGame starts the game process asynchronously using the configured operating mode and runner.
func (e *Engine) LaunchGame(ctx context.Context, opts runner.LaunchOptions) (*exec.Cmd, error) {
	return runner.Launch(ctx, e.Config, opts)
}

// RunGame executes the game process synchronously, streaming output to stdout/stderr.
func (e *Engine) RunGame(ctx context.Context, opts runner.LaunchOptions) error {
	return runner.Run(ctx, e.Config, opts)
}

// MigrateOperatingMode updates the launch configuration and Lutris registration for a game to match targetMode ("standalone" or "lutris").
func (e *Engine) MigrateOperatingMode(ctx context.Context, gameDir string, targetMode string, unregisterLutris bool) error {
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	info, err := detector.Detect(ctx, gameDir)
	if err != nil {
		return fmt.Errorf("detect game: %w", err)
	}

	// Load existing launch profile or initialize from detected info
	profile, _ := runner.GetLaunchProfile(info.GameDir)
	if profile == nil {
		profile = &runner.LaunchOptions{
			GameDir:  info.GameDir,
			ExePath:  info.ExePath,
			ExeArgs:  info.ExeArgs,
			AppID:    info.AppID,
			Name:     info.Name,
			Platform: info.Platform,
		}
	}

	profile.OperatingMode = targetMode

	targetDir := e.Config.LutrisDir
	if !filepath.IsAbs(targetDir) {
		home, err := os.UserHomeDir()
		if err == nil {
			targetDir = filepath.Join(home, targetDir)
		}
	}

	if targetMode == runner.ModeLutris {
		runnerType := "wine"
		if info.Platform == "linux" {
			runnerType = "linux"
		}

		exePath := profile.ExePath
		if exePath == "" {
			exePath = info.ExePath
		}
		if !filepath.IsAbs(exePath) {
			exePath = filepath.Join(info.GameDir, exePath)
		}

		targetExe := exePath
		envMap := make(map[string]string)
		for k, v := range profile.CustomEnv {
			envMap[k] = v
		}

		if info.Platform != "linux" {
			loaderX64 := filepath.Join(info.GameDir, "steamclient_loader_x64.exe")
			loaderX86 := filepath.Join(info.GameDir, "steamclient_loader_x86.exe")
			coldIni := filepath.Join(info.GameDir, "ColdClientLoader.ini")

			if util.FileExists(coldIni) {
				if info.Platform == "win32" && util.FileExists(loaderX86) {
					targetExe = loaderX86
					envMap["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
				} else if util.FileExists(loaderX64) {
					targetExe = loaderX64
					envMap["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
				}
			}
		}

		prefixPath := profile.WinePrefix
		if prefixPath == "" {
			prefixPath = lutris.GetExistingWinePrefix(lutris.Slugify(info.Name), targetDir)
		}

		lcfg := lutris.Config{
			Name:       info.Name,
			Slug:       lutris.Slugify(info.Name),
			GamePath:   targetExe,
			PrefixPath: prefixPath,
			Runner:     runnerType,
			Arch:       info.Platform,
			Args:       profile.ExeArgs,
			Env:        envMap,
			Version:    "Gamux",
		}
		if err := lutris.Write(lcfg, targetDir); err != nil {
			slog.Warn("Failed to register Lutris during migration", "error", err)
		}
	} else if targetMode == runner.ModeStandalone && unregisterLutris {
		slug := lutris.Slugify(info.Name)
		if err := lutris.UnregisterLutris(slug, targetDir); err != nil {
			slog.Warn("Failed to unregister Lutris during standalone migration", "slug", slug, "error", err)
		}
	}

	return runner.SaveLaunchProfile(info.GameDir, *profile)
}

// GetLaunchProfile retrieves the custom launch configuration for a game directory.
func (e *Engine) GetLaunchProfile(ctx context.Context, gameDir string) (*runner.LaunchOptions, error) {
	return gamecfg.Load(gameDir)
}

// SaveLaunchProfile persists custom launch configuration and synchronizes on-disk integrations.
func (e *Engine) SaveLaunchProfile(ctx context.Context, gameDir string, opts runner.LaunchOptions) error {
	opts.GameDir = gameDir
	return gamecfg.Sync(ctx, &opts, e.Config)
}

// ResetLaunchProfile removes custom launch configuration from <gameDir>/steam_settings/launch_profile.json.
func (e *Engine) ResetLaunchProfile(ctx context.Context, gameDir string) error {
	return gamecfg.Reset(gameDir)
}

// ApplyGame performs the end-to-end One-Shot lifecycle: Inspect -> ACF -> GBE Setup -> Lutris Register.
func (e *Engine) ApplyGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	status, err := e.InspectStatus(ctx, opts.Path)
	if err != nil {
		slog.Warn("Apply inspection warning", "error", err)
	}

	if status != nil && status.HasUpdate {
		slog.Info("Apply detected available Steam depot update", "title", status.Name, "localManifest", status.ManifestID)
	}

	return e.ProcessGame(ctx, opts)
}

// InspectStatusEx extends InspectStatus with customizable news history depth.
func (e *Engine) InspectStatusEx(ctx context.Context, targetPath string, newsCount int) (*GameStatus, error) {
	ctx = e.wrapContext(ctx)
	status, err := e.InspectStatus(ctx, targetPath)
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

// VerifyIntegrity runs an on-demand full depot integrity comparison for a game directory.
func (e *Engine) VerifyIntegrity(ctx context.Context, targetPath string) (*manifest.IntegrityReport, error) {
	ctx = e.wrapContext(ctx)
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

// GetPatchNote fetches and retrieves a single patch note at itemIndex (1-indexed) for a game path or AppID.
func (e *Engine) GetPatchNote(ctx context.Context, targetPath string, itemIndex int) (*steam.NewsItem, string, error) {
	ctx = e.wrapContext(ctx)
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

// Rollback restores original backup files and cleans up generated configuration files in a game directory.
func (e *Engine) Rollback(ctx context.Context, targetPath string, dryRun bool) error {
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("detect game for rollback: %w", err)
	}

	slog.Info("Starting rollback for game", "name", info.Name, "dir", info.GameDir)

	var restoredCount int
	err = filepath.WalkDir(info.GameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(d.Name(), ".ORIGINAL") {
			idx := strings.Index(path, ".20")
			origPath := path
			if idx != -1 {
				origPath = path[:idx]
			} else {
				origPath = strings.TrimSuffix(path, ".ORIGINAL")
			}

			if dryRun {
				slog.Info("[DRY RUN] Would restore original file", "from", path, "to", origPath)
			} else {
				if err := os.Rename(path, origPath); err != nil {
					slog.Error("Failed to restore original file", "from", path, "to", origPath, "error", err)
				} else {
					slog.Info("Restored original file", "restored", origPath)
					restoredCount++
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir during rollback: %w", err)
	}

	// Clean up loader files and configurations
	loaderFiles := []string{
		"ColdClientLoader.ini",
		"steamclient_loader_x64.exe",
		"steamclient_loader_x86.exe",
		"steamclient64.dll",
		"steamclient.dll",
		"GameOverlayRenderer64.dll",
		"GameOverlayRenderer.dll",
		"steam_interfaces.txt",
		".steam_interfaces.hash",
	}

	for _, fname := range loaderFiles {
		p := filepath.Join(info.GameDir, fname)
		if util.FileExists(p) {
			if dryRun {
				slog.Info("[DRY RUN] Would remove loader file", "path", p)
			} else {
				_ = os.Remove(p)
				slog.Info("Removed loader file", "path", p)
			}
		}
	}

	steamSettingsDir := filepath.Join(info.GameDir, "steam_settings")
	if _, err := os.Stat(steamSettingsDir); err == nil {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_settings directory", "path", steamSettingsDir)
		} else {
			if err := os.RemoveAll(steamSettingsDir); err == nil {
				slog.Info("Removed steam_settings directory", "path", steamSettingsDir)
			}
		}
	}

	appIDTxt := filepath.Join(info.GameDir, "steam_appid.txt")
	if util.FileExists(appIDTxt) {
		if dryRun {
			slog.Info("[DRY RUN] Would remove steam_appid.txt", "path", appIDTxt)
		} else {
			_ = os.Remove(appIDTxt)
			slog.Info("Removed steam_appid.txt", "path", appIDTxt)
		}
	}

	// Clean up Lutris integration if present
	slug := lutris.Slugify(info.Name)
	targetDir := e.Config.LutrisDir
	if !filepath.IsAbs(targetDir) {
		home, err := os.UserHomeDir()
		if err == nil {
			targetDir = filepath.Join(home, targetDir)
		}
	}

	if dryRun {
		slog.Info("[DRY RUN] Would unregister game from Lutris and clean up Lutris configs", "slug", slug)
	} else {
		if err := lutris.UnregisterLutris(slug, targetDir); err == nil {
			slog.Info("Unregistered game from Lutris and cleaned up Lutris configs/shortcuts", "slug", slug)
		}
	}

	slog.Info("Rollback completed", "restoredFiles", restoredCount)
	return nil
}

// LibraryUpdateStatus represents update availability status for a game in the library.
type LibraryUpdateStatus struct {
	GameTitle       string
	AppID           string
	GameDir         string
	LocalManifestID string
	HasUpdate       bool
}

// CheckLibraryUpdates scans a directory containing game folders and checks for available Steam updates.
func (e *Engine) CheckLibraryUpdates(ctx context.Context, parentDir string) ([]LibraryUpdateStatus, error) {
	ctx = e.wrapContext(ctx)
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

// NotifyLaunch inspects a game directory for updates or configuration issues, and triggers a console/desktop notification if found.
func (e *Engine) NotifyLaunch(ctx context.Context, gameDir string) error {
	status, err := e.InspectStatus(ctx, gameDir)
	if err != nil {
		slog.Warn("NotifyLaunch status inspection failed", "dir", gameDir, "error", err)
		return nil // Don't block launch if inspection fails
	}

	var issues []string
	if status.HasUpdate {
		issues = append(issues, "An update is available for this game on Steam.")
	}
	if len(status.MissingFiles) > 0 {
		issues = append(issues, fmt.Sprintf("%d official game files are missing.", len(status.MissingFiles)))
	}
	if len(status.ModifiedFiles) > 0 {
		issues = append(issues, fmt.Sprintf("%d game files have been modified.", len(status.ModifiedFiles)))
	}

	if len(issues) == 0 {
		return nil
	}

	msg := fmt.Sprintf("Game: %s (AppID: %s)\nPath: %s\n\nIssues Detected:\n", status.Name, status.AppID, status.GameDir)
	for _, issue := range issues {
		msg += fmt.Sprintf(" • %s\n", issue)
	}
	msg += fmt.Sprintf("\nSuggested Action: Run 'gamux apply %q' to resolve.", status.GameDir)

	slog.Info("Pre-launch notification sent", "title", status.Name, "issuesCount", len(issues))
	if e.Notifier != nil {
		return e.Notifier(ctx, status.Name, msg)
	}

	return nil
}


func (e *Engine) resolveGameName(ctx context.Context, path string) string {
	if info, err := detector.Detect(ctx, path); err == nil && info != nil && info.Name != "" {
		return info.Name
	}
	return filepath.Base(path)
}

// SubmitApply schedules an asynchronous game post-processing setup task.
func (e *Engine) SubmitApply(opts ProcessOptions) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := e.resolveGameName(context.Background(), opts.Path)
	title := fmt.Sprintf("Apply Setup (%s)", gameName)
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeApply,
		Title:        title,
		GameName:     gameName,
		GameDir:      opts.Path,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Configuring Game", fmt.Sprintf("Applying GBE and launcher profile for %s...", gameName))
			_, err := e.ProcessGame(ctx, opts)
			return err
		},
	})
}

// SubmitUpdate schedules an asynchronous game update task.
func (e *Engine) SubmitUpdate(gamePath string) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := e.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Update %s", gameName)
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeUpdate,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Checking Updates", fmt.Sprintf("Checking Steam depot changes for %s...", gameName))
			_, err := e.UpdateGame(ctx, gamePath)
			return err
		},
	})
}

// SubmitVerify schedules an asynchronous game depot integrity verification task.
func (e *Engine) SubmitVerify(gamePath string) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := e.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Verify Integrity (%s)", gameName)
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeVerify,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Verifying Integrity", fmt.Sprintf("Scanning files in %s...", gameName))
			_, err := e.VerifyIntegrity(ctx, gamePath)
			return err
		},
	})
}

// SubmitNormalize schedules an asynchronous directory normalization task.
func (e *Engine) SubmitNormalize(gamePath string) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := e.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Normalize Directory (%s)", gameName)
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeNormalize,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Normalizing Directory", fmt.Sprintf("Standardizing directory naming for %s...", gameName))
			opts := ProcessOptions{
				Path:         gamePath,
				NormalizeDir: true,
				ApplyGBE:     false,
				AddLutris:    false,
				NoSteamless:  true,
				AutoYes:      true,
			}
			res, err := e.ProcessGame(ctx, opts)
			if err == nil && res != nil && res.Info != nil && res.Info.GameDir != "" {
				ctrl.ReportStage("Completed", fmt.Sprintf("Directory normalized to %s", res.Info.GameDir))
			}
			return err
		},
	})
}

// SubmitRollback schedules an asynchronous game rollback task.
func (e *Engine) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := e.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Rollback %s", gameName)
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeRollback,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Restoring Backups", fmt.Sprintf("Reverting patched binaries for %s...", gameName))
			return e.Rollback(ctx, gamePath, dryRun)
		},
	})
}

// SubmitMigrateMode schedules an asynchronous operating mode migration task.
func (e *Engine) SubmitMigrateMode(gamePath string, targetMode string, unregisterLutris bool) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := e.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Migrate Mode (%s)", gameName)
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeMigrate,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Migrating Mode", fmt.Sprintf("Switching %s to %s mode...", gameName, targetMode))
			return e.MigrateOperatingMode(ctx, gamePath, targetMode, unregisterLutris)
		},
	})
}

// SubmitDownload schedules an asynchronous depot download transfer with state persistence, speed telemetry, and pause/resume support.
func (e *Engine) SubmitDownload(opts DownloadGameOptions) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	if opts.AppID == 0 {
		return "", fmt.Errorf("invalid AppID 0")
	}
	if opts.TargetDir == "" {
		return "", fmt.Errorf("target directory cannot be empty")
	}

	gameName := fmt.Sprintf("Steam AppID %d", opts.AppID)
	if p, ok := e.StoreRegistry.Get("steam"); ok {
		if meta, err := p.GetMetadata(context.Background(), fmt.Sprintf("%d", opts.AppID)); err == nil && meta.Name != "" {
			gameName = meta.Name
		}
	}

	appIDStr := fmt.Sprintf("%d", opts.AppID)
	for _, existing := range e.Tasks.List() {
		if existing.Type == tasks.TypeDownload && existing.AppID == appIDStr {
			if existing.Status == tasks.StatusRunning || existing.Status == tasks.StatusQueued || existing.Status == tasks.StatusPaused {
				slog.Info("Download already active/queued, reusing existing task", "appID", opts.AppID, "taskID", existing.ID)
				return existing.ID, nil
			}
		}
	}

	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeDownload,
		Title:        fmt.Sprintf("Download %s", gameName),
		GameName:     gameName,
		GameDir:      opts.TargetDir,
		AppID:        fmt.Sprintf("%d", opts.AppID),
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapPersist | tasks.CapBandwidthLimit,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			wrappedCtx := e.wrapContext(ctx)
			ctrl.ReportStage("Resolving Keys & Manifests", "Contacting depot database...")

			opts.ByteProgressCallback = func(currBytes, totalBytes int64, speed float64, eta int, pct float64, item string) {
				if err := ctrl.CheckPauseOrCancel(); err != nil {
					return
				}
				_ = ctrl.Throttle(int(speed / 10))
				ctrl.ReportProgress(currBytes, totalBytes, 0, 0, speed, eta, pct)
				ctrl.ReportStage("Downloading Depots", item)
			}

			origCallback := opts.ProgressCallback
			opts.ProgressCallback = func(curr, total int, item string) {
				if origCallback != nil {
					origCallback(curr, total, item)
				}
			}

			_, err := e.DownloadGame(wrappedCtx, opts)
			return err
		},
	})
}

// SearchStoreGames searches across all registered store providers (Steam, GOG, etc.).
func (e *Engine) SearchStoreGames(ctx context.Context, query string) ([]store.SearchResult, error) {
	if e.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return e.StoreRegistry.SearchAll(e.wrapContext(ctx), query)
}

// ResolveArtwork resolves covers, banners, heroes, and logos for a game.
func (e *Engine) ResolveArtwork(ctx context.Context, game *domain.Game) (*domain.GameArtwork, error) {
	if e.ArtworkService == nil {
		return nil, fmt.Errorf("artwork service not initialized")
	}
	return e.ArtworkService.Resolve(e.wrapContext(ctx), game)
}

// SearchArtworkCandidates queries registered providers (SteamGridDB, Steam, GOG) for artwork matching a title.
func (e *Engine) SearchArtworkCandidates(ctx context.Context, query string) ([]store.SearchResult, error) {
	if e.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return e.StoreRegistry.SearchAll(e.wrapContext(ctx), query)
}

// QueryArtworkOptions queries candidate providers for specific asset options (covers, heroes, banners, logos).
func (e *Engine) QueryArtworkOptions(ctx context.Context, query domain.ArtworkQuery) ([]domain.ArtworkOption, error) {
	if e.StoreRegistry == nil {
		return nil, fmt.Errorf("store registry not initialized")
	}
	return e.StoreRegistry.QueryArtworkOptions(e.wrapContext(ctx), query)
}

// ApplyCustomArtwork sets custom artwork on a game and persists the choice in the launch profile.
func (e *Engine) ApplyCustomArtwork(ctx context.Context, gameDir string, assetType string, imageSource string) (*domain.GameArtwork, error) {
	ctx = e.wrapContext(ctx)
	if e.ArtworkService == nil {
		return nil, fmt.Errorf("artwork service not initialized")
	}

	status, err := e.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, fmt.Errorf("inspect game: %w", err)
	}

	art, err := e.ArtworkService.ApplyCustomArtwork(ctx, status, assetType, imageSource)
	if err != nil {
		return nil, err
	}

	// Persist to launch profile
	profile, err := gamecfg.Load(gameDir)
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
	_ = gamecfg.Save(profile)

	return art, nil
}

// LinkGameMetadata links a game installation to a specific metadata provider and ID.
func (e *Engine) LinkGameMetadata(ctx context.Context, gameDir, storeName, storeID string) (*domain.Game, error) {
	ctx = e.wrapContext(ctx)
	profile, err := gamecfg.Load(gameDir)
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
		if e.StoreRegistry != nil {
			if p, ok := e.StoreRegistry.Get("gog"); ok {
				if meta, err := p.GetMetadata(ctx, storeID); err == nil && meta != nil && meta.Name != "" {
					profile.Name = meta.Name
				}
			}
		}
	case "standalone", "custom":
		profile.AppID = "0"
		profile.StoreAppID = ""
		profile.GOGID = ""
	}

	if err := gamecfg.Save(profile); err != nil {
		return nil, fmt.Errorf("save launch profile: %w", err)
	}
	_ = gamecfg.Sync(ctx, profile, e.Config)

	status, err := e.InspectStatus(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	return status, nil
}

// SubmitToolUpdate schedules an asynchronous tool update download (GBE or Steamless) through the unified Tasks engine.
func (e *Engine) SubmitToolUpdate(toolKey string) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	tool, err := e.Tools().GetTool(toolKey)
	if err != nil {
		return "", err
	}

	targetDir := ""
	switch strings.ToLower(toolKey) {
	case "gbe":
		targetDir = util.ExpandPath(e.Config.GbeDir)
	case "steamless":
		targetDir = util.ExpandPath(e.Config.SteamlessDir)
	case "umu":
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".local", "share", "umu")
	default:
		targetDir = util.ExpandPath(e.Config.GbeDir)
	}

	displayName := tool.Name()
	if strings.ToLower(toolKey) == "gbe" {
		displayName = "Goldberg Steam Emulator (GBE)"
	} else if strings.ToLower(toolKey) == "steamless" {
		displayName = "Steamless-rs DRM Unpacker"
	} else if strings.ToLower(toolKey) == "umu" {
		displayName = "UMU Proton Launcher (umu-run)"
	}

	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeToolUpdate,
		Title:        fmt.Sprintf("Update %s", displayName),
		GameName:     displayName,
		GameDir:      targetDir,
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapBandwidthLimit,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			wrappedCtx := e.wrapContext(ctx)
			ctrl.ReportStage("Checking GitHub Releases", fmt.Sprintf("Querying latest release for %s...", tool.Name()))

			if err := ctrl.CheckPauseOrCancel(); err != nil {
				return err
			}

			ctrl.ReportStage("Downloading Tool Release", fmt.Sprintf("Fetching %s assets to %s", tool.Name(), targetDir))
			if err := tool.Update(wrappedCtx); err != nil {
				return fmt.Errorf("update %s: %w", tool.Name(), err)
			}

			ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
			ctrl.ReportStage("Completed", fmt.Sprintf("%s updated successfully", tool.Name()))
			return nil
		},
	})
}

// SubmitWorkshop schedules an asynchronous Steam workshop mod download.
func (e *Engine) SubmitWorkshop(itemURLOrID, targetDir string, dryRun bool) (string, error) {
	if e.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	if targetDir == "" {
		targetDir = "."
	}
	return e.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeWorkshop,
		Title:        fmt.Sprintf("Download Workshop Item (%s)", itemURLOrID),
		GameDir:      targetDir,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Fetching Workshop Details", itemURLOrID)
			return steam.DownloadWorkshopItem(ctx, itemURLOrID, targetDir, dryRun)
		},
	})
}



