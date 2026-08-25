package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
)

type mockStage struct {
	name     string
	executed bool
	fail     bool
}

func (m *mockStage) Name() string { return m.name }
func (m *mockStage) Execute(ctx context.Context, pctx *Context) error {
	m.executed = true
	if m.fail {
		return errors.New("stage failed")
	}
	return nil
}

type mockRollbackStage struct {
	mockStage
	rolledBack bool
}

func (m *mockRollbackStage) Rollback(ctx context.Context, pctx *Context) error {
	m.rolledBack = true
	return nil
}

func TestPipeline_SequentialExecution(t *testing.T) {
	s1 := &mockStage{name: "Stage 1"}
	s2 := &mockStage{name: "Stage 2"}

	pipe := New(s1, s2)
	pctx := &Context{
		Game: domain.NewGame("TestGame", "/tmp/test", "123", "linux"),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	if !s1.executed || !s2.executed {
		t.Errorf("expected both stages to be executed: s1=%v s2=%v", s1.executed, s2.executed)
	}
}

func TestPipeline_RollbackOnError(t *testing.T) {
	s1 := &mockRollbackStage{mockStage: mockStage{name: "Stage 1"}}
	s2 := &mockStage{name: "Stage 2", fail: true}

	pipe := New(s1, s2)
	pctx := &Context{
		Game: domain.NewGame("TestGame", "/tmp/test", "123", "linux"),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err == nil {
		t.Fatal("expected pipeline to fail")
	}

	if !s1.executed {
		t.Error("expected Stage 1 to have executed")
	}
	if !s1.rolledBack {
		t.Error("expected Stage 1 to have been rolled back")
	}
}

func TestPipeline_ConcreteStages_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "Cyberpunk.2077.v2.12-P2P")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "Cyberpunk2077.exe"), []byte("exe"), 0755)

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"1091500\"\n\t\"name\"\t\"Cyberpunk 2077\"\n}"
	_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_1091500.acf"), []byte(acfContent), 0644)

	bus := events.NewBus()
	game := domain.NewGame("", gameDir, "", "")

	pipe := New(
		&ScanStage{},
		&ManifestStage{},
		&NormalizeStage{},
		&GBEStage{},
		&LutrisStage{},
	)

	cfg := config.DefaultConfig()
	pctx := &Context{
		Options: Options{
			DryRun:       true,
			NormalizeDir: true,
			ApplyGBE:     true,
			AddLutris:    true,
		},
		Game:     game,
		Config:   cfg,
		EventBus: bus,
	}

	err := pipe.Execute(context.Background(), pctx)
	if err != nil {
		t.Fatalf("Pipeline dry-run failed: %v", err)
	}

	if pctx.Game.Name != "Cyberpunk 2077" {
		t.Errorf("expected detected name 'Cyberpunk 2077', got %s", pctx.Game.Name)
	}
	if !pctx.DirectoryNormalized {
		t.Error("expected DirectoryNormalized to be true")
	}
	if !pctx.Patched {
		t.Error("expected Patched to be true")
	}
	if !pctx.LutrisAdded {
		t.Error("expected LutrisAdded to be true")
	}
}

func TestPipeline_NonSteamGame_CleanExecution(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "Factorio_Standalone")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "factorio.exe"), []byte("exe"), 0755)

	game := domain.NewGame("Factorio", gameDir, "", "windows")
	game.Store = "Standalone"

	pipe := New(
		&ScanStage{},
		&ManifestStage{},
		&NormalizeStage{},
		&SteamlessStage{},
		&GBEStage{},
		&LutrisStage{},
	)

	cfg := config.DefaultConfig()
	pctx := &Context{
		Options: Options{
			DryRun:    true,
			ApplyGBE:  true,
			AddLutris: true,
		},
		Game:   game,
		Config: cfg,
	}

	err := pipe.Execute(context.Background(), pctx)
	if err != nil {
		t.Fatalf("Pipeline execution for non-Steam game failed: %v", err)
	}

	// Non-steam game should not have created unnecessary [Manifests] folder
	manifestsDir := filepath.Join(gameDir, "[Manifests]")
	if _, statErr := os.Stat(manifestsDir); !os.IsNotExist(statErr) {
		t.Errorf("expected [Manifests] directory not to be created for non-Steam game")
	}
}
