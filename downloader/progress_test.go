package downloader

import "testing"

func TestProgressStats(t *testing.T) {
	tests := []struct {
		name               string
		curr, total, net   int64
		elapsed            float64
		wantPct, wantSpeed float64
		wantETA            int
	}{
		{"too early for speed", 50, 100, 50, 0.4, 50, 0, 0},
		{"speed and eta", 50, 100, 100, 2, 50, 50, 1},
		{"percent capped at 100", 150, 100, 150, 1, 100, 150, 0},
		{"unknown total", 10, 0, 10, 1, 0, 10, 0},
		{"no network bytes", 10, 100, 0, 2, 10, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct, speed, eta := progressStats(tt.curr, tt.total, tt.net, tt.elapsed)
			if pct != tt.wantPct || speed != tt.wantSpeed || eta != tt.wantETA {
				t.Errorf("progressStats = (%v, %v, %d), want (%v, %v, %d)", pct, speed, eta, tt.wantPct, tt.wantSpeed, tt.wantETA)
			}
		})
	}
}
