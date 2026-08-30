package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/pipeline"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util"
)

// ProcessGame processes a game directory according to the supplied options.
func (eng *Engine) ProcessGame(ctx context.Context, opts ProcessOptions) (*ProcessResult, error) {
	ctx = eng.WrapContext(ctx)
	targetPath := opts.Path
	if targetPath == "" {
		targetPath = "."
	}
	targetPath = util.CanonicalPath(targetPath)

	game := domain.NewGame("", targetPath, "", "")
	pctx := &pipeline.Context{
		Options:  opts,
		Game:     game,
		Config:   eng.Config,
		EventBus: eng.EventBus,
		Reporter: opts.Reporter,
	}

	pipe := pipeline.NewDefault()
	pipeErr := pipe.Execute(ctx, pctx)

	pctx.Game.RecomputeState()

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

		// Skip hidden folders
		if strings.HasPrefix(entry.Name(), ".") {
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
	status, err := eng.InspectStatus(ctx, opts.Path)
	if err != nil {
		slog.Warn("Apply inspection warning", "error", err)
	}

	if status != nil && status.HasUpdate {
		slog.Info("Apply detected available Steam depot update", "title", status.Name, "localManifest", status.ManifestID)
	}

	return eng.ProcessGame(ctx, opts)
}

// Rollback restores original backup files and cleans up generated configuration files in a game directory via Rollback pipeline recipe.
func (eng *Engine) Rollback(ctx context.Context, targetPath string, dryRun bool) error {
	ctx = eng.WrapContext(ctx)
	targetPath = util.CanonicalPath(targetPath)
	if targetPath == "" {
		targetPath = "."
	}

	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path:   targetPath,
			DryRun: dryRun,
		},
		Game:     domain.NewGame("", targetPath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}

	pipe := pipeline.NewRollback()
	return pipe.Execute(ctx, pctx)
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
	return eng.runnerManager().Start(ctx, eng.Config, opts)
}

// EnsureAndLaunchGame performs one-click Quick Play: if the game is unpatched/unconfigured,
// it auto-applies setup first, then starts the game asynchronously.
func (eng *Engine) EnsureAndLaunchGame(ctx context.Context, opts runner.LaunchOptions) (*exec.Cmd, error) {
	if opts.GameDir != "" {
		status, err := eng.InspectStatus(ctx, opts.GameDir)
		if err == nil && status != nil && (status.IsClean() || !status.IsReady) {
			slog.Info("EnsureAndLaunchGame auto-applying setup before launch", "dir", opts.GameDir, "game", status.Name)
			applyOpts := ProcessOptions{
				Path:          opts.GameDir,
				OperatingMode: domain.ModePortable,
				WinePrefix:    opts.WinePrefix,
				ExePath:       opts.ExePath,
				ExeArgs:       opts.ExeArgs,
			}
			pctx := &pipeline.Context{
				Options:  applyOpts,
				Game:     domain.NewGame(status.Name, opts.GameDir, "", ""),
				Config:   eng.Config,
				EventBus: eng.EventBus,
			}
			applyOp := pipeline.NewSetupOperation(status.Name, opts.GameDir, pctx)
			if eng.Tasks != nil {
				if err := eng.Tasks.RunSyncOp(ctx, applyOp); err != nil {
					return nil, fmt.Errorf("auto-apply before launch failed: %w", err)
				}
			} else {
				if _, applyErr := eng.ApplyGame(ctx, applyOpts); applyErr != nil {
					return nil, fmt.Errorf("auto-apply before launch failed: %w", applyErr)
				}
			}
		}
	}

	return eng.LaunchGame(ctx, opts)
}

// RunGame executes the game process synchronously, streaming output to stdout/stderr.
func (eng *Engine) RunGame(ctx context.Context, opts runner.LaunchOptions) error {
	return runner.Run(ctx, eng.Config, opts)
}

// GetLaunchProfile retrieves the custom launch configuration for a game directory.
func (eng *Engine) GetLaunchProfile(ctx context.Context, gameDir string) (*domain.LaunchProfile, error) {
	return compat.LoadProfile(gameDir)
}

