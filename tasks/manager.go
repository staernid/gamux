package tasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/staernid/gamux/events"
)

// Options configures Manager initialization.
type Options struct {
	MaxConcurrentDownloads  int
	MaxConcurrentOperations int
	Store                   Store
	EventSink               EventSink
	Limiter                 Limiter
}

// Manager is the unified task supervisor.
type Manager struct {
	mu           sync.RWMutex
	tasks        []*Task
	controllers  map[string]*Controller
	cancels      map[string]context.CancelFunc
	runners      map[TaskType]TaskRunner
	store        Store
	eventSink    EventSink
	limiter      Limiter
	ioSem        chan struct{}
	cpuSem       chan struct{}
	closed       atomic.Bool
	stopCh       chan struct{}
	notifyCh     chan struct{}
	lastProgress map[string]int64 // TaskID -> timestamp in ms for 10Hz coalescing
}

func generateTaskID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	if prefix == "" {
		prefix = "task"
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

// NewManager creates a Manager with the supplied options.
func NewManager(opts Options) *Manager {
	if opts.MaxConcurrentDownloads <= 0 {
		opts.MaxConcurrentDownloads = 2
	}
	if opts.MaxConcurrentOperations <= 0 {
		opts.MaxConcurrentOperations = 4
	}
	if opts.Store == nil {
		opts.Store = NewMemoryStore()
	}
	if opts.EventSink == nil {
		opts.EventSink = NewNoopEventSink()
	}
	if opts.Limiter == nil {
		opts.Limiter = NewTokenBucketLimiter(0)
	}

	m := &Manager{
		tasks:        make([]*Task, 0),
		controllers:  make(map[string]*Controller),
		cancels:      make(map[string]context.CancelFunc),
		runners:      make(map[TaskType]TaskRunner),
		store:        opts.Store,
		eventSink:    opts.EventSink,
		limiter:      opts.Limiter,
		ioSem:        make(chan struct{}, opts.MaxConcurrentDownloads),
		cpuSem:       make(chan struct{}, opts.MaxConcurrentOperations),
		stopCh:       make(chan struct{}),
		notifyCh:     make(chan struct{}, 1),
		lastProgress: make(map[string]int64),
	}

	// Load existing persistent tasks
	if loaded, err := m.store.Load(); err == nil && len(loaded) > 0 {
		for _, t := range loaded {
			if t.Status == StatusRunning {
				t.Status = StatusPaused
				t.Stage = "Paused"
			}
			m.tasks = append(m.tasks, t)
		}
	}

	go m.schedulerLoop()
	return m
}

// RegisterRunner registers an executable handler for tasks of a given TaskType.
func (m *Manager) RegisterRunner(taskType TaskType, runner TaskRunner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runners[taskType] = runner
}

func (m *Manager) notifyScheduler() {
	if m.closed.Load() {
		return
	}

	select {
	case m.notifyCh <- struct{}{}:
	default:
	}
}

func (m *Manager) schedulerLoop() {
	for {
		select {
		case <-m.stopCh:
			return
		case <-m.notifyCh:
			m.mu.Lock()
			if m.closed.Load() {
				m.mu.Unlock()
				return
			}

			for _, t := range m.tasks {
				if t.Status != StatusQueued {
					continue
				}

				// Determine concurrency category
				var targetSem chan struct{}
				if t.Type == TypeDownload || t.Type == TypeToolUpdate {
					targetSem = m.ioSem
				} else {
					targetSem = m.cpuSem
				}

				// Try non-blocking slot acquisition
				select {
				case targetSem <- struct{}{}:
					// Acquired slot, start worker
					t.Status = StatusRunning
					t.StartedAt = time.Now().UnixMilli()
					if t.Stage == "" {
						t.Stage = "Running"
					}
					ctx, cancel := context.WithCancel(context.Background())
					m.cancels[t.ID] = cancel

					taskCopy := *t
					m.emitEvent(events.EventTaskStarted, &taskCopy)
					_ = m.store.Save(m.tasks)

					go m.runWorker(ctx, t, targetSem)
				default:
					// Worker pool is saturated for this category, wait for slots
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *Manager) runWorker(ctx context.Context, t *Task, slot chan struct{}) {
	defer func() {
		<-slot
		m.notifyScheduler()
	}()

	taskID := t.ID
	caps := t.Capabilities

	ctrl := NewController(
		ctx,
		taskID,
		caps,
		m.limiter,
		func(currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64) {
			m.handleProgress(taskID, currBytes, totalBytes, currItems, totalItems, speed, eta, pct)
		},
		func(stage, details string) {
			m.handleStage(taskID, stage, details)
		},
		func(step, total int, stage, details string) {
			m.handleStep(taskID, step, total, stage, details)
		},
	)

	m.mu.Lock()
	m.controllers[taskID] = ctrl
	runFn := t.Run
	if runFn == nil && m.runners != nil {
		if r, ok := m.runners[t.Type]; ok {
			runFn = func(c context.Context, cctrl *Controller) error {
				return r(c, t, cctrl)
			}
		}
	}
	m.mu.Unlock()

	var runErr error
	if runFn != nil {
		runErr = runFn(ctx, ctrl)
	} else {
		runErr = fmt.Errorf("tasks: no executable runner available for task %s (%s)", t.ID, t.Type)
	}

	m.mu.Lock()
	delete(m.controllers, taskID)
	delete(m.cancels, taskID)

	t.FinishedAt = time.Now().UnixMilli()

	if ctx.Err() == context.Canceled || (runErr != nil && errors.Is(runErr, context.Canceled)) {
		t.Status = StatusCancelled
		t.Details = "Task cancelled by user"
		taskCopy := *t
		m.emitEvent(events.EventTaskCancelled, &taskCopy)
	} else if runErr != nil {
		t.Status = StatusFailed
		t.Error = runErr.Error()
		taskCopy := *t
		m.emitEvent(events.EventTaskFailed, &taskCopy)
	} else {
		t.Status = StatusCompleted
		t.Progress.Percent = 100.0
		if t.Stage == "" || t.Stage == "Running" {
			t.Stage = "Completed"
		}
		if resGame := ctrl.ResultGame(); resGame != nil {
			t.Game = resGame
		}
		if meta := ctrl.Metadata(); len(meta) > 0 {
			if t.Metadata == nil {
				t.Metadata = make(map[string]string)
			}
			for k, v := range meta {
				t.Metadata[k] = v
			}
		}
		taskCopy := *t
		m.emitEvent(events.EventTaskCompleted, &taskCopy)
	}

	_ = m.store.Save(m.tasks)
	m.mu.Unlock()
}

func (m *Manager) handleProgress(taskID string, currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64) {
	now := time.Now().UnixMilli()

	m.mu.Lock()
	t := m.findTaskLocked(taskID)
	if t == nil || t.Status != StatusRunning {
		m.mu.Unlock()
		return
	}

	t.Progress.CurrentBytes = currBytes
	t.Progress.TotalBytes = totalBytes
	t.Progress.CurrentItems = currItems
	t.Progress.TotalItems = totalItems
	t.Progress.BytesPerSec = speed
	t.Progress.ETASeconds = eta
	t.Progress.Percent = pct

	lastTick := m.lastProgress[taskID]
	// Throttle event emissions to 10Hz (100ms)
	shouldEmit := (now - lastTick) >= 100
	if shouldEmit {
		m.lastProgress[taskID] = now
		taskCopy := *t
		m.mu.Unlock()
		m.emitEvent(events.EventTaskProgress, &taskCopy)
		return
	}
	m.mu.Unlock()
}

func (m *Manager) handleStage(taskID string, stage, details string) {
	m.mu.Lock()
	t := m.findTaskLocked(taskID)
	if t == nil {
		m.mu.Unlock()
		return
	}
	t.Stage = stage
	t.Details = details
	taskCopy := *t
	m.mu.Unlock()

	m.emitEvent(events.EventTaskStageChange, &taskCopy)
}

func (m *Manager) handleStep(taskID string, step, total int, stage, details string) {
	m.mu.Lock()
	t := m.findTaskLocked(taskID)
	if t == nil {
		m.mu.Unlock()
		return
	}
	t.Stage = stage
	t.Details = details
	t.Progress.CurrentItems = step
	t.Progress.TotalItems = total
	if total > 0 {
		t.Progress.Percent = float64(step) / float64(total) * 100
	}
	taskCopy := *t
	m.mu.Unlock()

	m.emitEvent(events.EventTaskProgress, &taskCopy)
}

func (m *Manager) findTaskLocked(id string) *Task {
	for _, t := range m.tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (m *Manager) emitEvent(typ events.EventType, t *Task) {
	if m.eventSink == nil || t == nil {
		return
	}

	ev := events.Event{
		Type:      typ,
		TaskID:    t.ID,
		Operation: string(t.Type),
		AppID:     t.AppID,
		GameDir:   t.GameDir,
		GameName:  t.GameName,
		Stage:     t.Stage,
		Details:   t.Details,
		Error:     t.Error,
		Progress: events.TaskProgress{
			CurrentBytes: t.Progress.CurrentBytes,
			TotalBytes:   t.Progress.TotalBytes,
			CurrentItems: t.Progress.CurrentItems,
			TotalItems:   t.Progress.TotalItems,
			BytesPerSec:  t.Progress.BytesPerSec,
			ETASeconds:   t.Progress.ETASeconds,
			Percent:      t.Progress.Percent,
		},
		Metadata:  t.Metadata,
		Game:      t.Game,
		Timestamp: time.Now().UnixMilli(),
	}

	m.eventSink.Emit(ev)
}

// SubmitOp enqueues a self-contained Operation into the supervisor.
func (m *Manager) SubmitOp(op Operation) (string, error) {
	if op == nil {
		return "", fmt.Errorf("tasks: cannot submit nil operation")
	}
	spec := op.TaskSpec()
	spec.Op = op
	if spec.Run == nil {
		spec.Run = op.Execute
	}
	return m.Submit(spec)
}

// RunSyncOp executes a self-contained Operation synchronously within the caller context.
func (m *Manager) RunSyncOp(ctx context.Context, op Operation) error {
	if op == nil {
		return fmt.Errorf("tasks: cannot run nil operation")
	}
	spec := op.TaskSpec()
	spec.Op = op
	if spec.Run == nil {
		spec.Run = op.Execute
	}
	return m.RunSync(ctx, spec)
}

// Submit enqueues a new task blueprint.
func (m *Manager) Submit(spec TaskSpec) (string, error) {
	if m.closed.Load() {
		return "", ErrManagerClosed
	}

	if spec.Run == nil && spec.Op != nil {
		spec.Run = spec.Op.Execute
	}

	id := spec.ID
	if id == "" {
		prefix := string(spec.Type)
		if prefix == "" {
			prefix = "task"
		}
		id = generateTaskID(prefix)
	}

	m.mu.Lock()
	if m.findTaskLocked(id) != nil {
		m.mu.Unlock()
		return "", ErrDuplicateTaskID
	}

	task := &Task{
		ID:           id,
		Type:         spec.Type,
		Title:        spec.Title,
		GameName:     spec.GameName,
		GameDir:      spec.GameDir,
		AppID:        spec.AppID,
		Status:       StatusQueued,
		Stage:        "Queued",
		Capabilities: spec.Capabilities,
		CreatedAt:    time.Now().UnixMilli(),
		Metadata:     spec.Metadata,
		Run:          spec.Run,
	}

	// Respect priority: higher priority placed earlier in queued partition
	inserted := false
	if spec.Priority > 0 {
		for i, existing := range m.tasks {
			if existing.Status == StatusQueued {
				m.tasks = append(m.tasks[:i], append([]*Task{task}, m.tasks[i:]...)...)
				inserted = true
				break
			}
		}
	}
	if !inserted {
		m.tasks = append(m.tasks, task)
	}

	_ = m.store.Save(m.tasks)
	m.mu.Unlock()

	m.notifyScheduler()

	slog.Info("Submitted task to unified manager", "id", task.ID, "type", task.Type, "title", task.Title)
	return task.ID, nil
}

// RunSync executes a task blueprint synchronously within the caller context,
// driving progress through a Controller and emitting standard progress telemetry.
func (m *Manager) RunSync(ctx context.Context, spec TaskSpec) error {
	if spec.Run == nil && spec.Op != nil {
		spec.Run = spec.Op.Execute
	}

	id := spec.ID
	if id == "" {
		prefix := string(spec.Type)
		if prefix == "" {
			prefix = "task"
		}
		id = generateTaskID(prefix)
	}

	task := &Task{
		ID:           id,
		Type:         spec.Type,
		Title:        spec.Title,
		GameName:     spec.GameName,
		GameDir:      spec.GameDir,
		AppID:        spec.AppID,
		Status:       StatusRunning,
		Stage:        "Running",
		Capabilities: spec.Capabilities,
		CreatedAt:    time.Now().UnixMilli(),
		StartedAt:    time.Now().UnixMilli(),
		Metadata:     spec.Metadata,
	}

	taskCopy := *task
	m.emitEvent(events.EventTaskStarted, &taskCopy)

	ctrl := NewController(
		ctx,
		id,
		spec.Capabilities,
		m.limiter,
		func(currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64) {
			m.handleProgress(id, currBytes, totalBytes, currItems, totalItems, speed, eta, pct)
		},
		func(stage, details string) {
			m.handleStage(id, stage, details)
		},
		func(step, total int, stage, details string) {
			m.handleStep(id, step, total, stage, details)
		},
	)

	var runErr error
	if spec.Run != nil {
		runErr = spec.Run(ctx, ctrl)
	}

	task.FinishedAt = time.Now().UnixMilli()
	if ctx.Err() == context.Canceled || (runErr != nil && runErr == context.Canceled) {
		task.Status = StatusCancelled
		task.Details = "Task cancelled"
		tc := *task
		m.emitEvent(events.EventTaskCancelled, &tc)
	} else if runErr != nil {
		task.Status = StatusFailed
		task.Error = runErr.Error()
		task.Details = fmt.Sprintf("Failed: %v", runErr)
		tc := *task
		m.emitEvent(events.EventTaskFailed, &tc)
	} else {
		task.Status = StatusCompleted
		task.Stage = "Completed"
		task.Progress.Percent = 100.0
		tc := *task
		m.emitEvent(events.EventTaskCompleted, &tc)
	}

	return runErr
}

// Pause suspends a running or queued task if CapPausable is supported.
func (m *Manager) Pause(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := m.findTaskLocked(id)
	if t == nil {
		return ErrTaskNotFound
	}
	if !t.Capabilities.Has(CapPausable) {
		return ErrTaskNotPausable
	}

	if t.Status == StatusPaused {
		return nil
	}

	if t.Status == StatusQueued {
		t.Status = StatusPaused
		t.Stage = "Paused"
		_ = m.store.Save(m.tasks)
		taskCopy := *t
		m.emitEvent(events.EventTaskPaused, &taskCopy)
		return nil
	}

	if t.Status == StatusRunning {
		if ctrl, ok := m.controllers[id]; ok {
			ctrl.SetPaused(true)
			t.Status = StatusPaused
			t.Stage = "Paused"
			_ = m.store.Save(m.tasks)
			taskCopy := *t
			m.emitEvent(events.EventTaskPaused, &taskCopy)
			return nil
		}
	}

	return ErrInvalidTransition
}

// Resume unpauses a paused task.
func (m *Manager) Resume(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := m.findTaskLocked(id)
	if t == nil {
		return ErrTaskNotFound
	}
	if t.Status != StatusPaused {
		return ErrInvalidTransition
	}

	if ctrl, ok := m.controllers[id]; ok {
		ctrl.SetPaused(false)
		t.Status = StatusRunning
		t.Stage = "Resumed"
		_ = m.store.Save(m.tasks)
		taskCopy := *t
		m.emitEvent(events.EventTaskResumed, &taskCopy)
		return nil
	}

	// If it was paused in queue, requeue it
	t.Status = StatusQueued
	t.Stage = "Queued"
	_ = m.store.Save(m.tasks)
	m.notifyScheduler()
	taskCopy := *t
	m.emitEvent(events.EventTaskResumed, &taskCopy)
	return nil
}

// Cancel terminates a running or queued task.
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t := m.findTaskLocked(id)
	if t == nil {
		return ErrTaskNotFound
	}

	if t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled {
		return nil
	}

	if cancel, ok := m.cancels[id]; ok {
		// Cancel active worker context and unpause controller to allow exit
		if ctrl, ok := m.controllers[id]; ok {
			ctrl.SetPaused(false)
		}
		cancel()
		return nil
	}

	// Task was in queue/paused
	t.Status = StatusCancelled
	t.FinishedAt = time.Now().UnixMilli()
	t.Details = "Cancelled by user"
	_ = m.store.Save(m.tasks)
	taskCopy := *t
	m.emitEvent(events.EventTaskCancelled, &taskCopy)
	return nil
}

// Remove deletes a finished task from the history list.
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, t := range m.tasks {
		if t.ID == id {
			if t.Status == StatusRunning {
				return fmt.Errorf("cannot remove running task %s; cancel it first", id)
			}
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrTaskNotFound
	}

	m.tasks = append(m.tasks[:idx], m.tasks[idx+1:]...)
	_ = m.store.Save(m.tasks)
	return nil
}

// Reorder rearranges queued tasks according to the supplied ID order.
func (m *Manager) Reorder(ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idMap := make(map[string]*Task)
	for _, t := range m.tasks {
		idMap[t.ID] = t
	}

	var reordered []*Task
	for _, id := range ids {
		if t, ok := idMap[id]; ok && t.Status == StatusQueued {
			reordered = append(reordered, t)
			delete(idMap, id)
		}
	}

	// Append remaining tasks maintaining their relative order
	for _, t := range m.tasks {
		if _, ok := idMap[t.ID]; ok {
			reordered = append(reordered, t)
		}
	}

	m.tasks = reordered
	_ = m.store.Save(m.tasks)
	m.notifyScheduler()
	return nil
}

// ClearFinished removes all completed, failed, and cancelled tasks from the list.
func (m *Manager) ClearFinished() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []*Task
	for _, t := range m.tasks {
		if t.Status == StatusRunning || t.Status == StatusQueued || t.Status == StatusPaused {
			active = append(active, t)
		}
	}

	m.tasks = active
	return m.store.Save(m.tasks)
}

// Get returns a clone of the task matching the specified ID.
func (m *Manager) Get(id string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t := m.findTaskLocked(id)
	if t == nil {
		return nil, false
	}
	cloned := *t
	return &cloned, true
}

// List returns a filtered snapshot slice of tasks.
func (m *Manager) List(filter ...Filter) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []*Task
	for _, t := range m.tasks {
		if len(filter) == 0 || filter[0].Matches(t) {
			cloned := *t
			res = append(res, &cloned)
		}
	}
	return res
}

// ActiveCount returns the number of currently running tasks.
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, t := range m.tasks {
		if t.Status == StatusRunning {
			count++
		}
	}
	return count
}

// SetSpeedLimit updates the global bandwidth limit for tasks with CapBandwidthLimit.
func (m *Manager) SetSpeedLimit(bytesPerSec int64) {
	if m.limiter != nil {
		m.limiter.SetLimit(bytesPerSec)
	}
}

// SpeedLimit returns the active global download bandwidth limit.
func (m *Manager) SpeedLimit() int64 {
	if m.limiter != nil {
		return m.limiter.Limit()
	}
	return 0
}

// Shutdown gracefully cancels active tasks and closes worker pools.
func (m *Manager) Shutdown(ctx context.Context) error {
	if m.closed.Swap(true) {
		return nil
	}
	close(m.stopCh)

	m.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(m.cancels))
	for _, c := range m.cancels {
		cancels = append(cancels, c)
	}
	m.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}

	return nil
}
