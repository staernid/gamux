package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/util"
)

type TaskService struct {
	sc      *ServiceContext
	library *LibraryService
	game    *GameService
}

func NewTaskService(sc *ServiceContext, library *LibraryService, game *GameService) *TaskService {
	return &TaskService{sc: sc, library: library, game: game}
}

// SubmitApply schedules an asynchronous game post-processing setup task.
func (ts *TaskService) SubmitApply(opts ProcessOptions) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := ts.library.resolveGameName(context.Background(), opts.Path)
	title := fmt.Sprintf("Apply Setup (%s)", gameName)
	return ts.sc.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeApply,
		Title:        title,
		GameName:     gameName,
		GameDir:      opts.Path,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Configuring Game", fmt.Sprintf("Applying GBE and launcher profile for %s...", gameName))
			opts.Reporter = ctrl
			res, err := ts.game.ProcessGame(ctx, opts)
			if err == nil && res != nil {
				ctrl.ReportStage("Completed", fmt.Sprintf("Setup applied successfully for %s", gameName))
				ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
			}
			return err
		},
	})
}

// SubmitUpdate schedules an asynchronous game update task.
func (ts *TaskService) SubmitUpdate(gamePath string) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := ts.library.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Update %s", gameName)
	return ts.sc.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeUpdate,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Checking Updates", fmt.Sprintf("Checking Steam depot changes for %s...", gameName))
			res, err := ts.game.UpdateGame(ctx, gamePath)
			if err == nil && res != nil {
				ctrl.ReportStage("Completed", fmt.Sprintf("Update completed for %s", gameName))
				ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
			}
			return err
		},
	})
}

// SubmitVerify schedules an asynchronous game depot integrity verification task.
func (ts *TaskService) SubmitVerify(gamePath string) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := ts.library.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Verify Integrity (%s)", gameName)
	return ts.sc.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeVerify,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Verifying Integrity", fmt.Sprintf("Scanning files in %s...", gameName))
			rep, err := ts.library.VerifyIntegrity(ctx, gamePath)
			if err == nil && rep != nil {
				ctrl.ReportProgress(int64(rep.IntactCount), int64(rep.OfficialCount), rep.IntactCount, rep.OfficialCount, 0, 0, 100)
				ctrl.ReportStage("Completed", fmt.Sprintf("%d/%d files intact", rep.IntactCount, rep.OfficialCount))
			}
			return err
		},
	})
}

// SubmitNormalize schedules an asynchronous directory normalization task.
func (ts *TaskService) SubmitNormalize(gamePath string) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := ts.library.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Normalize Directory (%s)", gameName)
	return ts.sc.Tasks.Submit(tasks.TaskSpec{
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
				NoSteamless:  true,
				AutoYes:      true,
				Reporter:     ctrl,
			}
			res, err := ts.game.ProcessGame(ctx, opts)
			if err == nil && res != nil && res.Info != nil && res.Info.GameDir != "" {
				ctrl.ReportStage("Completed", fmt.Sprintf("Directory normalized to %s", res.Info.GameDir))
				ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
			}
			return err
		},
	})
}

// SubmitRollback schedules an asynchronous game rollback task.
func (ts *TaskService) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := ts.library.resolveGameName(context.Background(), gamePath)
	title := fmt.Sprintf("Rollback %s", gameName)
	return ts.sc.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeRollback,
		Title:        title,
		GameName:     gameName,
		GameDir:      gamePath,
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Restoring Backups", fmt.Sprintf("Reverting patched binaries for %s...", gameName))
			err := ts.game.Rollback(ctx, gamePath, dryRun)
			if err == nil {
				ctrl.ReportStage("Completed", fmt.Sprintf("Rollback completed for %s", gameName))
				ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
			}
			return err
		},
	})
}

// SubmitDownload schedules an asynchronous depot download transfer with state persistence, speed telemetry, and pause/resume support.
func (ts *TaskService) SubmitDownload(opts DownloadGameOptions) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	if opts.AppID == 0 {
		return "", fmt.Errorf("invalid AppID 0")
	}
	if opts.TargetDir == "" {
		return "", fmt.Errorf("target directory cannot be empty")
	}

	gameName := fmt.Sprintf("Steam AppID %d", opts.AppID)
	if p, ok := ts.sc.StoreRegistry.Get("steam"); ok {
		if meta, err := p.GetMetadata(context.Background(), fmt.Sprintf("%d", opts.AppID)); err == nil && meta.Name != "" {
			gameName = meta.Name
		}
	}

	appIDStr := fmt.Sprintf("%d", opts.AppID)
	for _, existing := range ts.sc.Tasks.List() {
		if existing.Type == tasks.TypeDownload && existing.AppID == appIDStr {
			if existing.Status == tasks.StatusRunning || existing.Status == tasks.StatusQueued || existing.Status == tasks.StatusPaused {
				slog.Info("Download already active/queued, reusing existing task", "appID", opts.AppID, "taskID", existing.ID)
				return existing.ID, nil
			}
		}
	}

	return ts.sc.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeDownload,
		Title:        fmt.Sprintf("Download %s", gameName),
		GameName:     gameName,
		GameDir:      opts.TargetDir,
		AppID:        fmt.Sprintf("%d", opts.AppID),
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapPersist | tasks.CapBandwidthLimit,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			wrappedCtx := ts.sc.WrapContext(ctx)
			ctrl.ReportStage("Resolving Keys & Manifests", "Contacting depot database...")

			var lastCurrItems, lastTotalItems int
			origCallback := opts.ProgressCallback
			opts.ProgressCallback = func(curr, total int, item string) {
				lastCurrItems = curr
				lastTotalItems = total
				if origCallback != nil {
					origCallback(curr, total, item)
				}
			}

			opts.ByteProgressCallback = func(currBytes, totalBytes int64, speed float64, eta int, pct float64, item string) {
				if err := ctrl.CheckPauseOrCancel(); err != nil {
					return
				}
				_ = ctrl.Throttle(int(speed / 10))
				ctrl.ReportProgress(currBytes, totalBytes, lastCurrItems, lastTotalItems, speed, eta, pct)
				ctrl.ReportStage("Downloading Depots", item)
			}

			_, err := ts.game.DownloadGame(wrappedCtx, opts)
			return err
		},
	})
}

