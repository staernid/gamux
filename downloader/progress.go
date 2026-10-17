package downloader

import (
	"sync"
	"sync/atomic"
	"time"
)

// progressStats derives completion percentage, network speed (bytes/s) and ETA
// (seconds) from byte counters. Speed and ETA stay zero until enough time has
// elapsed to be meaningful.
func progressStats(curr, total, netBytes int64, elapsed float64) (pct, speed float64, eta int) {
	if elapsed > 0.5 {
		speed = float64(netBytes) / elapsed
		if speed > 0 && total > curr {
			eta = int(float64(total-curr) / speed)
		}
	}
	if total > 0 {
		pct = float64(curr) / float64(total) * 100.0
		if pct > 100 {
			pct = 100
		}
	}
	return pct, speed, eta
}

// progressTracker holds the shared counters of a download and throttles
// telemetry callbacks. All methods are safe for concurrent use.
type progressTracker struct {
	onProgress func(ProgressUpdate)
	totalBytes int64
	totalItems int
	start      time.Time

	downloadedBytes atomic.Int64
	networkBytes    atomic.Int64
	processedCount  atomic.Int32
	updatedCount    atomic.Int32

	mu   sync.Mutex
	last time.Time
}

func newProgressTracker(onProgress func(ProgressUpdate), totalBytes int64, totalItems int) *progressTracker {
	return &progressTracker{
		onProgress: onProgress,
		totalBytes: totalBytes,
		totalItems: totalItems,
		start:      time.Now(),
	}
}

// report emits a progress update, rate-limited to one per 100ms.
func (p *progressTracker) report(item string) { p.emit(item, false) }

// flush emits a progress update unconditionally.
func (p *progressTracker) flush(item string) { p.emit(item, true) }

func (p *progressTracker) emit(item string, force bool) {
	if p.onProgress == nil {
		return
	}
	p.mu.Lock()
	now := time.Now()
	if !force && now.Sub(p.last) < 100*time.Millisecond {
		p.mu.Unlock()
		return
	}
	p.last = now
	p.mu.Unlock()

	curr := p.downloadedBytes.Load()
	pct, speed, eta := progressStats(curr, p.totalBytes, p.networkBytes.Load(), time.Since(p.start).Seconds())
	p.onProgress(ProgressUpdate{
		DownloadedBytes: curr,
		TotalBytes:      p.totalBytes,
		CurrentItems:    int(p.processedCount.Load()),
		TotalItems:      p.totalItems,
		SpeedBps:        speed,
		ETASeconds:      eta,
		Percent:         pct,
		Item:            item,
	})
}