// SaveLaunchProfile persists custom launch configuration and synchronizes lightweight on-disk settings.
//
// NOTE (Architecture Invariant):
// Profile saves must NOT trigger heavyweight binary modifications (gbe.Apply) unless the OperatingMode
// explicitly transitioned (e.g. from portable to loader, or loader to standalone/none).
// Normal configuration updates (launch arguments, HUD options, environment variables, artwork) are persisted
// purely via SyncSettings/SaveProfile.
func (eng *Engine) SaveLaunchProfile(ctx context.Context, gameDir string, opts runner.LaunchOptions) error {
	opts.GameDir = util.CanonicalPath(gameDir)
	if opts.GameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	existingProf, _ := compat.LoadProfile(opts.GameDir)
	prevMode := ""
	if existingProf != nil {
		prevMode = existingProf.OperatingMode
	}
	newMode := opts.OperatingMode

	// Always sync profile and lightweight text configs (steam_appid.txt, force_language.txt)
	if err := compat.SyncSettings(ctx, &opts, eng.Config); err != nil {
		return fmt.Errorf("sync launch profile settings: %w", err)
	}

	// Only trigger binary DRM application if the operating mode transitioned
	if newMode != "" && !strings.EqualFold(newMode, prevMode) {
		game, _ := eng.InspectLocal(ctx, opts.GameDir)
		return compat.Apply(ctx, eng.Config, game, &opts, false)
	}

	return nil
}

