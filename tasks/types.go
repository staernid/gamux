package tasks

import (
	"context"
	"errors"

	"github.com/staernid/gamux/domain"
)

// Standard typed error modes across the seam.
var (
	ErrTaskNotFound      = errors.New("tasks: task not found")
	ErrTaskNotPausable   = errors.New("tasks: task does not support pause/resume")
	ErrInvalidTransition = errors.New("tasks: invalid state transition")
	ErrManagerClosed     = errors.New("tasks: manager is shut down")
	ErrDuplicateTaskID   = errors.New("tasks: task id already exists")
)

// TaskType identifies the operational domain of a task.
type TaskType string

const (
	TypeDownload   TaskType = "download"
	TypeApply      TaskType = "apply"
	TypeUpdate     TaskType = "update"
	TypeVerify     TaskType = "verify"
	TypeToolUpdate TaskType = "tool_update"
	TypeWorkshop   TaskType = "workshop"
	TypeNormalize  TaskType = "normalize"
	TypeRollback   TaskType = "rollback"
	TypeCustom     TaskType = "custom"
)

// TaskStatus represents the lifecycle state machine.
type TaskStatus string

const (
	StatusQueued    TaskStatus = "queued"
	StatusRunning   TaskStatus = "running"
	StatusPaused    TaskStatus = "paused"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
)

// Capability bitmask declares runtime control features supported by a task.
type Capability uint32

const (
	CapNone        Capability = 0
	CapCancellable Capability = 1 << iota
	CapPausable
	CapPersist
	CapBandwidthLimit
)

// Has returns true if the capability bit is set.
func (c Capability) Has(target Capability) bool {
	return c&target != 0
}

// Progress captures unified numeric, byte-rate, and step-based telemetry.
type Progress struct {
	CurrentBytes int64   `json:"current_bytes"`
	TotalBytes   int64   `json:"total_bytes"`
	CurrentItems int     `json:"current_items"`
	TotalItems   int     `json:"total_items"`
	BytesPerSec  float64 `json:"bytes_per_sec"`
	ETASeconds   int     `json:"eta_seconds"`
	Percent      float64 `json:"percent"`
}

// Task is the canonical entity returned to callers and emitted in telemetry.
type Task struct {
	ID           string            `json:"id"`
	Type         TaskType          `json:"type"`
	Title        string            `json:"title"`
	GameName     string            `json:"game_name,omitempty"`
	GameDir      string            `json:"game_dir,omitempty"`
	AppID        string            `json:"app_id,omitempty"`
	Status       TaskStatus        `json:"status"`
	Stage        string            `json:"stage"`
	Details      string            `json:"details"`
	Capabilities Capability        `json:"capabilities"`
	Progress     Progress          `json:"progress"`
	Error        string            `json:"error,omitempty"`
	CreatedAt    int64             `json:"created_at"`
	StartedAt    int64             `json:"started_at,omitempty"`
	FinishedAt   int64             `json:"finished_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Game         *domain.Game      `json:"game,omitempty"`

	// Run is the internal execution closure (not exported to JSON).
	Run TaskFunc `json:"-"`
}

// TaskFunc is the execution closure executed by the worker pool.
type TaskFunc func(ctx context.Context, ctrl *Controller) error

// Operation represents a self-contained, typed executable task with its own metadata, capabilities, and execution lifecycle.
type Operation interface {
	// TaskSpec returns the specification describing this operation.
	TaskSpec() TaskSpec

	// Execute runs the operation within the task supervisor's controller context.
	Execute(ctx context.Context, ctrl *Controller) error
}

// TaskSpec configures an arbitrary task submission blueprint.
type TaskSpec struct {
	ID           string
	Type         TaskType
	Title        string
	GameName     string
	GameDir      string
	AppID        string
	Capabilities Capability
	Priority     int
	Metadata     map[string]string
	Op           Operation
	Run          TaskFunc
}

// Filter allows precise querying of historical and active tasks.
type Filter struct {
	IDs      []string     `json:"ids,omitempty"`
	Types    []TaskType   `json:"types,omitempty"`
	Statuses []TaskStatus `json:"statuses,omitempty"`
	AppID    string       `json:"app_id,omitempty"`
}

// Matches returns true if the snapshot satisfies the filter.
func (f Filter) Matches(t *Task) bool {
	if len(f.IDs) > 0 {
		matched := false
		for _, id := range f.IDs {
			if t.ID == id {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.Types) > 0 {
		matched := false
		for _, typ := range f.Types {
			if t.Type == typ {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.Statuses) > 0 {
		matched := false
		for _, st := range f.Statuses {
			if t.Status == st {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if f.AppID != "" && t.AppID != f.AppID {
		return false
	}
	return true
}

// ProgressReporter is an interface for reporting progress in stages and tasks.
type ProgressReporter interface {
	UpdateStage(stage, details string)
	UpdateProgress(percent float64, currentBytes int64, currentItems, totalItems, totalBytes int64, speed float64, eta int)
	ReportStep(step, total int, stage, details string)
}

// TaskRunner executes a task instance within a Controller.
type TaskRunner func(ctx context.Context, t *Task, ctrl *Controller) error
