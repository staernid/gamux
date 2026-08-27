package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/pipeline"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/util"
)

type GameService struct {
	sc      *ServiceContext
	library *LibraryService
}

func NewGameService(sc *ServiceContext, library *LibraryService) *GameService {
	return &GameService{sc: sc, library: library}
}

// ProcessGame processes a game directory according to the supplied options.
func (gs *GameService) ProcessGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	ctx = gs.sc.WrapContext(ctx)
	targetPath := opts.Path
	if targetPath == "" {
		targetPath = "."
	}

	game := domain.NewGame("", targetPath, "", "")
	pctx := &pipeline.Context{
		Options:  opts,
		Game:     game,
		Config:   gs.sc.Config,
		EventBus: gs.sc.EventBus,
		Reporter: opts.Reporter,
	}

	pipe := pipeline.ForGame(game)
	pipeErr := pipe.Execute(ctx, pctx)

	pctx.Game.RecomputeState()

	res := &ProcessResult{
		Info:                pctx.Game,
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
func (gs *GameService) BatchProcess(ctx context.Context, parentDir string, opts ProcessOptions) ([]*ProcessResult, error) {
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

		res, err := gs.ProcessGame(ctx, subOpts)
		if err != nil {
			slog.Warn("Failed to batch process game directory", "path", gamePath, "error", err)
			continue
		}
		results = append(results, res)
	}

	return results, nil
}

// ApplyGame performs the end-to-end One-Shot lifecycle: Inspect -> ACF -> GBE Setup.
func (gs *GameService) ApplyGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	status, err := gs.library.InspectStatus(ctx, opts.Path)
	if err != nil {
		slog.Warn("Apply inspection warning", "error", err)
	}

	if status != nil && status.HasUpdate {
		slog.Info("Apply detected available Steam depot update", "title", status.Name, "localManifest", status.ManifestID)
	}

	return gs.ProcessGame(ctx, opts)
}

// Rollback restores original backup files and cleans up generated configuration files in a game directory.
func (gs *GameService) Rollback(ctx context.Context, targetPath string, dryRun bool) error {
	info, err := detector.Detect(ctx, targetPath)
	if err != nil {
		return fmt.Errorf("detect game for rollback: %w", err)
	}

	slog.Info("Starting rollback for game", "name", info.Name, "dir", info.GameDir)
	restoredCount, err := gbe.Revert(info.GameDir, dryRun)
	if err != nil {
		return fmt.Errorf("rollback game %s: %w", info.Name, err)
	}

	slog.Info("Rollback completed", "game", info.Name, "restoredFiles", restoredCount)
	return nil
}

func (gs *GameService) runnerManager() *runner.Manager {
	if gs.sc.RunnerMgr != nil {
		return gs.sc.RunnerMgr
	}
	return runner.DefaultManager()
}

// LaunchGame starts the game process asynchronously using the configured operating mode and runner.
// Profile loading, executable resolution fallback, and environment synthesis are encapsulated in the runner.
func (gs *GameService) LaunchGame(ctx context.Context, opts runner.LaunchOptions) (*exec.Cmd, error) {
	if opts.GameDir == "" && opts.ExePath == "" {
		return nil, fmt.Errorf("game directory or executable path cannot be empty")
	}
	return gs.runnerManager().Start(ctx, gs.sc.Config, opts)
}

// EnsureAndLaunchGame performs one-click Quick Play: if the game is unpatched/unconfigured,
// it auto-applies setup first, then starts the game asynchronously.
func (gs *GameService) EnsureAndLaunchGame(ctx context.Context, opts runner.LaunchOptions) (*exec.Cmd, error) {
	if opts.GameDir != "" {
		status, err := gs.library.InspectStatus(ctx, opts.GameDir)
		if err == nil && status != nil && (status.IsClean() || !status.IsReady) {
			slog.Info("EnsureAndLaunchGame auto-applying setup before launch", "dir", opts.GameDir, "game", status.Name)
			applyOpts := ProcessOptions{
				Path:       opts.GameDir,
				ApplyGBE:   true,
				Portable:   true,
				WinePrefix: opts.WinePrefix,
				ExePath:    opts.ExePath,
				ExeArgs:    opts.ExeArgs,
			}
			if _, applyErr := gs.ApplyGame(ctx, applyOpts); applyErr != nil {
				return nil, fmt.Errorf("auto-apply before launch failed: %w", applyErr)
			}
		}
	}

	return gs.LaunchGame(ctx, opts)
}

// RunGame executes the game process synchronously, streaming output to stdout/stderr.
func (gs *GameService) RunGame(ctx context.Context, opts runner.LaunchOptions) error {
	return runner.Run(ctx, gs.sc.Config, opts)
}

