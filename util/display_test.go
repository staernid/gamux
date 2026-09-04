package util

import (
	"testing"
)

func TestDetectPrimaryDisplay(t *testing.T) {
	info := DetectPrimaryDisplay()
	if info.Width <= 0 || info.Height <= 0 {
		t.Fatalf("expected positive width and height, got %dx%d", info.Width, info.Height)
	}
	if info.RefreshRate <= 0 {
		t.Fatalf("expected positive refresh rate, got %d", info.RefreshRate)
	}
}
