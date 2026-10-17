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
	Gamescope          bool              `json:"gamescope,omitempty"`
	GamescopeWidth     int               `json:"gamescope_width,omitempty"`
	GamescopeHeight    int               `json:"gamescope_height,omitempty"`
	GamescopeFPS       int               `json:"gamescope_fps,omitempty"`
	TransferSpeedLimit int64             `json:"transfer_speed_limit,omitempty"`
	GlobalEnv          map[string]string `json:"global_env,omitempty"`
	InhibitSleep       bool              `json:"inhibit_sleep"`
	PreferSDL          bool              `json:"prefer_sdl,omitempty"`
	EnableWayland      bool              `json:"enable_wayland,omitempty"`

	// Emulation Profile Flags
	Language       string `json:"language,omitempty"`
	DisableOverlay bool   `json:"disable_overlay,omitempty"`
	OfflineMode    bool   `json:"offline_mode,omitempty"`

	// Save & Backup Settings
	AutoBackupSaves      bool   `json:"auto_backup_saves"`
	BackupRetentionCount int    `json:"backup_retention_count,omitempty"`
	BackupDir            string `json:"backup_dir,omitempty"`
	SaveManifestURL      string `json:"save_manifest_url,omitempty"`

	// Multi-Seat & Split-Screen Settings
	Multiseat MultiseatConfig `json:"multiseat,omitempty"`
}

// SeatPreference defines persistent preferences and device bindings for an individual player seat.
type SeatPreference struct {
	SeatID          string `json:"seat_id"`
	Mode            string `json:"mode,omitempty"`
	DisplayIndex    int    `json:"display_index"`
	ScreenSlot      int    `json:"screen_slot"`
	Disabled        bool   `json:"disabled,omitempty"`
	GamepadNode     string `json:"gamepad_node,omitempty"`
	KeyboardNode    string `json:"keyboard_node,omitempty"`
	MouseNode       string `json:"mouse_node,omitempty"`
	ForceGrabCursor bool   `json:"force_grab_cursor,omitempty"`
	GrabKeyboard    bool   `json:"grab_keyboard,omitempty"`
	Compositor      string `json:"compositor,omitempty"`
	NestCage        bool   `json:"nest_cage,omitempty"`
	ShowCursor      bool   `json:"show_cursor,omitempty"`
	ProfileID       string `json:"profile_id,omitempty"`
	SunshinePort    int    `json:"sunshine_port,omitempty"`
}

