package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
)

func TestResolveEffectiveOptions(t *testing.T) {
	cfg := &config.Config{
		OperatingMode: ModeStandalone,
		DefaultRunner: RunnerUMU,
		WinePrefix:    "~/.wine",
		MangoHud:      true,
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

func TestBuildCommand_Gamescope(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gamescope = true
	cfg.GamescopeWidth = 3440
	cfg.GamescopeHeight = 1440
	cfg.GamescopeFPS = 100

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "game.bin")
	_ = os.WriteFile(binPath, []byte("#!/bin/sh\necho ok"), 0755)

	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.bin",
		Platform:      "linux",
		OperatingMode: ModeStandalone,
		Runner:        RunnerLinux,
	}

	spec, err := BuildLaunchSpec(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("unexpected error building spec: %v", err)
	}

	// If gamescope binary exists on the system, spec.Args should wrap with gamescope
	if _, err := exec.LookPath("gamescope"); err == nil {
		if len(spec.Args) < 9 || spec.Args[0] != "/usr/bin/gamescope" && !strings.HasSuffix(spec.Args[0], "gamescope") {
			t.Fatalf("expected gamescope wrapper in args, got: %v", spec.Args)
		}
		cmdStr := strings.Join(spec.Args, " ")
		if !strings.Contains(cmdStr, "-w 3440 -h 1440 -W 3440 -H 1440") {
			t.Errorf("expected 1:1 resolution in args, got: %s", cmdStr)
		}
		if !strings.Contains(cmdStr, "-r 100") {
			t.Errorf("expected refresh rate in args, got: %s", cmdStr)
		}
		if !strings.Contains(cmdStr, "-f --") {
			t.Errorf("expected fullscreen and separator in args, got: %s", cmdStr)
		}
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

	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.exe",
		AppID:         "123",
		Platform:      "win64",
		OperatingMode: ModeStandalone,
		Runner:        RunnerLinux, // Test exe resolution directly
		WinePrefix:    filepath.Join(tmpDir, "pfx"),
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

	var hasLoaderDisable, hasLargeAddr, hasSDL, hasEsync, hasFsync, hasShaderCache, hasWineDebug bool
	for _, env := range cmd.Env {
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
		AppID:         "730",
		Name:          "Counter-Strike 2",
		Platform:      "win64",
		OperatingMode: ModeStandalone,
		Runner:        RunnerUMU,
		WinePrefix:    "~/.wine-cs2",
		MangoHud:      &trueVal,
		DisableOverlay: &trueVal,
		OfflineMode:    &trueVal,
		CustomEnv: map[string]string{
			"CUSTOM_VAR": "1",
		},
	}

	if err := compat.SaveProfile(&opts); err != nil {
		t.Fatalf("failed to save launch profile: %v", err)
	}

	loaded, err := compat.LoadProfile(tmpDir)
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
	if loaded.MangoHud == nil || !*loaded.MangoHud {
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

	p, err := compat.LoadProfile(tmpDir)
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

	if err := compat.SyncSettings(context.Background(), p, cfg); err != nil {
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
	if err := compat.SyncSettings(context.Background(), p, cfg); err != nil {
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

	if err := compat.SaveProfile(p); err != nil {
		t.Fatalf("SaveProfile failed: %v", err)
	}

	profilePath := filepath.Join(tmpDir, ".gamux", "launch_profile.json")
	if !util.FileExists(profilePath) {
		t.Fatalf("expected profile to exist before reset")
	}

	if err := compat.ResetProfile(tmpDir); err != nil {
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
		MangoHud:   &trueVal,
		DisableOverlay: &trueVal,
		OfflineMode:    &trueVal,
		CustomEnv: map[string]string{
			"CASCADED_VAR": "profile_val",
		},
	}
	if err := compat.SaveProfile(&profile); err != nil {
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

	// Create valid 32-bit PE binary with DOS header and COFF machine 0x014c
	pe32Bin := filepath.Join(tmpDir, "win32_bin.exe")
	pe32Data := make([]byte, 128)
	copy(pe32Data[:2], "MZ")
	pe32Data[0x3C] = 0x40 // e_lfanew = 64
	copy(pe32Data[0x40:], "PE\x00\x00")
	pe32Data[0x44] = 0x4c // IMAGE_FILE_MACHINE_I386
	pe32Data[0x45] = 0x01
	_ = os.WriteFile(pe32Bin, pe32Data, 0755)

	txtFile := filepath.Join(tmpDir, "text.txt")
	_ = os.WriteFile(txtFile, []byte("plain text"), 0644)

	if p := detectBinaryPlatform(elfBin); p != "linux" {
		t.Errorf("expected linux for ELF, got %s", p)
	}
	if p := detectBinaryPlatform(peBin); p != "win64" {
		t.Errorf("expected win64 for fallback PE, got %s", p)
	}
	if p := detectBinaryPlatform(pe32Bin); p != "win32" {
		t.Errorf("expected win32 for PE32 binary, got %s", p)
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
	if !strings.Contains(foundOverride, "steamclient,steamclient64,GameOverlayRenderer,GameOverlayRenderer64=n,b") {
		t.Errorf("expected loader redirection in WINEDLLOVERRIDES, got: %s", foundOverride)
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

func TestBuildCommand_NoDllOverridesByDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "game.exe")
	_ = os.WriteFile(exePath, []byte("MZ"), 0755)

	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "game.exe",
		Platform:      "win64",
		Runner:        RunnerWine,
		OperatingMode: ModeStandalone,
	}

	spec, err := BuildLaunchSpec(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BuildLaunchSpec failed: %v", err)
	}

	if overrides, exists := spec.Env["WINEDLLOVERRIDES"]; exists {
		t.Errorf("expected no WINEDLLOVERRIDES by default, got: %s", overrides)
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

	// ModeLoader -> should have libX11.so.6 in LD_PRELOAD
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
	var foundUser string
	var foundClientLaunch string
	for _, env := range cmdLoader.Env {
		if strings.HasPrefix(env, "LD_PRELOAD=") {
			foundPreload = strings.TrimPrefix(env, "LD_PRELOAD=")
		}
		if strings.HasPrefix(env, "SteamAppUser=") {
			foundUser = strings.TrimPrefix(env, "SteamAppUser=")
		}
		if strings.HasPrefix(env, "SteamClientLaunch=") {
			foundClientLaunch = strings.TrimPrefix(env, "SteamClientLaunch=")
		}
	}
	if !strings.Contains(foundPreload, "libX11.so.6") {
		t.Errorf("expected libX11.so.6 in LD_PRELOAD for linux loader mode, got: %s", foundPreload)
	}
	if foundUser != "client_player" {
		t.Errorf("expected SteamAppUser=client_player, got: %s", foundUser)
	}
	if foundClientLaunch != "1" {
		t.Errorf("expected SteamClientLaunch=1, got: %s", foundClientLaunch)
	}

	// ModeNone -> should NOT have libX11.so.6 in LD_PRELOAD
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
			if strings.Contains(env, "libX11.so.6") {
				t.Errorf("expected ModeNone to NOT have libX11.so.6 in LD_PRELOAD, got: %s", env)
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

	err := compat.Apply(context.Background(), cfg, game, profilePortable, false)
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

	err = compat.Apply(context.Background(), cfg, game, profileLoader, false)
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

	err = compat.Apply(context.Background(), cfg, game, profileNone, false)
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

	restored, err := compat.Revert(context.Background(), tmpDir, false)
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

func TestBuildCommand_UMUProtonfixesAndSharedPrefix(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "Cyberpunk2077.exe")
	_ = os.WriteFile(exePath, []byte("MZ"), 0755)

	// 1. Steam title with AppID -> should set GAMEID=1091500, STORE=steam, and default shared WINEPREFIX
	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "Cyberpunk2077.exe",
		AppID:         "1091500",
		Platform:      "win64",
		OperatingMode: domain.ModeLoader,
		Runner:        RunnerUMU,
	}

	cmd, err := BuildCommand(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	envMap := make(map[string]string)
	for _, env := range cmd.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["GAMEID"] != "1091500" {
		t.Errorf("expected GAMEID '1091500', got %s", envMap["GAMEID"])
	}
	if envMap["STORE"] != "steam" {
		t.Errorf("expected STORE 'steam', got %s", envMap["STORE"])
	}
	if envMap["SteamAppId"] != "1091500" {
		t.Errorf("expected SteamAppId '1091500', got %s", envMap["SteamAppId"])
	}
	if !strings.HasSuffix(envMap["WINEPREFIX"], "Games/umu/1091500") {
		t.Errorf("expected WINEPREFIX to end with Games/umu/1091500, got %s", envMap["WINEPREFIX"])
	}
	if !strings.HasSuffix(envMap["STEAM_COMPAT_DATA_PATH"], "Games/umu/1091500") {
		t.Errorf("expected STEAM_COMPAT_DATA_PATH to end with Games/umu/1091500, got %s", envMap["STEAM_COMPAT_DATA_PATH"])
	}

	// 2. Standalone / non-Steam title without AppID -> should fallback to GAMEID=umu-default, STORE=none
	standaloneOpts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       "Cyberpunk2077.exe",
		AppID:         "",
		Platform:      "win64",
		OperatingMode: domain.ModeStandalone,
		Runner:        RunnerUMU,
	}

	standaloneCmd, err := BuildCommand(context.Background(), cfg, standaloneOpts)
	if err != nil {
		t.Fatalf("BuildCommand for standalone failed: %v", err)
	}

	standaloneEnv := make(map[string]string)
	for _, env := range standaloneCmd.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			standaloneEnv[parts[0]] = parts[1]
		}
	}

	if standaloneEnv["GAMEID"] != "umu-default" {
		t.Errorf("expected standalone GAMEID 'umu-default', got %s", standaloneEnv["GAMEID"])
	}
	if standaloneEnv["STORE"] != "none" {
		t.Errorf("expected standalone STORE 'none', got %s", standaloneEnv["STORE"])
	}
}

func TestManager_PlaytimeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "game.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 0.1\n"), 0755)

	bus := events.NewBus()
	mgr := NewManager(bus)

	opts := LaunchOptions{
		GameDir:       tmpDir,
		Name:          "Short Game",
		ExePath:       "game.sh",
		Platform:      "linux",
		OperatingMode: domain.ModeStandalone,
		Runner:        RunnerLinux,
	}

	cfg := config.DefaultConfig()

	eventsCh, unsub := bus.Subscribe(events.EventGameExited)
	defer unsub()

	cmd, err := mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}
	if cmd == nil || cmd.Process == nil {
		t.Fatalf("expected non-nil process")
	}

	select {
	case <-eventsCh:
		// success
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for game exit event")
	}

	// Verify profile recorded last played timestamp
	prof, err := compat.LoadProfile(tmpDir)
	if err != nil {
		t.Fatalf("LoadLaunchProfile failed: %v", err)
	}
	if prof.LastPlayed <= 0 {
		t.Errorf("expected LastPlayed > 0, got %d", prof.LastPlayed)
	}
}

