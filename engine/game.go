package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/staernid/gamux/cache"
	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/detector"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/ini"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/pipeline"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// ProcessGame processes a game directory according to the supplied options.
func (eng *Engine) ProcessGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	return eng.ApplyGame(ctx, opts)
}

// BatchProcess scans a parent directory containing multiple game folders and processes each discovered game.
func (eng *Engine) BatchProcess(ctx context.Context, parentDir string, opts ProcessOptions) ([]*ProcessResult, error) {
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

		if detector.IsIgnoredLibraryDir(gamePath) {
			continue
		}

		slog.Info("Batch scanning candidate game directory", "path", gamePath)
		subOpts := opts
		subOpts.Path = gamePath

		res, err := eng.ProcessGame(ctx, subOpts)
		if err != nil {
			slog.Warn("Failed to batch process game directory", "path", gamePath, "error", err)
			continue
		}
		results = append(results, res)
	}

	return results, nil
}

// ApplyGame performs the end-to-end One-Shot lifecycle: Inspect -> ACF -> GBE Setup.
func (eng *Engine) ApplyGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	ctx = eng.WrapContext(ctx)
	targetPath := opts.Path
	if targetPath == "" {
		targetPath = "."
	}
	targetPath = util.CanonicalPath(targetPath)
	opts.Path = targetPath

	gameName := eng.resolveGameName(ctx, targetPath)
	game := domain.NewGame(gameName, targetPath, "", "")
	pctx := &pipeline.Context{
		Options:  opts,
		Game:     game,
		Config:   eng.Config,
		EventBus: eng.EventBus,
		Reporter: opts.Reporter,
	}

	applyOp := pipeline.NewSetupOperation(gameName, targetPath, pctx)
	var pipeErr error
	if eng.Tasks != nil {
		pipeErr = eng.Tasks.RunSyncOp(ctx, applyOp)
	} else {
		pipeErr = applyOp.Pipeline.Execute(ctx, pctx)
	}

	pctx.Game.RecomputeState()
	cache.InvalidateGameCache(pctx.Game.GameDir)
	if targetPath != pctx.Game.GameDir {
		cache.InvalidateGameCache(targetPath)
	}

	res := &ProcessResult{
		Info:                pctx.Game,
		Patched:             pctx.Results.Patched,
		SteamlessUnpacked:   pctx.Results.SteamlessUnpacked,
		ManifestsMoved:      true,
		DirectoryNormalized: pctx.Results.DirectoryNormalized,
		ACFSynthesized:      pctx.Results.ACFSynthesized,
		Errors:              pctx.Results.Errors,
	}

	if pipeErr != nil {
		return res, pipeErr
	}

	return res, nil
}

// Rollback restores original backup files and cleans up generated configuration files in a game directory via Rollback pipeline recipe.
func (eng *Engine) Rollback(ctx context.Context, targetPath string, dryRun bool) error {
	ctx = eng.WrapContext(ctx)
	targetPath = util.CanonicalPath(targetPath)
	if targetPath == "" {
		targetPath = "."
	}

	gameName := eng.resolveGameName(ctx, targetPath)
	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path:   targetPath,
			DryRun: dryRun,
		},
		Game:     domain.NewGame(gameName, targetPath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}

	rollbackOp := pipeline.NewRollbackOperation(gameName, targetPath, pctx)
	var err error
	if eng.Tasks != nil {
		err = eng.Tasks.RunSyncOp(ctx, rollbackOp)
	} else {
		err = rollbackOp.Pipeline.Execute(ctx, pctx)
	}
	cache.InvalidateGameCache(targetPath)
	return err
}

func (eng *Engine) runnerManager() *runner.Manager {
	if eng.RunnerMgr != nil {
		return eng.RunnerMgr
	}
	return runner.DefaultManager()
}

