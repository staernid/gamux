package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
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

	// Direct executable should be launched rather than delegating to loader subprocess
	var foundGameExe bool
	for _, arg := range cmd.Args {
		if arg == gameExe {
			foundGameExe = true
			break
		}
	}
	if cmd.Path == gameExe {
		foundGameExe = true
	}
	if !foundGameExe {
		t.Errorf("expected command args or path to contain direct game exe %s, got path: %s, args: %v", gameExe, cmd.Path, cmd.Args)
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

func TestGetLog_AndPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "log_game.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\necho \"Hello from Game STDOUT\"\necho \"Warning from Game STDERR\" >&2\n"), 0755)

	mgr := NewManager()
	events := make(chan ProcessEvent, 5)
	mgr.Subscribe(func(ev ProcessEvent) {
		events <- ev
	})

	cfg := config.DefaultConfig()
	cfg.GbeDir = tmpDir
	opts := LaunchOptions{
		GameDir:  tmpDir,
		ExePath:  scriptPath,
		Name:     "Test Logging Game",
		AppID:    "999999",
		Platform: "linux",
		Runner:   RunnerLinux,
		CustomEnv: map[string]string{
			"LD_PRELOAD": "",
		},
	}


	_, err := mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("failed to start game: %v", err)
	}

	// Wait for process stop event
	timeout := time.After(3 * time.Second)
	stopped := false
	for !stopped {
		select {
		case ev := <-events:
			if ev.GameDir == tmpDir && !ev.IsRunning {
				stopped = true
			}
		case <-timeout:
			t.Fatalf("timed out waiting for game process to stop")
		}
	}

	time.Sleep(50 * time.Millisecond)

	logContent, err := GetLog(tmpDir, "Test Logging Game", "999999")
	if err != nil {
		t.Fatalf("failed to get log: %v", err)
	}

	if logContent == "" {
		t.Fatalf("expected non-empty log content")
	}

	if !strings.Contains(logContent, "=== Gamux Launch Log ===") {
		t.Errorf("expected header in log, got: %s", logContent)
	}
	if !strings.Contains(logContent, "Hello from Game STDOUT") {
		t.Errorf("expected stdout in log, got: %s", logContent)
	}
	if !strings.Contains(logContent, "Warning from Game STDERR") {
		t.Errorf("expected stderr in log, got: %s", logContent)
	}
	if !strings.Contains(logContent, "=== Process Finished ===") {
		t.Errorf("expected footer in log, got: %s", logContent)
	}
}

func TestLaunchProfile_FallbackHydration(t *testing.T) {
	tmpDir := t.TempDir()
	settingsDir := filepath.Join(tmpDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)

	_ = os.WriteFile(filepath.Join(settingsDir, "steam_appid.txt"), []byte("999888\n"), 0644)
	_ = os.WriteFile(filepath.Join(settingsDir, "force_language.txt"), []byte("german\n"), 0644)

	realExe := filepath.Join(tmpDir, "MyGame.exe")
	_ = os.WriteFile(realExe, []byte("exe"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "ColdClientLoader.ini"), []byte("[Loader]\nExe = MyGame.exe\n"), 0644)

	p, err := LoadLaunchProfile(tmpDir)
	if err != nil {
		t.Fatalf("LoadLaunchProfile failed: %v", err)
	}

	if p.AppID != "999888" {
		t.Errorf("expected fallback AppID 999888, got %s", p.AppID)
	}
	if p.Language != "german" {
		t.Errorf("expected fallback Language german, got %s", p.Language)
	}
	if p.ExePath != realExe {
		t.Errorf("expected fallback ExePath %s, got %s", realExe, p.ExePath)
	}
}

func TestLaunchProfile_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()

	realExe := filepath.Join(tmpDir, "ActionGame.exe")
	_ = os.WriteFile(realExe, []byte("exe"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "ColdClientLoader.ini"), []byte("[Loader]\nExe = old.exe\n"), 0644)

	p := &LaunchProfile{
		GameDir:       tmpDir,
		AppID:         "777",
		Name:          "Action Game",
		ExePath:       realExe,
		Language:      "spanish",
		OperatingMode: ModeStandalone,
		Runner:        RunnerWine,
	}

	if err := SyncLaunchProfile(context.Background(), p, cfg); err != nil {
		t.Fatalf("SyncLaunchProfile failed: %v", err)
	}

	// Verify steam_appid.txt synced
	appIDData, err := os.ReadFile(filepath.Join(tmpDir, "steam_settings", "steam_appid.txt"))
	if err != nil || strings.TrimSpace(string(appIDData)) != "777" {
		t.Errorf("expected steam_appid.txt 777, got %s (err: %v)", string(appIDData), err)
	}

	// Verify force_language.txt synced
	langData, err := os.ReadFile(filepath.Join(tmpDir, "steam_settings", "force_language.txt"))
	if err != nil || strings.TrimSpace(string(langData)) != "spanish" {
		t.Errorf("expected force_language.txt spanish, got %s (err: %v)", string(langData), err)
	}

	// Now set Language to "" and re-sync: force_language.txt should be removed
	p.Language = ""
	if err := SyncLaunchProfile(context.Background(), p, cfg); err != nil {
		t.Fatalf("SyncLaunchProfile failed on language reset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "steam_settings", "force_language.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_language.txt to be removed when Language is empty")
	}
}