func TestRunner_Run_PlaytimeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "game.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 0.05\n"), 0755)

	opts := LaunchOptions{
		GameDir:       tmpDir,
		Name:          "CLI Game",
		ExePath:       "game.sh",
		Platform:      "linux",
		OperatingMode: domain.ModeStandalone,
		Runner:        RunnerLinux,
	}

	cfg := config.DefaultConfig()
	if err := Run(context.Background(), cfg, opts); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	prof, err := compat.LoadProfile(tmpDir)
	if err != nil {
		t.Fatalf("LoadLaunchProfile failed: %v", err)
	}
	if prof.LastPlayed <= 0 {
		t.Errorf("expected LastPlayed > 0, got %d", prof.LastPlayed)
	}
}

func TestGenerateLaunchScript_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "WinGame")
	_ = os.MkdirAll(gameDir, 0755)
	exePath := filepath.Join(gameDir, "Binaries", "Win64", "Game.exe")
	_ = os.MkdirAll(filepath.Dir(exePath), 0755)
	_ = os.WriteFile(exePath, []byte("fake_exe"), 0755)

	trueVal := true
	profile := LaunchOptions{
		GameDir:       gameDir,
		Name:          "Test Win Game",
		AppID:         "1091500",
		Platform:      "win64",
		ExePath:       exePath,
		ExeArgs:       "-fullscreen -v",
		OperatingMode: ModeLoader,
		Runner:        RunnerUMU,
		WinePrefix:    "prefix",
		MangoHud:      &trueVal,
		CustomEnv: map[string]string{
			"MY_CUSTOM_VAR": "custom_val",
		},
	}

	cfg := config.DefaultConfig()
	if err := GenerateLaunchScript(context.Background(), cfg, profile); err != nil {
		t.Fatalf("GenerateLaunchScript failed: %v", err)
	}

	scriptPath := filepath.Join(gameDir, ".gamux", "launch.sh")
	if !util.FileExists(scriptPath) {
		t.Fatalf("expected launch.sh to be created at %s", scriptPath)
	}

	fi, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&0111 == 0 {
		t.Errorf("expected launch.sh to be executable (0755), got %v", fi.Mode())
	}

	contentBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, `export SteamAppId="1091500"`) {
		t.Errorf("expected SteamAppId in launch.sh")
	}
	if !strings.Contains(content, `export MANGOHUD="1"`) {
		t.Errorf("expected MANGOHUD in launch.sh")
	}
	if !strings.Contains(content, `export MY_CUSTOM_VAR="custom_val"`) {
		t.Errorf("expected MY_CUSTOM_VAR in launch.sh")
	}
	if !strings.Contains(content, "steamclient,steamclient64,GameOverlayRenderer,GameOverlayRenderer64=n,b") {
		t.Errorf("expected loader dll overrides in launch.sh")
	}
	if !strings.Contains(content, `"$GAME_DIR/Binaries/Win64/Game.exe"`) {
		t.Errorf("expected relative game exe path in launch.sh, got content:\n%s", content)
	}
	if !strings.Contains(content, `"$@"`) {
		t.Errorf("expected $@ pass-through arguments in launch.sh")
	}
}

