package gui

import (
	"fmt"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
)

// -----------------------------------------------------------------------------
// Asynchronous Task Pipeline Submissions
// -----------------------------------------------------------------------------

// SubmitApply enqueues an asynchronous game post-processing setup task.
func (a *App) SubmitApply(opts engine.ProcessOptions) (string, error) {
	opts.Path = util.CanonicalPath(opts.Path)
	if opts.Path == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitApply(opts)
}

// SubmitNormalize enqueues an asynchronous directory normalization task.
func (a *App) SubmitNormalize(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitNormalize(gamePath)
}

// SubmitRollback enqueues an asynchronous game rollback task.
func (a *App) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitRollback(gamePath, dryRun)
}

// SubmitVerify enqueues an asynchronous depot integrity verification task.
func (a *App) SubmitVerify(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitVerify(gamePath)
}

// SubmitUpdate enqueues an asynchronous game update task.
func (a *App) SubmitUpdate(gamePath string) (string, error) {
	gamePath = util.CanonicalPath(gamePath)
	if gamePath == "" {
		return "", fmt.Errorf("game path cannot be empty")
	}
	return a.Engine.SubmitUpdate(gamePath)
}

// StartDownloadTransfer enqueues a new resumable file/depot download in the unified tasks engine.
func (a *App) StartDownloadTransfer(opts engine.DownloadGameOptions) (string, error) {
	return a.Engine.SubmitDownload(opts)
}

// UpdateTool enqueues an asynchronous tool update download in the unified Tasks engine.
func (a *App) UpdateTool(toolKey string) (string, error) {
	return a.Engine.SubmitToolUpdate(toolKey)
}

// SubmitBatchLibraryOp enqueues a batch library operation.
// Supported kinds: "achievements", "artwork", "normalize", "verify".
func (a *App) SubmitBatchLibraryOp(kind string) (string, error) {
	return a.Engine.SubmitBatchLibraryOp(kind)
}

// -----------------------------------------------------------------------------
// Unified Task Management
// -----------------------------------------------------------------------------

// ListTasks returns all background tasks tracked by the unified tasks manager.
func (a *App) ListTasks() []*tasks.Task {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return []*tasks.Task{}
	}
	return a.Engine.Tasks.List()
}

// GetTask retrieves details for a single task.
func (a *App) GetTask(id string) (*tasks.Task, bool) {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return nil, false
	}
	return a.Engine.Tasks.Get(id)
}

// PauseTask suspends an active or queued task.
func (a *App) PauseTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Pause(id)
}

// ResumeTask continues a paused task.
func (a *App) ResumeTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Resume(id)
}

// CancelTask terminates a running, queued, or paused task.
func (a *App) CancelTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Cancel(id)
}

// RemoveTask removes a finished task from the history list.
func (a *App) RemoveTask(id string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Remove(id)
}

// ReorderTasks rearranges queued tasks by ID priority.
func (a *App) ReorderTasks(ids []string) error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks.Reorder(ids)
}

// ClearFinishedTasks purges finished tasks from manager memory.
func (a *App) ClearFinishedTasks() error {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return nil
	}
	return a.Engine.Tasks.ClearFinished()
}

// SetTransferSpeedLimit configures the download bandwidth limiter (bytes/sec, 0 for unlimited) and persists it.
func (a *App) SetTransferSpeedLimit(bps int64) {
	if a.Engine != nil && a.Engine.Tasks != nil {
		a.Engine.Tasks.SetSpeedLimit(bps)
	}
	if a.Config != nil {
		a.Config.TransferSpeedLimit = bps
		_ = config.SaveConfig(a.Config, "")
	}
}

// GetTransferSpeedLimit returns the active download bandwidth limit.
func (a *App) GetTransferSpeedLimit() int64 {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return 0
	}
	return a.Engine.Tasks.SpeedLimit()
}

// GetToolsStatus returns version, release notes, and update status for GBE fork and steamless-rs.
func (a *App) GetToolsStatus() ([]tools.ToolStatus, error) {
	return a.Engine.GetToolsStatus(a.getContext())
}
