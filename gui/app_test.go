package gui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restoreEnv := testutil.SetupTestEnvironment()
	defer restoreEnv()
	restore := testutil.SilenceLogging()
	
	// Stub HTTP requests to avoid live network latency and external rate limits
	mockClient := testutil.NewMockHTTPClient(map[string]testutil.MockResponse{
		"GetNewsForApp": {
			StatusCode: 200,
			Body:       `{"appnews":{"appid":12345,"newsitems":[{"title":"Patch Note","url":"https://steam.com","contents":"Fixed bug","feedlabel":"Community","date":1672531199}]}}`,
		},
		"appdetails": {
			StatusCode: 200,
			Body:       `{"12345":{"success":true,"data":{"name":"Test Game","type":"game"}}}`,
		},
		"storesearch": {
			StatusCode: 200,
			Body:       `{"total":1,"items":[{"id":12345,"name":"Test Game"}]}`,
		},
		"releases/latest": {
			StatusCode: 200,
			Body:       `{"tag_name":"v1.0.0","name":"v1.0.0","body":"Release notes","assets":[]}`,
		},
	})
	steam.HTTPClient = mockClient
	github.HTTPClient = mockClient
	manifest.HTTPClient = mockClient

	code := m.Run()
	restore()
	restoreEnv()
	os.Exit(code)
}

func TestApp_GetAndSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "gamux", "config.json")

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := config.DefaultConfig()
	cfg.GbeDir = "/custom/gbe"

	app := NewApp(cfg)

	gotCfg, err := app.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}
	if gotCfg.GbeDir != "/custom/gbe" {
		t.Errorf("GetConfig() got GbeDir %q, want %q", gotCfg.GbeDir, "/custom/gbe")
	}

	// Test saving nil config
	if err := app.SaveConfig(nil); err == nil {
		t.Error("SaveConfig(nil) expected error, got nil")
	}

	// Test saving modified config
	cfg.SteamWebAPIKey = "test-api-key"
	cfg.MangoHud = true
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatalf("app.SaveConfig() unexpected error: %v", err)
	}

	if app.Config.SteamWebAPIKey != "test-api-key" || !app.Config.MangoHud {
		t.Errorf("app.Config mismatch: %+v", app.Config)
	}

	loadedFromFile, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to reload config from file: %v", err)
	}
	if loadedFromFile.SteamWebAPIKey != "test-api-key" || !loadedFromFile.MangoHud {
		t.Errorf("config on disk did not persist, got: %+v", loadedFromFile)
	}
}

func TestApp_SaveConfig_UpdatesStateAndDisk(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, "gamux")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "config.json")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Pre-create config file with HubcapAPIKey and LibraryPaths
	existingJSON := `{
		"hubcap_api_key": "LIVE_HUBCAP_KEY_123",
		"library_paths": ["/My/Games"],
		"operating_mode": "loader"
	}`
	if err := os.WriteFile(cfgPath, []byte(existingJSON), 0644); err != nil {
		t.Fatal(err)
	}

	app := NewApp(nil)
	if app.Config.HubcapAPIKey != "LIVE_HUBCAP_KEY_123" {
		t.Errorf("expected initial loaded HubcapAPIKey 'LIVE_HUBCAP_KEY_123', got %q", app.Config.HubcapAPIKey)
	}

	// Update configuration: change operating mode and clear library paths
	updatedCfg := *app.Config
	updatedCfg.OperatingMode = "standalone"
	updatedCfg.MangoHud = true
	updatedCfg.LibraryPaths = nil

	if err := app.SaveConfig(&updatedCfg); err != nil {
		t.Fatalf("app.SaveConfig failed: %v", err)
	}

	// Verify app.Config retained HubcapAPIKey, updated OperatingMode, and cleared LibraryPaths
	if app.Config.HubcapAPIKey != "LIVE_HUBCAP_KEY_123" {
		t.Errorf("expected app.Config to retain HubcapAPIKey, got %q", app.Config.HubcapAPIKey)
	}
	if app.Config.OperatingMode != "standalone" {
		t.Errorf("expected app.Config OperatingMode 'standalone', got %q", app.Config.OperatingMode)
	}
	if len(app.Config.LibraryPaths) != 0 {
		t.Errorf("expected app.Config LibraryPaths to be cleared, got %+v", app.Config.LibraryPaths)
	}

	// Verify disk contents
	loaded, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.HubcapAPIKey != "LIVE_HUBCAP_KEY_123" {
		t.Errorf("expected disk HubcapAPIKey 'LIVE_HUBCAP_KEY_123', got %q", loaded.HubcapAPIKey)
	}
	if len(loaded.LibraryPaths) != 0 {
		t.Errorf("expected disk LibraryPaths cleared, got %+v", loaded.LibraryPaths)
	}
	if loaded.OperatingMode != "standalone" {
		t.Errorf("expected disk OperatingMode 'standalone', got %q", loaded.OperatingMode)
	}
}