// LaunchGame starts the game process asynchronously using the configured operating mode and runner.
// Profile loading, executable resolution fallback, and environment synthesis are encapsulated in the runner.
func (eng *Engine) LaunchGame(ctx context.Context, opts runner.LaunchOptions) (*exec.Cmd, error) {
	if opts.GameDir == "" && opts.ExePath == "" {
		return nil, fmt.Errorf("game directory or executable path cannot be empty")
	}
	if eng.Profiles != nil {
		active := eng.Profiles.GetActive()
		if active != nil {
			_ = eng.Profiles.Materialize(active.ID)
			if opts.ProfileID == "" {
				opts.ProfileID = active.ID
			}
		}
	}
	return eng.runnerManager().Start(ctx, eng.Config, opts)
}

// EnsureAndLaunchGame performs one-click Quick Play: if the game is unpatched/unconfigured,
// it auto-applies setup first, then starts the game asynchronously.
func (eng *Engine) EnsureAndLaunchGame(ctx context.Context, opts runner.LaunchOptions) (*exec.Cmd, error) {
	if opts.GameDir != "" {
		status, err := eng.InspectStatus(ctx, opts.GameDir)
		if err == nil && status != nil && (status.IsClean() || !status.IsReady) {
			slog.Info("EnsureAndLaunchGame auto-applying setup before launch", "dir", opts.GameDir, "game", status.Name)
			opMode := domain.ModeLoader
			if opts.OperatingMode != "" {
				opMode = domain.NormalizeOperatingMode(opts.OperatingMode)
			} else if status != nil && status.EffectiveOperatingMode() != "" {
				opMode = status.EffectiveOperatingMode()
			} else if eng.Config != nil && eng.Config.OperatingMode != "" {
				opMode = domain.NormalizeOperatingMode(eng.Config.OperatingMode)
			}

			applyOpts := ProcessOptions{
				Path:          opts.GameDir,
				OperatingMode: opMode,
				WinePrefix:    opts.WinePrefix,
				ExePath:       opts.ExePath,
				ExeArgs:       opts.ExeArgs,
			}
			if _, applyErr := eng.ApplyGame(ctx, applyOpts); applyErr != nil {
				return nil, fmt.Errorf("auto-apply before launch failed: %w", applyErr)
			}
		}
	}

	return eng.LaunchGame(ctx, opts)
}

// RunGame executes the game process synchronously, streaming output to stdout/stderr.
func (eng *Engine) RunGame(ctx context.Context, opts runner.LaunchOptions) error {
	if eng.Profiles != nil {
		active := eng.Profiles.GetActive()
		if active != nil {
			_ = eng.Profiles.Materialize(active.ID)
			if opts.ProfileID == "" {
				opts.ProfileID = active.ID
			}
		}
	}
	return runner.Run(ctx, eng.Config, opts)
}

// PreviewLaunch computes the effective resolved execution specification and active configuration overrides for a game.
func (eng *Engine) PreviewLaunch(ctx context.Context, gameDir string) (*runner.LaunchPreview, error) {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	profile, err := compat.LoadProfile(gameDir)
	if err != nil || profile == nil {
		profile = &domain.LaunchProfile{GameDir: gameDir}
	}

	opts := runner.LaunchOptions{
		GameDir: gameDir,
	}
	opts.MergeFrom(profile)

	if eng.Profiles != nil {
		active := eng.Profiles.GetActive()
		if active != nil {
			if opts.ProfileID == "" {
				opts.ProfileID = active.ID
			}
		}
	}

	spec, err := runner.BuildLaunchSpec(ctx, eng.Config, opts)
	if err != nil {
		return nil, fmt.Errorf("build launch spec: %w", err)
	}

	overrides := runner.DetectOverrides(profile, eng.Config)

	return &runner.LaunchPreview{
		Command:       runner.FormatCommandLine(spec),
		TargetExe:     spec.TargetExe,
		Args:          spec.Args,
		Env:           spec.Env,
		WorkingDir:    spec.WorkingDir,
		Runner:        spec.Runner,
		OperatingMode: spec.OperatingMode,
		Platform:      spec.Platform,
		Overrides:     overrides,
	}, nil
}