func TestLaunchProfile_Reset(t *testing.T) {
	tmpDir := t.TempDir()
	p := &LaunchProfile{
		GameDir: tmpDir,
		Name:    "To Reset",
		ExePath: "game.exe",
	}

	if err := SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	profilePath := filepath.Join(tmpDir, "steam_settings", "launch_profile.json")
	if !util.FileExists(profilePath) {
		t.Fatalf("expected profile to exist before reset")
	}

	if err := ResetLaunchProfile(tmpDir); err != nil {
		t.Fatalf("ResetLaunchProfile failed: %v", err)
	}

	if util.FileExists(profilePath) {
		t.Errorf("expected launch_profile.json to be deleted after reset")
	}
}

func TestLaunchProfile_ResolveEffective(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OperatingMode = ModeStandalone
	cfg.DefaultRunner = RunnerUMU
	cfg.WinePrefix = "~/.default_wine"
	cfg.GlobalEnv = map[string]string{
		"GLOBAL_VAR": "global",
		"OVERRIDDEN": "from_global",
	}

	p := LaunchOptions{
		Platform: "win64",
		CustomEnv: map[string]string{
			"OVERRIDDEN": "from_profile",
			"LOCAL_VAR":  "local",
		},
	}

	eff := ResolveEffectiveOptions(cfg, p)
	if eff.OperatingMode != ModeStandalone {
		t.Errorf("expected OperatingMode standalone, got %s", eff.OperatingMode)
	}
	if eff.Runner != RunnerUMU {
		t.Errorf("expected Runner umu, got %s", eff.Runner)
	}
	if eff.WinePrefix != "~/.default_wine" {
		t.Errorf("expected WinePrefix ~/.default_wine, got %s", eff.WinePrefix)
	}
	if eff.CustomEnv["GLOBAL_VAR"] != "global" {
		t.Errorf("expected GLOBAL_VAR=global, got %s", eff.CustomEnv["GLOBAL_VAR"])
	}
	if eff.CustomEnv["OVERRIDDEN"] != "from_profile" {
		t.Errorf("expected OVERRIDDEN=from_profile, got %s", eff.CustomEnv["OVERRIDDEN"])
	}
	if eff.CustomEnv["LOCAL_VAR"] != "local" {
		t.Errorf("expected LOCAL_VAR=local, got %s", eff.CustomEnv["LOCAL_VAR"])
	}
}

