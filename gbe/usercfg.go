package gbe

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/ini"
	"github.com/staernid/gamux/util"
)

// GSEUserConfig represents Goldberg Steam Emulator user persona configuration
// stored in ~/.local/share/GSE Saves/settings/configs.user.ini.
type GSEUserConfig struct {
	AccountName    string `json:"account_name"`
	AccountSteamID string `json:"account_steamid"`
	Language       string `json:"language"`
	IPCountry      string `json:"ip_country"`
}

// GetGSEUserConfigPath returns the canonical path to Goldberg's global user config file.
func GetGSEUserConfigPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			dataHome = filepath.Join(homeDir, ".local", "share")
		} else {
			dataHome = filepath.Join(os.TempDir(), ".local", "share")
		}
	}
	return filepath.Join(dataHome, "GSE Saves", "settings", "configs.user.ini")
}

// LoadGSEUserConfig reads and parses Goldberg's global configs.user.ini.
func LoadGSEUserConfig() (*GSEUserConfig, error) {
	path := GetGSEUserConfigPath()
	return LoadGSEUserConfigFromPath(path)
}

// LoadGSEUserConfigFromPath reads and parses a specific configs.user.ini file.
func LoadGSEUserConfigFromPath(path string) (*GSEUserConfig, error) {
	cfg := &GSEUserConfig{
		AccountName:    "GamuxPlayer",
		AccountSteamID: "76561197960287930",
		Language:       "english",
		IPCountry:      "US",
	}

	if !util.FileExists(path) {
		return cfg, nil
	}

	f, err := ini.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configs.user.ini: %w", err)
	}

	sec, ok := f.GetSection("user::general")
	if !ok {
		// Fallback to default/global section if user omitted section header
		if defaultSec, ok := f.GetSection(""); ok && len(defaultSec.Keys()) > 0 {
			sec = defaultSec
		} else if len(f.Sections()) > 0 {
			sec = f.Sections()[0]
		}
	}

	if sec != nil {
		if val, ok := sec.Get("account_name"); ok && val != "" {
			cfg.AccountName = val
		}
		if val, ok := sec.Get("account_steamid"); ok && val != "" {
			cfg.AccountSteamID = val
		}
		if val, ok := sec.Get("language"); ok && val != "" {
			cfg.Language = val
		}
		if val, ok := sec.Get("ip_country"); ok && val != "" {
			cfg.IPCountry = val
		}
	}

	return cfg, nil
}

// SaveGSEUserConfig persists Goldberg's global user persona configuration to configs.user.ini.
func SaveGSEUserConfig(cfg *GSEUserConfig) error {
	if cfg == nil {
		return fmt.Errorf("gse user config cannot be nil")
	}
	path := GetGSEUserConfigPath()
	return SaveGSEUserConfigToPath(path, cfg)
}

// SaveGSEUserConfigToPath persists the configuration to a specific file path.
func SaveGSEUserConfigToPath(path string, cfg *GSEUserConfig) error {
	if cfg == nil {
		return fmt.Errorf("gse user config cannot be nil")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create gse settings directory: %w", err)
	}

	f, err := ini.LoadFile(path)
	if err != nil {
		f = ini.NewFile()
	}

	sec := f.Section("user::general")

	accountSteamID := strings.TrimSpace(cfg.AccountSteamID)
	if accountSteamID == "" {
		accountSteamID = "76561197960287930"
	}
	sec.Set("account_steamid", accountSteamID)

	accountName := strings.TrimSpace(cfg.AccountName)
	if accountName == "" {
		accountName = "GamuxPlayer"
	}
	sec.Set("account_name", accountName)

	language := strings.ToLower(strings.TrimSpace(cfg.Language))
	if language == "" {
		language = "english"
	}
	sec.Set("language", language)

	ipCountry := strings.ToUpper(strings.TrimSpace(cfg.IPCountry))
	if ipCountry == "" {
		ipCountry = "US"
	}
	sec.Set("ip_country", ipCountry)

	if err := f.SaveFile(path); err != nil {
		return fmt.Errorf("write configs.user.ini: %w", err)
	}

	slog.Info("Saved Goldberg global user configuration", "path", path, "account", cfg.AccountName)
	return nil
}
