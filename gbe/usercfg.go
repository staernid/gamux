package gbe

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configs.user.ini: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "account_name":
			cfg.AccountName = val
		case "account_steamid":
			cfg.AccountSteamID = val
		case "language":
			cfg.Language = val
		case "ip_country":
			cfg.IPCountry = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse configs.user.ini: %w", err)
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

	var buf bytes.Buffer
	buf.WriteString("[user::general]\n")

	if cfg.AccountSteamID != "" {
		buf.WriteString(fmt.Sprintf("account_steamid=%s\n", strings.TrimSpace(cfg.AccountSteamID)))
	} else {
		buf.WriteString("account_steamid=76561197960287930\n")
	}

	if cfg.AccountName != "" {
		buf.WriteString(fmt.Sprintf("account_name=%s\n", strings.TrimSpace(cfg.AccountName)))
	} else {
		buf.WriteString("account_name=GamuxPlayer\n")
	}

	if cfg.Language != "" {
		buf.WriteString(fmt.Sprintf("language=%s\n", strings.ToLower(strings.TrimSpace(cfg.Language))))
	} else {
		buf.WriteString("language=english\n")
	}

	if cfg.IPCountry != "" {
		buf.WriteString(fmt.Sprintf("ip_country=%s\n", strings.ToUpper(strings.TrimSpace(cfg.IPCountry))))
	} else {
		buf.WriteString("ip_country=US\n")
	}

	if err := util.WriteFileAtomic(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write configs.user.ini: %w", err)
	}

	slog.Info("Saved Goldberg global user configuration", "path", path, "account", cfg.AccountName)
	return nil
}