// GetLaunchProfile retrieves the custom launch configuration for a game directory.
func (eng *Engine) GetLaunchProfile(ctx context.Context, gameDir string) (*domain.LaunchProfile, error) {
	return compat.LoadProfile(gameDir)
}

// SaveLaunchProfile persists custom launch configuration and synchronizes lightweight on-disk settings.
//
// NOTE (Architecture Invariant):
// Profile saves must NOT trigger heavyweight binary modifications (gbe.Apply) unless the OperatingMode
// explicitly transitioned (e.g. from portable to loader, or loader to standalone/none), or if the game is unconfigured.
// Normal configuration updates (launch arguments, HUD options, environment variables, artwork) are persisted
// purely via SyncSettings/SaveProfile.
func (eng *Engine) SaveLaunchProfile(ctx context.Context, gameDir string, opts runner.LaunchOptions) error {
	opts.GameDir = util.CanonicalPath(gameDir)
	if opts.GameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	existingProf, _ := compat.LoadProfile(opts.GameDir)
	prevMode := ""
	if existingProf != nil && existingProf.OperatingMode != "" {
		prevMode = domain.NormalizeOperatingMode(existingProf.OperatingMode)
	}
	newMode := ""
	if opts.OperatingMode != "" {
		newMode = domain.NormalizeOperatingMode(opts.OperatingMode)
		opts.OperatingMode = newMode
	}

	// Always sync profile and lightweight text configs (steam_appid.txt, force_language.txt)
	if err := compat.SyncSettings(ctx, &opts, eng.Config); err != nil {
		return fmt.Errorf("sync launch profile settings: %w", err)
	}

	// Ensure portable .gamux/launch.sh is regenerated with new profile options
	_ = runner.GenerateLaunchScript(ctx, eng.Config, opts)

	game, _ := eng.InspectLocal(ctx, opts.GameDir)
	isConfigured := false
	if game != nil {
		if newMode == domain.ModeLoader {
			isConfigured = game.IsLoaderConfigured()
		} else if newMode == domain.ModePortable {
			isConfigured = game.IsPatched
		}
	}

	// Trigger binary DRM application if the operating mode transitioned, or if the game is not yet configured for the selected mode
	if newMode != "" && (!strings.EqualFold(newMode, prevMode) || !isConfigured) {
		if newMode == domain.ModeNone || newMode == domain.ModeStandalone {
			return eng.Rollback(ctx, opts.GameDir, false)
		}

		applyOpts := ProcessOptions{
			Path:          opts.GameDir,
			OperatingMode: newMode,
			WinePrefix:    opts.WinePrefix,
			ExePath:       opts.ExePath,
			ExeArgs:       opts.ExeArgs,
		}
		_, err := eng.ApplyGame(ctx, applyOpts)
		return err
	}

	return nil
}

// GetGameDepots retrieves the structured inventory of all content depots for a game directory.
func (eng *Engine) GetGameDepots(ctx context.Context, gameDir string) ([]domain.DepotSummary, error) {
	ctx = eng.WrapContext(ctx)
	status, err := eng.InspectLocal(ctx, gameDir)
	var numAppID uint32
	if err == nil && status != nil {
		numAppID, _ = util.ParseAppID(status.AppID)
	} else {
		numAppID, _ = util.ParseAppID(filepath.Base(gameDir))
	}
	if numAppID == 0 {
		if prof, pErr := compat.LoadProfile(gameDir); pErr == nil && prof != nil {
			numAppID, _ = util.ParseAppID(prof.AppID)
		}
	}

	var metaMap map[uint32]manifest.DepotMeta
	if numAppID > 0 {
		if remoteDepots, sErr := steam.FetchAppDepotMetadata(ctx, numAppID); sErr == nil && len(remoteDepots) > 0 {
			metaMap = make(map[uint32]manifest.DepotMeta, len(remoteDepots))
			for dID, dInfo := range remoteDepots {
				metaMap[dID] = manifest.DepotMeta{
					DepotID:      dInfo.DepotID,
					Name:         dInfo.Name,
					Language:     dInfo.Language,
					OSList:       dInfo.OSList,
					OSArch:       dInfo.OSArch,
					Optional:     dInfo.Optional,
					IsDLC:        dInfo.IsDLC,
					DLCAppID:     dInfo.DLCAppID,
					DLCName:      dInfo.DLCName,
					Size:         dInfo.Size,
					DownloadSize: dInfo.DownloadSize,
				}
			}
		}
	}

	targetDir := gameDir
	if status != nil && status.GameDir != "" {
		targetDir = status.GameDir
	}
	return manifest.InspectGameDepotsWithMetadata(targetDir, numAppID, metaMap)
}

