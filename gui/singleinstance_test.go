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
	var receivedTarget atomic.Pointer[string]

	dummyGetter := func() context.Context {
		focused.Store(true)
		return nil
	}
	targetHandler := func(target string) {
		receivedTarget.Store(&target)
	}

	// 1. Acquire primary instance
	guard1, isPrimary1 := AcquireOrActivateSocket(sockPath, nil, dummyGetter, targetHandler)
	if !isPrimary1 || guard1 == nil {
		t.Fatalf("expected guard1 to be primary, got isPrimary=%v", isPrimary1)
	}
	defer guard1.Release()

	// 2. Second instance attempts to acquire with an open target URL
	guard2, isPrimary2 := AcquireOrActivateSocket(sockPath, []string{"gamux://app/730"}, dummyGetter, targetHandler)
	if isPrimary2 || guard2 != nil {
		t.Fatalf("expected guard2 to NOT be primary, got isPrimary=%v", isPrimary2)
	}

	// Wait briefly for IPC command processing
	time.Sleep(50 * time.Millisecond)
	if !focused.Load() {
		t.Errorf("expected dummyGetter to be triggered by second instance focus command")
	}
	if p := receivedTarget.Load(); p == nil || *p != "gamux://app/730" {
		t.Errorf("expected receivedTarget to be 'gamux://app/730', got %v", p)
	}

	// 3. Release primary
	guard1.Release()

	// 4. Third instance should now succeed as primary
	guard3, isPrimary3 := AcquireOrActivateSocket(sockPath, nil, dummyGetter, targetHandler)
	if !isPrimary3 || guard3 == nil {
		t.Fatalf("expected guard3 to be primary after release, got isPrimary=%v", isPrimary3)
	}
	defer guard3.Release()
}

func TestExtractOpenTarget(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{}, want: ""},
		{args: []string{"--debug", "-v"}, want: ""},
		{args: []string{"gamux://app/730"}, want: "gamux://app/730"},
		{args: []string{"--flag", "steam://store/1245620"}, want: "steam://store/1245620"},
		{args: []string{"https://store.steampowered.com/app/570"}, want: "https://store.steampowered.com/app/570"},
		{args: []string{"730"}, want: "730"},
	}

	for _, tc := range tests {
		got := ExtractOpenTarget(tc.args)
		if got != tc.want {
			t.Errorf("ExtractOpenTarget(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}
