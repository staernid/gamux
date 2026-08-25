package tasks_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/staernid/gamux/tasks"
)

func TestTasksManager_BasicLifecycle(t *testing.T) {
	sink := tasks.NewChannelEventSink(50)
	store := tasks.NewMemoryStore()
	mgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads:  2,
		MaxConcurrentOperations: 2,
		Store:                   store,
		EventSink:               sink,
	})
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	done := make(chan struct{})
	id, err := mgr.Submit(tasks.TaskSpec{
		Type:         tasks.TypeApply,
		Title:        "Apply Goldberg",
		GameName:     "Hades II",
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			ctrl.ReportStage("Patching", "Injecting emulator...")
			ctrl.ReportProgress(100, 100, 1, 1, 0, 0, 100)
			close(done)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not complete in time")
	}

	// Give a moment for completion state to persist
	time.Sleep(50 * time.Millisecond)

	task, ok := mgr.Get(id)
	if !ok {
		t.Fatalf("task %s not found", id)
	}
	if task.Status != tasks.StatusCompleted {
		t.Fatalf("expected completed status, got %s", task.Status)
	}
	if task.Progress.Percent != 100.0 {
		t.Fatalf("expected 100 percent, got %f", task.Progress.Percent)
	}
}

func TestTasksManager_PauseAndResume(t *testing.T) {
	sink := tasks.NewChannelEventSink(50)
	mgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads: 1,
		EventSink:              sink,
	})
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	started := make(chan struct{})
	resumed := make(chan struct{})
	completed := make(chan struct{})

	id, err := mgr.Submit(tasks.TaskSpec{
		Type:         tasks.TypeDownload,
		Title:        "Download Game",
		Capabilities: tasks.CapCancellable | tasks.CapPausable | tasks.CapPersist,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			close(started)
			for i := 0; i < 5; i++ {
				if err := ctrl.CheckPauseOrCancel(); err != nil {
					return err
				}
				if i == 2 {
					// Wait for resume signal verification
					select {
					case <-resumed:
					case <-time.After(3 * time.Second):
						return fmt.Errorf("timeout waiting for resume")
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			close(completed)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to submit: %v", err)
	}

	<-started
	// Pause task
	if err := mgr.Pause(id); err != nil {
		t.Fatalf("failed to pause: %v", err)
	}

	task, _ := mgr.Get(id)
	if task.Status != tasks.StatusPaused {
		t.Fatalf("expected status paused, got %s", task.Status)
	}

	// Resume task
	close(resumed)
	if err := mgr.Resume(id); err != nil {
		t.Fatalf("failed to resume: %v", err)
	}

	select {
	case <-completed:
	case <-time.After(3 * time.Second):
		t.Fatal("task did not finish after resume")
	}

	time.Sleep(50 * time.Millisecond)
	task, _ = mgr.Get(id)
	if task.Status != tasks.StatusCompleted {
		t.Fatalf("expected completed status, got %s", task.Status)
	}
}

func TestTasksManager_PauseNonPausableRejection(t *testing.T) {
	mgr := tasks.NewManager(tasks.Options{})
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	id, err := mgr.Submit(tasks.TaskSpec{
		Type:         tasks.TypeApply,
		Title:        "In-Place DRM Strip",
		Capabilities: tasks.CapCancellable, // No CapPausable
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	// Attempt pause should fail with ErrTaskNotPausable
	err = mgr.Pause(id)
	if err != tasks.ErrTaskNotPausable {
		t.Fatalf("expected ErrTaskNotPausable, got %v", err)
	}
}

func TestTasksManager_Cancellation(t *testing.T) {
	mgr := tasks.NewManager(tasks.Options{})
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	started := make(chan struct{})
	done := make(chan struct{})

	id, err := mgr.Submit(tasks.TaskSpec{
		Type:         tasks.TypeDownload,
		Title:        "Large Transfer",
		Capabilities: tasks.CapCancellable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			close(started)
			<-ctx.Done()
			close(done)
			return ctx.Err()
		},
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	<-started
	if err := mgr.Cancel(id); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit on cancellation")
	}

	time.Sleep(50 * time.Millisecond)
	task, _ := mgr.Get(id)
	if task.Status != tasks.StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", task.Status)
	}
}

func TestTasksManager_DualPoolConcurrency(t *testing.T) {
	mgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads:  1,
		MaxConcurrentOperations: 2,
	})
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	var activeDownloads atomic.Int32
	var maxConcurrentDownloads atomic.Int32
	var activeOperations atomic.Int32
	var maxConcurrentOperations atomic.Int32

	dlFunc := func(ctx context.Context, ctrl *tasks.Controller) error {
		cur := activeDownloads.Add(1)
		for {
			old := maxConcurrentDownloads.Load()
			if cur <= old || maxConcurrentDownloads.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		activeDownloads.Add(-1)
		return nil
	}

	opFunc := func(ctx context.Context, ctrl *tasks.Controller) error {
		cur := activeOperations.Add(1)
		for {
			old := maxConcurrentOperations.Load()
			if cur <= old || maxConcurrentOperations.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		activeOperations.Add(-1)
		return nil
	}

	// Submit 3 downloads and 3 operations
	_, _ = mgr.Submit(tasks.TaskSpec{Type: tasks.TypeDownload, Title: "DL 1", Run: dlFunc})
	_, _ = mgr.Submit(tasks.TaskSpec{Type: tasks.TypeDownload, Title: "DL 2", Run: dlFunc})
	_, _ = mgr.Submit(tasks.TaskSpec{Type: tasks.TypeDownload, Title: "DL 3", Run: dlFunc})

	_, _ = mgr.Submit(tasks.TaskSpec{Type: tasks.TypeApply, Title: "Op 1", Run: opFunc})
	_, _ = mgr.Submit(tasks.TaskSpec{Type: tasks.TypeVerify, Title: "Op 2", Run: opFunc})
	_, _ = mgr.Submit(tasks.TaskSpec{Type: tasks.TypeUpdate, Title: "Op 3", Run: opFunc})

	// Wait for all to finish
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if mgr.ActiveCount() == 0 && len(mgr.List(tasks.Filter{Statuses: []tasks.TaskStatus{tasks.StatusQueued}})) == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if maxConcurrentDownloads.Load() > 1 {
		t.Fatalf("expected max 1 concurrent download, got %d", maxConcurrentDownloads.Load())
	}
	if maxConcurrentOperations.Load() > 2 {
		t.Fatalf("expected max 2 concurrent operations, got %d", maxConcurrentOperations.Load())
	}
}

func TestTasksManager_FileStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "tasks.json")

	store1 := tasks.NewFileStore(storePath)
	mgr1 := tasks.NewManager(tasks.Options{
		Store: store1,
	})

	id, err := mgr1.Submit(tasks.TaskSpec{
		ID:           "dl_persist_123",
		Type:         tasks.TypeDownload,
		Title:        "Persisted Download",
		Capabilities: tasks.CapPersist | tasks.CapPausable,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			time.Sleep(1 * time.Second)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	_ = mgr1.Shutdown(context.Background())

	// Start mgr2 with same store file
	store2 := tasks.NewFileStore(storePath)
	mgr2 := tasks.NewManager(tasks.Options{
		Store: store2,
	})
	defer func() { _ = mgr2.Shutdown(context.Background()) }()

	task, ok := mgr2.Get(id)
	if !ok {
		t.Fatalf("persisted task %s not recovered", id)
	}
	if task.Status != tasks.StatusPaused {
		t.Fatalf("expected recovered in-flight task to be paused, got %s", task.Status)
	}
}

func TestTasksManager_ReorderAndClear(t *testing.T) {
	mgr := tasks.NewManager(tasks.Options{
		MaxConcurrentDownloads: 1,
	})
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	blockCh := make(chan struct{})
	defer close(blockCh)

	// Block the 1 download worker
	_, _ = mgr.Submit(tasks.TaskSpec{
		ID:   "blocker",
		Type: tasks.TypeDownload,
		Run: func(ctx context.Context, ctrl *tasks.Controller) error {
			<-blockCh
			return nil
		},
	})

	time.Sleep(20 * time.Millisecond)

	id1, _ := mgr.Submit(tasks.TaskSpec{ID: "task_1", Type: tasks.TypeDownload, Title: "Task 1"})
	id2, _ := mgr.Submit(tasks.TaskSpec{ID: "task_2", Type: tasks.TypeDownload, Title: "Task 2"})
	id3, _ := mgr.Submit(tasks.TaskSpec{ID: "task_3", Type: tasks.TypeDownload, Title: "Task 3"})

	// Reorder 3, 1, 2
	if err := mgr.Reorder([]string{id3, id1, id2}); err != nil {
		t.Fatalf("reorder failed: %v", err)
	}

	list := mgr.List(tasks.Filter{Statuses: []tasks.TaskStatus{tasks.StatusQueued}})
	if len(list) != 3 || list[0].ID != id3 || list[1].ID != id1 || list[2].ID != id2 {
		t.Fatalf("unexpected order after reorder: %v, %v, %v", list[0].ID, list[1].ID, list[2].ID)
	}

	// Cancel one and remove
	_ = mgr.Cancel(id2)
	if err := mgr.Remove(id2); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	remaining := mgr.List(tasks.Filter{Statuses: []tasks.TaskStatus{tasks.StatusQueued}})
	if len(remaining) != 2 {
		t.Fatalf("expected 2 queued tasks after removal, got %d", len(remaining))
	}
}
