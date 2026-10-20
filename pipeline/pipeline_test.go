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
	"github.com/staernid/gamux/tasks"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restoreEnv := testutil.SetupTestEnvironment()
	restoreLogs := testutil.SilenceLogging()
	code := m.Run()
	restoreLogs()
	restoreEnv()
	os.Exit(code)
}

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
			DryRun:        true,
			NormalizeDir:  true,
			OperatingMode: domain.ModeLoader,
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
			DryRun:        true,
			OperatingMode: domain.ModeLoader,
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

func TestPipeline_NewRollback_Recipe(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "Portal")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "hl2.exe"), []byte("MZ"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api.dll.ORIGINAL"), []byte("ORIGINAL_DLL"), 0644)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api.dll"), []byte("GBE_DLL"), 0644)

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"400\"\n\t\"name\"\t\"Portal\"\n}"
	_ = os.WriteFile(filepath.Join(gameDir, "appmanifest_400.acf"), []byte(acfContent), 0644)

	pipe := NewRollback()
	pctx := &Context{
		Options: Options{
			Path:   gameDir,
			DryRun: false,
		},
		Game:   domain.NewGame("", gameDir, "", ""),
		Config: config.DefaultConfig(),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err != nil {
		t.Fatalf("NewRollback pipeline execution failed: %v", err)
	}

	if pctx.Results.RestoredCount < 1 {
		t.Errorf("expected at least 1 restored file, got %d", pctx.Results.RestoredCount)
	}

	// Verify original dll is restored
	data, _ := os.ReadFile(filepath.Join(gameDir, "steam_api.dll"))
	if string(data) != "ORIGINAL_DLL" {
		t.Errorf("expected steam_api.dll to contain ORIGINAL_DLL, got %s", string(data))
	}
}

func TestPipeline_NewNormalize_Recipe(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir := filepath.Join(tmpDir, "portal_download")
	_ = os.MkdirAll(originalDir, 0755)
	_ = os.WriteFile(filepath.Join(originalDir, "hl2.exe"), []byte("MZ"), 0755)

	acfContent := "\"AppState\"\n{\n\t\"appid\"\t\"400\"\n\t\"name\"\t\"Portal\"\n}"
	_ = os.WriteFile(filepath.Join(originalDir, "appmanifest_400.acf"), []byte(acfContent), 0644)

	pipe := NewNormalize()
	pctx := &Context{
		Options: Options{
			Path:         originalDir,
			NormalizeDir: true,
			DryRun:       false,
		},
		Game:   domain.NewGame("", originalDir, "", ""),
		Config: config.DefaultConfig(),
	}

	err := pipe.Execute(context.Background(), pctx)
	if err != nil {
		t.Fatalf("NewNormalize pipeline execution failed: %v", err)
	}

	if !pctx.Results.DirectoryNormalized {
		t.Error("expected DirectoryNormalized to be true")
	}
	expectedDir := filepath.Join(tmpDir, "Portal")
	if pctx.Game.GameDir != expectedDir {
		t.Errorf("expected normalized dir %s, got %s", expectedDir, pctx.Game.GameDir)
	}
}

func TestPipeline_SteamlessStage_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "Game")
	_ = os.MkdirAll(gameDir, 0755)

	exePath := filepath.Join(gameDir, "game.exe")
	backupPath := filepath.Join(gameDir, "game.exe.20260101-120000.ORIGINAL")

	_ = os.WriteFile(exePath, []byte("MODIFIED_EXE"), 0755)
	_ = os.WriteFile(backupPath, []byte("ORIGINAL_EXE"), 0755)

	stage := &SteamlessStage{}
	pctx := &Context{
		Options: Options{
			Path:   gameDir,
			DryRun: false,
		},
		Game: domain.NewGame("Game", gameDir, "100", "win64"),
		Results: Results{
			SteamlessUnpacked: true,
		},
	}

	err := stage.Rollback(context.Background(), pctx)
	if err != nil {
		t.Fatalf("SteamlessStage.Rollback failed: %v", err)
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("failed to read restored exe: %v", err)
	}
	if string(data) != "ORIGINAL_EXE" {
		t.Errorf("expected restored content ORIGINAL_EXE, got %s", string(data))
	}
	if pctx.Results.SteamlessUnpacked {
		t.Error("expected SteamlessUnpacked to be reset to false after rollback")
	}
}

func TestPipeline_Operation_ExecuteAndProgress(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "TestGame")
	_ = os.MkdirAll(gameDir, 0755)

	s1 := &mockStage{name: "Stage 1"}
	s2 := &mockStage{name: "Stage 2"}
	pipe := New(s1, s2)

	pctx := &Context{
		Game: domain.NewGame("TestGame", gameDir, "123", "linux"),
	}

	op := NewOperation(tasks.TaskSpec{
		Type:     tasks.TypeApply,
		Title:    "Test Operation",
		GameName: "TestGame",
		GameDir:  gameDir,
	}, pipe, pctx)

	var reportedPct float64
	var completedStage string
	ctrl := tasks.NewController(
		context.Background(),
		"test-task-1",
		tasks.CapCancellable,
		nil,
		func(currBytes, totalBytes int64, currItems, totalItems int, speed float64, eta int, pct float64) {
			reportedPct = pct
		},
		func(stage, details string) {
			completedStage = stage
		},
		nil,
	)

	err := op.Execute(context.Background(), ctrl)
	if err != nil {
		t.Fatalf("Operation execution failed: %v", err)
	}

	if completedStage != "Completed" {
		t.Errorf("expected completed stage 'Completed', got %q", completedStage)
	}
	if reportedPct != 100 {
		t.Errorf("expected final reported percent 100, got %v", reportedPct)
	}
	if ctrl.ResultGame() == nil || ctrl.ResultGame().Name != "TestGame" {
		t.Errorf("expected result game to be set on controller")
	}
}

func TestPipeline_Operation_DetachedRollbackOnCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "CancelledGame")
	_ = os.MkdirAll(gameDir, 0755)

	ctx, cancel := context.WithCancel(context.Background())

	s1 := &mockRollbackStage{mockStage: mockStage{name: "Stage 1"}}
	// s2 cancels context during execution and returns context.Canceled
	s2 := &mockCancelStage{name: "Stage 2", cancelFunc: cancel}

	pipe := New(s1, s2)
	pctx := &Context{
		Game: domain.NewGame("CancelledGame", gameDir, "123", "linux"),
	}

	op := NewSetupOperation("CancelledGame", gameDir, pctx)
	op.Pipeline = pipe

	ctrl := tasks.NewController(ctx, "test-cancel", tasks.CapCancellable, nil, nil, nil, nil)
	err := op.Execute(ctx, ctrl)
	if err == nil {
		t.Fatal("expected operation to return context error on cancelled context")
	}

	// s1 must have executed and successfully rolled back using detached context
	if !s1.executed {
		t.Error("expected Stage 1 to have executed")
	}
	if !s1.rolledBack {
		t.Error("expected Stage 1 to have rolled back cleanly using detached context")
	}
}

type mockCancelStage struct {
	name       string
	cancelFunc func()
}

func (m *mockCancelStage) Name() string { return m.name }
func (m *mockCancelStage) Execute(ctx context.Context, pctx *Context) error {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	return ctx.Err()
}