func TestManager_BusEventDispatch(t *testing.T) {
	bus := events.NewBus()
	chLaunch, unsubLaunch := bus.Subscribe(events.EventGameLaunched)
	defer unsubLaunch()
	chExit, unsubExit := bus.Subscribe(events.EventGameExited)
	defer unsubExit()

	mgr := NewManager(bus)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "dummy_game.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	opts := LaunchOptions{
		GameDir:  tmpDir,
		ExePath:  scriptPath,
		Runner:   RunnerLinux,
		Platform: "linux",
		Name:     "Telemetry Game",
		AppID:    "77777",
	}

	cmd, err := mgr.Start(context.Background(), nil, opts)
	if err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}
	if cmd == nil || cmd.Process == nil {
		t.Fatalf("expected active process handle")
	}

	// Verify EventGameLaunched
	select {
	case ev := <-chLaunch:
		if ev.Type != events.EventGameLaunched {
			t.Errorf("expected EventGameLaunched, got %s", ev.Type)
		}
		if ev.GameDir != tmpDir {
			t.Errorf("expected GameDir %s, got %s", tmpDir, ev.GameDir)
		}
		if ev.GameName != "Telemetry Game" {
			t.Errorf("expected GameName 'Telemetry Game', got %s", ev.GameName)
		}
		if ev.AppID != "77777" {
			t.Errorf("expected AppID 77777, got %s", ev.AppID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for EventGameLaunched")
	}

	// Verify EventGameExited
	select {
	case ev := <-chExit:
		if ev.Type != events.EventGameExited {
			t.Errorf("expected EventGameExited, got %s", ev.Type)
		}
		if ev.GameDir != tmpDir {
			t.Errorf("expected GameDir %s, got %s", tmpDir, ev.GameDir)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EventGameExited")
	}
}

func TestResolveEffectiveOptions_CascadingProfile(t *testing.T) {
	tmpDir := t.TempDir()
	trueVal := true
	profile := LaunchOptions{
		GameDir:           tmpDir,
		ExePath:           filepath.Join(tmpDir, "SavedGame.exe"),
		AppID:             "88888",
		Name:              "Cascaded Game",
		Platform:          "win64",
		OperatingMode:     ModeStandalone,
		Runner:            RunnerWine,
		WinePrefix:        "~/.wine-cascaded",
		MangoHud:          &trueVal,
		WineFullscreenFSR: &trueVal,
		DXVKHud:           "fps",
		CustomEnv: map[string]string{
			"CASCADED_VAR": "profile_val",
		},
	}
	if err := SaveLaunchProfile(tmpDir, profile); err != nil {
		t.Fatalf("SaveLaunchProfile failed: %v", err)
	}

	cfg := config.DefaultConfig()
	opts := LaunchOptions{
		GameDir: tmpDir, // ExePath and other fields omitted
	}

	resolved := ResolveEffectiveOptions(cfg, opts)
	if resolved.Name != "Cascaded Game" {
		t.Errorf("expected Name 'Cascaded Game', got %s", resolved.Name)
	}
	if resolved.AppID != "88888" {
		t.Errorf("expected AppID 88888, got %s", resolved.AppID)
	}
	if resolved.ExePath != filepath.Join(tmpDir, "SavedGame.exe") {
		t.Errorf("expected ExePath %s, got %s", filepath.Join(tmpDir, "SavedGame.exe"), resolved.ExePath)
	}
	if resolved.Runner != RunnerWine {
		t.Errorf("expected Runner %s, got %s", RunnerWine, resolved.Runner)
	}
	if resolved.WinePrefix != "~/.wine-cascaded" {
		t.Errorf("expected WinePrefix ~/.wine-cascaded, got %s", resolved.WinePrefix)
	}
	if resolved.MangoHud == nil || !*resolved.MangoHud {
		t.Errorf("expected MangoHud true")
	}
	if resolved.CustomEnv["CASCADED_VAR"] != "profile_val" {
		t.Errorf("expected custom env CASCADED_VAR=profile_val, got %s", resolved.CustomEnv["CASCADED_VAR"])
	}
}

func TestResolveEffectiveOptions_CandidateFallback(t *testing.T) {
	tmpDir := t.TempDir()
	gameExe := filepath.Join(tmpDir, "MainBinary.exe")
	_ = os.WriteFile(gameExe, []byte("MZ...fake-pe"), 0755)

	cfg := config.DefaultConfig()
	opts := LaunchOptions{
		GameDir: tmpDir,
	}

	resolved := ResolveEffectiveOptions(cfg, opts)
	if resolved.ExePath != gameExe {
		t.Errorf("expected discovered ExePath %s, got %s", gameExe, resolved.ExePath)
	}
	if resolved.Platform != "win64" {
		t.Errorf("expected platform win64 from MZ magic bytes, got %s", resolved.Platform)
	}
}

func TestDetectBinaryPlatform(t *testing.T) {
	tmpDir := t.TempDir()

	elfBin := filepath.Join(tmpDir, "linux_bin")
	_ = os.WriteFile(elfBin, []byte("\x7fELFfakebinary"), 0755)

	peBin := filepath.Join(tmpDir, "win_bin.exe")
	_ = os.WriteFile(peBin, []byte("MZfakebinary"), 0755)

	txtFile := filepath.Join(tmpDir, "text.txt")
	_ = os.WriteFile(txtFile, []byte("plain text"), 0644)

	if p := detectBinaryPlatform(elfBin); p != "linux" {
		t.Errorf("expected linux for ELF, got %s", p)
	}
	if p := detectBinaryPlatform(peBin); p != "win64" {
		t.Errorf("expected win64 for PE, got %s", p)
	}
	if p := detectBinaryPlatform(txtFile); p != "" {
		t.Errorf("expected empty platform for text, got %s", p)
	}
}

func TestManager_LaunchGameDomain(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "domain_game.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755)

	mgr := NewManager()
	cfg := config.DefaultConfig()

	game := domain.NewGame("Domain Game", tmpDir, "11111", "linux")
	game.GameRecord.ExePath = scriptPath
	game.LaunchProfile = &domain.LaunchProfile{
		OperatingMode: ModeStandalone,
		Runner:        RunnerLinux,
	}

	events := make(chan ProcessEvent, 5)
	mgr.Subscribe(func(ev ProcessEvent) {
		events <- ev
	})

	cmd, err := mgr.Launch(context.Background(), cfg, game)
	if err != nil {
		t.Fatalf("mgr.Launch failed: %v", err)
	}
	if cmd == nil || cmd.Process == nil {
		t.Fatalf("expected non-nil process")
	}

	timeout := time.After(2 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.GameDir == tmpDir && !ev.IsRunning {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for domain game exit")
		}
	}
}

func TestBuildCommand_DllOverrides(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "game.exe")
	_ = os.WriteFile(exePath, []byte("MZ"), 0755)

	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.exe",
		Platform:      "win64",
		Runner:        RunnerWine,
		DllOverrides:  "steam_api64=n,b;d3d11=n,b",
		OperatingMode: ModeLoader,
	}

	cmd, err := BuildCommand(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	var foundOverride string
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "WINEDLLOVERRIDES=") {
			foundOverride = strings.TrimPrefix(env, "WINEDLLOVERRIDES=")
			break
		}
	}

	if foundOverride == "" {
		t.Fatalf("expected WINEDLLOVERRIDES in env, got none")
	}

	if !strings.Contains(foundOverride, "steam_api64=n,b") {
		t.Errorf("expected steam_api64=n,b in WINEDLLOVERRIDES, got: %s", foundOverride)
	}
	if !strings.Contains(foundOverride, "d3d11=n,b") {
		t.Errorf("expected d3d11=n,b in WINEDLLOVERRIDES, got: %s", foundOverride)
	}
	if !strings.Contains(foundOverride, "steamclient,steamclient64=n,b") {
		t.Errorf("expected loader redirection steamclient,steamclient64=n,b in WINEDLLOVERRIDES, got: %s", foundOverride)
	}
}

