package gbe

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/steam"
	"github.com/staernid/gamux/util/testutil"
)

func TestMain(m *testing.M) {
	restore := testutil.SilenceLogging()
	code := m.Run()
	restore()
	os.Exit(code)
}


func TestApplyGBE_InvalidPlatform(t *testing.T) {
	err := Apply(context.Background(), Options{
		Config:    config.DefaultConfig(),
		TargetDir: ".",
		Platform:  "invalid_platform",
		AppID:     "123",
		DryRun:    true,
		Portable:  false,
	})
	if err == nil {
		t.Fatal("expected error for invalid platform")
	}
}

func TestApplyGBE_ExplicitTargetDirectory(t *testing.T) {
	// Create mock GBE directory structure
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	linuxGbe := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	if err := os.MkdirAll(linuxGbe, 0755); err != nil {
		t.Fatal(err)
	}

	mockSourceLib := filepath.Join(linuxGbe, "libsteam_api.so")
	if err := os.WriteFile(mockSourceLib, []byte("emulated_gbe_lib"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create mock game directory in a completely separate folder
	gameDir := filepath.Join(tmpDir, "MyGame")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetLib := filepath.Join(gameDir, "libsteam_api.so")
	if err := os.WriteFile(targetLib, []byte("original_steam_lib"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		GbeDir: gbeDir,
	}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	// Run Apply targeting gameDir in portable mode with dry-run
	err := Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "linux",
		AppID:     "12345",
		DryRun:    true,
		Portable:  true,
	})

	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify original file remained intact due to dry-run
	data, err := os.ReadFile(targetLib)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original_steam_lib" {
		t.Errorf("expected original lib content, got %s", string(data))
	}
}

