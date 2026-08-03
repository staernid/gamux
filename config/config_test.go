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

	expectedLutris := filepath.Join(configHome, "lutris", "games")
	if cfg.LutrisDir != expectedLutris {
		t.Errorf("expected LutrisDir %q, got %q", expectedLutris, cfg.LutrisDir)
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
		"lutris_dir": "/custom/lutris",
		"steam_userdata": "/custom/steam"
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
	if cfg.LutrisDir != "/custom/lutris" {
		t.Errorf("expected LutrisDir '/custom/lutris', got %s", cfg.LutrisDir)
	}
	if cfg.SteamUserdata != "/custom/steam" {
		t.Errorf("expected SteamUserdata '/custom/steam', got %s", cfg.SteamUserdata)
	}
}