// GetAppDepots retrieves structured depot metadata for a Steam AppID from remote metadata.
func (eng *Engine) GetAppDepots(ctx context.Context, appID uint32) ([]domain.DepotSummary, error) {
	ctx = eng.WrapContext(ctx)
	if appID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID")
	}

	var metaMap map[uint32]manifest.DepotMeta
	if remoteDepots, sErr := steam.FetchAppDepotMetadata(ctx, appID); sErr == nil && len(remoteDepots) > 0 {
		metaMap = make(map[uint32]manifest.DepotMeta, len(remoteDepots))
		for dID, dInfo := range remoteDepots {
			metaMap[dID] = manifest.DepotMeta{
				DepotID:      dInfo.DepotID,
				Name:         dInfo.Name,
				Language:     dInfo.Language,
				OSList:       dInfo.OSList,
				OSArch:       dInfo.OSArch,
				Optional:     dInfo.Optional,
				IsDLC:        dInfo.IsDLC,
				DLCAppID:     dInfo.DLCAppID,
				DLCName:      dInfo.DLCName,
				Size:         dInfo.Size,
				DownloadSize: dInfo.DownloadSize,
			}
		}
	}
	return manifest.InspectGameDepotsWithMetadata("", appID, metaMap)
}

// UninstallDepot removes on-disk files belonging to a specific depot ID while preserving shared assets and manifests.
func (eng *Engine) UninstallDepot(ctx context.Context, gameDir string, depotID uint32) error {
	ctx = eng.WrapContext(ctx)
	status, err := eng.InspectLocal(ctx, gameDir)
	if err != nil {
		return err
	}
	numAppID, _ := util.ParseAppID(status.AppID)
	deleted, freed, err := manifest.UninstallDepot(status.GameDir, numAppID, depotID)
	if err != nil {
		return fmt.Errorf("uninstall depot %d: %w", depotID, err)
	}
	slog.Info("Uninstalled depot", "gameDir", gameDir, "depotID", depotID, "deletedFiles", deleted, "freedBytes", freed)
	return nil
}

// SwitchGamePlatform configures the game launch profile for a new target platform (e.g. "linux", "win64", "macos")
// and automatically switches the runner and primary executable.
func (eng *Engine) SwitchGamePlatform(ctx context.Context, gameDir string, targetPlatform string) error {
	ctx = eng.WrapContext(ctx)
	profile, err := compat.LoadProfile(gameDir)
	if err != nil || profile == nil {
		profile = &runner.LaunchOptions{GameDir: gameDir}
	}

	game, err := eng.InspectLocal(ctx, gameDir)
	if err != nil {
		return fmt.Errorf("inspect game: %w", err)
	}

	numAppID, _ := util.ParseAppID(game.AppID)
	targetPlatClean := strings.ToLower(strings.TrimSpace(targetPlatform))
	if targetPlatClean == "windows" {
		targetPlatClean = "win64"
	}

	profile.Platform = targetPlatClean
	if targetPlatClean == "linux" {
		profile.Runner = runner.RunnerLinux
	} else if targetPlatClean == "win64" || targetPlatClean == "win32" {
		if profile.Runner == runner.RunnerLinux || profile.Runner == "" {
			profile.Runner = runner.RunnerUMU
		}
	}

	// Discover matching executable for target platform
	if numAppID > 0 {
		candidates := manifest.DiscoverManifestExecutables(gameDir, numAppID, targetPlatClean)
		if len(candidates) > 0 {
			profile.ExePath = candidates[0].Executable
		}
	}

	if err := eng.SaveLaunchProfile(ctx, gameDir, *profile); err != nil {
		return fmt.Errorf("save launch profile for platform switch: %w", err)
	}

	slog.Info("Successfully switched game platform", "gameDir", gameDir, "platform", targetPlatClean, "runner", profile.Runner, "exe", profile.ExePath)
	return nil
}