// MultiseatConfig holds preferences and layout/device options for 2-player local and streaming sessions.
type MultiseatConfig struct {
	Player2ProfileID     string           `json:"player2_profile_id,omitempty"`
	DefaultSeat1Mode     string           `json:"default_seat1_mode,omitempty"` // "local" | "stream"
	DefaultSeat2Mode     string           `json:"default_seat2_mode,omitempty"` // "local" | "stream"
	DefaultLayout        string           `json:"default_layout,omitempty"`     // "dual_monitor" | "side_by_side" | "top_bottom" | "quad_grid" | "three_player" | "windowed"
	Seat1Gamepad         string           `json:"seat1_gamepad,omitempty"`
	Seat2Gamepad         string           `json:"seat2_gamepad,omitempty"`
	Seat1Keyboard        string           `json:"seat1_keyboard,omitempty"`
	Seat2Keyboard        string           `json:"seat2_keyboard,omitempty"`
	Seat1Mouse           string           `json:"seat1_mouse,omitempty"`
	Seat2Mouse           string           `json:"seat2_mouse,omitempty"`
	Seat1ForceGrabCursor bool             `json:"seat1_force_grab_cursor"`
	Seat1GrabKeyboard    bool             `json:"seat1_grab_keyboard"`
	Seat1ShowCursor      bool             `json:"seat1_show_cursor"`
	Seat2ForceGrabCursor bool             `json:"seat2_force_grab_cursor,omitempty"`
	Seat2GrabKeyboard    bool             `json:"seat2_grab_keyboard,omitempty"`
	Seat2Compositor      string           `json:"seat2_compositor,omitempty"`
	Seat2NestCage        bool             `json:"seat2_nest_cage,omitempty"`
	Seat2ShowCursor      bool             `json:"seat2_show_cursor"`
	Seat1DisplayIndex    int              `json:"seat1_display_index"`
	Seat2DisplayIndex    int              `json:"seat2_display_index"`
	SunshinePortP1       int              `json:"sunshine_port_p1,omitempty"`
	SunshinePortP2       int              `json:"sunshine_port_p2,omitempty"`
	Seats                []SeatPreference `json:"seats,omitempty"`
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

	cfg := &Config{
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
		InhibitSleep:         true,
		PreferSDL:            false,
		EnableWayland:        false,
		Multiseat: MultiseatConfig{
			Player2ProfileID:     "p2",
			DefaultSeat1Mode:     "local",
			DefaultSeat2Mode:     "local",
			DefaultLayout:        "dual_monitor",
			Seat1Gamepad:         "auto",
			Seat2Gamepad:         "auto",
			Seat1Keyboard:        "auto",
			Seat2Keyboard:        "auto",
			Seat1Mouse:           "auto",
			Seat2Mouse:           "auto",
			Seat1ForceGrabCursor: false,
			Seat1GrabKeyboard:    false,
			Seat1ShowCursor:      true,
			Seat2ForceGrabCursor: false,
			Seat2GrabKeyboard:    false,
			Seat2Compositor:      "auto",
			Seat2NestCage:        false,
			Seat2ShowCursor:      true,
			Seat1DisplayIndex:    0,
			Seat2DisplayIndex:    1,
			SunshinePortP1:       47989,
			SunshinePortP2:       48010,
		},
	}
	syncMultiseatSeats(&cfg.Multiseat)
	return cfg
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

	// Migrate legacy layout names directly (AGENTS.md rule 6)
	if cfg.Multiseat.DefaultLayout == "horizontal_split" {
		cfg.Multiseat.DefaultLayout = "side_by_side"
	} else if cfg.Multiseat.DefaultLayout == "vertical_split" {
		cfg.Multiseat.DefaultLayout = "top_bottom"
	}
	if cfg.Multiseat.Seat2Compositor == "" {
		cfg.Multiseat.Seat2Compositor = "auto"
	}

	syncMultiseatSeats(&cfg.Multiseat)
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
	syncMultiseatSeats(&cfg.Multiseat)
	path = GetConfigPath(path)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return util.WriteFileAtomic(path, data, 0644)
}

