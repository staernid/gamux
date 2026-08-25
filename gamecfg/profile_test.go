package gamecfg

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/util"
)

func TestProfile_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	trueVal := true

	p := &Profile{
		GameDir:           tmpDir,
		AppID:             "12345",
		Name:              "Test Adventure",
		Platform:          "win64",
		ExePath:           "Binaries/Win64/Game.exe",
		ExeArgs:           "-vulkan",
		OperatingMode:     ModeStandalone,
		Runner:            RunnerUMU,
		WinePrefix:        "~/.wine",
		Language:          "french",
		MangoHud:          &trueVal,
		DXVKHud:           "fps",
		WineFullscreenFSR: &trueVal,
		ProtonNVAPI:       &trueVal,
		CustomEnv: map[string]string{
			"MY_CUSTOM_VAR": "1",
		},
	}

	if err := Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.AppID != "12345" {
		t.Errorf("expected AppID 12345, got %s", loaded.AppID)
	}
	if loaded.Name != "Test Adventure" {
		t.Errorf("expected Name 'Test Adventure', got %s", loaded.Name)
	}
	if loaded.ExePath != "Binaries/Win64/Game.exe" {
		t.Errorf("expected ExePath Binaries/Win64/Game.exe, got %s", loaded.ExePath)
	}
	if loaded.Language != "french" {
		t.Errorf("expected Language french, got %s", loaded.Language)
	}
	if loaded.MangoHud == nil || !*loaded.MangoHud {
		t.Errorf("expected MangoHud=true")
	}
	if loaded.CustomEnv["MY_CUSTOM_VAR"] != "1" {
		t.Errorf("expected custom env MY_CUSTOM_VAR=1, got %v", loaded.CustomEnv)
	}
}

func TestProfile_FallbackHydration(t *testing.T) {
	tmpDir := t.TempDir()
	settingsDir := filepath.Join(tmpDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)

	_ = os.WriteFile(filepath.Join(settingsDir, "steam_appid.txt"), []byte("999888\n"), 0644)
	_ = os.WriteFile(filepath.Join(settingsDir, "force_language.txt"), []byte("german\n"), 0644)

	realExe := filepath.Join(tmpDir, "MyGame.exe")
	_ = os.WriteFile(realExe, []byte("exe"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "ColdClientLoader.ini"), []byte("[Loader]\nExe = MyGame.exe\n"), 0644)

	p, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
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

func TestProfile_Sync(t *testing.T) {
	tmpDir := t.TempDir()
	lutrisDir := filepath.Join(tmpDir, "lutris_games")
	_ = os.MkdirAll(lutrisDir, 0755)

	cfg := config.DefaultConfig()
	cfg.LutrisDir = lutrisDir

	realExe := filepath.Join(tmpDir, "ActionGame.exe")
	_ = os.WriteFile(realExe, []byte("exe"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "ColdClientLoader.ini"), []byte("[Loader]\nExe = old.exe\n"), 0644)

	p := &Profile{
		GameDir:       tmpDir,
		AppID:         "777",
		Name:          "Action Game",
		ExePath:       realExe,
		Language:      "spanish",
		OperatingMode: ModeLutris,
		Runner:        RunnerWine,
	}

	if err := Sync(context.Background(), p, cfg); err != nil {
		t.Fatalf("Sync failed: %v", err)
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

	// Verify ColdClientLoader.ini Exe replaced
	iniData, err := os.ReadFile(filepath.Join(tmpDir, "ColdClientLoader.ini"))
	if err != nil || !strings.Contains(string(iniData), "ActionGame.exe") {
		t.Errorf("expected ColdClientLoader.ini to contain ActionGame.exe, got: %s", string(iniData))
	}

	// Verify Lutris YAML generated
	lutrisFile := filepath.Join(lutrisDir, "action-game.yml")
	if !util.FileExists(lutrisFile) {
		t.Errorf("expected Lutris config file at %s", lutrisFile)
	}

	// Now set Language to "" and re-sync: force_language.txt should be removed
	p.Language = ""
	if err := Sync(context.Background(), p, cfg); err != nil {
		t.Fatalf("Sync failed on language reset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "steam_settings", "force_language.txt")); !os.IsNotExist(err) {
		t.Errorf("expected force_language.txt to be removed when Language is empty")
	}
}

func TestProfile_Reset(t *testing.T) {
	tmpDir := t.TempDir()
	p := &Profile{
		GameDir: tmpDir,
		Name:    "To Reset",
		ExePath: "game.exe",
	}

	if err := Save(p); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	profilePath := filepath.Join(tmpDir, "steam_settings", "launch_profile.json")
	if !util.FileExists(profilePath) {
		t.Fatalf("expected profile to exist before reset")
	}

	if err := Reset(tmpDir); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	if util.FileExists(profilePath) {
		t.Errorf("expected launch_profile.json to be deleted after reset")
	}
}

func TestProfile_ResolveEffective(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.OperatingMode = ModeStandalone
	cfg.DefaultRunner = RunnerUMU
	cfg.WinePrefix = "~/.default_wine"
	cfg.GlobalEnv = map[string]string{
		"GLOBAL_VAR": "global",
		"OVERRIDDEN": "from_global",
	}

	p := &Profile{
		Platform: "win64",
		CustomEnv: map[string]string{
			"OVERRIDDEN": "from_profile",
			"LOCAL_VAR":  "local",
		},
	}

	eff := ResolveEffective(cfg, p)
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
