package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Default URL & Path Constants
const (
	DefaultSteamStoreAPI      = "https://store.steampowered.com/api"
	DefaultSteamWebAPI        = "https://api.steampowered.com"
	DefaultGithubAPIURL       = "https://api.github.com/repos/Detanup01/gbe_fork/releases/latest"
	DefaultSteamlessGithubAPI = "https://api.github.com/repos/staernid/steamless-rs/releases/latest"
)

// Config holds user-configurable paths and settings for gamux.
type Config struct {
	// Paths
	GbeDir             string `json:"gbe_dir"`
	SteamlessDir       string `json:"steamless_dir"`
	LutrisDir          string `json:"lutris_dir"`
	SteamUserdata      string `json:"steam_userdata"`
	SteamStoreAPI      string `json:"steam_store_api"`
	SteamWebAPI        string `json:"steam_web_api"`
	GithubAPIURL       string `json:"github_api_url"`
	SteamlessGithubAPI string `json:"steamless_github_api"`

	// API Keys
	SteamWebAPIKey string `json:"steam_web_api_key"`
	HubcapAPIKey   string `json:"hubcap_api_key"`

	// Notifications
	EnableLaunchNotify bool   `json:"enable_launch_notify"`
	LaunchNotifyMode   string `json:"launch_notify_mode"`

	// Operational Settings
	GbeMode      string `json:"gbe_mode"`
	Lutris       bool   `json:"lutris"`
	Steamless    bool   `json:"steamless"`
	Achievements bool   `json:"achievements"`
	Normalize    bool   `json:"normalize"`
	Platform     string `json:"platform"`
	Runner       string `json:"runner"`
	WinePrefix   string `json:"wine_prefix"`
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

	lutrisPath := ".config/lutris/games"
	if configHome != "" {
		lutrisPath = filepath.Join(configHome, "lutris", "games")
	}

	steamUserDataPath := ".local/share/Steam/userdata"
	if dataHome != "" {
		steamUserDataPath = filepath.Join(dataHome, "Steam", "userdata")
	}

	return &Config{
		GbeDir:             gbePath,
		SteamlessDir:       steamlessPath,
		LutrisDir:          lutrisPath,
		SteamUserdata:      steamUserDataPath,
		SteamStoreAPI:      DefaultSteamStoreAPI,
		SteamWebAPI:        DefaultSteamWebAPI,
		GithubAPIURL:       DefaultGithubAPIURL,
		SteamlessGithubAPI: DefaultSteamlessGithubAPI,
		SteamWebAPIKey:     "",
		HubcapAPIKey:       "",
		EnableLaunchNotify: true,
		LaunchNotifyMode:   "notify",
		GbeMode:            "loader",
		Lutris:             true,
		Steamless:          true,
		Achievements:       true,
		Normalize:          true,
		Platform:           "win64",
		Runner:             "wine",
		WinePrefix:         "",
	}
}

// GetConfigPath returns the absolute path to the configuration file.
func GetConfigPath(customPath string) string {
	if customPath != "" {
		return customPath
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
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}