func TestApp_GameOperations(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "TestGame")
	_ = os.MkdirAll(gameDir, 0755)

	exePath := filepath.Join(gameDir, "game.exe")
	_ = os.WriteFile(exePath, []byte("fake exe"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll"), []byte("fake dll"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_appid.txt"), []byte("12345\n"), 0644)

	cfg := config.DefaultConfig()
	cfg.GbeDir = tmpDir

	app := NewApp(cfg)

	// 1. Detect Game
	info, err := app.DetectGame(gameDir)
	if err != nil {
		t.Fatalf("DetectGame() failed: %v", err)
	}
	if info.AppID != "12345" {
		t.Errorf("DetectGame() AppID = %s, want 12345", info.AppID)
	}

	// 2. Empty path handling
	if _, err := app.DetectGame(""); err == nil {
		t.Error("DetectGame(\"\") expected error, got nil")
	}
	if _, err := app.InspectGame("", 0); err == nil {
		t.Error("InspectGame(\"\") expected error, got nil")
	}
	if _, err := app.InspectLibraries([]string{""}); err == nil {
		t.Error("InspectLibraries([\"\"]) expected error, got nil")
	}
	if _, err := app.SubmitApply(engine.ProcessOptions{}); err == nil {
		t.Error("SubmitApply(empty) expected error, got nil")
	}
	if _, err := app.SubmitRollback("", false); err == nil {
		t.Error("SubmitRollback(\"\") expected error, got nil")
	}

	// 3. Inspect Game (with news count 0 and >0)
	status, err := app.InspectGame(gameDir, 0)
	if err != nil {
		t.Fatalf("InspectGame() failed: %v", err)
	}
	if status.AppID != "12345" {
		t.Errorf("InspectGame() AppID = %s, want 12345", status.AppID)
	}

	statusWithNews, err := app.InspectGame(gameDir, 2)
	if err != nil {
		t.Fatalf("InspectGame(newsCount=2) failed: %v", err)
	}
	if statusWithNews.AppID != "12345" {
		t.Errorf("InspectGame(newsCount=2) AppID = %s, want 12345", statusWithNews.AppID)
	}

	// 4. Batch Inspect Libraries
	statuses, err := app.InspectLibraries([]string{tmpDir})
	if err != nil {
		t.Fatalf("InspectLibraries() failed: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("InspectLibraries() returned %d games, want 1", len(statuses))
	}
	if statuses[0].AppID != "12345" {
		t.Errorf("InspectLibraries() status AppID = %s, want 12345", statuses[0].AppID)
	}

	// 5. Submit Apply & Rollback Tasks
	applyTaskID, err := app.SubmitApply(engine.ProcessOptions{
		Path:          gameDir,
		DryRun:        true,
		OperatingMode: domain.ModeLoader,
	})
	if err != nil {
		t.Fatalf("SubmitApply() dry run failed: %v", err)
	}
	if applyTaskID == "" {
		t.Error("SubmitApply returned empty task ID")
	}

	rollbackTaskID, err := app.SubmitRollback(gameDir, true)
	if err != nil {
		t.Fatalf("SubmitRollback() dry run failed: %v", err)
	}
	if rollbackTaskID == "" {
		t.Error("SubmitRollback returned empty task ID")
	}
}

func TestApp_StartupAndNotifier(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	// Verify Startup doesn't panic
	app.Startup(context.Background())

	if app.Engine.Notifier == nil {
		t.Fatal("Engine.Notifier not configured after Startup")
	}

	// Calling notifier with dummy ctx should not fail
	if err := app.Engine.Notifier(context.Background(), "Test Alert", "Sample Message"); err != nil {
		t.Errorf("app.Engine.Notifier returned error: %v", err)
	}
}

func TestApp_ValidationAndHelpers(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	// Test empty path for OpenDirectory
	if err := app.OpenDirectory(""); err == nil {
		t.Error("OpenDirectory with empty path expected error, got nil")
	}
	if err := app.OpenDirectory("/nonexistent/directory/that/does/not/exist"); err == nil {
		t.Error("OpenDirectory with non-existent path expected error, got nil")
	}
}

func TestApp_LaunchProfileAndRunners(t *testing.T) {
	tmpDir := t.TempDir()
	app := NewApp(config.DefaultConfig())

	// Test GetAvailableRunners
	runners := app.GetAvailableRunners()
	if len(runners) == 0 {
		t.Errorf("expected at least one available runner")
	}

	// Test empty game path
	if _, err := app.GetLaunchProfile(""); err == nil {
		t.Error("expected error with empty game path, got nil")
	}
	if _, err := app.SaveLaunchProfile("", LaunchOptions{}); err == nil {
		t.Error("expected error with empty game path, got nil")
	}
	if err := app.LaunchGame(LaunchOptions{}); err == nil {
		t.Error("expected error with empty options, got nil")
	}

	// Test Save & Get profile
	trueVal := true
	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.bin",
		OperatingMode: "standalone",
		Runner:        "linux",
		MangoHud:      &trueVal,
		DXVKHud:       "fps",
	}

	if _, err := app.SaveLaunchProfile(tmpDir, opts); err != nil {
		t.Fatalf("SaveLaunchProfile failed: %v", err)
	}

	loaded, err := app.GetLaunchProfile(tmpDir)
	if err != nil {
		t.Fatalf("GetLaunchProfile failed: %v", err)
	}
	if loaded == nil || loaded.MangoHud == nil || !*loaded.MangoHud || loaded.DXVKHud != "fps" {
		t.Fatalf("loaded profile mismatch: %+v", loaded)
	}
}