func TestBuildCommand_ModeNone_WineNoLoaderOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "game.exe")
	_ = os.WriteFile(exePath, []byte("MZ"), 0755)

	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.exe",
		Platform:      "win64",
		Runner:        RunnerWine,
		OperatingMode: ModeNone,
	}

	cmd, err := BuildCommand(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "WINEDLLOVERRIDES=") {
			if strings.Contains(env, "steamclient") {
				t.Errorf("expected ModeNone to NOT contain steamclient override in WINEDLLOVERRIDES, got: %s", env)
			}
		}
	}
}

func TestBuildCommand_Linux_LoaderPreload(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	linuxGbe := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	_ = os.MkdirAll(linuxGbe, 0755)
	soPath := filepath.Join(linuxGbe, "steamclient.so")
	_ = os.WriteFile(soPath, []byte("so_mock"), 0755)

	cfg := &config.Config{GbeDir: gbeDir}
	binPath := filepath.Join(tmpDir, "game.x86_64")
	_ = os.WriteFile(binPath, []byte("\x7fELF..."), 0755)

	// ModeLoader -> should have LD_PRELOAD
	optsLoader := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.x86_64",
		Platform:      "linux",
		Runner:        RunnerLinux,
		OperatingMode: ModeLoader,
	}

	cmdLoader, err := BuildCommand(context.Background(), cfg, optsLoader)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	var foundPreload string
	for _, env := range cmdLoader.Env {
		if strings.HasPrefix(env, "LD_PRELOAD=") {
			foundPreload = strings.TrimPrefix(env, "LD_PRELOAD=")
		}
	}
	if !strings.Contains(foundPreload, "steamclient.so") {
		t.Errorf("expected steamclient.so in LD_PRELOAD for linux loader mode, got: %s", foundPreload)
	}

	// ModeNone -> should NOT have LD_PRELOAD
	optsNone := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.x86_64",
		Platform:      "linux",
		Runner:        RunnerLinux,
		OperatingMode: ModeNone,
	}

	cmdNone, err := BuildCommand(context.Background(), cfg, optsNone)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	for _, env := range cmdNone.Env {
		if strings.HasPrefix(env, "LD_PRELOAD=") {
			if strings.Contains(env, "steamclient.so") {
				t.Errorf("expected ModeNone to NOT have steamclient.so in LD_PRELOAD, got: %s", env)
			}
		}
	}
}