// ResetLaunchProfile removes custom launch configuration from <gameDir>/steam_settings/launch_profile.json and synchronizes default DRM state.
func (eng *Engine) ResetLaunchProfile(ctx context.Context, gameDir string) error {
	if err := compat.ResetProfile(gameDir); err != nil {
		return err
	}
	_, err := eng.ApplyGame(ctx, ProcessOptions{Path: gameDir})
	return err
}

// NotifyLaunch inspects a game directory for updates or configuration issues, and triggers a console/desktop notification if found.
func (eng *Engine) NotifyLaunch(ctx context.Context, gameDir string) error {
	status, err := eng.InspectStatus(ctx, gameDir)
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
	return eng.Notify(ctx, status.Name, msg)
}

// GetGameLog reads the persisted output log for the last/current run of a game.
func (eng *Engine) GetGameLog(ctx context.Context, gameDir string) (string, error) {
	if gameDir == "" {
		return "", fmt.Errorf("game directory cannot be empty")
	}

	gameName := ""
	appID := ""
	if status, err := eng.InspectStatus(ctx, gameDir); err == nil && status != nil {
		gameName = status.Name
		appID = status.AppID
	}

	return runner.GetLog(gameDir, gameName, appID)
}

// GetGameAchievements reads the achievement schema and unlocked status for a game.
func (eng *Engine) GetGameAchievements(ctx context.Context, gameDir string) ([]domain.AchievementItem, error) {
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	gameDir = util.CanonicalPath(gameDir)
	status, _ := eng.InspectStatus(ctx, gameDir)

	var settingsDirs []string
	var appID uint32
	if status != nil {
		settingsDirs = status.SettingsDirs
		if status.AppID != "" {
			appID, _ = util.ParseAppID(status.AppID)
		}
	}

	apiKey := ""
	if eng.Config != nil {
		apiKey = eng.Config.SteamWebAPIKey
	}

	return gbe.LoadAchievements(ctx, gameDir, appID, settingsDirs, apiKey)
}

// ResolveDownloadTarget computes the effective filesystem target directory for downloading a Steam AppID.
// It detects whether targetDir is empty, ".", or points directly to a configured games library root,
// queries Steam metadata for canonical installdir and soundtrack type, and routes soundtracks to MusicDir.
func ResolveDownloadTarget(ctx context.Context, cfg *config.Config, appID uint32, targetDir, platform string) (string, error) {
	var libDirs []string
	if cfg != nil {
		libDirs = cfg.ResolvedLibraryDirs()
	}

	isLibraryRoot := false
	if len(libDirs) > 0 && targetDir != "" {
		canonicalTarget := util.CanonicalPath(targetDir)
		for _, lib := range libDirs {
			if canonicalTarget != "" && canonicalTarget == util.CanonicalPath(lib) {
				isLibraryRoot = true
				break
			}
		}
	}

	if targetDir == "" || targetDir == "." || isLibraryRoot {
		if appID > 0 {
			name, appType, installDir, err := steam.FetchAppInfo(ctx, fmt.Sprintf("%d", appID))
			if err == nil && installDir != "" {
				if steam.IsSoundtrack(appType) || platform == "soundtrack" {
					targetDir = filepath.Join(util.GetMusicDir(), installDir)
				} else if isLibraryRoot {
					targetDir = filepath.Join(targetDir, installDir)
				} else if targetDir == "" && len(libDirs) > 0 {
					targetDir = filepath.Join(libDirs[0], installDir)
				} else {
					targetDir = installDir
				}
			} else if name != "" {
				cleanName := util.SanitizeInstallDir(name)
				if isLibraryRoot {
					targetDir = filepath.Join(targetDir, cleanName)
				} else if targetDir == "" && len(libDirs) > 0 {
					targetDir = filepath.Join(libDirs[0], cleanName)
				} else {
					targetDir = cleanName
				}
			} else if platform == "soundtrack" {
				targetDir = util.GetMusicDir()
			} else if targetDir == "" {
				targetDir = fmt.Sprintf("./%d", appID)
			}
		} else if platform == "soundtrack" {
			targetDir = util.GetMusicDir()
		} else if targetDir == "" {
			targetDir = "."
		}
	}

	return util.CanonicalPath(targetDir), nil
}