func TestApp_AssetHandler(t *testing.T) {
	tmpDir := t.TempDir()
	artDir := filepath.Join(tmpDir, "art")
	gameArtDir := filepath.Join(artDir, "test-game")
	_ = os.MkdirAll(gameArtDir, 0755)

	coverPath := filepath.Join(gameArtDir, "cover.jpg")
	_ = os.WriteFile(coverPath, []byte("fake-image-bytes"), 0644)

	cfg := config.DefaultConfig()
	cfg.ArtworkDir = artDir
	app := NewApp(cfg)

	handler := app.AssetHandler()

	// 1. Valid request for existing cover
	req := httptest.NewRequest(http.MethodGet, "/art/test-game/cover.jpg", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "fake-image-bytes" {
		t.Errorf("expected body 'fake-image-bytes', got %s", rec.Body.String())
	}

	// 2. Request for non-existent image
	req404 := httptest.NewRequest(http.MethodGet, "/art/non-existent/cover.jpg", nil)
	rec404 := httptest.NewRecorder()
	handler.ServeHTTP(rec404, req404)

	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec404.Code)
	}

	// 3. Request with directory traversal
	reqTrav := httptest.NewRequest(http.MethodGet, "/art/../../etc/passwd", nil)
	recTrav := httptest.NewRecorder()
	handler.ServeHTTP(recTrav, reqTrav)

	if recTrav.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for traversal, got %d", recTrav.Code)
	}
}

func TestApp_InspectLibraries(t *testing.T) {
	app := NewApp(config.DefaultConfig())

	// Empty slice returns empty slice without error
	games, err := app.InspectLibraries([]string{})
	if err != nil {
		t.Fatalf("InspectLibraries([]) unexpected error: %v", err)
	}
	if len(games) != 0 {
		t.Errorf("InspectLibraries([]) len = %d, want 0", len(games))
	}

	// Slice with only empty strings returns error
	if _, err := app.InspectLibraries([]string{""}); err == nil {
		t.Error("InspectLibraries([\"\"]) expected error, got nil")
	}
}

func TestApp_SubmitTasks(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	tmpDir := t.TempDir()

	// 1. Empty game path checks
	if _, err := app.SubmitApply(engine.ProcessOptions{}); err == nil {
		t.Error("SubmitApply(empty) expected error, got nil")
	}
	if _, err := app.SubmitNormalize(""); err == nil {
		t.Error("SubmitNormalize(\"\") expected error, got nil")
	}
	if _, err := app.SubmitRollback("", false); err == nil {
		t.Error("SubmitRollback(\"\") expected error, got nil")
	}
	if _, err := app.SubmitVerify(""); err == nil {
		t.Error("SubmitVerify(\"\") expected error, got nil")
	}
	if _, err := app.SubmitUpdate(""); err == nil {
		t.Error("SubmitUpdate(\"\") expected error, got nil")
	}

	// 2. Valid task submissions
	taskID, err := app.SubmitNormalize(tmpDir)
	if err != nil {
		t.Fatalf("SubmitNormalize failed: %v", err)
	}
	if taskID == "" {
		t.Error("SubmitNormalize returned empty taskID")
	}

	task, exists := app.GetTask(taskID)
	if !exists || task == nil {
		t.Errorf("GetTask(%s) not found", taskID)
	}
}

func TestApp_MetadataAndSpeedLimit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	app := NewApp(config.DefaultConfig())

	// Test GetGameMetadata with invalid/unlinked provider
	if _, err := app.GetGameMetadata("none", "12345"); err == nil {
		t.Error("expected error for unlinked provider, got nil")
	}
	if _, err := app.GetGameMetadata("steam", ""); err == nil {
		t.Error("expected error for empty id, got nil")
	}

	// Test GetGameMetadata with valid mocked steam provider
	meta, err := app.GetGameMetadata("steam", "12345")
	if err != nil {
		t.Fatalf("GetGameMetadata failed: %v", err)
	}
	if meta.Name != "Test Game" {
		t.Errorf("expected game name 'Test Game', got %q", meta.Name)
	}

	// Test TransferSpeedLimit
	if limit := app.GetTransferSpeedLimit(); limit != 0 {
		t.Errorf("expected default speed limit 0, got %d", limit)
	}
	app.SetTransferSpeedLimit(5242880)
	if limit := app.GetTransferSpeedLimit(); limit != 5242880 {
		t.Errorf("expected updated speed limit 5242880, got %d", limit)
	}
}