// SubmitToolUpdate schedules an asynchronous tool update download (GBE or Steamless) through the unified Tasks engine.
func (ts *TaskService) SubmitToolUpdate(toolKey string) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	tool, err := ts.sc.Tools().GetTool(toolKey)
	if err != nil {
		return "", err
	}

	targetDir := ""
	switch strings.ToLower(toolKey) {
	case "gbe":
		targetDir = util.ExpandPath(ts.sc.Config.GbeDir)
	case "steamless":
		targetDir = util.ExpandPath(ts.sc.Config.SteamlessDir)
	case "umu":
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".local", "share", "umu")
	default:
		targetDir = util.ExpandPath(ts.sc.Config.GbeDir)
	}

	displayName := tool.Name()
	if strings.ToLower(toolKey) == "gbe" {
		displayName = "Goldberg Steam Emulator (GBE)"
	} else if strings.ToLower(toolKey) == "steamless" {
		displayName = "Steamless-rs DRM Unpacker"
	} else if strings.ToLower(toolKey) == "umu" {
		displayName = "UMU Proton Launcher (umu-run)"
	}

	return ts.sc.Tasks.Submit(tasks.TaskSpec{
		Type:         tasks.TypeToolUpdate,
		Title:        fmt.Sprintf("Update %s", displayName),
		GameName:     displayName,
		GameDir:      targetDir,
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapBandwidthLimit,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			wrappedCtx := ts.sc.WrapContext(ctx)
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
func (ts *TaskService) SubmitWorkshop(itemURLOrID, targetDir string, dryRun bool) (string, error) {
	if ts.sc.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}
	if targetDir == "" {
		targetDir = "."
	}
	return ts.sc.Tasks.Submit(tasks.TaskSpec{
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

func (ts *TaskService) registerDefaultTaskRunners() {
	if ts.sc.Tasks == nil {
		return
	}

	ts.sc.Tasks.RegisterRunner(tasks.TypeDownload, func(ctx context.Context, t *tasks.Task, ctrl *tasks.Controller) error {
		appIDVal, ok := util.ParseAppID(t.AppID)
		if !ok || appIDVal == 0 {
			return fmt.Errorf("invalid AppID for download task %s: %q", t.ID, t.AppID)
		}
		if t.GameDir == "" {
			return fmt.Errorf("target directory cannot be empty for download task %s", t.ID)
		}

		wrappedCtx := ts.sc.WrapContext(ctx)
		ctrl.ReportStage("Resolving Keys & Manifests", "Contacting depot database...")

		dlOpts := DownloadGameOptions{
			AppID:       appIDVal,
			TargetDir:   t.GameDir,
			AutoApply:   true,
			CheckCancel: ctrl.CheckPauseOrCancel,
			Throttle:    ctrl.Throttle,
			OnProgress: func(p downloader.ProgressUpdate) {
				ctrl.ReportProgress(p.DownloadedBytes, p.TotalBytes, p.CurrentItems, p.TotalItems, p.SpeedBps, p.ETASeconds, p.Percent)
				ctrl.ReportStage("Downloading Depots", p.Item)
			},
		}

		_, err := ts.game.DownloadGame(wrappedCtx, dlOpts)
		return err
	})

	ts.sc.Tasks.RegisterRunner(tasks.TypeToolUpdate, func(ctx context.Context, t *tasks.Task, ctrl *tasks.Controller) error {
		toolKey := t.Metadata["tool_key"]
		if toolKey == "" {
			toolKey = "gbe"
		}
		tool, err := ts.sc.Tools().GetTool(toolKey)
		if err != nil {
			return err
		}
		ctrl.ReportStage("Updating Tool", fmt.Sprintf("Downloading %s...", tool.Name()))
		return tool.Update(ts.sc.WrapContext(ctx))
	})
}
