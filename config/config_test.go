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

func TestSaveConfig_OverwritesAndClearsFields(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Pre-populate disk config with keys
	initialJSON := `{
		"hubcap_api_key": "SECRET_HUBCAP_KEY",
		"library_paths": ["/Games/Library1", "/Games/Library2"],
		"operating_mode": "loader",
		"mangohud": true
	}`
	if err := os.WriteFile(configFile, []byte(initialJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Incoming update with cleared library_paths and modified operating_mode
	update := &Config{
		OperatingMode: "standalone",
		MangoHud:      false,
		Language:      "german",
		LibraryPaths:  nil, // Cleared
	}

	if err := SaveConfig(update, configFile); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify that the file on disk reflects cleared and updated fields
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, exists := rawMap["library_paths"]; exists {
		t.Errorf("expected library_paths to be omitted/cleared, got %v", rawMap["library_paths"])
	}
	if rawMap["operating_mode"] != "standalone" {
		t.Errorf("expected operating_mode to be updated to 'standalone', got %v", rawMap["operating_mode"])
	}
	if rawMap["language"] != "german" {
		t.Errorf("expected language to be 'german', got %v", rawMap["language"])
	}

	loaded, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(loaded.LibraryPaths) != 0 {
		t.Errorf("expected empty LibraryPaths, got %v", loaded.LibraryPaths)
	}
	if loaded.OperatingMode != "standalone" {
		t.Errorf("expected loaded OperatingMode 'standalone', got %q", loaded.OperatingMode)
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

func TestMultiseatConfig_PersistenceAndDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	if !cfg.Multiseat.Seat2ShowCursor {
		t.Errorf("expected default Seat2ShowCursor to be true, got false")
	}
	if cfg.Multiseat.Seat2Compositor != "auto" {
		t.Errorf("expected default Seat2Compositor to be 'auto', got %q", cfg.Multiseat.Seat2Compositor)
	}

	// Mutate to non-default values (including boolean false for Seat2ShowCursor)
	cfg.Multiseat.Seat2ShowCursor = false
	cfg.Multiseat.Seat2ForceGrabCursor = true
	cfg.Multiseat.Seat2GrabKeyboard = true
	cfg.Multiseat.Seat2Compositor = "cage"
	cfg.Multiseat.Seat2DisplayIndex = 0

	if err := SaveConfig(cfg, configFile); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Check the raw JSON on disk to verify false/0 are not dropped by omitempty
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	multiseatRaw, ok := rawMap["multiseat"].(map[string]any)
	if !ok {
		t.Fatalf("multiseat key missing from saved JSON")
	}
	if val, exists := multiseatRaw["seat2_show_cursor"]; !exists || val != false {
		t.Errorf("expected seat2_show_cursor to be explicitly persisted as false in JSON, got exists=%v val=%v", exists, val)
	}
	if val, exists := multiseatRaw["seat2_display_index"]; !exists || val != float64(0) {
		t.Errorf("expected seat2_display_index to be explicitly persisted as 0 in JSON, got exists=%v val=%v", exists, val)
	}

	// Now reload and ensure loaded struct preserves the values
	loaded, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Multiseat.Seat2ShowCursor != false {
		t.Errorf("expected loaded.Multiseat.Seat2ShowCursor to be false, got %v", loaded.Multiseat.Seat2ShowCursor)
	}
	if !loaded.Multiseat.Seat2ForceGrabCursor {
		t.Errorf("expected loaded.Multiseat.Seat2ForceGrabCursor to be true")
	}
	if !loaded.Multiseat.Seat2GrabKeyboard {
		t.Errorf("expected loaded.Multiseat.Seat2GrabKeyboard to be true")
	}
	if loaded.Multiseat.Seat2Compositor != "cage" {
		t.Errorf("expected loaded.Multiseat.Seat2Compositor 'cage', got %q", loaded.Multiseat.Seat2Compositor)
	}
}


