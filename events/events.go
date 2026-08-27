package events

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// EventType represents the category of a published system or task event.
type EventType string

const (
	EventTaskStarted       EventType = "task:started"
	EventTaskProgress      EventType = "task:progress"
	EventTaskStageChange   EventType = "task:stage_change"
	EventTaskPaused        EventType = "task:paused"
	EventTaskResumed       EventType = "task:resumed"
	EventTaskCompleted     EventType = "task:completed"
	EventTaskFailed        EventType = "task:failed"
	EventTaskCancelled     EventType = "task:cancelled"
	EventTransferStarted   EventType = "transfer:started"
	EventTransferProgress  EventType = "transfer:progress"
	EventTransferPaused    EventType = "transfer:paused"
	EventTransferResumed   EventType = "transfer:resumed"
	EventTransferCompleted EventType = "transfer:completed"
	EventTransferFailed    EventType = "transfer:failed"
	EventTransferCancelled EventType = "transfer:cancelled"
	EventGameLaunched      EventType = "game:launched"
	EventGameExited        EventType = "game:exited"
	EventNotification      EventType = "system:notification"
)

// TaskProgress holds fine-grained numeric telemetry for downloads, hashing, and batch steps.
type TaskProgress struct {
	CurrentBytes int64   `json:"current_bytes"`
	TotalBytes   int64   `json:"total_bytes"`
	CurrentItems int     `json:"current_items"`
	TotalItems   int     `json:"total_items"`
	BytesPerSec  float64 `json:"bytes_per_sec"`
	ETASeconds   int     `json:"eta_seconds"`
	Percent      float64 `json:"percent"`
}