// syncMultiseatSeats ensures that the Multiseat.Seats slice and top-level Multiseat fields stay bidirectionally synchronized.
func syncMultiseatSeats(ms *MultiseatConfig) {
	if ms == nil {
		return
	}
	if len(ms.Seats) == 0 {
		p2Prof := ms.Player2ProfileID
		if p2Prof == "" {
			p2Prof = "p2"
		}
		s1Mode := ms.DefaultSeat1Mode
		if s1Mode == "" {
			s1Mode = "local"
		}
		s2Mode := ms.DefaultSeat2Mode
		if s2Mode == "" {
			s2Mode = "local"
		}
		p1Port := ms.SunshinePortP1
		if p1Port == 0 {
			p1Port = 47989
		}
		p2Port := ms.SunshinePortP2
		if p2Port == 0 {
			p2Port = 48010
		}
		ms.Seats = []SeatPreference{
			{
				SeatID:          "p1",
				Mode:            s1Mode,
				DisplayIndex:    ms.Seat1DisplayIndex,
				ScreenSlot:      0,
				Disabled:        false,
				GamepadNode:     ms.Seat1Gamepad,
				KeyboardNode:    ms.Seat1Keyboard,
				MouseNode:       ms.Seat1Mouse,
				ForceGrabCursor: ms.Seat1ForceGrabCursor,
				GrabKeyboard:    ms.Seat1GrabKeyboard,
				ShowCursor:      ms.Seat1ShowCursor,
				ProfileID:       "main",
				SunshinePort:    p1Port,
			},
			{
				SeatID:          "p2",
				Mode:            s2Mode,
				DisplayIndex:    ms.Seat2DisplayIndex,
				ScreenSlot:      1,
				Disabled:        false,
				GamepadNode:     ms.Seat2Gamepad,
				KeyboardNode:    ms.Seat2Keyboard,
				MouseNode:       ms.Seat2Mouse,
				ForceGrabCursor: false,
				GrabKeyboard:    false,
				Compositor:      "gamescope",
				NestCage:        false,
				ShowCursor:      ms.Seat2ShowCursor,
				ProfileID:       p2Prof,
				SunshinePort:    p2Port,
			},
		}
		return
	}

	// When Seats has at least 1 seat, harmonize seat 1
	if len(ms.Seats) > 0 {
		s1 := &ms.Seats[0]
		if ms.DefaultSeat1Mode != "" {
			s1.Mode = ms.DefaultSeat1Mode
		} else if s1.Mode != "" {
			ms.DefaultSeat1Mode = s1.Mode
		}
		s1.DisplayIndex = ms.Seat1DisplayIndex
		if ms.Seat1Gamepad != "" {
			s1.GamepadNode = ms.Seat1Gamepad
		} else if s1.GamepadNode != "" {
			ms.Seat1Gamepad = s1.GamepadNode
		}
		if ms.Seat1Keyboard != "" {
			s1.KeyboardNode = ms.Seat1Keyboard
		} else if s1.KeyboardNode != "" {
			ms.Seat1Keyboard = s1.KeyboardNode
		}
		if ms.Seat1Mouse != "" {
			s1.MouseNode = ms.Seat1Mouse
		} else if s1.MouseNode != "" {
			ms.Seat1Mouse = s1.MouseNode
		}
		s1.ForceGrabCursor = ms.Seat1ForceGrabCursor
		s1.GrabKeyboard = ms.Seat1GrabKeyboard
		s1.ShowCursor = ms.Seat1ShowCursor
		if ms.SunshinePortP1 != 0 {
			s1.SunshinePort = ms.SunshinePortP1
		} else if s1.SunshinePort != 0 {
			ms.SunshinePortP1 = s1.SunshinePort
		}
	}

	// When Seats has at least 2 seats, harmonize seat 2 and guest seats
	if len(ms.Seats) > 1 {
		s1 := &ms.Seats[0]
		s2 := &ms.Seats[1]
		if s1.DisplayIndex == s2.DisplayIndex && s1.ScreenSlot == s2.ScreenSlot {
			if s1.ScreenSlot == 1 {
				s2.ScreenSlot = 0
			} else {
				s2.ScreenSlot = 1
			}
		}
		if ms.Player2ProfileID != "" {
			s2.ProfileID = ms.Player2ProfileID
		} else if s2.ProfileID != "" {
			ms.Player2ProfileID = s2.ProfileID
		}
		if ms.DefaultSeat2Mode != "" {
			s2.Mode = ms.DefaultSeat2Mode
		} else if s2.Mode != "" {
			ms.DefaultSeat2Mode = s2.Mode
		}
		s2.DisplayIndex = ms.Seat2DisplayIndex
		if ms.Seat2Gamepad != "" {
			s2.GamepadNode = ms.Seat2Gamepad
		} else if s2.GamepadNode != "" {
			ms.Seat2Gamepad = s2.GamepadNode
		}
		if ms.Seat2Keyboard != "" {
			s2.KeyboardNode = ms.Seat2Keyboard
		} else if s2.KeyboardNode != "" {
			ms.Seat2Keyboard = s2.KeyboardNode
		}
		if ms.Seat2Mouse != "" {
			s2.MouseNode = ms.Seat2Mouse
		} else if s2.MouseNode != "" {
			ms.Seat2Mouse = s2.MouseNode
		}
		if ms.SunshinePortP2 != 0 {
			s2.SunshinePort = ms.SunshinePortP2
		} else if s2.SunshinePort != 0 {
			ms.SunshinePortP2 = s2.SunshinePort
		}

		// Non-main (guest) seats use Gamescope, no desktop cursor grab, and share guest show_cursor setting
		for i := 1; i < len(ms.Seats); i++ {
			sg := &ms.Seats[i]
			sg.ForceGrabCursor = false
			sg.GrabKeyboard = false
			sg.Compositor = "gamescope"
			sg.NestCage = false
			sg.ShowCursor = ms.Seat2ShowCursor
		}
	}
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

// ResolvedLibraryDirs returns all configured non-empty library paths, falling back to ~/Games if none configured and ~/Games exists.
func (cfg *Config) ResolvedLibraryDirs() []string {
	var dirs []string
	if cfg != nil && len(cfg.LibraryPaths) > 0 {
		for _, p := range cfg.LibraryPaths {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				dirs = append(dirs, trimmed)
			}
		}
	}
	if len(dirs) == 0 {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			defaultGames := filepath.Join(home, "Games")
			if util.FileExists(defaultGames) {
				dirs = append(dirs, defaultGames)
			}
		}
	}
	return dirs
}