// DownloadGame orchestrates key resolution, manifest persistence, depot chunk downloading, and optional post-download setup.
func (eng *Engine) DownloadGame(ctx context.Context, opts DownloadGameOptions) (*downloader.Result, error) {
	ctx = eng.WrapContext(ctx)
	if opts.AppID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID")
	}

	resolvedTarget, err := ResolveDownloadTarget(ctx, eng.Config, opts.AppID, opts.TargetDir, opts.Platform)
	if err != nil {
		return nil, err
	}
	opts.TargetDir = resolvedTarget

	hubcapKey := ""
	if eng.Config != nil {
		hubcapKey = eng.Config.HubcapAPIKey
	}

	prefLang := opts.Language
	if prefLang == "" {
		if prof, lErr := compat.LoadProfile(opts.TargetDir); lErr == nil && prof != nil && prof.Language != "" {
			prefLang = prof.Language
		}
	}
	if prefLang == "" && eng.Config != nil && eng.Config.Language != "" {
		prefLang = eng.Config.Language
	}
	if prefLang == "" || prefLang == "auto" {
		prefLang = manifest.ResolveSystemLanguage()
	}

	var depotMetaMap map[uint32]manifest.DepotMeta
	if remoteDepots, sErr := steam.FetchAppDepotMetadata(ctx, opts.AppID); sErr == nil && len(remoteDepots) > 0 {
		depotMetaMap = make(map[uint32]manifest.DepotMeta, len(remoteDepots))
		for dID, dInfo := range remoteDepots {
			depotMetaMap[dID] = manifest.DepotMeta{
				DepotID:      dInfo.DepotID,
				Name:         dInfo.Name,
				Language:     dInfo.Language,
				OSList:       dInfo.OSList,
				OSArch:       dInfo.OSArch,
				Optional:     dInfo.Optional,
				IsDLC:        dInfo.IsDLC,
				DLCAppID:     dInfo.DLCAppID,
				DLCName:      dInfo.DLCName,
				Size:         dInfo.Size,
				DownloadSize: dInfo.DownloadSize,
			}
		}
	}

	targetBranch := opts.Branch
	if targetBranch == "" && opts.TargetDir != "" {
		if prof, lErr := compat.LoadProfile(opts.TargetDir); lErr == nil && prof != nil && prof.Branch != "" {
			targetBranch = prof.Branch
		}
	}
	if targetBranch == "" {
		targetBranch = "public"
	}

	var branchManifestGIDs map[uint32]string
	if gids, bErr := steam.FetchAppBranchManifestGIDs(ctx, opts.AppID, targetBranch); bErr == nil && len(gids) > 0 {
		branchManifestGIDs = gids
	}

	req := downloader.DownloadRequest{
		AppID:           opts.AppID,
		TargetDir:       opts.TargetDir,
		Platform:        opts.Platform,
		Language:        prefLang,
		IncludedDepots:  opts.IncludedDepots,
		ExcludedDepots:  opts.ExcludedDepots,
		IncludeOptional: opts.IncludeOptional,
		Branch:          targetBranch,
		ManifestGIDs:    branchManifestGIDs,
		DepotMetadata:   depotMetaMap,
		LuaPath:         opts.LuaPath,
		HubcapAPIKey:    hubcapKey,
		WorkerLimit:     opts.WorkerLimit,
		DryRun:          opts.DryRun,
		HTTPClient:      eng.HTTPClient,
		CheckCancel:     opts.CheckCancel,
		Throttle:        opts.Throttle,
		OnStage: func(stage, details string) {
			eng.reportStep(ctx, 1, 3, stage, details)
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
		return nil, err
	}

	if opts.AutoApply {
		eng.reportStep(ctx, 3, 3, "Running Post-Download Setup", fmt.Sprintf("Target: %s", opts.TargetDir))
		shouldNormalize := util.IsPlaceholderTitle(filepath.Base(opts.TargetDir), fmt.Sprintf("%d", opts.AppID))
		applyOpts := ProcessOptions{
			Path:              opts.TargetDir,
			OperatingMode:     domain.ModeLoader,
			NormalizeDir:      shouldNormalize,
			DryRun:            opts.DryRun,
			AutoYes:           true,
			FetchAchievements: true,
		}
		if opts.ApplyOptions != nil {
			applyOpts = *opts.ApplyOptions
			applyOpts.Path = opts.TargetDir
		}
		if _, err := eng.ProcessGame(ctx, applyOpts); err != nil {
			slog.Warn("Auto-apply setup failed following download", "targetDir", opts.TargetDir, "error", err)
		}
	}

	if opts.Branch != "" && opts.Branch != "public" {
		if p, lErr := compat.LoadProfile(opts.TargetDir); lErr == nil && p != nil {
			p.Branch = opts.Branch
			_ = compat.SaveProfile(p)
		}
	}

	return res, nil
}

