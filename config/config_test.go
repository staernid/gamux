package config

import (
	"encoding/json"
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

func TestHydrateAPIKeys_EnvironmentAndFiles(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "gamux")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("STEAMGRIDDB_API_KEY", "env_sgdb_key")
	t.Setenv("STEAM_WEB_API_KEY", "env_web_key")

	_ = os.WriteFile(filepath.Join(configDir, "hubcap_key.txt"), []byte("file_hubcap_key\n"), 0644)
	_ = os.WriteFile(filepath.Join(configDir, "rawg_key.txt"), []byte("  file_rawg_key  \n"), 0644)

	cfg := &Config{}
	hydrateAPIKeys(cfg)

	if cfg.SteamGridDBAPIKey != "env_sgdb_key" {
		t.Errorf("expected SteamGridDBAPIKey 'env_sgdb_key', got %q", cfg.SteamGridDBAPIKey)
	}
	if cfg.SteamWebAPIKey != "env_web_key" {
		t.Errorf("expected SteamWebAPIKey 'env_web_key', got %q", cfg.SteamWebAPIKey)
	}
	if cfg.HubcapAPIKey != "file_hubcap_key" {
		t.Errorf("expected HubcapAPIKey 'file_hubcap_key', got %q", cfg.HubcapAPIKey)
	}
	if cfg.RAWGAPIKey != "file_rawg_key" {
		t.Errorf("expected RAWGAPIKey 'file_rawg_key', got %q", cfg.RAWGAPIKey)
	}
}

func TestSaveConfig_MergeOnSave_PreservesExistingKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Pre-populate disk config with API keys and custom fields
	initialJSON := `{
		"hubcap_api_key": "SECRET_HUBCAP_KEY",
		"steam_web_api_key": "SECRET_STEAM_KEY",
		"custom_user_option": "custom_value",
		"library_paths": ["/Games/Library1"],
		"operating_mode": "loader",
		"mangohud": false
	}`
	if err := os.WriteFile(configFile, []byte(initialJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Incoming partial update (e.g. from UI or partial struct with omitted API keys)
	partialUpdate := &Config{
		OperatingMode: "standalone",
		MangoHud:      true,
		Language:      "german",
	}

	if err := SaveConfig(partialUpdate, configFile); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify that the file on disk preserved existing keys and unmapped fields
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read merged config: %v", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("Failed to unmarshal merged JSON: %v", err)
	}

	if rawMap["hubcap_api_key"] != "SECRET_HUBCAP_KEY" {
		t.Errorf("expected hubcap_api_key to be preserved, got %v", rawMap["hubcap_api_key"])
	}
	if rawMap["steam_web_api_key"] != "SECRET_STEAM_KEY" {
		t.Errorf("expected steam_web_api_key to be preserved, got %v", rawMap["steam_web_api_key"])
	}
	if rawMap["custom_user_option"] != "custom_value" {
		t.Errorf("expected custom_user_option to be preserved, got %v", rawMap["custom_user_option"])
	}
	if rawMap["operating_mode"] != "standalone" {
		t.Errorf("expected operating_mode to be updated to 'standalone', got %v", rawMap["operating_mode"])
	}
	if rawMap["mangohud"] != true {
		t.Errorf("expected mangohud to be updated to true, got %v", rawMap["mangohud"])
	}
	if rawMap["language"] != "german" {
		t.Errorf("expected language to be updated to 'german', got %v", rawMap["language"])
	}

	// Also verify LoadConfig produces correct merged Config struct
	loaded, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.HubcapAPIKey != "SECRET_HUBCAP_KEY" {
		t.Errorf("expected loaded HubcapAPIKey 'SECRET_HUBCAP_KEY', got %q", loaded.HubcapAPIKey)
	}
	if loaded.SteamWebAPIKey != "SECRET_STEAM_KEY" {
		t.Errorf("expected loaded SteamWebAPIKey 'SECRET_STEAM_KEY', got %q", loaded.SteamWebAPIKey)
	}
	if loaded.OperatingMode != "standalone" {
		t.Errorf("expected loaded OperatingMode 'standalone', got %q", loaded.OperatingMode)
	}
	if !loaded.MangoHud {
		t.Errorf("expected loaded MangoHud to be true")
	}
}

func TestUpdateConfig_ReadModifyWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	initialJSON := `{
		"hubcap_api_key": "HUBCAP_TEST",
		"transfer_speed_limit": 1000000
	}`
	if err := os.WriteFile(configFile, []byte(initialJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Update only transfer speed limit via UpdateConfig
	updated, err := UpdateConfig(configFile, func(c *Config) error {
		c.TransferSpeedLimit = 5000000
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	if updated.TransferSpeedLimit != 5000000 {
		t.Errorf("expected updated.TransferSpeedLimit = 5000000, got %d", updated.TransferSpeedLimit)
	}

	// Verify persisted file has both updated speed limit and preserved hubcap key
	loaded, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.TransferSpeedLimit != 5000000 {
		t.Errorf("expected loaded.TransferSpeedLimit = 5000000, got %d", loaded.TransferSpeedLimit)
	}
	if loaded.HubcapAPIKey != "HUBCAP_TEST" {
		t.Errorf("expected loaded.HubcapAPIKey = 'HUBCAP_TEST', got %q", loaded.HubcapAPIKey)
	}
}