// GetLaunchProfile retrieves the custom launch configuration for a game directory.
func (gs *GameService) GetLaunchProfile(ctx context.Context, gameDir string) (*runner.LaunchOptions, error) {
	return runner.LoadLaunchProfile(gameDir)
}

// SaveLaunchProfile persists custom launch configuration and synchronizes on-disk integrations.
func (gs *GameService) SaveLaunchProfile(ctx context.Context, gameDir string, opts runner.LaunchOptions) error {
	opts.GameDir = gameDir
	return runner.SyncLaunchProfile(ctx, &opts, gs.sc.Config)
}

// ResetLaunchProfile removes custom launch configuration from <gameDir>/steam_settings/launch_profile.json.
func (gs *GameService) ResetLaunchProfile(ctx context.Context, gameDir string) error {
	return runner.ResetLaunchProfile(gameDir)
}

// NotifyLaunch inspects a game directory for updates or configuration issues, and triggers a console/desktop notification if found.
func (gs *GameService) NotifyLaunch(ctx context.Context, gameDir string) error {
	status, err := gs.library.InspectStatus(ctx, gameDir)
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
	return gs.sc.Notify(ctx, status.Name, msg)
}

// GetGameLog reads the persisted output log for the last/current run of a game.
func (gs *GameService) GetGameLog(ctx context.Context, gameDir string) (string, error) {
	if gameDir == "" {
		return "", fmt.Errorf("game directory cannot be empty")
	}

	gameName := ""
	appID := ""
	if status, err := gs.library.InspectStatus(ctx, gameDir); err == nil && status != nil {
		gameName = status.Name
		appID = status.AppID
	}

	return runner.GetLog(gameDir, gameName, appID)
}

// DownloadGame orchestrates key resolution, manifest persistence, depot chunk downloading, and optional post-download setup.
func (gs *GameService) DownloadGame(ctx context.Context, opts DownloadGameOptions) (*downloader.Result, error) {
	ctx = gs.sc.WrapContext(ctx)
	if opts.AppID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID")
	}
	if opts.TargetDir == "" {
		opts.TargetDir = "."
	}

	hubcapKey := ""
	if gs.sc.Config != nil {
		hubcapKey = gs.sc.Config.HubcapAPIKey
	}

	req := downloader.DownloadRequest{
		AppID:        opts.AppID,
		TargetDir:    opts.TargetDir,
		Platform:     opts.Platform,
		LuaPath:      opts.LuaPath,
		HubcapAPIKey: hubcapKey,
		WorkerLimit:  opts.WorkerLimit,
		DryRun:       opts.DryRun,
		HTTPClient:   gs.sc.HTTPClient,
		CheckCancel:  opts.CheckCancel,
		Throttle:     opts.Throttle,
		OnStage: func(stage, details string) {
			gs.sc.reportStep(ctx, 1, 3, stage, details)
		},
		OnProgress: func(p downloader.ProgressUpdate) {
			if opts.OnProgress != nil {
				opts.OnProgress(p)
			}
			if opts.ByteProgressCallback != nil {
				opts.ByteProgressCallback(p.DownloadedBytes, p.TotalBytes, p.SpeedBps, p.ETASeconds, p.Percent, p.Item)
			}
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(p.CurrentItems, p.TotalItems, p.Item)
			}
		},
	}

	res, err := downloader.Download(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("download game depot: %w", err)
	}

	if opts.AutoApply {
		gs.sc.reportStep(ctx, 3, 3, "Running Post-Download Setup", fmt.Sprintf("Target: %s", opts.TargetDir))
		applyOpts := ProcessOptions{
			Path:              opts.TargetDir,
			ApplyGBE:          true,
			NormalizeDir:      false,
			DryRun:            opts.DryRun,
			AutoYes:           true,
			FetchAchievements: true,
		}
		if opts.ApplyOptions != nil {
			applyOpts = *opts.ApplyOptions
			applyOpts.Path = opts.TargetDir
		}
		if _, err := gs.ProcessGame(ctx, applyOpts); err != nil {
			slog.Warn("Auto-apply setup failed following download", "targetDir", opts.TargetDir, "error", err)
		}
	}

	return res, nil
}

// UpdateGame checks for available Steam depot updates for an existing game directory, downloads changed files, and re-applies setup.
func (gs *GameService) UpdateGame(ctx context.Context, gamePath string) (*GameStatus, error) {
	ctx = gs.sc.WrapContext(ctx)
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

	if _, dlErr := gs.DownloadGame(ctx, dlOpts); dlErr != nil {
		return nil, fmt.Errorf("update download failed: %w", dlErr)
	}

	return gs.library.InspectStatus(ctx, info.GameDir)
}