// UpdateGame checks for available Steam depot updates for an existing game directory, downloads changed files, and re-applies setup via the Update pipeline recipe.
func (eng *Engine) UpdateGame(ctx context.Context, gamePath string) (*GameStatus, error) {
	ctx = eng.WrapContext(ctx)
	targetPath := gamePath
	if targetPath == "" {
		targetPath = "."
	}
	targetPath = util.CanonicalPath(targetPath)

	slog.Info("Updating game installation via pipeline", "target", targetPath)

	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path:          targetPath,
			OperatingMode: domain.ModePortable,
			NormalizeDir:  true,
		},
		Game:     domain.NewGame("", targetPath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}

	pipe := pipeline.NewUpdate()
	if err := pipe.Execute(ctx, pctx); err != nil {
		return nil, fmt.Errorf("update pipeline failed: %w", err)
	}

	return eng.InspectStatus(ctx, pctx.Game.GameDir)
}

// GetGSEUserConfig returns the Goldberg user persona settings from configs.user.ini or the active profile.
func (eng *Engine) GetGSEUserConfig() (*gbe.GSEUserConfig, error) {
	if eng.Profiles != nil {
		active := eng.Profiles.GetActive()
		if active != nil {
			return &gbe.GSEUserConfig{
				AccountName:    active.Name,
				AccountSteamID: active.SteamID,
				Language:       active.Language,
				IPCountry:      active.IPCountry,
			}, nil
		}
	}
	return gbe.LoadGSEUserConfig()
}

// SaveGSEUserConfig persists the Goldberg user persona settings and synchronizes them with the active profile.
func (eng *Engine) SaveGSEUserConfig(cfg *gbe.GSEUserConfig) error {
	if cfg == nil {
		return fmt.Errorf("gse user config cannot be nil")
	}
	if eng.Profiles != nil {
		active := eng.Profiles.GetActive()
		if active != nil {
			active.Name = cfg.AccountName
			active.SteamID = cfg.AccountSteamID
			active.Language = cfg.Language
			active.IPCountry = cfg.IPCountry
			_ = eng.Profiles.Update(active)
		}
	}
	return gbe.SaveGSEUserConfig(cfg)
}

// GetAvailableRunners returns list of detected execution runners on the host.
func (eng *Engine) GetAvailableRunners() []string {
	return runner.GetAvailableRunners()
}

