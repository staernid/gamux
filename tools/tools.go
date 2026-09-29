package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/staernid/gamux/config"
	"golang.org/x/sync/errgroup"
)

// ToolStatus holds version, release metadata, and changelog notes for external post-processing tools.
type ToolStatus struct {
	Name          string `json:"name"`
	Key           string `json:"key"` // "gbe" | "steamless"
	InstalledPath string `json:"installed_path"`
	InstalledTag  string `json:"installed_tag"`
	LatestTag     string `json:"latest_tag"`
	ReleaseName   string `json:"release_name"`
	ReleaseNotes  string `json:"release_notes"`
	ReleaseURL    string `json:"release_url"`
	PublishedAt   string `json:"published_at"`
	HasUpdate     bool   `json:"has_update"`
	IsInstalled   bool   `json:"is_installed"`
}

// Tool defines the unified lifecycle interface for an external helper tool.
type Tool interface {
	Name() string
	Key() string
	Status(ctx context.Context) (*ToolStatus, error)
	EnsureAvailable(ctx context.Context) (string, error)
	Update(ctx context.Context) error
}

// Manager aggregates and coordinates post-processing tools.
type Manager struct {
	Config *config.Config
	tools  map[string]Tool
	order  []string
}

// NewManager creates a tool manager with default GBE and Steamless tool adapters registered.
func NewManager(cfg *config.Config) *Manager {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	m := &Manager{
		Config: cfg,
		tools:  make(map[string]Tool),
	}
	m.Register(NewGBETool(cfg))
	m.Register(NewSteamlessTool(cfg))
	m.Register(NewUMUTool(cfg))
	m.Register(NewProtonTool(cfg, FlavorGE))
	m.Register(NewProtonTool(cfg, FlavorCachyOS))
	return m
}

// Register registers a tool adapter with the manager.
func (m *Manager) Register(t Tool) {
	key := strings.ToLower(t.Key())
	if _, exists := m.tools[key]; !exists {
		m.order = append(m.order, key)
	}
	m.tools[key] = t
}

// GetTool retrieves a tool adapter by key.
func (m *Manager) GetTool(key string) (Tool, error) {
	k := strings.ToLower(strings.TrimSpace(key))
	t, ok := m.tools[k]
	if !ok {
		return nil, fmt.Errorf("unknown tool key: %q", key)
	}
	return t, nil
}

// GetToolsStatus returns current installation and release update statuses across all registered tools.
func (m *Manager) GetToolsStatus(ctx context.Context) ([]ToolStatus, error) {
	results := make([]ToolStatus, len(m.order))
	g, gCtx := errgroup.WithContext(ctx)

	for i, key := range m.order {
		idx := i
		t := m.tools[key]
		g.Go(func() error {
			st, err := t.Status(gCtx)
			if err != nil || st == nil {
				results[idx] = ToolStatus{
					Name:          t.Name(),
					Key:           t.Key(),
					InstalledPath: "",
					InstalledTag:  "Unknown",
					ReleaseNotes:  "Could not retrieve release status from GitHub.",
				}
				return nil
			}
			results[idx] = *st
			return nil
		})
	}

	_ = g.Wait()
	return results, nil
}

// Update updates a specific tool identified by key.
func (m *Manager) Update(ctx context.Context, key string) error {
	t, err := m.GetTool(key)
	if err != nil {
		return err
	}
	return t.Update(ctx)
}

// UpdateAll updates all registered tools concurrently.
func (m *Manager) UpdateAll(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	for _, key := range m.order {
		t := m.tools[key]
		g.Go(func() error {
			return t.Update(gCtx)
		})
	}
	return g.Wait()
}
