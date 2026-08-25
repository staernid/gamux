package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/staernid/gamux/config"
)

func TestResolveEffectiveOptions(t *testing.T) {
	cfg := &config.Config{
		OperatingMode:     ModeStandalone,
		DefaultRunner:     RunnerUMU,
		WinePrefix:        "~/.wine",
		MangoHud:          true,
		WineFullscreenFSR: true,
		ProtonNVAPI:       true,
		DXVKHud:           "fps",
	}

	opts := LaunchOptions{
		GameDir:  "/tmp/game",
		ExePath:  "game.exe",
		Platform: "win64",
	}

	resolved := ResolveEffectiveOptions(cfg, opts)

	if resolved.OperatingMode != ModeStandalone {
		t.Errorf("expected OperatingMode %s, got %s", ModeStandalone, resolved.OperatingMode)
	}
	if resolved.Runner != RunnerUMU {
		t.Errorf("expected Runner %s, got %s", RunnerUMU, resolved.Runner)
	}
	if resolved.MangoHud == nil || !*resolved.MangoHud {
		t.Errorf("expected MangoHud to be true")
	}
	if resolved.WineFullscreenFSR == nil || !*resolved.WineFullscreenFSR {
		t.Errorf("expected WineFullscreenFSR to be true")
	}
	if resolved.ProtonNVAPI == nil || !*resolved.ProtonNVAPI {
		t.Errorf("expected ProtonNVAPI to be true")
	}
	if resolved.DXVKHud != "fps" {
		t.Errorf("expected DXVKHud 'fps', got %s", resolved.DXVKHud)
	}
	if resolved.WorkingDir != "/tmp/game" {
		t.Errorf("expected WorkingDir '/tmp/game', got %s", resolved.WorkingDir)
	}
}

func TestResolveEffectiveOptions_LinuxNative(t *testing.T) {
	cfg := &config.Config{
		OperatingMode: ModeStandalone,
		DefaultRunner: RunnerUMU,
	}

	opts := LaunchOptions{
		GameDir:  "/tmp/game",
		ExePath:  "game.x86_64",
		Platform: "linux",
	}

	resolved := ResolveEffectiveOptions(cfg, opts)
	if resolved.Runner != RunnerLinux {
		t.Errorf("expected linux platform to resolve to RunnerLinux, got %s", resolved.Runner)
	}
}

func TestBuildCommand_StandaloneLinux(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OperatingMode = ModeStandalone

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "game.bin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\necho ok"), 0755)

	trueVal := true
	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.bin",
		ExeArgs:       "-fullscreen -v",
		AppID:         "12345",
		Platform:      "linux",
		OperatingMode: ModeStandalone,
		Runner:        RunnerLinux,
		MangoHud:      &trueVal,
	}

	cmd, err := BuildCommand(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("unexpected error building command: %v", err)
	}

	if cmd.Path != binPath {
		t.Errorf("expected cmd.Path %s, got %s", binPath, cmd.Path)
	}

	if len(cmd.Args) != 3 || cmd.Args[1] != "-fullscreen" || cmd.Args[2] != "-v" {
		t.Errorf("unexpected cmd.Args: %v", cmd.Args)
	}

	var hasMangoHud, hasGameID bool
	for _, env := range cmd.Env {
		if env == "MANGOHUD=1" {
			hasMangoHud = true
		}
		if env == "GAMEID=12345" {
			hasGameID = true
		}
	}

	if !hasMangoHud {
		t.Errorf("expected MANGOHUD=1 in env")
	}
	if !hasGameID {
		t.Errorf("expected GAMEID=12345 in env")
	}
}

