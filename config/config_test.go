package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_XDGResolution(t *testing.T) {
	tmpDir := t.TempDir()
	dataHome := filepath.Join(tmpDir, "share")
	configHome := filepath.Join(tmpDir, "config")

	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	cfg := DefaultConfig()

	expectedGbe := filepath.Join(dataHome, "gbe_fork")
	if cfg.GbeDir != expectedGbe {
		t.Errorf("expected GbeDir %q, got %q", expectedGbe, cfg.GbeDir)
	}

	expectedSteam := filepath.Join(dataHome, "Steam", "userdata")
	if cfg.SteamUserdata != expectedSteam {
		t.Errorf("expected SteamUserdata %q, got %q", expectedSteam, cfg.SteamUserdata)
	}
}

func TestLoadConfig_CustomJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	content := `{
		"gbe_dir": "/custom/gbe",
		"steam_userdata": "/custom/steam",
		"steam_web_api_key": "TESTKEY123",
		"hubcap_api_key": "HUBCAP123"
	}`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.GbeDir != "/custom/gbe" {
		t.Errorf("expected GbeDir '/custom/gbe', got %s", cfg.GbeDir)
	}
	if cfg.SteamUserdata != "/custom/steam" {
		t.Errorf("expected SteamUserdata '/custom/steam', got %s", cfg.SteamUserdata)
	}
	if cfg.SteamWebAPIKey != "TESTKEY123" {
		t.Errorf("expected SteamWebAPIKey 'TESTKEY123', got %s", cfg.SteamWebAPIKey)
	}
	if cfg.HubcapAPIKey != "HUBCAP123" {
		t.Errorf("expected HubcapAPIKey 'HUBCAP123', got %s", cfg.HubcapAPIKey)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.OperatingMode = "standalone"
	cfg.MangoHud = true
	cfg.Language = "english"

	if err := SaveConfig(cfg, configFile); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if !loaded.MangoHud || loaded.Language != "english" {
		t.Errorf("expected saved settings to persist, got %+v", loaded)
	}
}
