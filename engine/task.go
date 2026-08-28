package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/util"
)

// SubmitApply schedules an asynchronous game post-processing setup task.
func (eng *Engine) SubmitApply(opts ProcessOptions) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), opts.Path)
	return eng.Tasks.SubmitOp(&ApplyOperation{
		Engine:   eng,
		Options:  opts,
		GameName: gameName,
	})
}

// SubmitUpdate schedules an asynchronous game update task.
func (eng *Engine) SubmitUpdate(gamePath string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	return eng.Tasks.SubmitOp(&UpdateOperation{
		Engine:   eng,
		GamePath: gamePath,
		GameName: gameName,
	})
}

// SubmitVerify schedules an asynchronous game depot integrity verification task.
func (eng *Engine) SubmitVerify(gamePath string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	return eng.Tasks.SubmitOp(&VerifyOperation{
		Engine:   eng,
		GamePath: gamePath,
		GameName: gameName,
	})
}

// SubmitNormalize schedules an asynchronous directory normalization task.
func (eng *Engine) SubmitNormalize(gamePath string) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	return eng.Tasks.SubmitOp(&NormalizeOperation{
		Engine:   eng,
		GamePath: gamePath,
		GameName: gameName,
	})
}

// SubmitRollback schedules an asynchronous game rollback task.
func (eng *Engine) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	gameName := eng.resolveGameName(context.Background(), gamePath)
	return eng.Tasks.SubmitOp(&RollbackOperation{
		Engine:   eng,
		GamePath: gamePath,
		GameName: gameName,
		DryRun:   dryRun,
	})
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

func (eng *Engine) getLibraryDirs() []string {
	var dirs []string
	if eng.Config != nil && len(eng.Config.LibraryPaths) > 0 {
		dirs = append(dirs, eng.Config.LibraryPaths...)
	}
	if len(dirs) == 0 {
		home, _ := os.UserHomeDir()
		defaultGames := filepath.Join(home, "Games")
		if util.FileExists(defaultGames) {
			dirs = append(dirs, defaultGames)
		}
	}
	return dirs
}

// SubmitBatchAchievements schedules an asynchronous batch task to fetch schemas and download achievement images for all library games.
func (eng *Engine) SubmitBatchAchievements() (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	return eng.Tasks.SubmitOp(&BatchAchievementsOperation{
		Engine:      eng,
		LibraryDirs: eng.getLibraryDirs(),
	})
}

// SubmitBatchArtwork schedules an asynchronous batch task to query and cache missing covers, heroes, and banners for all games.
func (eng *Engine) SubmitBatchArtwork() (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	return eng.Tasks.SubmitOp(&BatchArtworkOperation{
		Engine:      eng,
		LibraryDirs: eng.getLibraryDirs(),
	})
}

// SubmitBatchNormalize schedules an asynchronous batch task to normalize directory names across all library games.
func (eng *Engine) SubmitBatchNormalize() (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	return eng.Tasks.SubmitOp(&BatchNormalizeOperation{
		Engine:      eng,
		LibraryDirs: eng.getLibraryDirs(),
	})
}

// SubmitBatchVerify schedules an asynchronous batch integrity verification task across all games with ACF manifests.
func (eng *Engine) SubmitBatchVerify() (string, error) {
	if eng.Tasks == nil {
		return "", fmt.Errorf("task manager not initialized")
	}

	return eng.Tasks.SubmitOp(&BatchVerifyOperation{
		Engine:      eng,
		LibraryDirs: eng.getLibraryDirs(),
	})
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

		op := &DownloadOperation{
			Engine: eng,
			Options: DownloadGameOptions{
				AppID:     appIDVal,
				TargetDir: t.GameDir,
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
