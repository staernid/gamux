package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Global Configuration
var (
	GbeDir        = ".local/share/gbe_fork"
	SteamStoreAPI = "https://store.steampowered.com/api"
	GithubAPIURL  = "https://api.github.com/repos/Detanup01/gbe_fork/releases/latest"
	LutrisDir     = ".config/lutris/games"
	SteamUserdata = ".local/share/Steam/userdata"
)

// InitConfig initializes the configuration or loads it from a file.
func InitConfig(customPath string) error {
	path := customPath
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(homeDir, ".config", "gbe_fork_helper", "config.json")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if customPath != "" {
			return err
		}
		return nil // Use defaults if default config file doesn't exist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var conf struct {
		GbeDir        string `json:"gbe_dir"`
		LutrisDir     string `json:"lutris_dir,omitempty"`
		SteamUserdata string `json:"steam_userdata,omitempty"`
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return err
	}

	if conf.GbeDir != "" {
		GbeDir = conf.GbeDir
	}
	if conf.LutrisDir != "" {
		LutrisDir = conf.LutrisDir
	}
	if conf.SteamUserdata != "" {
		SteamUserdata = conf.SteamUserdata
	}

	return nil
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
