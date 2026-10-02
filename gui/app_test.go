package gui

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/runner"
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

func TestApp_GetConfigSchema(t *testing.T) {
	app := NewApp(nil)
	schema := app.GetConfigSchema()
	if len(schema) == 0 {
		t.Fatalf("expected non-empty schema from App.GetConfigSchema(), got 0 items")
	}
	foundKey := false
	for _, prop := range schema {
		if prop.Key == "operating_mode" {
			foundKey = true
			if prop.Type != "select" {
				t.Errorf("operating_mode expected type 'select', got %q", prop.Type)
			}
			break
		}
	}
	if !foundKey {
		t.Errorf("expected operating_mode to be present in schema")
	}
}

func TestApp_GetGameConfigSchema(t *testing.T) {
	app := NewApp(nil)
	schema := app.GetGameConfigSchema()
	if len(schema) == 0 {
		t.Fatalf("expected non-empty schema from App.GetGameConfigSchema(), got 0 items")
	}
	foundRunner := false
	for _, prop := range schema {
		if prop.Key == "runner" {
			foundRunner = true
			if prop.Type != "select" {
				t.Errorf("runner expected type 'select', got %q", prop.Type)
			}
			if prop.Scope != "game" {
				t.Errorf("runner expected scope 'game', got %q", prop.Scope)
			}
			break
		}
	}
	if !foundRunner {
		t.Errorf("expected runner to be present in game schema")
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

	originalEngine := app.Engine
	originalBus := app.Engine.EventBus

	if err := app.SaveConfig(&updatedCfg); err != nil {
		t.Fatalf("app.SaveConfig failed: %v", err)
	}

	if app.Engine != originalEngine {
		t.Errorf("expected app.Engine instance to be preserved across SaveConfig")
	}
	if app.Engine.EventBus != originalBus {
		t.Errorf("expected app.Engine.EventBus to be preserved across SaveConfig")
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

func TestApp_TrashGame(t *testing.T) {
	tmpDir := t.TempDir()
	fakeDataHome := filepath.Join(tmpDir, "data_home")
	t.Setenv("XDG_DATA_HOME", fakeDataHome)

	gameDir := filepath.Join(tmpDir, "GameToTrash")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "game.exe"), []byte("payload"), 0755)

	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	// Populate cache
	app.UpdateCachedGame(domain.NewGame("GameToTrash", gameDir, "12345", "win64"))

	// Empty path error
	if err := app.TrashGame(""); err == nil {
		t.Error("TrashGame(\"\") expected error, got nil")
	}

	// Trash valid game
	if err := app.TrashGame(gameDir); err != nil {
		t.Fatalf("TrashGame() failed: %v", err)
	}

	// Verify target no longer exists at original path
	if _, err := os.Stat(gameDir); !os.IsNotExist(err) {
		t.Errorf("gameDir still exists after TrashGame")
	}

	// Verify removed from cache
	app.gamesMu.RLock()
	for _, g := range app.cachedGames {
		if g != nil && g.GameDir == gameDir {
			t.Errorf("game still found in cachedGames")
		}
	}
	app.gamesMu.RUnlock()
}

func TestApp_StartupAndNotifier(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	// Verify Startup doesn't panic
	app.Startup(context.Background())

	ch, unsub := app.Engine.EventBus.Subscribe(events.EventNotification)
	defer unsub()

	// Calling Notify publishes to EventBus
	if err := app.Engine.Notify(context.Background(), "Test Alert", "Sample Message"); err != nil {
		t.Errorf("app.Engine.Notify returned error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Stage != "Test Alert" {
			t.Errorf("expected stage 'Test Alert', got %s", ev.Stage)
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("timeout waiting for notification on EventBus")
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
		DisableOverlay: &trueVal,
		OfflineMode:    &trueVal,
	}

	if _, err := app.SaveLaunchProfile(tmpDir, opts); err != nil {
		t.Fatalf("SaveLaunchProfile failed: %v", err)
	}

	loaded, err := app.GetLaunchProfile(tmpDir)
	if err != nil {
		t.Fatalf("GetLaunchProfile failed: %v", err)
	}
	if loaded == nil || loaded.MangoHud == nil || !*loaded.MangoHud || loaded.DisableOverlay == nil || !*loaded.DisableOverlay || loaded.OfflineMode == nil || !*loaded.OfflineMode {
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

func TestApp_AssetHandler_Thumbnail(t *testing.T) {
	tmpDir := t.TempDir()
	artDir := filepath.Join(tmpDir, "art")
	gameArtDir := filepath.Join(artDir, "test-game")
	_ = os.MkdirAll(gameArtDir, 0755)

	// Write real 800x1200 JPEG cover
	img := image.NewRGBA(image.Rect(0, 0, 800, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 800; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	coverPath := filepath.Join(gameArtDir, "cover.jpg")
	f, err := os.Create(coverPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Write real 400x600 PNG for game-54321
	appArtDir := filepath.Join(artDir, "game-54321")
	_ = os.MkdirAll(appArtDir, 0755)
	pngImg := image.NewRGBA(image.Rect(0, 0, 400, 600))
	pngPath := filepath.Join(appArtDir, "cover.png")
	pf, err := os.Create(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(pf, pngImg); err != nil {
		t.Fatal(err)
	}
	pf.Close()

	cfg := config.DefaultConfig()
	cfg.ArtworkDir = artDir
	app := NewApp(cfg)
	app.UpdateCachedGame(domain.NewGame("Test Game", "/games/test", "12345", "win64"))

	pngGame := domain.NewGame("PNG Icon Game", "/games/png", "54321", "win64")
	pngGame.LocalCover = pngPath
	app.UpdateCachedGame(pngGame)

	handler := app.AssetHandler()

	// 1. Direct path id: test-game/cover.jpg with w=300
	req1 := httptest.NewRequest(http.MethodGet, "/art/thumb?id=test-game/cover.jpg&w=300", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec1.Code)
	}
	if ct := rec1.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("expected Content-Type image/jpeg, got %s", ct)
	}
	decoded1, _, err := image.Decode(rec1.Body)
	if err != nil {
		t.Fatalf("failed to decode response image: %v", err)
	}
	if decoded1.Bounds().Dx() != 300 {
		t.Errorf("expected thumb width 300, got %d", decoded1.Bounds().Dx())
	}
	if decoded1.Bounds().Dy() != 450 {
		t.Errorf("expected thumb height 450, got %d", decoded1.Bounds().Dy())
	}

	// 2. Slug id: test-game with w=200
	req2 := httptest.NewRequest(http.MethodGet, "/art/thumb?id=test-game&w=200", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 for slug resolution, got %d", rec2.Code)
	}

	// 3. AppID resolution: id=54321 with w=150 (resolves cover.png)
	req3 := httptest.NewRequest(http.MethodGet, "/art/thumb?id=54321&w=150", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected status 200 for AppID resolution, got %d", rec3.Code)
	}
	if ct := rec3.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", ct)
	}

	// 4. Missing ID parameter
	reqMissing := httptest.NewRequest(http.MethodGet, "/art/thumb", nil)
	recMissing := httptest.NewRecorder()
	handler.ServeHTTP(recMissing, reqMissing)
	if recMissing.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing id, got %d", recMissing.Code)
	}

	// 5. Traversal attempt
	reqTrav := httptest.NewRequest(http.MethodGet, "/art/thumb?id=../../etc/passwd", nil)
	recTrav := httptest.NewRecorder()
	handler.ServeHTTP(recTrav, reqTrav)
	if recTrav.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for traversal, got %d", recTrav.Code)
	}

	// 6. Non-existent ID
	req404 := httptest.NewRequest(http.MethodGet, "/art/thumb?id=nonexistent-game", nil)
	rec404 := httptest.NewRecorder()
	handler.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for missing game, got %d", rec404.Code)
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

func TestApp_ProcessTrackingAndRunningGames(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "TrackedGame")
	_ = os.MkdirAll(gameDir, 0755)
	scriptPath := filepath.Join(gameDir, "sleep.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 10\n"), 0755)

	app := NewApp(config.DefaultConfig())

	// Initially not running
	if app.IsGameRunning(gameDir) {
		t.Errorf("expected game not to be running initially")
	}
	if app.IsGameRunning(gameDir + "/") {
		t.Errorf("expected game not to be running initially with trailing slash")
	}

	// Test empty path validation
	if app.IsGameRunning("") {
		t.Errorf("expected empty path IsGameRunning to be false")
	}
	if err := app.StopGame(""); err == nil {
		t.Errorf("expected StopGame with empty path to return error")
	}

	// Launch dummy game
	opts := runner.LaunchOptions{
		GameDir:  gameDir + "/",
		ExePath:  scriptPath,
		Platform: "linux",
		Runner:   runner.RunnerLinux,
	}
	_, err := app.Engine.RunnerMgr.Start(context.Background(), app.Config, opts)
	if err != nil {
		t.Fatalf("RunnerMgr.Start failed: %v", err)
	}

	// Verify running via canonical and non-canonical path
	if !app.IsGameRunning(gameDir) {
		t.Errorf("expected IsGameRunning(%q) = true", gameDir)
	}
	if !app.IsGameRunning(gameDir + "/") {
		t.Errorf("expected IsGameRunning(%q) with trailing slash = true", gameDir+"/")
	}

	// Verify GetRunningGames includes canonical path
	runningList := app.GetRunningGames()
	found := false
	for _, p := range runningList {
		if p == gameDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %q in GetRunningGames(), got: %v", gameDir, runningList)
	}

	// Stop game via path with trailing slash
	if err := app.StopGame(gameDir + "/"); err != nil {
		t.Fatalf("StopGame failed: %v", err)
	}

	// Verify stopped
	if app.IsGameRunning(gameDir) {
		t.Errorf("expected game to be stopped")
	}
}

func TestApp_CheckGameUpdates_and_DLCs(t *testing.T) {
	tmpDir := t.TempDir()
	app := &App{
		Engine: engine.New(nil),
	}

	gameDir := filepath.Join(tmpDir, "TestGame")
	sDir := filepath.Join(gameDir, "steam_settings")
	if err := os.MkdirAll(sDir, 0755); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(gameDir, "steam_appid.txt"), []byte("12345"), 0644)
	_ = os.WriteFile(filepath.Join(gameDir, "game.exe"), []byte("MZ"), 0755)

	iniContent := `[app::dlcs]
unlock_all=0
1001=Test DLC 1
`
	_ = os.WriteFile(filepath.Join(sDir, "configs.app.ini"), []byte(iniContent), 0644)

	// 1. GetGameDLCs
	dlcs, err := app.GetGameDLCs(gameDir)
	if err != nil {
		t.Fatalf("GetGameDLCs failed: %v", err)
	}
	if len(dlcs) != 1 || dlcs[0].AppID != "1001" {
		t.Errorf("expected 1 DLC with AppID 1001, got %+v", dlcs)
	}

	// 2. CheckGameUpdates
	status, err := app.CheckGameUpdates(gameDir)
	if err != nil {
		t.Fatalf("CheckGameUpdates failed: %v", err)
	}
	if status == nil {
		t.Fatalf("expected non-nil VersionStatus")
	}
}

func TestApp_Profiles(t *testing.T) {
	app := &App{
		Engine: engine.New(nil),
	}

	profs, err := app.GetProfiles()
	if err != nil {
		t.Fatalf("GetProfiles failed: %v", err)
	}
	if len(profs) < 1 {
		t.Fatalf("expected at least 1 profile, got %d", len(profs))
	}

	active, err := app.GetActiveProfile()
	if err != nil {
		t.Fatalf("GetActiveProfile failed: %v", err)
	}
	if active == nil || active.ID != "main" {
		t.Errorf("expected active profile to be main, got %+v", active)
	}

	// Create new profile
	p3, err := app.CreateProfile("PlayerThree", "76561198000000003", "english", "US", "")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	if p3.Name != "PlayerThree" {
		t.Errorf("expected PlayerThree, got %s", p3.Name)
	}

	// Switch active profile
	if err := app.SetActiveProfile(p3.ID); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}
	active, _ = app.GetActiveProfile()
	if active.ID != p3.ID {
		t.Errorf("expected active profile %s, got %s", p3.ID, active.ID)
	}

	// Save profile
	p3.Name = "UpdatedThree"
	if err := app.SaveProfile(*p3); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	// Delete profile
	if err := app.DeleteProfile(p3.ID); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}
	active, _ = app.GetActiveProfile()
	if active.ID != "main" {
		t.Errorf("expected active to fall back to main, got %s", active.ID)
	}
}