func TestApplyCompatibility_Modes(t *testing.T) {
	tmpDir := t.TempDir()
	gbeDir := filepath.Join(tmpDir, "gbe_fork")
	x64Release := filepath.Join(gbeDir, "win_release", "experimental", "x64")
	_ = os.MkdirAll(x64Release, 0755)
	_ = os.WriteFile(filepath.Join(x64Release, "steam_api64.dll"), []byte("emulated_gbe_dll"), 0755)
	_ = os.WriteFile(filepath.Join(x64Release, "steamclient64.dll"), []byte("emulated_steamclient_dll"), 0755)

	loaderDir := filepath.Join(gbeDir, "win_release", "steamclient_experimental")
	_ = os.MkdirAll(loaderDir, 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient64.dll"), []byte("loader_steamclient64"), 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "steamclient.dll"), []byte("loader_steamclient32"), 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "GameOverlayRenderer64.dll"), []byte("loader_overlay64"), 0755)
	_ = os.WriteFile(filepath.Join(loaderDir, "GameOverlayRenderer.dll"), []byte("loader_overlay32"), 0755)

	cfg := config.DefaultConfig()
	cfg.GbeDir = gbeDir

	gameDir := filepath.Join(tmpDir, "GameFolder")
	_ = os.MkdirAll(gameDir, 0755)
	gameExe := filepath.Join(gameDir, "game.exe")
	_ = os.WriteFile(gameExe, []byte("MZ"), 0755)
	libPath := filepath.Join(gameDir, "steam_api64.dll")
	_ = os.WriteFile(libPath, []byte("pristine_steam_lib"), 0755)

	game := domain.NewGame("Test Game", gameDir, "12345", "win64")
	game.GameRecord.ExePath = "game.exe"

	// 1. Apply Portable Mode
	profilePortable := &domain.LaunchProfile{
		GameDir:       gameDir,
		AppID:         "12345",
		Platform:      "win64",
		OperatingMode: domain.ModePortable,
		ExePath:       "game.exe",
		Language:      "english",
	}

	err := ApplyCompatibility(context.Background(), cfg, game, profilePortable, false)
	if err != nil {
		t.Fatalf("ApplyCompatibility(portable) failed: %v", err)
	}

	data, _ := os.ReadFile(libPath)
	if string(data) != "emulated_gbe_dll" {
		t.Fatalf("expected emulated lib in portable mode, got: %s", string(data))
	}

	langData, _ := os.ReadFile(filepath.Join(gameDir, "steam_settings", "force_language.txt"))
	if strings.TrimSpace(string(langData)) != "english" {
		t.Errorf("expected force_language.txt english, got: %s", string(langData))
	}

	// 2. Apply Loader Mode -> should restore original library and create loader symlinks
	profileLoader := &domain.LaunchProfile{
		GameDir:       gameDir,
		AppID:         "12345",
		Platform:      "win64",
		OperatingMode: domain.ModeLoader,
		ExePath:       "game.exe",
	}

	err = ApplyCompatibility(context.Background(), cfg, game, profileLoader, false)
	if err != nil {
		t.Fatalf("ApplyCompatibility(loader) failed: %v", err)
	}

	restoredData, _ := os.ReadFile(libPath)
	if string(restoredData) != "pristine_steam_lib" {
		t.Fatalf("expected pristine lib restored in loader mode, got: %s", string(restoredData))
	}
	if _, err := os.Stat(filepath.Join(gameDir, "steamclient64.dll")); err != nil {
		t.Fatalf("expected steamclient64.dll symlink to exist in loader mode")
	}

	// 3. Apply None Mode -> should clean up loader symlinks
	profileNone := &domain.LaunchProfile{
		GameDir:       gameDir,
		AppID:         "12345",
		Platform:      "win64",
		OperatingMode: domain.ModeNone,
		ExePath:       "game.exe",
	}

	err = ApplyCompatibility(context.Background(), cfg, game, profileNone, false)
	if err != nil {
		t.Fatalf("ApplyCompatibility(none) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(gameDir, "steamclient64.dll")); err == nil {
		t.Fatalf("expected steamclient64.dll to be removed in None mode")
	}
}

func TestRevertCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "steam_api64.dll"), []byte("emulated"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "steam_api64.dll.ORIGINAL"), []byte("pristine"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "steamclient64.dll"), []byte("loader_link"), 0755)

	restored, err := RevertCompatibility(context.Background(), tmpDir, false)
	if err != nil {
		t.Fatalf("RevertCompatibility failed: %v", err)
	}
	if restored == 0 {
		t.Errorf("expected at least 1 restored backup, got 0")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "steam_api64.dll"))
	if string(data) != "pristine" {
		t.Errorf("expected pristine library restored, got %s", string(data))
	}
	if util.FileExists(filepath.Join(tmpDir, "steamclient64.dll")) {
		t.Errorf("expected loader artifact to be removed")
	}
}






