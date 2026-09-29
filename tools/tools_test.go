package tools

import (
	"context"
	"testing"

	"github.com/staernid/gamux/config"
)

type mockTool struct {
	key     string
	name    string
	updated bool
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Key() string  { return m.key }
func (m *mockTool) Status(ctx context.Context) (*ToolStatus, error) {
	return &ToolStatus{
		Name:         m.name,
		Key:          m.key,
		InstalledTag: "v1.0.0",
		LatestTag:    "v1.0.0",
		HasUpdate:    false,
	}, nil
}
func (m *mockTool) EnsureAvailable(ctx context.Context) (string, error) {
	return "/bin/" + m.key, nil
}
func (m *mockTool) Update(ctx context.Context) error {
	m.updated = true
	return nil
}

func TestManager_RegisterAndGet(t *testing.T) {
	cfg := config.DefaultConfig()
	mgr := NewManager(cfg)

	gbe, err := mgr.GetTool("gbe")
	if err != nil || gbe == nil {
		t.Fatalf("expected gbe tool registered by default, got err: %v", err)
	}

	steamless, err := mgr.GetTool("steamless")
	if err != nil || steamless == nil {
		t.Fatalf("expected steamless tool registered by default, got err: %v", err)
	}

	umu, err := mgr.GetTool("umu")
	if err != nil || umu == nil {
		t.Fatalf("expected umu tool registered by default, got err: %v", err)
	}

	protonGE, err := mgr.GetTool("proton-ge")
	if err != nil || protonGE == nil {
		t.Fatalf("expected proton-ge tool registered by default, got err: %v", err)
	}

	protonCachyOS, err := mgr.GetTool("proton-cachyos")
	if err != nil || protonCachyOS == nil {
		t.Fatalf("expected proton-cachyos tool registered by default, got err: %v", err)
	}

	mock := &mockTool{key: "custom", name: "Custom Tool"}
	mgr.Register(mock)

	got, err := mgr.GetTool("custom")
	if err != nil || got.Name() != "Custom Tool" {
		t.Fatalf("expected Custom Tool, got %v, err: %v", got, err)
	}

	// Case-insensitivity check
	gotCase, err := mgr.GetTool("CUSTOM")
	if err != nil || gotCase.Key() != "custom" {
		t.Fatalf("expected case-insensitive lookup, got %v, err: %v", gotCase, err)
	}

	_, err = mgr.GetTool("nonexistent")
	if err == nil {
		t.Fatalf("expected error for nonexistent tool, got nil")
	}
}

func TestManager_GetToolsStatusAndUpdates(t *testing.T) {
	mgr := &Manager{
		tools: make(map[string]Tool),
	}
	m1 := &mockTool{key: "tool1", name: "Tool 1"}
	m2 := &mockTool{key: "tool2", name: "Tool 2"}
	mgr.Register(m1)
	mgr.Register(m2)

	ctx := context.Background()
	statuses, err := mgr.GetToolsStatus(ctx)
	if err != nil {
		t.Fatalf("GetToolsStatus failed: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}

	if err := mgr.Update(ctx, "tool1"); err != nil {
		t.Fatalf("update tool1 failed: %v", err)
	}
	if !m1.updated {
		t.Fatalf("expected tool1 to be marked updated")
	}

	if err := mgr.UpdateAll(ctx); err != nil {
		t.Fatalf("UpdateAll failed: %v", err)
	}
	if !m2.updated {
		t.Fatalf("expected tool2 to be marked updated")
	}
}
