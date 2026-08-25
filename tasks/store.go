package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store defines the persistence seam for saving and restoring task records.
type Store interface {
	Save(tasks []*Task) error
	Load() ([]*Task, error)
}

// FileStore persists durable tasks to a JSON file with atomic write semantics.
type FileStore struct {
	filePath string
	mu       sync.Mutex
}

// NewFileStore creates a new FileStore at filePath.
func NewFileStore(filePath string) *FileStore {
	return &FileStore{
		filePath: filePath,
	}
}

// Save atomically writes persistent tasks to disk.
func (s *FileStore) Save(allTasks []*Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var persistent []*Task
	for _, t := range allTasks {
		if t != nil && t.Capabilities.Has(CapPersist) {
			persistent = append(persistent, t)
		}
	}

	data, err := json.MarshalIndent(persistent, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp tasks file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("rename temp tasks file: %w", err)
	}

	return nil
}

// Load reads persisted tasks from disk, restoring in-flight tasks to StatusPaused.
func (s *FileStore) Load() ([]*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return []*Task{}, nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("read tasks file: %w", err)
	}

	if len(data) == 0 {
		return []*Task{}, nil
	}

	var persisted []*Task
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}

	for _, t := range persisted {
		// Restore in-flight tasks as Paused so user can resume at will
		if t.Status == StatusRunning || t.Status == StatusQueued {
			t.Status = StatusPaused
			t.Stage = "Paused"
			t.Details = "Recovered after restart"
		}
	}

	return persisted, nil
}

// MemoryStore is an in-memory Store adapter for unit tests and ephemeral environments.
type MemoryStore struct {
	mu    sync.Mutex
	tasks []*Task
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make([]*Task, 0),
	}
}

// Save stores tasks in memory.
func (m *MemoryStore) Save(allTasks []*Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = make([]*Task, 0, len(allTasks))
	for _, t := range allTasks {
		if t != nil && t.Capabilities.Has(CapPersist) {
			// Clone to avoid mutations
			cloned := *t
			m.tasks = append(m.tasks, &cloned)
		}
	}
	return nil
}

// Load returns stored in-memory tasks.
func (m *MemoryStore) Load() ([]*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := make([]*Task, len(m.tasks))
	for i, t := range m.tasks {
		cloned := *t
		res[i] = &cloned
	}
	return res, nil
}
