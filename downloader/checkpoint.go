package downloader

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/staernid/gamux/util"
)

// Checkpoint tracks completed chunk IDs for resume support.
type Checkpoint struct {
	AppID           uint32          `json:"app_id"`
	DepotID         uint32          `json:"depot_id"`
	CompletedChunks map[string]bool `json:"completed_chunks"`
	lastSaved       time.Time
	dirty           bool
	mu              sync.Mutex
}

func loadCheckpoint(path string) *Checkpoint {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Checkpoint{CompletedChunks: make(map[string]bool)}
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil || cp.CompletedChunks == nil {
		return &Checkpoint{CompletedChunks: make(map[string]bool)}
	}
	return &cp
}

func (cp *Checkpoint) isCompleted(key string) bool {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	return cp.CompletedChunks[key]
}

func (cp *Checkpoint) markCompleted(path string, key string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.CompletedChunks[key] {
		return
	}
	cp.CompletedChunks[key] = true
	cp.dirty = true

	// Throttle disk writes to at most once per second
	if time.Since(cp.lastSaved) >= 1*time.Second {
		cp.saveLocked(path)
	}
}

func (cp *Checkpoint) Flush(path string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.dirty {
		cp.saveLocked(path)
	}
}

func (cp *Checkpoint) saveLocked(path string) {
	data, err := json.Marshal(cp)
	if err == nil {
		if err := util.WriteFileAtomic(path, data, 0644); err == nil {
			cp.lastSaved = time.Now()
			cp.dirty = false
		}
	}
}
