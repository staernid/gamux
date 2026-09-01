package gui

import (
	"fmt"

	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/tools"
)

// -----------------------------------------------------------------------------
// Asynchronous Task Pipeline Submissions
// -----------------------------------------------------------------------------

// SubmitApply enqueues an asynchronous game post-processing setup task.
func (a *App) SubmitApply(opts engine.ProcessOptions) (string, error) {
	path, err := requireGamePath(opts.Path)
	if err != nil {
		return "", err
	}
	opts.Path = path
	return a.Engine.SubmitApply(opts)
}

// SubmitNormalize enqueues an asynchronous directory normalization task.
func (a *App) SubmitNormalize(gamePath string) (string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return "", err
	}
	return a.Engine.SubmitNormalize(path)
}

// SubmitRollback enqueues an asynchronous game rollback task.
func (a *App) SubmitRollback(gamePath string, dryRun bool) (string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return "", err
	}
	return a.Engine.SubmitRollback(path, dryRun)
}

// SubmitVerify enqueues an asynchronous depot integrity verification task.
func (a *App) SubmitVerify(gamePath string) (string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return "", err
	}
	return a.Engine.SubmitVerify(path)
}

// SubmitUpdate enqueues an asynchronous game update task.
func (a *App) SubmitUpdate(gamePath string) (string, error) {
	path, err := requireGamePath(gamePath)
	if err != nil {
		return "", err
	}
	return a.Engine.SubmitUpdate(path)
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

func (a *App) requireTasks() (*tasks.Manager, error) {
	if a.Engine == nil || a.Engine.Tasks == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	return a.Engine.Tasks, nil
}

// ListTasks returns all background tasks tracked by the unified tasks manager.
func (a *App) ListTasks() []*tasks.Task {
	if tm, err := a.requireTasks(); err == nil {
		return tm.List()
	}
	return []*tasks.Task{}
}

// GetTask retrieves details for a single task.
func (a *App) GetTask(id string) (*tasks.Task, bool) {
	if tm, err := a.requireTasks(); err == nil {
		return tm.Get(id)
	}
	return nil, false
}

// PauseTask suspends an active or queued task.
func (a *App) PauseTask(id string) error {
	tm, err := a.requireTasks()
	if err != nil {
		return err
	}
	return tm.Pause(id)
}

// ResumeTask continues a paused task.
func (a *App) ResumeTask(id string) error {
	tm, err := a.requireTasks()
	if err != nil {
		return err
	}
	return tm.Resume(id)
}

// CancelTask terminates a running, queued, or paused task.
func (a *App) CancelTask(id string) error {
	tm, err := a.requireTasks()
	if err != nil {
		return err
	}
	return tm.Cancel(id)
}

// RemoveTask removes a finished task from the history list.
func (a *App) RemoveTask(id string) error {
	tm, err := a.requireTasks()
	if err != nil {
		return err
	}
	return tm.Remove(id)
}

// ReorderTasks rearranges queued tasks by ID priority.
func (a *App) ReorderTasks(ids []string) error {
	tm, err := a.requireTasks()
	if err != nil {
		return err
	}
	return tm.Reorder(ids)
}

// ClearFinishedTasks purges finished tasks from manager memory.
func (a *App) ClearFinishedTasks() error {
	if tm, err := a.requireTasks(); err == nil {
		return tm.ClearFinished()
	}
	return nil
}

// SetTransferSpeedLimit configures the download bandwidth limiter (bytes/sec, 0 for unlimited) and persists it.
func (a *App) SetTransferSpeedLimit(bps int64) {
	if a.Engine != nil {
		_ = a.Engine.SetTransferSpeedLimit(bps)
	}
	if a.Config != nil {
		a.Config.TransferSpeedLimit = bps
	}
}

// GetTransferSpeedLimit returns the active download bandwidth limit.
func (a *App) GetTransferSpeedLimit() int64 {
	if a.Engine != nil {
		return a.Engine.GetTransferSpeedLimit()
	}
	if a.Config != nil {
		return a.Config.TransferSpeedLimit
	}
	return 0
}

// GetToolsStatus returns version, release notes, and update status for GBE fork and steamless-rs.
func (a *App) GetToolsStatus() ([]tools.ToolStatus, error) {
	return a.Engine.GetToolsStatus(a.getContext())
}
