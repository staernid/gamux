package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

	// Notifications
	EnableLaunchNotify bool `json:"enable_launch_notify"`

	// Operational Settings
	OperatingMode      string            `json:"operating_mode,omitempty"`
	DefaultRunner      string            `json:"default_runner,omitempty"`
	ProtonPath         string            `json:"proton_path,omitempty"`
	Steamless          bool              `json:"steamless"`
	Achievements       bool              `json:"achievements"`
	Normalize          bool              `json:"normalize"`
	WinePrefix         string            `json:"wine_prefix,omitempty"`
	MangoHud           bool              `json:"mangohud,omitempty"`
	DXVKHud            string            `json:"dxvk_hud,omitempty"`
	WineFullscreenFSR  bool              `json:"wine_fullscreen_fsr,omitempty"`
	ProtonNVAPI        bool              `json:"proton_nvapi,omitempty"`
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
		OperatingMode:        "loader",
		DefaultRunner:        "umu",
		Steamless:            true,
		Achievements:         true,
		Normalize:            true,
		WinePrefix:           "",
		MangoHud:             false,
		WineFullscreenFSR:    false,
		ProtonNVAPI:          false,
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
	if cfg.SteamGridDBAPIKey == "" {
		if envKey := os.Getenv("STEAMGRIDDB_API_KEY"); envKey != "" {
			cfg.SteamGridDBAPIKey = envKey
		} else if home, err := os.UserHomeDir(); err == nil {
			keyFile := filepath.Join(home, ".config", "gamux", "steamgriddb_key.txt")
			if data, err := os.ReadFile(keyFile); err == nil {
				cfg.SteamGridDBAPIKey = string(data)
			}
		}
	}
}

// SaveConfig writes a Config struct cleanly to a JSON file on disk.
func SaveConfig(cfg *Config, path string) error {
	if cfg == nil {
		return nil
	}
	path = GetConfigPath(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// InitConfig initializes the configuration or loads it from a file.
func InitConfig(customPath string) error {
	_, err := LoadConfig(customPath)
	return err
}
