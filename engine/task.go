package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/gbe"
	"github.com/staernid/gamux/pipeline"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/util"
)

// SubmitApply schedules an asynchronous game post-processing setup task.
func (eng *Engine) SubmitApply(opts ProcessOptions) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), opts.Path)
	pctx := &pipeline.Context{
		Options:  opts,
		Game:     domain.NewGame(gameName, opts.Path, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}
	return eng.Tasks.SubmitOp(pipeline.NewSetupOperation(gameName, opts.Path, pctx))
}

// SubmitUpdate schedules an asynchronous game update task.
func (eng *Engine) SubmitUpdate(gamePath string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path:          gamePath,
			OperatingMode: domain.ModePortable,
			NormalizeDir:  true,
		},
		Game:     domain.NewGame(gameName, gamePath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}
	return eng.Tasks.SubmitOp(pipeline.NewUpdateOperation(gameName, gamePath, pctx))
}

// SubmitVerify schedules an asynchronous game depot integrity verification task.
func (eng *Engine) SubmitVerify(gamePath string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path: gamePath,
		},
		Game:     domain.NewGame(gameName, gamePath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}
	return eng.Tasks.SubmitOp(pipeline.NewVerifyOperation(gameName, gamePath, pctx))
}

// SubmitNormalize schedules an asynchronous directory normalization task.
func (eng *Engine) SubmitNormalize(gamePath string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path:         gamePath,
			NormalizeDir: true,
			AutoYes:      true,
		},
		Game:     domain.NewGame(gameName, gamePath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}
	return eng.Tasks.SubmitOp(pipeline.NewNormalizeOperation(gameName, gamePath, pctx))
}

// SubmitRollback schedules an asynchronous game rollback task.
func (eng *Engine) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	pctx := &pipeline.Context{
		Options: pipeline.Options{
			Path:   gamePath,
			DryRun: dryRun,
		},
		Game:     domain.NewGame(gameName, gamePath, "", ""),
		Config:   eng.Config,
		EventBus: eng.EventBus,
	}
	return eng.Tasks.SubmitOp(pipeline.NewRollbackOperation(gameName, gamePath, pctx))
}

// SubmitDownload schedules an asynchronous depot download transfer with state persistence, speed telemetry, and pause/resume support.
func (eng *Engine) SubmitDownload(opts DownloadGameOptions) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	if opts.AppID == 0 {
		return "", fmt.Errorf("invalid AppID 0")
	}
	if opts.TargetDir == "" {
		return "", fmt.Errorf("target directory cannot be empty")
	}

	gameName := fmt.Sprintf("Steam AppID %d", opts.AppID)
	if p, ok := eng.StoreRegistry.Get("steam"); ok {
		if meta, err := p.GetMetadata(context.Background(), fmt.Sprintf("%d", opts.AppID)); err == nil && meta.Name != "" {
			gameName = meta.Name
		}
	}

	appIDStr := fmt.Sprintf("%d", opts.AppID)
	for _, existing := range eng.Tasks.List() {
		if existing.Type == tasks.TypeDownload && existing.AppID == appIDStr {
			if existing.Status == tasks.StatusRunning || existing.Status == tasks.StatusQueued || existing.Status == tasks.StatusPaused {
				slog.Info("Download already active/queued, reusing existing task", "appID", opts.AppID, "taskID", existing.ID)
				return existing.ID, nil
			}
		}
	}

	return eng.Tasks.SubmitOp(&DownloadOperation{
		Engine:   eng,
		Options:  opts,
		GameName: gameName,
	})
}

// SubmitToolUpdate schedules an asynchronous tool update download (GBE or Steamless) through the unified Tasks engine.
func (eng *Engine) SubmitToolUpdate(toolKey string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	tool, err := eng.Tools().GetTool(toolKey)
	if err != nil {
		return "", err
	}

	targetDir := ""
	switch strings.ToLower(toolKey) {
	case "gbe":
		targetDir = util.ExpandPath(eng.Config.GbeDir)
	case "steamless":
		targetDir = util.ExpandPath(eng.Config.SteamlessDir)
	case "umu":
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".local", "share", "umu")
	default:
		targetDir = util.ExpandPath(eng.Config.GbeDir)
	}

	displayName := tool.Name()
	if strings.ToLower(toolKey) == "gbe" {
		displayName = "Goldberg Steam Emulator (GBE)"
	} else if strings.ToLower(toolKey) == "steamless" {
		displayName = "Steamless-rs DRM Unpacker"
	} else if strings.ToLower(toolKey) == "umu" {
		displayName = "UMU Proton Launcher (umu-run)"
	}

	return eng.Tasks.SubmitOp(&ToolUpdateOperation{
		Engine:      eng,
		ToolKey:     toolKey,
		DisplayName: displayName,
		TargetDir:   targetDir,
		Tool:        tool,
	})
}