// GetGameDepots retrieves the structured inventory of all content depots for a game directory.
func (eng *Engine) GetGameDepots(ctx context.Context, gameDir string) ([]domain.DepotSummary, error) {
	ctx = eng.WrapContext(ctx)
	status, err := eng.InspectLocal(ctx, gameDir)
	if err != nil {
		return nil, err
	}
	numAppID, _ := util.ParseAppID(status.AppID)
	return manifest.InspectGameDepots(status.GameDir, numAppID)
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
	game, _ := eng.InspectLocal(ctx, gameDir)
	return compat.Apply(ctx, eng.Config, game, nil, false)
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

	var appID uint32
	if status != nil && status.AppID != "" {
		appID, _ = util.ParseAppID(status.AppID)
	}

	// Fallback to steam_appid.txt if appID wasn't parsed from ACF/status
	if appID == 0 {
		for _, p := range []string{
			filepath.Join(gameDir, "steam_appid.txt"),
			filepath.Join(gameDir, "steam_settings", "steam_appid.txt"),
		} {
			if data, err := os.ReadFile(p); err == nil {
				txt := strings.TrimSpace(string(data))
				if parsed, ok := util.ParseAppID(txt); ok && parsed > 0 {
					appID = parsed
					break
				}
			}
		}
	}

	// Fallback to launch_profile.json
	if appID == 0 {
		if prof, err := compat.LoadProfile(gameDir); err == nil && prof != nil && prof.AppID != "" {
			if parsed, ok := util.ParseAppID(prof.AppID); ok && parsed > 0 {
				appID = parsed
			}
		}
	}

	achievementsFile := filepath.Join(gameDir, "steam_settings", "achievements.json")
	var schemas []steam.AchievementSchema

	if util.FileExists(achievementsFile) {
		if data, err := os.ReadFile(achievementsFile); err == nil {
			if err := json.Unmarshal(data, &schemas); err != nil {
				// Try map format {"ACH_NAME": {displayName: "...", ...}}
				var schemaMap map[string]steam.AchievementSchema
				if err2 := json.Unmarshal(data, &schemaMap); err2 == nil {
					for k, v := range schemaMap {
						if v.Name == "" {
							v.Name = k
						}
						schemas = append(schemas, v)
					}
				}
			}
		}
	}

	if len(schemas) == 0 && appID > 0 {
		apiKey := ""
		if eng.Config != nil {
			apiKey = eng.Config.SteamWebAPIKey
		}
		if fetched, err := steam.FetchAchievementSchema(ctx, appID, apiKey, "english"); err == nil && len(fetched) > 0 {
			schemas = fetched
		}
	}

	if len(schemas) == 0 {
		return []domain.AchievementItem{}, nil
	}

	// Discover unlocked achievements from standard GSE emulator save paths
	unlockedMap := make(map[string]int64)

	home, _ := os.UserHomeDir()
	candidatePaths := []string{
		filepath.Join(gameDir, "steam_settings", "achievements_unlocked.json"),
	}

	if appID > 0 && home != "" {
		appIDStr := fmt.Sprintf("%d", appID)
		candidatePaths = append(candidatePaths,
			filepath.Join(home, ".local", "share", "GSE Saves", appIDStr, "achievements.json"),
			filepath.Join(home, ".local", "share", "GSE Saves", appIDStr, "remote", "achievements.json"),
			filepath.Join(home, ".local", "share", "GSE Saves", "settings", "achievements.json"),
			filepath.Join(home, "Games", "umu", "umu-default", "drive_c", "users", "steamuser", "AppData", "Roaming", "GSE Saves", appIDStr, "achievements.json"),
		)

		// Expand nested remote folder patterns (e.g. GSE Saves/<appID>/remote/*/achievements.json)
		if matches, err := filepath.Glob(filepath.Join(home, ".local", "share", "GSE Saves", appIDStr, "remote", "*", "achievements.json")); err == nil {
			candidatePaths = append(candidatePaths, matches...)
		}
	}

	for _, p := range candidatePaths {
		if !util.FileExists(p) {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		var simpleMap map[string]any
		if err := json.Unmarshal(data, &simpleMap); err == nil {
			for k, v := range simpleMap {
				switch val := v.(type) {
				case float64:
					if val > 0 {
						unlockedMap[strings.ToUpper(k)] = int64(val)
					}
				case bool:
					if val {
						unlockedMap[strings.ToUpper(k)] = 1
					}
				case map[string]any:
					earned := false
					if e, ok := val["earned"].(bool); ok && e {
						earned = true
					} else if u, ok := val["unlocked"].(bool); ok && u {
						earned = true
					}

					if earned {
						var t int64 = 1
						if et, ok := val["earned_time"].(float64); ok && et > 0 {
							t = int64(et)
						} else if ut, ok := val["unlock_time"].(float64); ok && ut > 0 {
							t = int64(ut)
						}
						unlockedMap[strings.ToUpper(k)] = t
					}
				default:
					unlockedMap[strings.ToUpper(k)] = 1
				}
			}
		} else {
			var arr []string
			if err := json.Unmarshal(data, &arr); err == nil {
				for _, k := range arr {
					unlockedMap[strings.ToUpper(k)] = 1
				}
			}
		}
	}

	items := make([]domain.AchievementItem, 0, len(schemas))
	for _, s := range schemas {
		unlockedTime, isUnlocked := unlockedMap[strings.ToUpper(s.Name)]

		iconURL := s.Icon
		iconGrayURL := s.IconGray

		cleanIcon := strings.TrimPrefix(s.Icon, "img/")
		cleanGray := strings.TrimPrefix(s.IconGray, "img/")

		// Check if local image files exist
		localImgPath := filepath.Join(gameDir, "steam_settings", "img", cleanIcon)
		if !util.FileExists(localImgPath) {
			localImgPath = filepath.Join(gameDir, "steam_settings", s.Icon)
		}
		if util.FileExists(localImgPath) {
			iconURL = "/local-img" + localImgPath
		} else if appID > 0 && !strings.HasPrefix(iconURL, "http") {
			iconURL = fmt.Sprintf("https://shared.fastly.steamstatic.com/community_assets/images/apps/%d/%s", appID, cleanIcon)
		}

		localGrayPath := filepath.Join(gameDir, "steam_settings", "img", cleanGray)
		if !util.FileExists(localGrayPath) {
			localGrayPath = filepath.Join(gameDir, "steam_settings", s.IconGray)
		}
		if util.FileExists(localGrayPath) {
			iconGrayURL = "/local-img" + localGrayPath
		} else if appID > 0 && !strings.HasPrefix(iconGrayURL, "http") {
			iconGrayURL = fmt.Sprintf("https://shared.fastly.steamstatic.com/community_assets/images/apps/%d/%s", appID, cleanGray)
		}

		items = append(items, domain.AchievementItem{
			ID:          s.Name,
			Name:        s.DisplayName,
			Description: s.Description,
			IconURL:     iconURL,
			IconGrayURL: iconGrayURL,
			Hidden:      s.Hidden == 1,
			Unlocked:    isUnlocked,
			UnlockTime:  unlockedTime,
		})
	}

	// Sort achievements: completed/unlocked first (sorted by most recent unlock time desc), then locked
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Unlocked != items[j].Unlocked {
			return items[i].Unlocked // true comes before false
		}
		if items[i].Unlocked {
			return items[i].UnlockTime > items[j].UnlockTime // most recently unlocked first
		}
		return false // preserve schema order for locked items
	})

	return items, nil
}

