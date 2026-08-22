package gui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/engine"
	"github.com/staernid/gamux/github"
	"github.com/staernid/gamux/manifest"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
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
	os.Exit(code)
}

func TestApp_GetAndSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := config.DefaultConfig()
	cfg.GbeDir = "/custom/gbe"
	cfg.LutrisDir = "/custom/lutris"

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
	if err := config.SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig() unexpected error: %v", err)
	}

	app.Config = cfg
	if app.Config.SteamWebAPIKey != "test-api-key" {
		t.Errorf("app.Config.SteamWebAPIKey got %q, want %q", app.Config.SteamWebAPIKey, "test-api-key")
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
	cfg.LutrisDir = tmpDir

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
	if _, err := app.InspectGame(""); err == nil {
		t.Error("InspectGame(\"\") expected error, got nil")
	}
	if _, err := app.BatchInspect(""); err == nil {
		t.Error("BatchInspect(\"\") expected error, got nil")
	}
	if _, err := app.ProcessGame(engine.ProcessOptions{}); err == nil {
		t.Error("ProcessGame(empty) expected error, got nil")
	}
	if _, err := app.ApplyGame(engine.ProcessOptions{}); err == nil {
		t.Error("ApplyGame(empty) expected error, got nil")
	}
	if err := app.Rollback("", false); err == nil {
		t.Error("Rollback(\"\") expected error, got nil")
	}
	if _, err := app.CheckLibraryUpdates(""); err == nil {
		t.Error("CheckLibraryUpdates(\"\") expected error, got nil")
	}
	if _, err := app.EnsureDirectoryExists(""); err == nil {
		t.Error("EnsureDirectoryExists(\"\") expected error, got nil")
	}

	// 3. Inspect Game
	status, err := app.InspectGame(gameDir)
	if err != nil {
		t.Fatalf("InspectGame() failed: %v", err)
	}
	if status.AppID != "12345" {
		t.Errorf("InspectGame() AppID = %s, want 12345", status.AppID)
	}

	// 4. Batch Inspect
	statuses, err := app.BatchInspect(tmpDir)
	if err != nil {
		t.Fatalf("BatchInspect() failed: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("BatchInspect() returned %d games, want 1", len(statuses))
	}
	if statuses[0].AppID != "12345" {
		t.Errorf("BatchInspect() status AppID = %s, want 12345", statuses[0].AppID)
	}

	// 5. Process Game (Dry Run)
	res, err := app.ProcessGame(engine.ProcessOptions{
		Path:     gameDir,
		DryRun:   true,
		ApplyGBE: true,
	})
	if err != nil {
		t.Fatalf("ProcessGame() dry run failed: %v", err)
	}
	if res.Info == nil || res.Info.AppID != "12345" {
		t.Errorf("ProcessGame() result Info invalid: %+v", res)
	}

	// 6. Rollback (Dry Run)
	if err := app.Rollback(gameDir, true); err != nil {
		t.Fatalf("Rollback() dry run failed: %v", err)
	}

	// 7. EnsureDirectoryExists
	created, err := app.EnsureDirectoryExists(filepath.Join(tmpDir, "new_sub_dir"))
	if err != nil || !created {
		t.Errorf("EnsureDirectoryExists() failed: created=%v, err=%v", created, err)
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

func TestApp_DownloadAndUpdateValidation(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	// Test invalid AppID
	if _, err := app.DownloadGame(DownloadGameOptions{AppID: 0, TargetDir: "/tmp"}); err == nil {
		t.Error("DownloadGame with AppID 0 expected error, got nil")
	}

	// Test empty TargetDir
	if _, err := app.DownloadGame(DownloadGameOptions{AppID: 12345, TargetDir: ""}); err == nil {
		t.Error("DownloadGame with empty TargetDir expected error, got nil")
	}

	// Test empty path for UpdateGame
	if _, err := app.UpdateGame(""); err == nil {
		t.Error("UpdateGame with empty path expected error, got nil")
	}

	// Test empty path for OpenFolder
	if err := app.OpenFolder(""); err == nil {
		t.Error("OpenFolder with empty path expected error, got nil")
	}
	if err := app.OpenFolder("/nonexistent/directory/that/does/not/exist"); err == nil {
		t.Error("OpenFolder with non-existent path expected error, got nil")
	}

	// Test empty slug for LaunchLutrisGame
	if err := app.LaunchLutrisGame(""); err == nil {
		t.Error("LaunchLutrisGame with empty slug expected error, got nil")
	}
}