// CheckGameUpdates inspects a game's update status using hybrid version checking.
func (eng *Engine) CheckGameUpdates(ctx context.Context, gameDir string) (*domain.VersionStatus, error) {
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}
	cleanDir := util.CanonicalPath(gameDir)
	game, err := eng.DetectGame(ctx, cleanDir)
	if err != nil {
		return nil, err
	}
	if game == nil || game.NumericAppID == 0 {
		return &domain.VersionStatus{Checked: false, IsUpToDate: true, Message: "Not a Steam game"}, nil
	}

	game = game.Clone()
	vs, err := steam.CheckGameVersion(ctx, game.NumericAppID, game.BuildID, game.InstalledAt, game.Branch, game.ManifestID)
	if err != nil {
		return nil, err
	}

	if vs != nil {
		game.VersionStatus = vs
		game.HasUpdate = vs.Checked && !vs.IsUpToDate
		if vs.RemoteManifestID != "" {
			game.RemoteManifestID = vs.RemoteManifestID
		}
		game.RecomputeState()

		var dirMtime, profileMtime int64
		if dirFi, sErr := os.Stat(cleanDir); sErr == nil {
			dirMtime = dirFi.ModTime().UnixNano()
		}
		if pFi, sErr := os.Stat(filepath.Join(cleanDir, ".gamux", "launch_profile.json")); sErr == nil {
			profileMtime = pFi.ModTime().UnixNano()
		} else if legFi, sErr := os.Stat(filepath.Join(cleanDir, "steam_settings", "launch_profile.json")); sErr == nil {
			profileMtime = legFi.ModTime().UnixNano()
		}
		_ = cache.SaveCachedGame(cleanDir, dirMtime, profileMtime, game)
	}

	return vs, nil
}

// GetGameDLCs retrieves configured DLCs from configs.app.ini, falling back to Store API discovery.
func (eng *Engine) GetGameDLCs(ctx context.Context, gameDir string) ([]domain.DLCItem, error) {
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}
	cleanDir := util.CanonicalPath(gameDir)
	game, err := eng.DetectGame(ctx, cleanDir)
	if err != nil {
		return nil, err
	}

	// 1. Read existing configs.app.ini if present
	candidateFiles := make([]string, 0, len(game.SettingsDirs)+2)
	for _, sDir := range game.SettingsDirs {
		candidateFiles = append(candidateFiles, filepath.Join(sDir, "configs.app.ini"))
	}
	candidateFiles = append(candidateFiles,
		filepath.Join(cleanDir, "steam_settings", "configs.app.ini"),
		filepath.Join(cleanDir, ".gamux", "configs.app.ini"),
	)

	for _, iniFile := range candidateFiles {
		if f, err := ini.LoadFile(iniFile); err == nil {
			sec := f.Section("app::dlcs")
			var items []domain.DLCItem
			for _, k := range sec.Keys() {
				if k != "unlock_all" {
					name := sec.GetString(k, "")
					if strings.TrimSpace(name) != "" {
						items = append(items, domain.DLCItem{AppID: k, Name: name})
					}
				}
			}
			if len(items) > 0 {
				sort.Slice(items, func(i, j int) bool {
					return items[i].AppID < items[j].AppID
				})
				return items, nil
			}
		}
	}

	// 2. If nothing configured locally yet and game has Steam AppID, discover via Steam Store API
	if game.NumericAppID > 0 {
		return steam.DiscoverDLCs(ctx, fmt.Sprintf("%d", game.NumericAppID))
	}

	return []domain.DLCItem{}, nil
}

// FetchGameDLCs fetches DLC definitions from Steam and saves them into configs.app.ini.
func (eng *Engine) FetchGameDLCs(ctx context.Context, gameDir string) error {
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}
	cleanDir := util.CanonicalPath(gameDir)
	game, err := eng.DetectGame(ctx, cleanDir)
	if err != nil {
		return err
	}
	if game == nil || game.NumericAppID == 0 {
		return fmt.Errorf("game at %s has no valid Steam AppID", cleanDir)
	}

	appIDStr := fmt.Sprintf("%d", game.NumericAppID)
	if err := steam.FetchDLCs(ctx, appIDStr, cleanDir, false); err != nil {
		return err
	}
	cache.InvalidateGameCache(cleanDir)
	return nil
}
