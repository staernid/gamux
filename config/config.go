package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Global Configuration fallback defaults
var (
	GbeDir             = ".local/share/gbe_fork"
	SteamlessDir       = ".local/share/steamless"
	SteamStoreAPI      = "https://store.steampowered.com/api"
	SteamWebAPI        = "https://api.steampowered.com"
	GithubAPIURL       = "https://api.github.com/repos/Detanup01/gbe_fork/releases/latest"
	SteamlessGithubAPI = "https://api.github.com/repos/staernid/steamless-rs/releases/latest"
	LutrisDir          = ".config/lutris/games"
	SteamUserdata      = ".local/share/Steam/userdata"
)

// Config holds user-configurable paths and settings for gamux.
type Config struct {
	GbeDir             string `json:"gbe_dir"`
	SteamlessDir       string `json:"steamless_dir"`
	LutrisDir          string `json:"lutris_dir"`
	SteamUserdata      string `json:"steam_userdata"`
	SteamStoreAPI      string `json:"steam_store_api"`
	GithubAPIURL       string `json:"github_api_url"`
	SteamlessGithubAPI string `json:"steamless_github_api"`
	SteamWebAPIKey     string `json:"steam_web_api_key"`
	HubcapAPIKey       string `json:"hubcap_api_key"`
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

	gbePath := GbeDir
	if dataHome != "" {
		gbePath = filepath.Join(dataHome, "gbe_fork")
	}

	steamlessPath := SteamlessDir
	if dataHome != "" {
		steamlessPath = filepath.Join(dataHome, "steamless")
	}

	lutrisPath := LutrisDir
	if configHome != "" {
		lutrisPath = filepath.Join(configHome, "lutris", "games")
	}

	steamUserDataPath := SteamUserdata
	if dataHome != "" {
		steamUserDataPath = filepath.Join(dataHome, "Steam", "userdata")
	}

	return &Config{
		GbeDir:             gbePath,
		SteamlessDir:       steamlessPath,
		LutrisDir:          lutrisPath,
		SteamUserdata:      steamUserDataPath,
		SteamStoreAPI:      SteamStoreAPI,
		GithubAPIURL:       GithubAPIURL,
		SteamlessGithubAPI: SteamlessGithubAPI,
		SteamWebAPIKey:     "",
		HubcapAPIKey:       "",
	}
}

// LoadConfig loads configuration from a JSON file or returns defaults.
func LoadConfig(customPath string) (*Config, error) {
	cfg := DefaultConfig()

	path := customPath
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return cfg, nil
		}
		path = filepath.Join(homeDir, ".config", "gamux", "config.json")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if customPath != "" {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		GbeDir             string `json:"gbe_dir"`
		SteamlessDir       string `json:"steamless_dir"`
		LutrisDir          string `json:"lutris_dir"`
		SteamUserdata      string `json:"steam_userdata"`
		SteamStoreAPI      string `json:"steam_store_api"`
		GithubAPIURL       string `json:"github_api_url"`
		SteamlessGithubAPI string `json:"steamless_github_api"`
		SteamWebAPIKey     string `json:"steam_web_api_key"`
		HubcapAPIKey       string `json:"hubcap_api_key"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	healed := false
	if raw.GbeDir != "" {
		cfg.GbeDir = raw.GbeDir
	} else {
		healed = true
	}
	if raw.SteamlessDir != "" {
		cfg.SteamlessDir = raw.SteamlessDir
	} else {
		healed = true
	}
	if raw.LutrisDir != "" {
		cfg.LutrisDir = raw.LutrisDir
	} else {
		healed = true
	}
	if raw.SteamUserdata != "" {
		cfg.SteamUserdata = raw.SteamUserdata
	} else {
		healed = true
	}
	if raw.SteamStoreAPI != "" {
		cfg.SteamStoreAPI = raw.SteamStoreAPI
	} else {
		healed = true
	}
	if raw.GithubAPIURL != "" {
		cfg.GithubAPIURL = raw.GithubAPIURL
	} else {
		healed = true
	}
	if raw.SteamWebAPIKey != "" {
		cfg.SteamWebAPIKey = raw.SteamWebAPIKey
	}
	if raw.HubcapAPIKey != "" {
		cfg.HubcapAPIKey = raw.HubcapAPIKey
	}

	if healed && customPath == "" {
		_ = SaveConfig(cfg, path)
	}

	// Update legacy globals for fallback compatibility
	GbeDir = cfg.GbeDir
	LutrisDir = cfg.LutrisDir
	SteamUserdata = cfg.SteamUserdata
	SteamStoreAPI = cfg.SteamStoreAPI
	GithubAPIURL = cfg.GithubAPIURL

	return cfg, nil
}

// SaveConfig writes a Config struct cleanly to a JSON file on disk.
func SaveConfig(cfg *Config, path string) error {
	if cfg == nil || path == "" {
		return nil
	}
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

// PlatformConfig maps platform names to their configuration.
var PlatformConfig = map[string]struct {
	Subdir, Target, Additional, Generator, Arch string
}{
	"linux": {
		Subdir:     "linux_release",
		Target:     "libsteam_api.so",
		Additional: "steamclient.so",
		Generator:  "generate_interfaces_x64",
		Arch:       "64",
	},
	"win64": {
		Subdir:     "win_release",
		Target:     "steam_api64.dll",
		Additional: "steamclient64.dll",
		Generator:  "generate_interfaces_x64.exe",
		Arch:       "64",
	},
	"win32": {
		Subdir:     "win_release",
		Target:     "steam_api.dll",
		Additional: "steamclient.dll",
		Generator:  "generate_interfaces_x32.exe",
		Arch:       "32",
	},
}

// Release represents a GitHub release.
type Release struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
}
