package gui

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingleInstanceGuard(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test-gamux.sock")

	var focused atomic.Bool

	dummyGetter := func() context.Context {
		focused.Store(true)
		return nil
	}

	// 1. Acquire primary instance
	guard1, isPrimary1 := AcquireOrActivateSocket(sockPath, dummyGetter)
	if !isPrimary1 || guard1 == nil {
		t.Fatalf("expected guard1 to be primary, got isPrimary=%v", isPrimary1)
	}
	defer guard1.Release()

	// 2. Second instance attempts to acquire
	guard2, isPrimary2 := AcquireOrActivateSocket(sockPath, dummyGetter)
	if isPrimary2 || guard2 != nil {
		t.Fatalf("expected guard2 to NOT be primary, got isPrimary=%v", isPrimary2)
	}

	// Wait briefly for IPC command processing
	time.Sleep(50 * time.Millisecond)
	if !focused.Load() {
		t.Errorf("expected dummyGetter to be triggered by second instance focus command")
	}

	// 3. Release primary
	guard1.Release()

	// 4. Third instance should now succeed as primary
	guard3, isPrimary3 := AcquireOrActivateSocket(sockPath, dummyGetter)
	if !isPrimary3 || guard3 == nil {
		t.Fatalf("expected guard3 to be primary after release, got isPrimary=%v", isPrimary3)
	}
	defer guard3.Release()
}