func TestGenerateLaunchScript_Linux(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "LinuxGame")
	_ = os.MkdirAll(gameDir, 0755)
	exePath := filepath.Join(gameDir, "game.x86_64")
	_ = os.WriteFile(exePath, []byte("#!/bin/sh\n"), 0755)

	profile := LaunchOptions{
		GameDir:       gameDir,
		Name:          "Test Linux Game",
		Platform:      "linux",
		ExePath:       exePath,
		OperatingMode: domain.ModeStandalone,
		Runner:        RunnerLinux,
	}

	cfg := config.DefaultConfig()
	if err := GenerateLaunchScript(context.Background(), cfg, profile); err != nil {
		t.Fatalf("GenerateLaunchScript failed: %v", err)
	}

	scriptPath := filepath.Join(gameDir, ".gamux", "launch.sh")
	contentBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, `exec "$GAME_DIR/game.x86_64" "$@"`) {
		t.Errorf("expected direct native exec in launch.sh, got:\n%s", content)
	}
}

func TestGenerateLaunchScript_LinuxLoader(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "LinuxGameLoader")
	_ = os.MkdirAll(gameDir, 0755)
	exePath := filepath.Join(gameDir, "game.x86_64")
	_ = os.WriteFile(exePath, []byte("#!/bin/sh\n"), 0755)

	profile := LaunchOptions{
		GameDir:       gameDir,
		Name:          "Test Linux Game Loader",
		Platform:      "linux",
		ExePath:       exePath,
		OperatingMode: domain.ModeLoader,
		Runner:        RunnerLinux,
	}

	cfg := config.DefaultConfig()
	if err := GenerateLaunchScript(context.Background(), cfg, profile); err != nil {
		t.Fatalf("GenerateLaunchScript failed: %v", err)
	}

	scriptPath := filepath.Join(gameDir, ".gamux", "launch.sh")
	contentBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, `cleanup_steam`) {
		t.Errorf("expected cleanup_steam trap in launch.sh for Linux loader mode")
	}
	if !strings.Contains(content, `trap cleanup_steam EXIT INT TERM`) {
		t.Errorf("expected trap cleanup_steam in launch.sh")
	}
	if !strings.Contains(content, `export LD_PRELOAD="libX11.so.6"`) {
		t.Errorf("expected libX11.so.6 preload in launch.sh")
	}
}

func TestPrepareLinuxSteamEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	fakeHome := filepath.Join(tmpDir, "fake_home")
	t.Setenv("HOME", fakeHome)

	gbeDir := filepath.Join(tmpDir, "gbe")
	x64Dir := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	x86Dir := filepath.Join(gbeDir, "linux_release", "experimental", "x86")
	_ = os.MkdirAll(x64Dir, 0755)
	_ = os.MkdirAll(x86Dir, 0755)
	_ = os.WriteFile(filepath.Join(x64Dir, "steamclient.so"), []byte("gbe_so_64"), 0755)
	_ = os.WriteFile(filepath.Join(x86Dir, "steamclient.so"), []byte("gbe_so_86"), 0755)

	gameDir := filepath.Join(tmpDir, "game")
	gameSettings := filepath.Join(gameDir, "steam_settings")
	_ = os.MkdirAll(gameSettings, 0755)

	// Pre-create an existing steam.pid to test backup/restore
	steamDir := filepath.Join(fakeHome, ".steam")
	_ = os.MkdirAll(steamDir, 0755)
	existingPidFile := filepath.Join(steamDir, "steam.pid")
	_ = os.WriteFile(existingPidFile, []byte("999999\n"), 0644)

	cfg := &config.Config{GbeDir: gbeDir}
	opts := LaunchOptions{
		GameDir:       gameDir,
		Platform:      "linux",
		OperatingMode: ModeLoader,
	}

	cleanup, err := PrepareLinuxSteamEnvironment(cfg, opts)
	if err != nil {
		t.Fatalf("PrepareLinuxSteamEnvironment failed: %v", err)
	}

	// Verify current PID is written
	pidBytes, err := os.ReadFile(existingPidFile)
	if err != nil {
		t.Fatalf("failed to read steam.pid: %v", err)
	}
	expectedPid := fmt.Sprintf("%d\n", os.Getpid())
	if string(pidBytes) != expectedPid {
		t.Errorf("expected steam.pid to be %q, got %q", expectedPid, string(pidBytes))
	}

	// Verify steam.pid.orig exists
	if !util.FileExists(filepath.Join(steamDir, "steam.pid.orig")) {
		t.Errorf("expected steam.pid.orig to exist")
	}

	// Verify steamclient.so symlinks exist
	sdk64Client := filepath.Join(steamDir, "sdk64", "steamclient.so")
	if !util.FileExists(sdk64Client) {
		t.Errorf("expected sdk64/steamclient.so to exist")
	}
	sdk32Client := filepath.Join(steamDir, "sdk32", "steamclient.so")
	if !util.FileExists(sdk32Client) {
		t.Errorf("expected sdk32/steamclient.so to exist")
	}

	// Verify steam_settings symlinks exist
	if !util.DirExists(filepath.Join(steamDir, "sdk64", "steam_settings")) {
		t.Errorf("expected sdk64/steam_settings to exist")
	}

	// Run cleanup
	cleanup()

	// Verify steam.pid was restored to original value
	restoredPid, _ := os.ReadFile(existingPidFile)
	if string(restoredPid) != "999999\n" {
		t.Errorf("expected restored steam.pid to be 999999\\n, got: %q", string(restoredPid))
	}
	if util.FileExists(sdk64Client) {
		t.Errorf("expected sdk64/steamclient.so to be removed after cleanup")
	}
	if util.DirExists(filepath.Join(steamDir, "sdk64", "steam_settings")) {
		t.Errorf("expected sdk64/steam_settings symlink to be removed after cleanup")
	}
}

