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
	)

	cfg := config.DefaultConfig()
	pctx := &Context{
		Options: Options{
			DryRun:       true,
			NormalizeDir: true,
			ApplyGBE:     true,
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
	if !pctx.Results.DirectoryNormalized {
		t.Error("expected DirectoryNormalized to be true")
	}
	if !pctx.Results.Patched {
		t.Error("expected Patched to be true")
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
	)

	cfg := config.DefaultConfig()
	pctx := &Context{
		Options: Options{
			DryRun:   true,
			ApplyGBE: true,
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

func TestPipeline_NormalizeStage_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := filepath.Join(tmpDir, "My Game Folder (Unclean)")
	_ = os.MkdirAll(originalDir, 0755)
	_ = os.WriteFile(filepath.Join(originalDir, "game.exe"), []byte("exe"), 0755)

	game := domain.NewGame("My Game", originalDir, "12345", "win64")
	normStage := &NormalizeStage{}
	failStage := &mockStage{name: "Failing Downstream", fail: true}

	pipe := New(normStage, failStage)
	pctx := &Context{
		Options: Options{
			NormalizeDir: true,
			DryRun:       false,
		},
		Game:   game,
		Config: config.DefaultConfig(),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err == nil {
		t.Fatal("expected pipeline to fail")
	}

	// Verify directory was rolled back to original name
	if _, err := os.Stat(originalDir); os.IsNotExist(err) {
		t.Errorf("expected original directory to be restored after rollback: %s", originalDir)
	}
	if pctx.Game.GameDir != originalDir {
		t.Errorf("expected pctx.Game.GameDir to be restored to %s, got %s", originalDir, pctx.Game.GameDir)
	}
	if pctx.Results.DirectoryNormalized {
		t.Error("expected DirectoryNormalized to be false after rollback")
	}
}

func TestPipeline_ManifestStage_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "TestGame")
	_ = os.MkdirAll(gameDir, 0755)

	game := domain.NewGame("TestGame", gameDir, "99999", "linux")
	manifestStage := &ManifestStage{}
	failStage := &mockStage{name: "Failing Downstream", fail: true}

	pipe := New(manifestStage, failStage)
	pctx := &Context{
		Options: Options{
			DryRun: false,
		},
		Game:   game,
		Config: config.DefaultConfig(),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err == nil {
		t.Fatal("expected pipeline to fail")
	}

	// Synthesized ACF manifest should be removed after rollback
	targetACF := filepath.Join(gameDir, "[Manifests]", "appmanifest_99999.acf")
	if _, err := os.Stat(targetACF); !os.IsNotExist(err) {
		t.Errorf("expected synthesized ACF to be removed during rollback: %s", targetACF)
	}
	if pctx.Results.ACFSynthesized {
		t.Error("expected ACFSynthesized to be false after rollback")
	}
}

func TestPipeline_NewDefault_DynamicExecution(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "GameFolder")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "game.exe"), []byte("MZ"), 0755)

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"400\"\n\t\"name\"\t\"Portal\"\n}"
	_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_400.acf"), []byte(acfContent), 0644)

	pipe := NewDefault()
	pctx := &Context{
		Options: Options{
			Path:          gameDir,
			OperatingMode: domain.ModeLoader,
			DryRun:        true,
			NormalizeDir:  true,
		},
		Game:   domain.NewGame("", gameDir, "", ""),
		Config: config.DefaultConfig(),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err != nil {
		t.Fatalf("NewDefault pipeline execution failed: %v", err)
	}

	if pctx.Game.Name != "Portal" {
		t.Errorf("expected detected name 'Portal', got %s", pctx.Game.Name)
	}
	if pctx.Game.AppID != "400" {
		t.Errorf("expected detected appID '400', got %s", pctx.Game.AppID)
	}
	if !pctx.Results.Patched {
		t.Error("expected Results.Patched to be true for Steam loader game")
	}
}