func TestGenerateAchievementsJSON_DryRun(t *testing.T) {
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			jsonResp := `{"game": {"availableGameStats": {"achievements": [{"name": "ACH1", "displayName": "Ach 1", "icon": "https://cdn.com/ach1.png"}]}}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(jsonResp)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	tmpDir := t.TempDir()
	err := GenerateAchievementsJSON(ctx, 12345, "MOCKKEY", tmpDir, true)
	if err != nil {
		t.Fatalf("GenerateAchievementsJSON dry-run failed: %v", err)
	}
}

func TestConfigureCustomSettings(t *testing.T) {
	tmpDir := t.TempDir()
	steamSettingsDir := filepath.Join(tmpDir, "steam_settings")

	cfg := &config.Config{
		Language:       "german",
		DisableOverlay: true,
		OfflineMode:    true,
	}

	supportedLanguages := []string{"english", "german", "french"}

	err := ConfigureCustomSettings(steamSettingsDir, cfg, supportedLanguages, false)
	if err != nil {
		t.Fatalf("ConfigureCustomSettings failed: %v", err)
	}

	// Verify force_language.txt
	langData, err := os.ReadFile(filepath.Join(steamSettingsDir, "force_language.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(langData)) != "german" {
		t.Errorf("expected language 'german', got %q", string(langData))
	}

	// Verify force_account_name.txt & force_steamid.txt are NOT generated per-game
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "force_account_name.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_account_name.txt not to exist")
	}
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "force_steamid.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_steamid.txt not to exist")
	}

	// Verify disable_overlay.txt & offline.txt
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "disable_overlay.txt")); os.IsNotExist(err) {
		t.Error("expected disable_overlay.txt to exist")
	}
	if _, err := os.Stat(filepath.Join(steamSettingsDir, "offline.txt")); os.IsNotExist(err) {
		t.Error("expected offline.txt to exist")
	}
}

func TestConfigureCustomSettings_FallbackUnsupportedLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	steamSettingsDir := filepath.Join(tmpDir, "steam_settings")

	cfg := &config.Config{
		Language: "japanese",
	}

	supportedLanguages := []string{"english", "german"} // Japanese not supported

	err := ConfigureCustomSettings(steamSettingsDir, cfg, supportedLanguages, false)
	if err != nil {
		t.Fatalf("ConfigureCustomSettings failed: %v", err)
	}

	langData, _ := os.ReadFile(filepath.Join(steamSettingsDir, "force_language.txt"))
	if strings.TrimSpace(string(langData)) != "english" {
		t.Errorf("expected fallback language 'english', got %q", string(langData))
	}
}

func TestConfigureCustomSettings_AutoLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	steamSettingsDir := filepath.Join(tmpDir, "steam_settings")
	_ = os.MkdirAll(steamSettingsDir, 0755)

	// Pre-existing force_language.txt should be removed on "auto"
	_ = os.WriteFile(filepath.Join(steamSettingsDir, "force_language.txt"), []byte("english"), 0644)

	cfg := &config.Config{
		Language: "auto",
	}

	err := ConfigureCustomSettings(steamSettingsDir, cfg, []string{"english", "german"}, false)
	if err != nil {
		t.Fatalf("ConfigureCustomSettings failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(steamSettingsDir, "force_language.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_language.txt to be removed on auto language")
	}
}

func TestRevert_PortableMode(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	linuxGbe := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	_ = os.MkdirAll(linuxGbe, 0755)

	_ = os.WriteFile(filepath.Join(linuxGbe, "libsteam_api.so"), []byte("emulated_gbe_so"), 0755)

	gameDir := filepath.Join(tmpDir, "PortableGame")
	_ = os.MkdirAll(gameDir, 0755)
	targetLib := filepath.Join(gameDir, "libsteam_api.so")
	_ = os.WriteFile(targetLib, []byte("original_unpatched_so"), 0755)

	cfg := &config.Config{GbeDir: gbeDir}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	// Apply in portable mode
	err := Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "linux",
		AppID:     "11111",
		Portable:  true,
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify replacement occurred
	appliedData, _ := os.ReadFile(targetLib)
	if string(appliedData) != "emulated_gbe_so" {
		t.Fatalf("expected emulated lib, got %s", string(appliedData))
	}
	backups, _ := filepath.Glob(targetLib + "*.ORIGINAL")
	if len(backups) == 0 {
		t.Fatalf("expected .ORIGINAL backup file matching %s*.ORIGINAL", targetLib)
	}

	// Revert
	restored, err := Revert(gameDir, false)
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	if restored != 1 {
		t.Errorf("expected 1 restored file, got %d", restored)
	}

	// Verify restored to original
	restoredData, _ := os.ReadFile(targetLib)
	if string(restoredData) != "original_unpatched_so" {
		t.Errorf("expected original content after revert, got %s", string(restoredData))
	}
	backupsAfter, _ := filepath.Glob(targetLib + "*.ORIGINAL")
	if len(backupsAfter) != 0 {
		t.Errorf("expected .ORIGINAL backups to be removed after revert")
	}
	if _, err := os.Stat(filepath.Join(gameDir, "steam_settings")); !os.IsNotExist(err) {
		t.Errorf("expected steam_settings to be deleted after revert")
	}
}

func TestRevert_LoaderMode(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	loaderDir := filepath.Join(gbeDir, "win_release", "steamclient_experimental")
	winGbe := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(loaderDir, 0755)
	_ = os.MkdirAll(winGbe, 0755)

	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient64.dll"), []byte("steamclient64_binary"), 0755)
	_ = os.WriteFile(filepath.Join(winGbe, "steam_api64.dll"), []byte("emulated_dll"), 0644)

	gameDir := filepath.Join(tmpDir, "LoaderGame")
	_ = os.MkdirAll(gameDir, 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "Game.exe"), []byte("original_exe"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "steam_api64.dll"), []byte("original_dll"), 0644)

	cfg := &config.Config{GbeDir: gbeDir}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	// Apply in loader mode
	err := Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "win64",
		AppID:     "22222",
		Portable:  false,
		ExePath:   filepath.Join(gameDir, "Game.exe"),
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify loader artifacts created
	if _, err := os.Stat(filepath.Join(gameDir, "steamclient64.dll")); err != nil {
		t.Fatalf("expected steamclient64.dll")
	}

	// Revert
	_, err = Revert(gameDir, false)
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}

	// Verify loader artifacts removed
	if _, err := os.Stat(filepath.Join(gameDir, "steamclient64.dll")); !os.IsNotExist(err) {
		t.Errorf("expected steamclient64.dll to be removed")
	}

	// Original files remain intact
	data, _ := os.ReadFile(filepath.Join(gameDir, "Game.exe"))
	if string(data) != "original_exe" {
		t.Errorf("Game.exe corrupted")
	}
}

func TestSetupLoaderMode_SubdirectoryExe(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	loaderDir := filepath.Join(gbeDir, "win_release", "steamclient_experimental")
	winGbe := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(loaderDir, 0755)
	_ = os.MkdirAll(winGbe, 0755)

	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient64.dll"), []byte("steamclient64_binary"), 0755)
	_ = os.WriteFile(filepath.Join(winGbe, "steam_api64.dll"), []byte("emulated_dll"), 0644)

	gameDir := filepath.Join(tmpDir, "DeepGame")
	binDir := filepath.Join(gameDir, "bin", "x64")
	pluginDir := filepath.Join(gameDir, "Plugins")
	_ = os.MkdirAll(binDir, 0755)
	_ = os.MkdirAll(pluginDir, 0755)

	exePath := filepath.Join(binDir, "DeepGame.exe")
	libPath := filepath.Join(pluginDir, "steam_api64.dll")
	_ = os.WriteFile(exePath, []byte("exe_binary"), 0755)
	_ = os.WriteFile(libPath, []byte("original_steam_lib"), 0644)

	cfg := &config.Config{GbeDir: gbeDir}
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	// Apply loader mode with exe in subdirectory
	err := Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "win64",
		AppID:     "33333",
		Portable:  false,
		ExePath:   exePath,
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify steamclient64.dll is symlinked in root, bin/x64, and Plugins
	if _, err := os.Stat(filepath.Join(gameDir, "steamclient64.dll")); err != nil {
		t.Errorf("missing steamclient64.dll in root dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(binDir, "steamclient64.dll")); err != nil {
		t.Errorf("missing steamclient64.dll in exe dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "steamclient64.dll")); err != nil {
		t.Errorf("missing steamclient64.dll in lib dir: %v", err)
	}

	// Verify original lib remains unmodified
	data, err := os.ReadFile(libPath)
	if err != nil || string(data) != "original_steam_lib" {
		t.Errorf("original lib was modified: %s", string(data))
	}
}

func TestTransition_PortableToLoader_RestoresOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	loaderDir := filepath.Join(gbeDir, "win_release", "steamclient_experimental")
	winGbe := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(loaderDir, 0755)
	_ = os.MkdirAll(winGbe, 0755)

	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient64.dll"), []byte("steamclient64_binary"), 0755)
	_ = os.WriteFile(filepath.Join(winGbe, "steam_api64.dll"), []byte("emulated_gbe_dll"), 0644)

	gameDir := filepath.Join(tmpDir, "TransitionGame")
	_ = os.MkdirAll(gameDir, 0755)
	libPath := filepath.Join(gameDir, "steam_api64.dll")
	_ = os.WriteFile(libPath, []byte("pristine_steam_lib"), 0644)

	cfg := &config.Config{GbeDir: gbeDir}
	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	// 1. First apply Portable mode (replaces steam_api64.dll and creates .ORIGINAL)
	err := Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "win64",
		AppID:     "44444",
		Portable:  true,
	})
	if err != nil {
		t.Fatalf("Portable Apply failed: %v", err)
	}

	data, _ := os.ReadFile(libPath)
	if string(data) != "emulated_gbe_dll" {
		t.Fatalf("expected emulated lib in portable mode, got: %s", string(data))
	}
	entries, _ := os.ReadDir(gameDir)
	var foundBackup bool
	for _, e := range entries {
		if strings.Contains(e.Name(), ".ORIGINAL") {
			foundBackup = true
			break
		}
	}
	if !foundBackup {
		t.Fatalf("expected .ORIGINAL backup to exist in %s", gameDir)
	}

	// 2. Now transition to Loader mode (should restore .ORIGINAL and install loader symlinks)
	err = Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "win64",
		AppID:     "44444",
		Portable:  false,
	})
	if err != nil {
		t.Fatalf("Loader Apply failed: %v", err)
	}

	restoredData, _ := os.ReadFile(libPath)
	if string(restoredData) != "pristine_steam_lib" {
		t.Errorf("expected pristine lib restored in loader mode, got: %s", string(restoredData))
	}
	if _, err := os.Stat(filepath.Join(gameDir, "steamclient64.dll")); err != nil {
		t.Errorf("expected steamclient64.dll loader symlink to exist")
	}
}

func TestApplyGBE_Linux32ELFAutoDetection(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")

	// Set up both 64-bit and 32-bit GBE Linux assets
	linux64Gbe := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	linux32Gbe := filepath.Join(gbeDir, "linux_release", "experimental", "x86")
	_ = os.MkdirAll(linux64Gbe, 0755)
	_ = os.MkdirAll(linux32Gbe, 0755)

	_ = os.WriteFile(filepath.Join(linux64Gbe, "libsteam_api.so"), []byte("gbe_x64_so"), 0755)
	_ = os.WriteFile(filepath.Join(linux32Gbe, "libsteam_api.so"), []byte("gbe_x86_so"), 0755)

	// Create 32-bit Linux ELF mock library (ELFCLASS32: \x7fELF\x01)
	gameDir := filepath.Join(tmpDir, "Linux32Game")
	_ = os.MkdirAll(gameDir, 0755)
	elf32Header := []byte("\x7fELF\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00original_steam_so_32")
	targetLib := filepath.Join(gameDir, "libsteam_api.so")
	_ = os.WriteFile(targetLib, elf32Header, 0755)

	cfg := &config.Config{GbeDir: gbeDir}

	mockClient := &http.Client{
		Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	ctx := steam.WithHTTPClient(context.Background(), mockClient)

	err := Apply(ctx, Options{
		Config:    cfg,
		TargetDir: gameDir,
		Platform:  "linux",
		AppID:     "12345",
		Portable:  true,
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify 32-bit x86 GBE was selected and copied
	data, err := os.ReadFile(targetLib)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "gbe_x86_so" {
		t.Errorf("expected 32-bit gbe_x86_so, got: %s", string(data))
	}
}

func TestRevert_PreservesGamuxProfile(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "GameWithProfile")
	gamuxDir := filepath.Join(gameDir, ".gamux")
	_ = os.MkdirAll(gamuxDir, 0755)

	profileJSON := filepath.Join(gamuxDir, "launch_profile.json")
	launchSh := filepath.Join(gamuxDir, "launch.sh")
	steamSettings := filepath.Join(gameDir, "steam_settings")
	_ = os.MkdirAll(steamSettings, 0755)

	_ = os.WriteFile(profileJSON, []byte(`{"name":"My Game","playtime_seconds":3600}`), 0644)
	_ = os.WriteFile(launchSh, []byte("#!/bin/sh\nexec ./game.exe"), 0755)
	_ = os.WriteFile(filepath.Join(steamSettings, "force_language.txt"), []byte("english"), 0644)

	// Call Revert
	restored, err := Revert(gameDir, false)
	if err != nil {
		t.Fatalf("Revert failed: %v", err)
	}
	_ = restored

	// Verify .gamux/launch_profile.json is preserved
	profData, err := os.ReadFile(profileJSON)
	if err != nil {
		t.Fatalf("expected .gamux/launch_profile.json to remain intact after Revert: %v", err)
	}
	if !strings.Contains(string(profData), "playtime_seconds") {
		t.Errorf("profile contents corrupted: %s", string(profData))
	}

	// Verify .gamux/launch.sh and steam_settings were removed
	if _, err := os.Stat(launchSh); !os.IsNotExist(err) {
		t.Errorf("expected .gamux/launch.sh to be removed during Revert")
	}
	if _, err := os.Stat(steamSettings); !os.IsNotExist(err) {
		t.Errorf("expected steam_settings directory to be removed during Revert")
	}
}