func TestBuildCommand_LoaderModeDetection(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OperatingMode = ModeStandalone

	tmpDir := t.TempDir()
	coldIni := filepath.Join(tmpDir, "ColdClientLoader.ini")
	loaderExe := filepath.Join(tmpDir, "steamclient_loader_x64.exe")
	gameExe := filepath.Join(tmpDir, "game.exe")

	_ = os.WriteFile(coldIni, []byte("[SteamClient]\nAppId=123"), 0644)
	_ = os.WriteFile(loaderExe, []byte("MZ..."), 0755)
	_ = os.WriteFile(gameExe, []byte("MZ..."), 0755)

	trueVal := true
	opts := LaunchOptions{
		GameDir:           tmpDir,
		ExePath:           "game.exe",
		AppID:             "123",
		Platform:          "win64",
		OperatingMode:     ModeStandalone,
		Runner:            RunnerLinux, // Test exe resolution directly
		WineFullscreenFSR: &trueVal,
		ProtonNVAPI:       &trueVal,
		DXVKHud:           "compiler",
		WinePrefix:        filepath.Join(tmpDir, "pfx"),
	}

	cmd, err := BuildCommand(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Loader mode should redirect target executable to steamclient_loader_x64.exe
	var foundLoader bool
	for _, arg := range cmd.Args {
		if arg == loaderExe {
			foundLoader = true
			break
		}
	}
	if cmd.Path == loaderExe {
		foundLoader = true
	}
	if !foundLoader {
		t.Errorf("expected command args or path to contain loader %s, got path: %s, args: %v", loaderExe, cmd.Path, cmd.Args)
	}

	var hasFSR, hasNVAPI, hasDXVK, hasLoaderDisable, hasLargeAddr, hasSDL, hasEsync, hasFsync, hasShaderCache, hasWineDebug bool
	for _, env := range cmd.Env {
		if env == "WINE_FULLSCREEN_FSR=1" {
			hasFSR = true
		}
		if env == "PROTON_ENABLE_NVAPI=1" {
			hasNVAPI = true
		}
		if env == "DXVK_HUD=compiler" {
			hasDXVK = true
		}
		if env == "PROTON_DISABLE_LSTEAMCLIENT=1" {
			hasLoaderDisable = true
		}
		if env == "WINE_LARGE_ADDRESS_AWARE=1" {
			hasLargeAddr = true
		}
		if env == "SDL_VIDEO_MINIMIZE_ON_FOCUS_LOSS=0" {
			hasSDL = true
		}
		if env == "WINEESYNC=1" {
			hasEsync = true
		}
		if env == "WINEFSYNC=1" {
			hasFsync = true
		}
		if env == "__GL_SHADER_DISK_CACHE=1" {
			hasShaderCache = true
		}
		if env == "WINEDEBUG=-all" {
			hasWineDebug = true
		}
	}

	if !hasFSR {
		t.Errorf("expected WINE_FULLSCREEN_FSR=1 in env")
	}
	if !hasNVAPI {
		t.Errorf("expected PROTON_ENABLE_NVAPI=1 in env")
	}
	if !hasDXVK {
		t.Errorf("expected DXVK_HUD=compiler in env")
	}
	if !hasLoaderDisable {
		t.Errorf("expected PROTON_DISABLE_LSTEAMCLIENT=1 in env")
	}
	if !hasLargeAddr {
		t.Errorf("expected WINE_LARGE_ADDRESS_AWARE=1 in env")
	}
	if !hasSDL {
		t.Errorf("expected SDL_VIDEO_MINIMIZE_ON_FOCUS_LOSS=0 in env")
	}
	if !hasEsync {
		t.Errorf("expected WINEESYNC=1 in env")
	}
	if !hasFsync {
		t.Errorf("expected WINEFSYNC=1 in env")
	}
	if !hasShaderCache {
		t.Errorf("expected __GL_SHADER_DISK_CACHE=1 in env")
	}
	if !hasWineDebug {
		t.Errorf("expected WINEDEBUG=-all in env")
	}
}

func TestLaunchProfile_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	trueVal := true
	opts := LaunchOptions{
		GameDir:           tmpDir,
		ExePath:           "Binaries/Win64/Game.exe",
		ExeArgs:           "-nostartupmovies",
		AppID:             "730",
		Name:              "Counter-Strike 2",
		Platform:          "win64",
		OperatingMode:     ModeStandalone,
		Runner:            RunnerUMU,
		WinePrefix:        "~/.wine-cs2",
		MangoHud:          &trueVal,
		WineFullscreenFSR: &trueVal,
		ProtonNVAPI:       &trueVal,
		DXVKHud:           "fps",
		CustomEnv: map[string]string{
			"CUSTOM_VAR": "1",
		},
	}

	if err := SaveLaunchProfile(tmpDir, opts); err != nil {
		t.Fatalf("failed to save launch profile: %v", err)
	}

	loaded, err := GetLaunchProfile(tmpDir)
	if err != nil {
		t.Fatalf("failed to get launch profile: %v", err)
	}
	if loaded == nil {
		t.Fatalf("expected loaded launch profile, got nil")
	}

	if loaded.Name != opts.Name {
		t.Errorf("expected Name %s, got %s", opts.Name, loaded.Name)
	}
	if loaded.OperatingMode != opts.OperatingMode {
		t.Errorf("expected OperatingMode %s, got %s", opts.OperatingMode, loaded.OperatingMode)
	}
	if loaded.Runner != opts.Runner {
		t.Errorf("expected Runner %s, got %s", opts.Runner, loaded.Runner)
	}
	if loaded.MangoHud == nil || !*loaded.MangoHud || loaded.WineFullscreenFSR == nil || !*loaded.WineFullscreenFSR || loaded.ProtonNVAPI == nil || !*loaded.ProtonNVAPI {
		t.Errorf("expected environment flags to persist")
	}
	if loaded.CustomEnv["CUSTOM_VAR"] != "1" {
		t.Errorf("expected custom env to persist")
	}
}

func TestGetAvailableRunners(t *testing.T) {
	runners := GetAvailableRunners()
	if len(runners) == 0 {
		t.Errorf("expected at least linux runner")
	}
	var hasLinux bool
	for _, r := range runners {
		if r == RunnerLinux {
			hasLinux = true
		}
	}
	if !hasLinux {
		t.Errorf("expected RunnerLinux in available runners")
	}
}

func TestManager_LifecycleAndEvents(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "sleep_game.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 5\n"), 0755)

	mgr := NewManager()
	events := make(chan ProcessEvent, 5)
	mgr.Subscribe(func(ev ProcessEvent) {
		events <- ev
	})

	cfg := config.DefaultConfig()
	opts := LaunchOptions{
		GameDir:  tmpDir,
		ExePath:  scriptPath,
		Platform: "linux",
		Runner:   RunnerLinux,
	}

	cmd, err := mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("Manager.Start failed: %v", err)
	}
	if cmd == nil || cmd.Process == nil {
		t.Fatalf("expected non-nil process")
	}

	if !mgr.IsRunning(tmpDir) {
		t.Errorf("expected game to be running")
	}

	// Verify Start event received
	select {
	case ev := <-events:
		if ev.GameDir != tmpDir || !ev.IsRunning {
			t.Errorf("unexpected start event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for start event")
	}

	// Stop game
	if err := mgr.Stop(tmpDir); err != nil {
		t.Fatalf("Manager.Stop failed: %v", err)
	}

	// Verify Stop event received
	select {
	case ev := <-events:
		if ev.GameDir != tmpDir || ev.IsRunning {
			t.Errorf("unexpected stop event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for stop event")
	}

	if mgr.IsRunning(tmpDir) {
		t.Errorf("expected game to no longer be running")
	}
}