// SubmitWorkshop schedules an asynchronous Steam workshop mod download.
func (eng *Engine) SubmitWorkshop(itemURLOrID, targetDir string, dryRun bool) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	if targetDir == "" {
		targetDir = "."
	}
	return eng.Tasks.SubmitOp(&WorkshopOperation{
		ItemURLOrID: itemURLOrID,
		TargetDir:   targetDir,
		DryRun:      dryRun,
	})
}

// SubmitBatchLibraryOp schedules an asynchronous batch library operation.
// Supported kinds: "achievements", "artwork", "normalize", "verify".
func (eng *Engine) SubmitBatchLibraryOp(kind string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	op, err := eng.newBatchLibraryOp(kind)
	if err != nil {
		return "", err
	}

	return eng.Tasks.SubmitOp(op)
}

// newBatchLibraryOp constructs a BatchLibraryOperation for the given kind.
func (eng *Engine) newBatchLibraryOp(kind string) (*BatchLibraryOperation, error) {
	dirs := eng.Config.ResolvedLibraryDirs()

	switch strings.ToLower(kind) {
	case "achievements":
		apiKey := ""
		if eng.Config != nil {
			apiKey = eng.Config.SteamWebAPIKey
		}
		return &BatchLibraryOperation{
			Engine:      eng,
			LibraryDirs: dirs,
			BatchTitle:  "Batch Download Missing Achievements",
			TaskType:    tasks.TypeCustom,
			ScanLabel:   "Scanning games for missing achievement data...",
			DoneLabel:   "Processed %d games (downloaded data for %d)",
			Metadata:    map[string]string{"batch_kind": "achievements"},
			Action: func(ctx context.Context, eng *Engine, game *GameStatus, ctrl *tasks.Controller, i, total int) (bool, error) {
				if game.AppID == "" || game.AppID == "0" || strings.EqualFold(game.AppID, "custom") {
					return false, nil
				}
				appIDNum, ok := util.ParseAppID(game.AppID)
				if !ok || appIDNum == 0 {
					return false, nil
				}
				achFile := filepath.Join(game.GameDir, "steam_settings", "achievements.json")
				imgDir := filepath.Join(game.GameDir, "steam_settings", "img")
				if util.FileExists(achFile) && util.FileExists(imgDir) {
					return false, nil
				}
				if err := gbe.GenerateAchievementsJSON(ctx, appIDNum, apiKey, game.GameDir, false); err != nil {
					return false, err
				}
				return true, nil
			},
		}, nil

	case "artwork":
		return &BatchLibraryOperation{
			Engine:      eng,
			LibraryDirs: dirs,
			BatchTitle:  "Batch Refresh Missing Artwork",
			TaskType:    tasks.TypeCustom,
			ScanLabel:   "Scanning games for missing poster, hero, and banner artwork...",
			DoneLabel:   "Artwork refresh completed for %d games (%d updated)",
			Metadata:    map[string]string{"batch_kind": "artwork"},
			Action: func(ctx context.Context, eng *Engine, game *GameStatus, ctrl *tasks.Controller, i, total int) (bool, error) {
				art, _ := eng.ResolveArtwork(ctx, game)
				if eng.ArtworkService != nil {
					_, _ = eng.ArtworkService.PruneOrphans(game)
				}
				return art != nil, nil
			},
		}, nil

	case "apply", "setup":
		return &BatchLibraryOperation{
			Engine:      eng,
			LibraryDirs: dirs,
			BatchTitle:  "Batch Apply DRM & Emulator Setup",
			TaskType:    tasks.TypeApply,
			ScanLabel:   "Scanning games for unpatched titles...",
			DoneLabel:   "Setup completed for %d games (%d patched)",
			Metadata:    map[string]string{"batch_kind": "apply"},
			Action: func(ctx context.Context, eng *Engine, game *GameStatus, ctrl *tasks.Controller, i, total int) (bool, error) {
				if game.IsPatched {
					return false, nil
				}
				opts := ProcessOptions{
					Path:          game.GameDir,
					OperatingMode: domain.ModeLoader,
					AutoYes:       true,
					Reporter:      ctrl,
				}
				if eng.Config != nil && !eng.Config.Steamless {
					opts.NoSteamless = true
				}
				res, err := eng.ProcessGame(ctx, opts)
				if err != nil {
					return false, err
				}
				return res != nil && res.Patched, nil
			},
		}, nil

	case "normalize":
		return &BatchLibraryOperation{
			Engine:      eng,
			LibraryDirs: dirs,
			BatchTitle:  "Batch Normalize Directory Names",
			TaskType:    tasks.TypeNormalize,
			ScanLabel:   "Scanning game directories for standard naming...",
			DoneLabel:   "Normalization finished for %d games (%d directories renamed)",
			Metadata:    map[string]string{"batch_kind": "normalize"},
			Action: func(ctx context.Context, eng *Engine, game *GameStatus, ctrl *tasks.Controller, i, total int) (bool, error) {
				opts := ProcessOptions{
					Path:          game.GameDir,
					OperatingMode: domain.ModeNone,
					NormalizeDir:  true,
					NoSteamless:   true,
					AutoYes:       true,
					Reporter:      ctrl,
				}
				res, err := eng.ProcessGame(ctx, opts)
				if err != nil {
					return false, err
				}
				return res != nil && res.Info != nil && res.Info.GameDir != game.GameDir, nil
			},
		}, nil

	case "verify":
		return &BatchLibraryOperation{
			Engine:      eng,
			LibraryDirs: dirs,
			BatchTitle:  "Batch Verify Depot Integrity",
			TaskType:    tasks.TypeVerify,
			ScanLabel:   "Scanning games for depot manifests...",
			DoneLabel:   "Integrity check finished for %d games (%d verified)",
			Metadata:    map[string]string{"batch_kind": "verify"},
			Action: func(ctx context.Context, eng *Engine, game *GameStatus, ctrl *tasks.Controller, i, total int) (bool, error) {
				rep, err := eng.VerifyIntegrity(ctx, game.GameDir)
				if err != nil {
					return false, err
				}
				return rep != nil && rep.OfficialCount > 0, nil
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown batch library operation kind: %q", kind)
	}
}

func (eng *Engine) registerDefaultTaskRunners() {
	if eng.Tasks == nil {
		return
	}

	eng.Tasks.RegisterRunner(tasks.TypeDownload, func(ctx context.Context, t *tasks.Task, ctrl *tasks.Controller) error {
		appIDVal, ok := util.ParseAppID(t.AppID)
		if !ok || appIDVal == 0 {
			return fmt.Errorf("invalid AppID for download task %s: %q", t.ID, t.AppID)
		}
		if t.GameDir == "" {
			return fmt.Errorf("target directory cannot be empty for download task %s", t.ID)
		}

		branch := ""
		if t.Metadata != nil {
			branch = t.Metadata["branch"]
		}
		if branch == "" && t.GameDir != "" {
			if lp, lErr := compat.LoadProfile(t.GameDir); lErr == nil && lp != nil && lp.Branch != "" {
				branch = lp.Branch
			}
		}

		op := &DownloadOperation{
			Engine: eng,
			Options: DownloadGameOptions{
				AppID:     appIDVal,
				TargetDir: t.GameDir,
				Branch:    branch,
				AutoApply: true,
			},
			GameName: t.GameName,
		}
		return op.Execute(ctx, ctrl)
	})

	eng.Tasks.RegisterRunner(tasks.TypeToolUpdate, func(ctx context.Context, t *tasks.Task, ctrl *tasks.Controller) error {
		toolKey := t.Metadata["tool_key"]
		if toolKey == "" {
			toolKey = "gbe"
		}
		tool, err := eng.Tools().GetTool(toolKey)
		if err != nil {
			return err
		}
		op := &ToolUpdateOperation{
			Engine:      eng,
			ToolKey:     toolKey,
			DisplayName: t.GameName,
			TargetDir:   t.GameDir,
			Tool:        tool,
		}
		return op.Execute(ctx, ctrl)
	})
}

// SetTransferSpeedLimit updates the task limiter speed and granularly persists the updated setting via read-modify-write.
func (eng *Engine) SetTransferSpeedLimit(bps int64) error {
	if eng.Tasks != nil {
		eng.Tasks.SetSpeedLimit(bps)
	}
	if eng.Config != nil {
		eng.Config.TransferSpeedLimit = bps
	}
	// Granularly update config on disk via read-modify-write without dumping arbitrary in-memory state
	_, err := config.UpdateConfig("", func(c *config.Config) error {
		c.TransferSpeedLimit = bps
		return nil
	})
	if err != nil {
		return fmt.Errorf("save config speed limit: %w", err)
	}
	return nil
}

// GetTransferSpeedLimit returns the current active download bandwidth limiter setting.
func (eng *Engine) GetTransferSpeedLimit() int64 {
	if eng.Tasks != nil {
		return eng.Tasks.SpeedLimit()
	}
	if eng.Config != nil {
		return eng.Config.TransferSpeedLimit
	}
	return 0
}
