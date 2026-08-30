package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/downloader"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
)



// DownloadOperation downloads game depots with progress streaming, throttling, pause/resume, and auto-setup.
type DownloadOperation struct {
	Engine   *Engine
	Options  DownloadGameOptions
	GameName string
	Result   *downloader.Result
}

func (op *DownloadOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeDownload,
		Title:        fmt.Sprintf("Download %s", op.GameName),
		GameName:     op.GameName,
		GameDir:      op.Options.TargetDir,
		AppID:        fmt.Sprintf("%d", op.Options.AppID),
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapPersist | tasks.CapBandwidthLimit,
	}
}

func (op *DownloadOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	wrappedCtx := op.Engine.WrapContext(ctx)
	ctrl.ReportStage("Resolving Keys & Manifests", "Contacting depot database...")

	origProgress := op.Options.OnProgress
	op.Options.OnProgress = func(p downloader.ProgressUpdate) {
		if err := ctrl.CheckPauseOrCancel(); err != nil {
			return
		}
		_ = ctrl.Throttle(int(p.SpeedBps / 10))
		ctrl.ReportProgress(p.DownloadedBytes, p.TotalBytes, p.CurrentItems, p.TotalItems, p.SpeedBps, p.ETASeconds, p.Percent)
		if p.Item != "" {
			ctrl.ReportStage("Downloading Depots", p.Item)
		}
		if origProgress != nil {
			origProgress(p)
		}
	}

	op.Options.CheckCancel = ctrl.CheckPauseOrCancel
	op.Options.Throttle = ctrl.Throttle

	res, err := op.Engine.DownloadGame(wrappedCtx, op.Options)
	op.Result = res
	if err == nil && op.Options.AutoApply && op.Engine != nil {
		if status, inspectErr := op.Engine.InspectStatus(ctx, op.Options.TargetDir); inspectErr == nil && status != nil {
			ctrl.SetResultGame(status)
		}
	}
	return err
}

// ToolUpdateOperation updates auxiliary utilities (GBE, Steamless, UMU).
type ToolUpdateOperation struct {
	Engine      *Engine
	ToolKey     string
	DisplayName string
	TargetDir   string
	Tool        tools.Tool
}

func (op *ToolUpdateOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeToolUpdate,
		Title:        fmt.Sprintf("Update %s", op.DisplayName),
		GameName:     op.DisplayName,
		GameDir:      op.TargetDir,
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapBandwidthLimit,
		Metadata:     map[string]string{"tool_key": op.ToolKey},
	}
}

func (op *ToolUpdateOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	wrappedCtx := op.Engine.WrapContext(ctx)
	ctrl.ReportStage("Checking GitHub Releases", fmt.Sprintf("Querying latest release for %s...", op.Tool.Name()))

	if err := ctrl.CheckPauseOrCancel(); err != nil {
		return err
	}

	ctrl.ReportStage("Downloading Tool Release", fmt.Sprintf("Fetching %s assets to %s", op.Tool.Name(), op.TargetDir))
	if err := op.Tool.Update(wrappedCtx); err != nil {
		return fmt.Errorf("update %s: %w", op.Tool.Name(), err)
	}

	ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf("%s updated successfully", op.Tool.Name()))
	return nil
}

// WorkshopOperation downloads a Steam Workshop item or mod.
type WorkshopOperation struct {
	ItemURLOrID string
	TargetDir   string
	DryRun      bool
}

func (op *WorkshopOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         tasks.TypeWorkshop,
		Title:        fmt.Sprintf("Download Workshop Item (%s)", op.ItemURLOrID),
		GameDir:      op.TargetDir,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *WorkshopOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Fetching Workshop Details", op.ItemURLOrID)
	return steam.DownloadWorkshopItem(ctx, op.ItemURLOrID, op.TargetDir, op.DryRun)
}

// BatchLibraryAction is a per-game callback for BatchLibraryOperation.
// It returns true if it performed meaningful work (for summary counting).
type BatchLibraryAction func(ctx context.Context, eng *Engine, game *domain.Game, ctrl *tasks.Controller, index, total int) (didWork bool, err error)

// BatchLibraryOperation iterates over all library games and applies a per-game action.
// This is the single generic skeleton for batch achievements, artwork, normalize, verify, etc.
type BatchLibraryOperation struct {
	Engine      *Engine
	LibraryDirs []string
	BatchTitle  string
	TaskType    tasks.TaskType
	ScanLabel   string // e.g. "Scanning games for missing achievement data..."
	DoneLabel   string // e.g. "Processed %d games (%d updated)" — must contain two %d verbs
	Action      BatchLibraryAction
}

func (op *BatchLibraryOperation) TaskSpec() tasks.TaskSpec {
	return tasks.TaskSpec{
		Type:         op.TaskType,
		Title:        op.BatchTitle,
		Capabilities: tasks.CapCancellable,
	}
}

func (op *BatchLibraryOperation) Execute(ctx context.Context, ctrl *tasks.Controller) error {
	ctrl.ReportStage("Scanning Library", op.ScanLabel)
	games, err := op.Engine.BatchInspectLibraries(ctx, op.LibraryDirs)
	if err != nil {
		return err
	}

	total := len(games)
	if total == 0 {
		ctrl.ReportStage("Completed", "No games in library")
		ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
		return nil
	}

	processedCount := 0
	successCount := 0

	for i, g := range games {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if g == nil {
			continue
		}

		pct := float64(i) / float64(total) * 100
		ctrl.ReportProgress(int64(i), int64(total), i, total, 0, 0, pct)
		ctrl.ReportStage(
			fmt.Sprintf("[%d/%d] %s", i+1, total, g.Name),
			fmt.Sprintf("Processing %s...", g.Name),
		)

		didWork, err := op.Action(ctx, op.Engine, g, ctrl, i, total)
		if err != nil {
			slog.Warn("Batch action failed for game", "game", g.Name, "error", err)
		}
		if didWork {
			successCount++
		}
		processedCount++
	}

	ctrl.ReportProgress(int64(total), int64(total), total, total, 0, 0, 100)
	ctrl.ReportStage("Completed", fmt.Sprintf(op.DoneLabel, processedCount, successCount))
	return nil
}