func TestPrepareLinuxSteamEnvironment_SecondaryProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, ".local", "share"))

	p2Home := filepath.Join(tmpDir, ".local", "share", "gamux", "profiles", "p2", "home")
	_ = os.MkdirAll(p2Home, 0755)

	gbeDir := filepath.Join(tmpDir, "gbe")
	x64Dir := filepath.Join(gbeDir, "linux_release", "experimental", "x64")
	_ = os.MkdirAll(x64Dir, 0755)
	_ = os.WriteFile(filepath.Join(x64Dir, "steamclient.so"), []byte("gbe_so_64"), 0755)

	gameDir := filepath.Join(tmpDir, "game")
	_ = os.MkdirAll(filepath.Join(gameDir, "steam_settings"), 0755)

	cfg := &config.Config{GbeDir: gbeDir}
	opts := LaunchOptions{
		GameDir:       gameDir,
		Platform:      "linux",
		OperatingMode: ModeLoader,
		ProfileID:     "p2",
	}

	cleanup, err := PrepareLinuxSteamEnvironment(cfg, opts)
	if err != nil {
		t.Fatalf("PrepareLinuxSteamEnvironment failed: %v", err)
	}
	defer cleanup()

	// Verify steamclient.so symlink exists in P2's home, NOT host home
	p2SteamDir := filepath.Join(p2Home, ".steam")
	p2Sdk64Client := filepath.Join(p2SteamDir, "sdk64", "steamclient.so")
	if !util.FileExists(p2Sdk64Client) {
		t.Errorf("expected sdk64/steamclient.so in p2 home at %s", p2Sdk64Client)
	}

	// Verify host home was untouched
	hostSteamSdk := filepath.Join(tmpDir, ".steam", "sdk64", "steamclient.so")
	if util.FileExists(hostSteamSdk) {
		t.Errorf("host home should NOT have received steamclient.so when running as p2")
	}
}

func TestGenerateLaunchScript_MatchesBuildCommand(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "MatchGame")
	_ = os.MkdirAll(gameDir, 0755)
	exePath := filepath.Join(gameDir, "game.exe")
	_ = os.WriteFile(exePath, []byte("binary"), 0755)

	cfg := config.DefaultConfig()
	cfg.GlobalEnv = map[string]string{
		"GLOBAL_KEY": "global_val",
	}

	opts := LaunchOptions{
		GameDir:       gameDir,
		Name:          "Match Game",
		AppID:         "55555",
		Platform:      "win64",
		ExePath:       "game.exe",
		ExeArgs:       "-arg1 -arg2",
		OperatingMode: ModeLoader,
		Runner:        RunnerWine,
		WinePrefix:    "prefix",
		CustomEnv: map[string]string{
			"LOCAL_KEY": "local_val",
		},
	}

	spec, err := BuildLaunchSpec(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BuildLaunchSpec failed: %v", err)
	}

	content := GenerateLaunchScriptContent(spec)

	// Verify all spec.Env keys are in generated script
	for k, v := range spec.Env {
		expectedVar := fmt.Sprintf("export %s=", k)
		if !strings.Contains(content, expectedVar) {
			t.Errorf("missing expected env var %s in generated script", k)
		}
		_ = v
	}

	// Verify global and local envs exist
	if !strings.Contains(content, `export GLOBAL_KEY="global_val"`) {
		t.Errorf("expected GLOBAL_KEY in script")
	}
	if !strings.Contains(content, `export LOCAL_KEY="local_val"`) {
		t.Errorf("expected LOCAL_KEY in script")
	}

	// Verify execution line has all args and $@
	if !strings.Contains(content, `exec`) || !strings.Contains(content, `"$@"`) {
		t.Errorf("invalid exec line in script:\n%s", content)
	}
}

func TestRun_ContextCancelNoBusySpin(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "sleep_loop.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 30\n"), 0755)

	cfg := config.DefaultConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	opts := LaunchOptions{
		GameDir:       tmpDir,
		ExePath:       scriptPath,
		Platform:      "linux",
		OperatingMode: ModeNative,
		Runner:        RunnerLinux,
	}

	start := time.Now()
	err := Run(ctx, cfg, opts)
	elapsed := time.Since(start)

	if err == nil {
		t.Errorf("expected context error on cancellation")
	}
	if elapsed > 3*time.Second {
		t.Errorf("Run took too long to terminate on cancellation: %v", elapsed)
	}
}