// DownloadGame orchestrates key resolution, manifest persistence, depot chunk downloading, and optional post-download setup.
func (eng *Engine) DownloadGame(ctx context.Context, opts DownloadGameOptions) (*downloader.Result, error) {
	ctx = eng.WrapContext(ctx)
	if opts.AppID == 0 {
		return nil, fmt.Errorf("invalid Steam AppID")
	}
	if opts.TargetDir == "" || opts.TargetDir == "." {
		if _, appType, installDir, err := steam.FetchAppInfo(ctx, fmt.Sprintf("%d", opts.AppID)); err == nil && installDir != "" {
			if steam.IsSoundtrack(appType) || opts.Platform == "soundtrack" {
				opts.TargetDir = filepath.Join(util.GetMusicDir(), installDir)
			} else {
				opts.TargetDir = installDir
			}
		} else if opts.Platform == "soundtrack" {
			opts.TargetDir = util.GetMusicDir()
		} else if opts.TargetDir == "" {
			opts.TargetDir = "."
		}
	}
	opts.TargetDir = util.CanonicalPath(opts.TargetDir)

	hubcapKey := ""
	if eng.Config != nil {
		hubcapKey = eng.Config.HubcapAPIKey
	}

	req := downloader.DownloadRequest{
		AppID:        opts.AppID,
		TargetDir:    opts.TargetDir,
		Platform:     opts.Platform,
		LuaPath:      opts.LuaPath,
		HubcapAPIKey: hubcapKey,
		WorkerLimit:  opts.WorkerLimit,
		DryRun:       opts.DryRun,
		HTTPClient:   eng.HTTPClient,
		CheckCancel:  opts.CheckCancel,
		Throttle:     opts.Throttle,
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
		return nil, fmt.Errorf("download game depot: %w", err)
	}

	if opts.AutoApply {
		eng.reportStep(ctx, 3, 3, "Running Post-Download Setup", fmt.Sprintf("Target: %s", opts.TargetDir))
		applyOpts := ProcessOptions{
			Path:              opts.TargetDir,
			OperatingMode:     domain.ModeLoader,
			NormalizeDir:      false,
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

// GetGSEUserConfig returns the global Goldberg user persona settings from configs.user.ini.
func (eng *Engine) GetGSEUserConfig() (*gbe.GSEUserConfig, error) {
	return gbe.LoadGSEUserConfig()
}

// SaveGSEUserConfig persists the global Goldberg user persona settings to configs.user.ini.
func (eng *Engine) SaveGSEUserConfig(cfg *gbe.GSEUserConfig) error {
	if cfg == nil {
		return fmt.Errorf("gse user config cannot be nil")
	}
	return gbe.SaveGSEUserConfig(cfg)
}

// GetAvailableRunners returns list of detected execution runners on the host.
func (eng *Engine) GetAvailableRunners() []string {
	return runner.GetAvailableRunners()
}
