package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/util"
)

// Config holds user-configurable paths and settings for gamux.
type Config struct {
	// Paths
	LibraryPaths  []string `json:"library_paths,omitempty"`
	GbeDir        string   `json:"gbe_dir,omitempty"`
	SteamlessDir  string   `json:"steamless_dir,omitempty"`
	SteamUserdata string   `json:"steam_userdata,omitempty"`

	// API Keys
	SteamWebAPIKey    string `json:"steam_web_api_key,omitempty"`
	HubcapAPIKey      string `json:"hubcap_api_key,omitempty"`
	SteamGridDBAPIKey string `json:"steamgriddb_api_key,omitempty"`
	RAWGAPIKey        string `json:"rawg_api_key,omitempty"`
	ArtworkDir        string `json:"artwork_dir,omitempty"`

	// Notifications & Process Lifecycle
	EnableLaunchNotify   bool   `json:"enable_launch_notify"`
	LaunchWindowBehavior string `json:"launch_window_behavior,omitempty"` // "keep" | "minimize" | "hide"
	CloseToTray          bool   `json:"close_to_tray"`

	// Operational Settings
	OperatingMode      string            `json:"operating_mode,omitempty"`
	DefaultRunner      string            `json:"default_runner,omitempty"`
	Steamless          bool              `json:"steamless"`
	Achievements       bool              `json:"achievements"`
	Normalize          bool              `json:"normalize"`
	WinePrefix         string            `json:"wine_prefix,omitempty"`
	MangoHud           bool              `json:"mangohud,omitempty"`
	TransferSpeedLimit int64             `json:"transfer_speed_limit,omitempty"`
	GlobalEnv          map[string]string `json:"global_env,omitempty"`

	// Emulation Profile Flags
	Language       string `json:"language,omitempty"`
	DisableOverlay bool   `json:"disable_overlay,omitempty"`
	OfflineMode    bool   `json:"offline_mode,omitempty"`

	// Save & Backup Settings
	AutoBackupSaves      bool   `json:"auto_backup_saves"`
	BackupRetentionCount int    `json:"backup_retention_count,omitempty"`
	BackupDir            string `json:"backup_dir,omitempty"`
	SaveManifestURL      string `json:"save_manifest_url,omitempty"`
}

// DefaultConfig resolves default paths respecting XDG environment variables.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" && homeDir != "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" && homeDir != "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	gbePath := ".local/share/gbe_fork"
	if dataHome != "" {
		gbePath = filepath.Join(dataHome, "gbe_fork")
	}

	steamlessPath := ".local/share/steamless"
	if dataHome != "" {
		steamlessPath = filepath.Join(dataHome, "steamless")
	}

	steamUserDataPath := ".local/share/Steam/userdata"
	if dataHome != "" {
		steamUserDataPath = filepath.Join(dataHome, "Steam", "userdata")
	}

	artworkPath := ".local/share/gamux/art"
	if dataHome != "" {
		artworkPath = filepath.Join(dataHome, "gamux", "art")
	}

	backupPath := ".local/share/gamux/backups"
	if dataHome != "" {
		backupPath = filepath.Join(dataHome, "gamux", "backups")
	}

	return &Config{
		LibraryPaths:         nil,
		GbeDir:               gbePath,
		SteamlessDir:         steamlessPath,
		SteamUserdata:        steamUserDataPath,
		ArtworkDir:           artworkPath,
		BackupDir:            backupPath,
		BackupRetentionCount: 5,
		AutoBackupSaves:      false,
		SteamWebAPIKey:       "",
		HubcapAPIKey:         "",
		SteamGridDBAPIKey:    "",
		EnableLaunchNotify:   true,
		LaunchWindowBehavior: "keep",
		CloseToTray:          false,
		OperatingMode:        "loader",
		DefaultRunner:        "umu",
		Steamless:            true,
		Achievements:         true,
		Normalize:            true,
		WinePrefix:           "",
		MangoHud:             false,
	}
}

// GetConfigPath returns the absolute path to the configuration file.
func GetConfigPath(customPath string) string {
	if customPath != "" {
		return customPath
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		return filepath.Join(configHome, "gamux", "config.json")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(homeDir, ".config", "gamux", "config.json")
}

// LoadConfig loads configuration from a JSON file or returns defaults.
func LoadConfig(customPath string) (*Config, error) {
	cfg := DefaultConfig()
	path := GetConfigPath(customPath)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		hydrateAPIKeys(cfg)
		if customPath != "" {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	hydrateAPIKeys(cfg)
	return cfg, nil
}

func hydrateAPIKeys(cfg *Config) {
	if cfg == nil {
		return
	}
	home, _ := os.UserHomeDir()
	configDir := ""
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		configDir = filepath.Join(configHome, "gamux")
	} else if home != "" {
		configDir = filepath.Join(home, ".config", "gamux")
	}

	loadKey := func(currentVal, envVar, fileName string) string {
		if currentVal != "" {
			return currentVal
		}
		if envKey := strings.TrimSpace(os.Getenv(envVar)); envKey != "" {
			return envKey
		}
		if configDir != "" {
			keyFile := filepath.Join(configDir, fileName)
			if data, err := os.ReadFile(keyFile); err == nil {
				return strings.TrimSpace(string(data))
			}
		}
		return ""
	}

	cfg.SteamGridDBAPIKey = loadKey(cfg.SteamGridDBAPIKey, "STEAMGRIDDB_API_KEY", "steamgriddb_key.txt")
	cfg.SteamWebAPIKey = loadKey(cfg.SteamWebAPIKey, "STEAM_WEB_API_KEY", "steam_web_key.txt")
	cfg.HubcapAPIKey = loadKey(cfg.HubcapAPIKey, "HUBCAP_API_KEY", "hubcap_key.txt")
	cfg.RAWGAPIKey = loadKey(cfg.RAWGAPIKey, "RAWG_API_KEY", "rawg_key.txt")
}

// SaveConfig writes a Config struct cleanly and atomically to a JSON file on disk.
func SaveConfig(cfg *Config, path string) error {
	if cfg == nil {
		return nil
	}
	path = GetConfigPath(path)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return util.WriteFileAtomic(path, data, 0644)
}

// UpdateConfig loads the existing configuration from disk (or uses DefaultConfig if non-existent),
// executes a mutation callback to update specific fields, and writes the updated config atomically.
func UpdateConfig(path string, mutate func(cfg *Config) error) (*Config, error) {
	resolvedPath := GetConfigPath(path)
	cfg, err := LoadConfig(resolvedPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load config for update: %w", err)
	}
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if err := mutate(cfg); err != nil {
		return nil, fmt.Errorf("mutate config: %w", err)
	}
	if err := SaveConfig(cfg, resolvedPath); err != nil {
		return nil, fmt.Errorf("save updated config: %w", err)
	}
	return cfg, nil
}

// InitConfig initializes the configuration or loads it from a file.
func InitConfig(customPath string) error {
	_, err := LoadConfig(customPath)
	return err
}