func TestManager_Stop_GracefulExit(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "game_mock.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\ntrap 'exit 0' TERM\nwhile true; do sleep 1; done\n"), 0755)

	cfg := config.DefaultConfig()
	mgr := NewManager()

	opts := LaunchOptions{
		Name:          "Graceful Game",
		GameDir:       tmpDir,
		ExePath:       scriptPath,
		Platform:      "linux",
		OperatingMode: ModeNative,
		Runner:        RunnerLinux,
	}

	cmd, err := mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("failed to start game: %v", err)
	}
	if cmd == nil || cmd.Process == nil {
		t.Fatalf("expected active command handle")
	}

	time.Sleep(100 * time.Millisecond)
	if !mgr.IsRunning(tmpDir) {
		t.Fatalf("expected game to be running")
	}

	start := time.Now()
	if err := mgr.Stop(tmpDir); err != nil {
		t.Fatalf("failed to stop game: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("expected fast graceful exit, took %v", elapsed)
	}

	// Give supervision goroutine a moment to clean up process map
	time.Sleep(200 * time.Millisecond)
	if mgr.IsRunning(tmpDir) {
		t.Errorf("expected game to be stopped in manager")
	}
}

func TestFindCandidateExecutable_ScoringAndFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "HadesGame")
	_ = os.MkdirAll(filepath.Join(gameDir, "bin", "x64"), 0755)

	// Create non-executable text files
	_ = os.WriteFile(filepath.Join(gameDir, "README"), []byte("readme content"), 0644)
	_ = os.WriteFile(filepath.Join(gameDir, "LICENSE"), []byte("license content"), 0644)

	// Create crash handlers / installers (should be ignored)
	_ = os.WriteFile(filepath.Join(gameDir, "UnityCrashHandler64.exe"), []byte("crash handler"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "unins000.exe"), []byte("uninstaller"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "dxsetup.exe"), []byte("setup"), 0755)

	// Create a secondary utility and the main game binary
	_ = os.WriteFile(filepath.Join(gameDir, "CrashDumper.exe"), []byte("crash dumper"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "ConfigTool.exe"), []byte("config tool"), 0755)
	_ = os.WriteFile(filepath.Join(gameDir, "bin", "x64", "Hades.exe"), []byte("main game"), 0755)

	cand := findCandidateExecutable(gameDir, "Hades")
	expected := filepath.Join(gameDir, "bin", "x64", "Hades.exe")
	if cand != expected {
		t.Errorf("expected %s as top candidate, got %s", expected, cand)
	}
}

func TestFindCandidateExecutable_LinuxPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "MyLinuxGame")
	_ = os.MkdirAll(gameDir, 0755)

	// Create non-executable file
	_ = os.WriteFile(filepath.Join(gameDir, "INSTALL"), []byte("install guide"), 0644)
	// Create executable script
	_ = os.WriteFile(filepath.Join(gameDir, "start.sh"), []byte("#!/bin/sh\n"), 0755)

	cand := findCandidateExecutable(gameDir, "MyLinuxGame")
	expected := filepath.Join(gameDir, "start.sh")
	if cand != expected {
		t.Errorf("expected %s, got %s", expected, cand)
	}
}

func TestManager_StopProcessGroupWithReapedParent(t *testing.T) {
	tmpDir := t.TempDir()
	wrapperScript := filepath.Join(tmpDir, "wrapper.sh")
	// Wrapper script starts a background sleep in the same process group and immediately exits
	_ = os.WriteFile(wrapperScript, []byte("#!/bin/sh\nsleep 30 &\nexit 0\n"), 0755)

	mgr := NewManager()
	cfg := config.DefaultConfig()
	opts := LaunchOptions{
		GameDir:  tmpDir,
		ExePath:  wrapperScript,
		Platform: "linux",
		Runner:   RunnerLinux,
	}

	cmd, err := mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	// Give the wrapper script a moment to exit and leave orphan child in pgid
	time.Sleep(100 * time.Millisecond)

	// Now call Stop on the gameDir
	if err := mgr.Stop(tmpDir); err != nil {
		t.Fatalf("mgr.Stop failed: %v", err)
	}

	// Verify manager is no longer tracking after stop
	time.Sleep(100 * time.Millisecond)
	if mgr.IsRunning(tmpDir) {
		t.Errorf("expected game to be stopped in manager")
	}
}

