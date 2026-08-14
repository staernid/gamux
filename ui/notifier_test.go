package ui

import (
	"context"
	"testing"
)

func TestNotifyIssue(t *testing.T) {
	ctx := context.Background()
	err := NotifyIssue(ctx, "Test Game", "This is a test notification message.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
