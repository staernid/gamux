package events

import (
	"sync"
	"testing"
	"time"
)

func TestBus_PublishAndSubscribe(t *testing.T) {
	bus := NewBus()

	ch, unsub := bus.Subscribe(EventTaskStarted, EventTaskCompleted)
	defer unsub()

	evStarted := NewTaskStarted("task_1", "download", "Hades II", "/games/Hades2", "1145360")
	bus.Publish(evStarted)

	select {
	case received := <-ch:
		if received.Type != EventTaskStarted {
			t.Errorf("expected EventTaskStarted, got %s", received.Type)
		}
		if received.TaskID != "task_1" {
			t.Errorf("expected TaskID task_1, got %s", received.TaskID)
		}
		if received.GameName != "Hades II" {
			t.Errorf("expected GameName Hades II, got %s", received.GameName)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Ignored event (filtered out)
	evIgnored := Event{Type: EventGameLaunched}
	bus.Publish(evIgnored)

	select {
	case unexpected := <-ch:
		t.Fatalf("received unexpected filtered event: %v", unexpected)
	case <-time.After(50 * time.Millisecond):
		// Expected to not receive
	}

	// Completed event
	evCompleted := NewTaskCompleted("task_1", "Downloaded 45 files")
	bus.Publish(evCompleted)

	select {
	case received := <-ch:
		if received.Type != EventTaskCompleted {
			t.Errorf("expected EventTaskCompleted, got %s", received.Type)
		}
		if received.Progress.Percent != 100.0 {
			t.Errorf("expected 100%% percent, got %f", received.Progress.Percent)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for completed event")
	}
}

func TestBus_SubscribeFuncAndUnsubscribe(t *testing.T) {
	bus := NewBus()

	var mu sync.Mutex
	var received []Event

	unsub := bus.SubscribeFunc(func(ev Event) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, ev)
	}, EventTaskProgress)

	bus.Publish(NewTaskProgress("task_x", "stage1", "details", TaskProgress{Percent: 50}))

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Progress.Percent != 50 {
		t.Errorf("expected 50%%, got %f", received[0].Progress.Percent)
	}
	mu.Unlock()

	// Unsubscribe
	unsub()

	bus.Publish(NewTaskProgress("task_x", "stage2", "details", TaskProgress{Percent: 75}))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(received) != 1 {
		t.Errorf("expected still 1 event after unsubscribe, got %d", len(received))
	}
	mu.Unlock()
}

func TestBus_ConcurrentPublish(t *testing.T) {
	bus := NewBus()
	ch, unsub := bus.Subscribe()
	defer unsub()

	const numPublishers = 20
	const eventsPerPublisher = 50

	var wg sync.WaitGroup
	wg.Add(numPublishers)

	for i := 0; i < numPublishers; i++ {
		go func(pubID int) {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				bus.Publish(NewTaskProgress("task_concurrent", "hashing", "chunk", TaskProgress{Percent: float64(j)}))
			}
		}(i)
	}

	wg.Wait()

	// Ensure channel drained without deadlocks or panics
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			return
		}
	}
}

func TestBus_TypedLifecycleConstructors(t *testing.T) {
	evLaunched := NewGameLaunched("/games/Cyberpunk2077", "Cyberpunk 2077", "1091500", 12345)
	if evLaunched.Type != EventGameLaunched {
		t.Errorf("expected EventGameLaunched, got %s", evLaunched.Type)
	}
	if evLaunched.Metadata["pid"] != "12345" {
		t.Errorf("expected pid metadata '12345', got %s", evLaunched.Metadata["pid"])
	}
	if evLaunched.GameDir != "/games/Cyberpunk2077" {
		t.Errorf("expected GameDir '/games/Cyberpunk2077', got %s", evLaunched.GameDir)
	}

	evExited := NewGameExited("/games/Cyberpunk2077", "Cyberpunk 2077", "1091500", nil)
	if evExited.Type != EventGameExited {
		t.Errorf("expected EventGameExited, got %s", evExited.Type)
	}
	if evExited.Error != "" {
		t.Errorf("expected empty error, got %s", evExited.Error)
	}

	evStep := NewStepProgress(2, 5, "Hashing Files", "Processing chunk 2")
	if evStep.Type != EventTaskProgress {
		t.Errorf("expected EventTaskProgress, got %s", evStep.Type)
	}
	if evStep.Progress.CurrentItems != 2 || evStep.Progress.TotalItems != 5 {
		t.Errorf("expected items 2/5, got %d/%d", evStep.Progress.CurrentItems, evStep.Progress.TotalItems)
	}
	if evStep.Progress.Percent != 40.0 {
		t.Errorf("expected percent 40.0, got %f", evStep.Progress.Percent)
	}
}