func TestBuildCommand_UMUDefaultProtonPath(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "game.exe")
	_ = os.WriteFile(exePath, []byte("MZ"), 0755)

	// Default: should set PROTONPATH to GE-Proton for UMU runner
	optsDefault := LaunchOptions{
		GameDir:  tmpDir,
		ExePath:  "game.exe",
		Platform: "win64",
		Runner:   RunnerUMU,
	}

	specDefault, err := BuildLaunchSpec(context.Background(), cfg, optsDefault)
	if err != nil {
		t.Fatalf("BuildLaunchSpec failed: %v", err)
	}
	if specDefault.Env["PROTONPATH"] != "GE-Proton" {
		t.Errorf("expected PROTONPATH to default to GE-Proton, got %q", specDefault.Env["PROTONPATH"])
	}

	// Custom: should respect custom ProtonPath
	optsCustom := LaunchOptions{
		GameDir:    tmpDir,
		ExePath:    "game.exe",
		Platform:   "win64",
		Runner:     RunnerUMU,
		ProtonPath: "/opt/custom-proton",
	}

	specCustom, err := BuildLaunchSpec(context.Background(), cfg, optsCustom)
	if err != nil {
		t.Fatalf("BuildLaunchSpec failed: %v", err)
	}
	if specCustom.Env["PROTONPATH"] != "/opt/custom-proton" {
		t.Errorf("expected PROTONPATH to be custom, got %q", specCustom.Env["PROTONPATH"])
	}
}

func TestIsHelperDaemon(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"wineserver", true},
		{"services.exe", true},
		{"SERVICES.EXE", true},
		{"C:\\windows\\system32\\explorer.exe", true},
		{"conhost.exe", true},
		{"gamemoded", true},
		{"bwrap", true},
		{"pressure-vessel-wrap", true},
		{"steam-runtime-launch-client", true},
		{"pv-bwrap", true},
		{"cyberpunk2077.exe", false},
		{"game.x86_64", false},
		{"hl2.exe", false},
		{"wine", false},
		{"wine64", false},
	}

	for _, c := range cases {
		if got := isHelperDaemon(c.name); got != c.expected {
			t.Errorf("isHelperDaemon(%q) = %v, want %v", c.name, got, c.expected)
		}
	}

	// Test isProcessHelperDaemon with wine loaders
	if !isProcessHelperDaemon("explorer.exe", []string{"explorer.exe", "/desktop"}) {
		t.Errorf("expected explorer.exe to be detected as helper daemon")
	}
	if !isProcessHelperDaemon("wine64-preloader", []string{"wine64-preloader", "C:\\windows\\system32\\services.exe"}) {
		t.Errorf("expected wine-launched services.exe to be detected as helper daemon")
	}
	if isProcessHelperDaemon("wine64-preloader", []string{"wine64-preloader", "Z:\\Games\\Cyberpunk\\bin\\x64\\Cyberpunk2077.exe"}) {
		t.Errorf("expected wine-launched Cyberpunk2077.exe NOT to be detected as helper daemon")
	}
}

func TestManager_PathCanonicalization(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "sleep.sh")
	_ = os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 5\n"), 0755)

	mgr := NewManager()
	cfg := config.DefaultConfig()

	// Launch with trailing slash
	dirWithSlash := tmpDir + "/"
	opts := LaunchOptions{
		GameDir:  dirWithSlash,
		ExePath:  scriptPath,
		Platform: "linux",
		Runner:   RunnerLinux,
	}

	cmd, err := mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil cmd")
	}

	// Verify IsRunning matches both with and without trailing slash
	cleanDir := filepath.Clean(tmpDir)
	if !mgr.IsRunning(cleanDir) {
		t.Errorf("expected IsRunning(%q) = true", cleanDir)
	}
	if !mgr.IsRunning(dirWithSlash) {
		t.Errorf("expected IsRunning(%q) = true", dirWithSlash)
	}

	// Verify RunningGames returns canonical path
	running := mgr.RunningGames()
	if len(running) != 1 || running[0] != cleanDir {
		t.Errorf("expected RunningGames() = [%q], got %v", cleanDir, running)
	}

	// Stop using the other path format
	if err := mgr.Stop(cleanDir); err != nil {
		t.Fatalf("mgr.Stop failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if mgr.IsRunning(cleanDir) || mgr.IsRunning(dirWithSlash) {
		t.Errorf("expected game to be stopped for both path variations")
	}
}

