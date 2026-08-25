package gbe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGSEUserConfig_LoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "settings", "configs.user.ini")

	// 1. Initial load from non-existent file returns defaults
	cfg, err := LoadGSEUserConfigFromPath(iniPath)
	if err != nil {
		t.Fatalf("LoadGSEUserConfigFromPath failed: %v", err)
	}
	if cfg.AccountName != "GamuxPlayer" {
		t.Errorf("expected default account name 'GamuxPlayer', got %q", cfg.AccountName)
	}

	// 2. Modify and Save
	cfg.AccountName = "AliceGamer"
	cfg.AccountSteamID = "76561198123456789"
	cfg.Language = "german"
	cfg.IPCountry = "DE"

	if err := SaveGSEUserConfigToPath(iniPath, cfg); err != nil {
		t.Fatalf("SaveGSEUserConfigToPath failed: %v", err)
	}

	// 3. Re-load and verify
	reloaded, err := LoadGSEUserConfigFromPath(iniPath)
	if err != nil {
		t.Fatalf("re-load failed: %v", err)
	}

	if reloaded.AccountName != "AliceGamer" {
		t.Errorf("expected 'AliceGamer', got %q", reloaded.AccountName)
	}
	if reloaded.AccountSteamID != "76561198123456789" {
		t.Errorf("expected '76561198123456789', got %q", reloaded.AccountSteamID)
	}
	if reloaded.Language != "german" {
		t.Errorf("expected 'german', got %q", reloaded.Language)
	}
	if reloaded.IPCountry != "DE" {
		t.Errorf("expected 'DE', got %q", reloaded.IPCountry)
	}
}

func TestGSEUserConfig_ParseExisting(t *testing.T) {
	tmpDir := t.TempDir()
	iniPath := filepath.Join(tmpDir, "configs.user.ini")

	content := `
# Custom User Settings
[user::general]
account_steamid=76561198077567604
account_name=cm9000
language=english
ip_country=US
`
	if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadGSEUserConfigFromPath(iniPath)
	if err != nil {
		t.Fatalf("LoadGSEUserConfigFromPath failed: %v", err)
	}

	if cfg.AccountName != "cm9000" {
		t.Errorf("expected 'cm9000', got %q", cfg.AccountName)
	}
	if cfg.AccountSteamID != "76561198077567604" {
		t.Errorf("expected '76561198077567604', got %q", cfg.AccountSteamID)
	}
}