// Event is the canonical data structure broadcast across the gamux application.
type Event struct {
	ID           string            `json:"id"`
	Type         EventType         `json:"type"`
	TaskID       string            `json:"task_id,omitempty"`
	ParentTaskID string            `json:"parent_task_id,omitempty"`
	Operation    string            `json:"operation,omitempty"`
	AppID        string            `json:"app_id,omitempty"`
	GameDir      string            `json:"game_dir,omitempty"`
	GameName     string            `json:"game_name,omitempty"`
	Stage        string            `json:"stage,omitempty"`
	Details      string            `json:"details,omitempty"`
	Progress     TaskProgress      `json:"progress"`
	Error        string            `json:"error,omitempty"`
	Timestamp    int64             `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type subscription struct {
	id      uint64
	ch      chan Event
	filter  map[EventType]bool
	handler func(Event)
}

// Bus is a thread-safe event broker supporting typed topic filtering.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[uint64]*subscription
	nextID      uint64
}

var (
	defaultBusOnce sync.Once
	defaultBusInst *Bus
)

// DefaultBus returns the global default event bus singleton.
func DefaultBus() *Bus {
	defaultBusOnce.Do(func() {
		defaultBusInst = NewBus()
	})
	return defaultBusInst
}

// NewBus creates a new, isolated event bus instance.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[uint64]*subscription),
	}
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Publish broadcasts an event to all matching subscribers non-blockingly.
func (b *Bus) Publish(ev Event) {
	if ev.ID == "" {
		ev.ID = "ev_" + generateID()
	}
	if ev.Timestamp == 0 {
		ev.Timestamp = time.Now().UnixMilli()
	}

	b.mu.RLock()
	subs := make([]*subscription, 0, len(b.subscribers))
	for _, s := range b.subscribers {
		subs = append(subs, s)
	}
	b.mu.RUnlock()

	for _, sub := range subs {
		if len(sub.filter) > 0 && !sub.filter[ev.Type] {
			continue
		}

		if sub.handler != nil {
			go sub.handler(ev)
		} else if sub.ch != nil {
			select {
			case sub.ch <- ev:
			default:
				// If channel buffer is full, drop to prevent hanging publishers
			}
		}
	}
}

// Subscribe returns an event channel and an unsubscribe closure.
// If filter is provided, only events matching one of the filter types will be delivered.
func (b *Bus) Subscribe(filter ...EventType) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID

	filterMap := make(map[EventType]bool, len(filter))
	for _, f := range filter {
		filterMap[f] = true
	}

	ch := make(chan Event, 64)
	sub := &subscription{
		id:     id,
		ch:     ch,
		filter: filterMap,
	}
	b.subscribers[id] = sub

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if s, ok := b.subscribers[id]; ok {
			delete(b.subscribers, id)
			close(s.ch)
		}
	}

	return ch, unsub
}

// SubscribeFunc registers a callback function to be called on matching events.
func (b *Bus) SubscribeFunc(handler func(Event), filter ...EventType) func() {
	if handler == nil {
		return func() {}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID

	filterMap := make(map[EventType]bool, len(filter))
	for _, f := range filter {
		filterMap[f] = true
	}

	ch := make(chan Event, 256)
	stopCh := make(chan struct{})

	sub := &subscription{
		id:     id,
		ch:     ch,
		filter: filterMap,
	}
	b.subscribers[id] = sub

	go func() {
		for {
			select {
			case <-stopCh:
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				handler(ev)
			}
		}
	}()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if s, ok := b.subscribers[id]; ok {
			delete(b.subscribers, id)
			close(stopCh)
			close(s.ch)
		}
	}
}

// String provides a human-readable representation of an EventType.
func (t EventType) String() string {
	return string(t)
}

// Helper constructors for common events

// NewTaskStarted creates a TaskStarted event.
func NewTaskStarted(taskID, op, gameName, gameDir, appID string) Event {
	return Event{
		Type:      EventTaskStarted,
		TaskID:    taskID,
		Operation: op,
		GameName:  gameName,
		GameDir:   gameDir,
		AppID:     appID,
	}
}

// NewTaskProgress creates a TaskProgress event.
func NewTaskProgress(taskID, stage, details string, prog TaskProgress) Event {
	return Event{
		Type:     EventTaskProgress,
		TaskID:   taskID,
		Stage:    stage,
		Details:  details,
		Progress: prog,
	}
}

// NewTaskCompleted creates a TaskCompleted event.
func NewTaskCompleted(taskID, details string) Event {
	return Event{
		Type:    EventTaskCompleted,
		TaskID:  taskID,
		Details: details,
		Progress: TaskProgress{
			Percent: 100.0,
		},
	}
}

// NewTaskFailed creates a TaskFailed event.
func NewTaskFailed(taskID string, err error) Event {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	return Event{
		Type:   EventTaskFailed,
		TaskID: taskID,
		Error:  errStr,
	}
}

// NewTaskCancelled creates a TaskCancelled event.
func NewTaskCancelled(taskID string) Event {
	return Event{
		Type:    EventTaskCancelled,
		TaskID:  taskID,
		Details: fmt.Sprintf("Task %s cancelled by user", taskID),
	}
}

// NewNotification creates a system notification event.
func NewNotification(title, message string) Event {
	return Event{
		Type:    EventNotification,
		Stage:   title,
		Details: message,
		Metadata: map[string]string{
			"title":   title,
			"message": message,
		},
	}
}

// NewGameLaunched creates an event representing a newly started game process.
func NewGameLaunched(gameDir, gameName, appID string, pid int) Event {
	return Event{
		Type:     EventGameLaunched,
		GameDir:  gameDir,
		GameName: gameName,
		AppID:    appID,
		Metadata: map[string]string{
			"pid": fmt.Sprintf("%d", pid),
		},
	}
}

// NewGameExited creates an event representing a finished game process.
func NewGameExited(gameDir, gameName, appID string, exitErr error) Event {
	errStr := ""
	if exitErr != nil {
		errStr = exitErr.Error()
	}
	return Event{
		Type:     EventGameExited,
		GameDir:  gameDir,
		GameName: gameName,
		AppID:    appID,
		Error:    errStr,
	}
}

// NewStepProgress creates an event representing granular step progress.
func NewStepProgress(step, total int, title, details string) Event {
	pct := 0.0
	if total > 0 {
		pct = float64(step) / float64(total) * 100
	}
	return Event{
		Type:    EventTaskProgress,
		Stage:   title,
		Details: details,
		Progress: TaskProgress{
			CurrentItems: step,
			TotalItems:   total,
			Percent:      pct,
		},
		Metadata: map[string]string{
			"step":  fmt.Sprintf("%d", step),
			"total": fmt.Sprintf("%d", total),
		},
	}
}