func TestManager_LingeringDaemonCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	// Create mock helper script that copies /bin/sleep to a binary named "wineserver"
	helperBin := filepath.Join(tmpDir, "wineserver")
	sleepBin, _ := exec.LookPath("sleep")
	if sleepBin == "" {
		t.Skip("sleep binary not available")
	}
	data, err := os.ReadFile(sleepBin)
	if err != nil {
		t.Skip("could not read sleep binary")
	}
	if err := os.WriteFile(helperBin, data, 0755); err != nil {
		t.Skip("could not create mock helper binary")
	}

	// Wrapper script runs the mock "wineserver" in background and immediately exits
	wrapperScript := filepath.Join(tmpDir, "launch.sh")
	scriptContent := fmt.Sprintf("#!/bin/sh\n%s 30 &\nexit 0\n", helperBin)
	_ = os.WriteFile(wrapperScript, []byte(scriptContent), 0755)

	bus := events.NewBus()
	exitEventCh := make(chan events.Event, 1)
	bus.SubscribeFunc(func(ev events.Event) {
		if ev.Type == events.EventGameExited {
			exitEventCh <- ev
		}
	})

	mgr := NewManager(bus)
	cfg := config.DefaultConfig()
	opts := LaunchOptions{
		GameDir:  tmpDir,
		ExePath:  wrapperScript,
		Platform: "linux",
		Runner:   RunnerLinux,
	}

	_, err = mgr.Start(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("mgr.Start failed: %v", err)
	}

	// The wrapper exits immediately. Only the mock "wineserver" remains in pgid.
	// The manager should recognize that no active game processes remain, terminate the daemon,
	// and emit EventGameExited within 3 seconds.
	select {
	case ev := <-exitEventCh:
		if ev.GameDir != util.CanonicalPath(tmpDir) {
			t.Errorf("expected exit event for %q, got %q", tmpDir, ev.GameDir)
		}
	case <-time.After(4 * time.Second):
		t.Fatalf("timeout: manager did not clean up lingering daemon and emit exit event")
	}

	if mgr.IsRunning(tmpDir) {
		t.Errorf("expected manager not to report running after daemon cleanup")
	}
}

func TestBuildLaunchSpec_SecondaryProfileIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, ".local", "share"))

	p2Home := filepath.Join(tmpDir, ".local", "share", "gamux", "profiles", "p2", "home")
	_ = os.MkdirAll(p2Home, 0755)

	gameDir := filepath.Join(tmpDir, "Torchlight2")
	_ = os.MkdirAll(gameDir, 0755)
	exePath := filepath.Join(gameDir, "Torchlight2.exe")
	_ = os.WriteFile(exePath, []byte("MZ"), 0755)

	cfg := config.DefaultConfig()
	opts := LaunchOptions{
		GameDir:       gameDir,
		ExePath:       "Torchlight2.exe",
		AppID:         "200710",
		Name:          "Torchlight II",
		Platform:      "win32",
		OperatingMode: "portable",
		Runner:        RunnerWine,
		ProfileID:     "p2",
	}

	spec, err := BuildLaunchSpec(context.Background(), cfg, opts)
	if err != nil {
		t.Fatalf("BuildLaunchSpec failed: %v", err)
	}

	expectedPrefix := filepath.Join(tmpDir, "Games", "umu-p2", "200710")
	if spec.Env["WINEPREFIX"] != expectedPrefix {
		t.Errorf("expected WINEPREFIX %s, got %s", expectedPrefix, spec.Env["WINEPREFIX"])
	}
	if spec.Env["STEAM_COMPAT_DATA_PATH"] != expectedPrefix {
		t.Errorf("expected STEAM_COMPAT_DATA_PATH %s, got %s", expectedPrefix, spec.Env["STEAM_COMPAT_DATA_PATH"])
	}
	if spec.Env["HOME"] != p2Home {
		t.Errorf("expected HOME %s, got %s", p2Home, spec.Env["HOME"])
	}
	expectedXdgData := filepath.Join(p2Home, ".local", "share")
	if spec.Env["XDG_DATA_HOME"] != expectedXdgData {
		t.Errorf("expected XDG_DATA_HOME %s, got %s", expectedXdgData, spec.Env["XDG_DATA_HOME"])
	}
}

func TestFormatCommandLine(t *testing.T) {
	spec := &LaunchSpec{
		Args: []string{"umu-run", "game.exe", "-dx11", "spaced arg"},
		Env: map[string]string{
			"WINEPREFIX": "/home/user/Games/umu",
			"MANGOHUD":   "1",
		},
	}
	formatted := FormatCommandLine(spec)
	if !strings.Contains(formatted, "WINEPREFIX=/home/user/Games/umu") {
		t.Errorf("expected WINEPREFIX in formatted command, got %q", formatted)
	}
	if !strings.Contains(formatted, "MANGOHUD=1") {
		t.Errorf("expected MANGOHUD in formatted command, got %q", formatted)
	}
	if !strings.Contains(formatted, `"spaced arg"`) {
		t.Errorf("expected quoted spaced arg, got %q", formatted)
	}
}

func TestDetectOverrides(t *testing.T) {
	cfg := &config.Config{
		DefaultRunner: "umu",
		OperatingMode: "loader",
		WinePrefix:    "~/Games/umu",
	}

	pClean := &domain.LaunchProfile{
		Runner:        "umu",
		OperatingMode: "loader",
	}
	overridesClean := DetectOverrides(pClean, cfg)
	if len(overridesClean) != 0 {
		t.Errorf("expected 0 overrides for matching config, got %v", overridesClean)
	}

	pCustom := &domain.LaunchProfile{
		Runner:    "wine",
		ExeArgs:   "-novid",
		CustomEnv: map[string]string{"FOO": "BAR"},
	}
	overridesCustom := DetectOverrides(pCustom, cfg)
	if len(overridesCustom) != 3 {
		t.Errorf("expected 3 overrides, got %v", overridesCustom)
	}
}

