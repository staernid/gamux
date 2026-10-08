package config

// PropertyScope represents the target configuration domain for a property.
type PropertyScope string

const (
	ScopeGlobal PropertyScope = "global"
	ScopeGame   PropertyScope = "game"
	ScopeBoth   PropertyScope = "both"
)

// PropertyOption represents a selectable option in an enumeration/select setting.
type PropertyOption struct {
	Value any    `json:"value"`
	Label string `json:"label"`
}

// PropertyDescriptor describes metadata, constraints, and presentation for a configuration property.
type PropertyDescriptor struct {
	Key          string           `json:"key"`
	Label        string           `json:"label"`
	Description  string           `json:"description"`
	Type         string           `json:"type"`     // "boolean", "select", "path", "path_list", "env", "string", "number"
	Category     string           `json:"category"` // "Paths", "API Keys", "Lifecycle", "Operational", "Emulation", "Saves & Backups", "Multi-Seat", "Runtime", "Graphics"
	Scope        PropertyScope    `json:"scope"`    // "global" | "game" | "both"
	Default      any              `json:"default"`
	Options      []PropertyOption `json:"options,omitempty"`
	InheritsFrom string           `json:"inherits_from,omitempty"`
	Min          *float64         `json:"min,omitempty"`
	Max          *float64         `json:"max,omitempty"`
	Step         *float64         `json:"step,omitempty"`
}

func floatPtr(v float64) *float64 {
	return &v
}

// masterSchema is the single canonical source of truth for all Gamux configuration descriptors.
var masterSchema = []PropertyDescriptor{
	// -------------------------------------------------------------------------
	// Paths & Storage (Global Only)
	// -------------------------------------------------------------------------
	{
		Key:         "library_paths",
		Label:       "Library Folders",
		Description: "Directories where Gamux scans and discovers installed game directories.",
		Type:        "path_list",
		Category:    "Paths",
		Scope:       ScopeGlobal,
		Default:     []string{},
	},
	{
		Key:         "gbe_dir",
		Label:       "Goldberg Emulator Directory",
		Description: "Path to Goldberg Steam Emulator binaries and overlay libraries.",
		Type:        "path",
		Category:    "Paths",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "steamless_dir",
		Label:       "Steamless Directory",
		Description: "Path to Steamless unpacker CLI tools for unpacking SteamStub DRM.",
		Type:        "path",
		Category:    "Paths",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "steam_userdata",
		Label:       "Steam Userdata Directory",
		Description: "Path to the host Steam client userdata folder for save synchronisation.",
		Type:        "path",
		Category:    "Paths",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "artwork_dir",
		Label:       "Artwork Cache Directory",
		Description: "Directory where downloaded posters, banners, and grid artwork are stored.",
		Type:        "path",
		Category:    "Paths",
		Scope:       ScopeGlobal,
		Default:     "",
	},

	// -------------------------------------------------------------------------
	// API Keys (Global Only)
	// -------------------------------------------------------------------------
	{
		Key:         "steam_web_api_key",
		Label:       "Steam Web API Key",
		Description: "API key used to query the official Steam Web API for store details and global stats.",
		Type:        "string",
		Category:    "API Keys",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "hubcap_api_key",
		Label:       "Hubcap API Key",
		Description: "API key used to retrieve community manifests and metadata via Hubcap.",
		Type:        "string",
		Category:    "API Keys",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "steamgriddb_api_key",
		Label:       "SteamGridDB API Key",
		Description: "API key for SteamGridDB to fetch custom high-res box art, grids, and hero graphics.",
		Type:        "string",
		Category:    "API Keys",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "rawg_api_key",
		Label:       "RAWG API Key",
		Description: "API key for RAWG Video Games Database to enrich metadata and descriptions.",
		Type:        "string",
		Category:    "API Keys",
		Scope:       ScopeGlobal,
		Default:     "",
	},

	// -------------------------------------------------------------------------
	// Notifications & Process Lifecycle (Global Only)
	// -------------------------------------------------------------------------
	{
		Key:         "enable_launch_notify",
		Label:       "Launch Notifications",
		Description: "Display a desktop notification when a game process launches or exits.",
		Type:        "boolean",
		Category:    "Lifecycle",
		Scope:       ScopeGlobal,
		Default:     true,
	},
	{
		Key:         "launch_window_behavior",
		Label:       "Launch Window Behavior",
		Description: "Action taken with the Gamux main window when a game is launched.",
		Type:        "select",
		Category:    "Lifecycle",
		Scope:       ScopeGlobal,
		Default:     "keep",
		Options: []PropertyOption{
			{Value: "keep", Label: "Keep Window Open"},
			{Value: "minimize", Label: "Minimize to Taskbar"},
			{Value: "hide", Label: "Hide Window (Restore on Exit)"},
		},
	},
	{
		Key:         "close_to_tray",
		Label:       "Close to System Tray",
		Description: "Minimize application to system tray when the main window is closed.",
		Type:        "boolean",
		Category:    "Lifecycle",
		Scope:       ScopeGlobal,
		Default:     false,
	},

	// -------------------------------------------------------------------------
	// Operational & Runners
	// -------------------------------------------------------------------------
	{
		Key:         "operating_mode",
		Label:       "Operating Mode",
		Description: "Default integration method for applying Goldberg emulator hooks.",
		Type:        "select",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     "loader",
		Options: []PropertyOption{
			{Value: "loader", Label: "Loader Mode (Clean / Non-Invasive)"},
			{Value: "portable", Label: "Portable Mode (Injected DLL Replacement)"},
			{Value: "none", Label: "DRM-Free / None"},
		},
	},
	{
		Key:         "default_runner",
		Label:       "Default Runner",
		Description: "Default execution environment for running Windows executables on Linux.",
		Type:        "select",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     "umu",
		Options: []PropertyOption{
			{Value: "umu", Label: "UMU (Unified Linux Wine/Proton Runtime)"},
			{Value: "wine", Label: "Wine (System Wine Binary)"},
		},
	},
	{
		Key:          "runner",
		Label:        "Runner Layer",
		Description:  "Execution environment used to run this title (UMU, Wine, Proton, Linux native).",
		Type:         "select",
		Category:     "Runtime",
		Scope:        ScopeGame,
		Default:      "umu",
		InheritsFrom: "default_runner",
		Options: []PropertyOption{
			{Value: "umu", Label: "UMU (Unified Linux Wine/Proton Runtime)"},
			{Value: "wine", Label: "Wine (System Wine Binary)"},
			{Value: "proton", Label: "Proton"},
			{Value: "linux", Label: "Linux Native"},
		},
	},
	{
		Key:         "proton_path",
		Label:       "Proton Version / Path",
		Description: "Proton compatibility tool alias or custom path (e.g. 'Proton-GE Latest', 'Proton-CachyOS Latest', or empty for default).",
		Type:        "select",
		Category:    "Runtime",
		Scope:       ScopeGame,
		Default:     "",
		Options: []PropertyOption{
			{Value: "", Label: "Default (UMU Auto / Managed)"},
			{Value: "Proton-GE Latest", Label: "Proton-GE Latest (GloriousEggroll)"},
			{Value: "Proton-CachyOS Latest", Label: "Proton-CachyOS Latest (CachyOS)"},
		},
	},
	{
		Key:          "prefer_sdl",
		Label:        "Prefer SDL Gamepad Driver",
		Description:  "Set PROTON_PREFER_SDL=1 to route controller inputs via SDL2 rather than DirectInput/RawInput.",
		Type:         "boolean",
		Category:     "Runtime",
		Scope:        ScopeBoth,
		Default:      false,
		InheritsFrom: "prefer_sdl",
	},
	{
		Key:         "steamless",
		Label:       "Unpack Executables (Steamless)",
		Description: "Automatically unpack SteamStub DRM protection during game processing.",
		Type:        "boolean",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     true,
	},
	{
		Key:         "achievements",
		Label:       "Achievements Tracking",
		Description: "Track unlock progress and trigger sound notifications when achievements occur.",
		Type:        "boolean",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     true,
	},
	{
		Key:         "normalize",
		Label:       "Normalize File Names",
		Description: "Clean and standardize file casing and names during game imports.",
		Type:        "boolean",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     true,
	},
	{
		Key:          "inhibit_sleep",
		Label:        "Inhibit System Sleep",
		Description:  "Prevent system screensaver and idle suspension while the game is running.",
		Type:         "boolean",
		Category:     "Operational",
		Scope:        ScopeBoth,
		Default:      true,
		InheritsFrom: "inhibit_sleep",
	},
	{
		Key:          "wine_prefix",
		Label:        "Wine Prefix",
		Description:  "Wine prefix directory path for Windows executables (leave empty for system default).",
		Type:         "path",
		Category:     "Operational",
		Scope:        ScopeBoth,
		Default:      "",
		InheritsFrom: "wine_prefix",
	},
	{
		Key:          "mangohud",
		Label:        "MangoHud Overlay",
		Description:  "Enable MangoHud performance and frame rate overlay for launched games.",
		Type:         "boolean",
		Category:     "Graphics",
		Scope:        ScopeBoth,
		Default:      false,
		InheritsFrom: "mangohud",
	},
	{
		Key:          "gamescope",
		Label:        "Enable Gamescope",
		Description:  "Run game inside Valve's Gamescope micro-compositor for resolution scaling and HDR.",
		Type:         "boolean",
		Category:     "Graphics",
		Scope:        ScopeBoth,
		Default:      false,
		InheritsFrom: "gamescope",
	},
	{
		Key:         "gamescope_width",
		Label:       "Gamescope Resolution Width",
		Description: "Render width in pixels inside Gamescope (0 for automatic detection).",
		Type:        "number",
		Category:    "Graphics",
		Scope:       ScopeGlobal,
		Default:     0,
		Min:         floatPtr(0),
		Max:         floatPtr(7680),
		Step:        floatPtr(1),
	},
	{
		Key:         "gamescope_height",
		Label:       "Gamescope Resolution Height",
		Description: "Render height in pixels inside Gamescope (0 for automatic detection).",
		Type:        "number",
		Category:    "Graphics",
		Scope:       ScopeGlobal,
		Default:     0,
		Min:         floatPtr(0),
		Max:         floatPtr(4320),
		Step:        floatPtr(1),
	},
	{
		Key:          "gamescope_fps",
		Label:        "Gamescope Framerate Limit",
		Description:  "Maximum target framerate cap for Gamescope (0 for unlimited).",
		Type:         "number",
		Category:     "Graphics",
		Scope:        ScopeBoth,
		Default:      0,
		InheritsFrom: "gamescope_fps",
		Min:          floatPtr(0),
		Max:          floatPtr(360),
		Step:         floatPtr(1),
	},
	{
		Key:          "enable_wayland",
		Label:        "Enable Wayland Driver",
		Description:  "Set PROTON_ENABLE_WAYLAND=1 to use the native Wine Wayland graphics driver instead of XWayland.",
		Type:         "boolean",
		Category:     "Graphics",
		Scope:        ScopeBoth,
		Default:      false,
		InheritsFrom: "enable_wayland",
	},
	{
		Key:         "transfer_speed_limit",
		Label:       "Download Speed Limit (KB/s)",
		Description: "Global bandwidth limit for downloads in kilobytes per second (0 for unlimited).",
		Type:        "number",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     int64(0),
		Min:         floatPtr(0),
		Max:         floatPtr(1000000),
		Step:        floatPtr(100),
	},
	{
		Key:         "global_env",
		Label:       "Global Environment Variables",
		Description: "Key-value environment variables passed to all launched games.",
		Type:        "env",
		Category:    "Operational",
		Scope:       ScopeGlobal,
		Default:     map[string]string{},
	},

	// -------------------------------------------------------------------------
	// Game-Specific Overrides (Game Only)
	// -------------------------------------------------------------------------
	{
		Key:         "exe_args",
		Label:       "Launch Arguments",
		Description: "Command line arguments passed directly to the game binary.",
		Type:        "string",
		Category:    "Runtime",
		Scope:       ScopeGame,
		Default:     "",
	},
	{
		Key:         "dll_overrides",
		Label:       "Wine DLL Overrides",
		Description: "Custom WINEDLLOVERRIDES string (e.g. 'd3d11=n,b;dxgi=n,b').",
		Type:        "string",
		Category:    "Runtime",
		Scope:       ScopeGame,
		Default:     "",
	},
	{
		Key:         "custom_env",
		Label:       "Environment Variables",
		Description: "Title-specific environment variables passed at runtime.",
		Type:        "env",
		Category:    "Runtime",
		Scope:       ScopeGame,
		Default:     map[string]string{},
	},

	// -------------------------------------------------------------------------
	// Emulation Profile Flags (Scope: Both)
	// -------------------------------------------------------------------------
	{
		Key:          "language",
		Label:        "Game Language",
		Description:  "Language reported to games via Goldberg steam_api emulation.",
		Type:         "select",
		Category:     "Emulation",
		Scope:        ScopeBoth,
		Default:      "auto",
		InheritsFrom: "language",
		Options: []PropertyOption{
			{Value: "auto", Label: "Auto (System Locale)"},
			{Value: "english", Label: "English"},
			{Value: "french", Label: "French"},
			{Value: "german", Label: "German"},
			{Value: "italian", Label: "Italian"},
			{Value: "spanish", Label: "Spanish"},
			{Value: "japanese", Label: "Japanese"},
			{Value: "schinese", Label: "Simplified Chinese"},
			{Value: "russian", Label: "Russian"},
			{Value: "brazilian", Label: "Brazilian Portuguese"},
			{Value: "koreana", Label: "Korean"},
			{Value: "polish", Label: "Polish"},
		},
	},
	{
		Key:          "disable_overlay",
		Label:        "Disable Steam Overlay",
		Description:  "Prevent Goldberg from hooking or injecting the Steam overlay.",
		Type:         "boolean",
		Category:     "Emulation",
		Scope:        ScopeBoth,
		Default:      false,
		InheritsFrom: "disable_overlay",
	},
	{
		Key:          "offline_mode",
		Label:        "Force Offline Mode",
		Description:  "Force Goldberg emulator into offline mode without LAN matchmaking.",
		Type:         "boolean",
		Category:     "Emulation",
		Scope:        ScopeBoth,
		Default:      false,
		InheritsFrom: "offline_mode",
	},

	// -------------------------------------------------------------------------
	// Save & Backup Settings (Global Only)
	// -------------------------------------------------------------------------
	{
		Key:         "auto_backup_saves",
		Label:       "Auto-Backup Saves",
		Description: "Automatically take snapshot archives of save files when a game process exits.",
		Type:        "boolean",
		Category:    "Saves & Backups",
		Scope:       ScopeGlobal,
		Default:     false,
	},
	{
		Key:         "backup_retention_count",
		Label:       "Backup Retention Count",
		Description: "Maximum number of save snapshots to retain per game before pruning oldest.",
		Type:        "select",
		Category:    "Saves & Backups",
		Scope:       ScopeGlobal,
		Default:     5,
		Options: []PropertyOption{
			{Value: 1, Label: "1 backup"},
			{Value: 2, Label: "2 backups"},
			{Value: 3, Label: "3 backups"},
			{Value: 5, Label: "5 backups (Recommended)"},
			{Value: 10, Label: "10 backups"},
			{Value: 15, Label: "15 backups"},
			{Value: 20, Label: "20 backups"},
			{Value: 30, Label: "30 backups"},
		},
	},
	{
		Key:         "backup_dir",
		Label:       "Save Backup Directory",
		Description: "Directory path where historical save backups are archived.",
		Type:        "path",
		Category:    "Saves & Backups",
		Scope:       ScopeGlobal,
		Default:     "",
	},
	{
		Key:         "save_manifest_url",
		Label:       "Custom Save Manifest URL",
		Description: "URL to remote community save game pattern definitions and schemas.",
		Type:        "string",
		Category:    "Saves & Backups",
		Scope:       ScopeGlobal,
		Default:     "",
	},

	// -------------------------------------------------------------------------
	// Multi-Seat & Split-Screen Settings (Global Only)
	// -------------------------------------------------------------------------
	{
		Key:         "multiseat.player2_profile_id",
		Label:       "Seat 2 Profile ID",
		Description: "Profile ID designated for Player 2 during local co-op and streaming sessions.",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "p2",
	},
	{
		Key:         "multiseat.default_seat1_mode",
		Label:       "Seat 1 Mode",
		Description: "Display mode for Player 1: local monitor or Sunshine network stream.",
		Type:        "select",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "local",
		Options: []PropertyOption{
			{Value: "local", Label: "Local Monitor"},
			{Value: "stream", Label: "Sunshine Stream"},
		},
	},
	{
		Key:         "multiseat.default_seat2_mode",
		Label:       "Seat 2 Mode",
		Description: "Display mode for Player 2: local monitor or Sunshine network stream.",
		Type:        "select",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "local",
		Options: []PropertyOption{
			{Value: "local", Label: "Local Monitor"},
			{Value: "stream", Label: "Sunshine Stream"},
		},
	},
	{
		Key:         "multiseat.default_layout",
		Label:       "Multi-Seat Layout",
		Description: "Screen layout arrangement for dual local players.",
		Type:        "select",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "dual_monitor",
		Options: []PropertyOption{
			{Value: "dual_monitor", Label: "Dual Dedicated Monitors"},
			{Value: "side_by_side", Label: "Side-by-Side (Horizontal Split)"},
			{Value: "top_bottom", Label: "Top & Bottom (Vertical Split)"},
			{Value: "windowed", Label: "Windowed Overlay"},
		},
	},
	{
		Key:         "multiseat.seat1_gamepad",
		Label:       "Seat 1 Gamepad",
		Description: "Gamepad device node assignment for Seat 1 (or 'auto').",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
	},
	{
		Key:         "multiseat.seat2_gamepad",
		Label:       "Seat 2 Gamepad",
		Description: "Gamepad device node assignment for Seat 2 (or 'auto').",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
	},
	{
		Key:         "multiseat.seat1_keyboard",
		Label:       "Seat 1 Keyboard",
		Description: "Keyboard device node assignment for Seat 1 (or 'auto').",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
	},
	{
		Key:         "multiseat.seat2_keyboard",
		Label:       "Seat 2 Keyboard",
		Description: "Keyboard device node assignment for Seat 2 (or 'auto').",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
	},
	{
		Key:         "multiseat.seat1_mouse",
		Label:       "Seat 1 Mouse",
		Description: "Mouse device node assignment for Seat 1 (or 'auto').",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
	},
	{
		Key:         "multiseat.seat2_mouse",
		Label:       "Seat 2 Mouse",
		Description: "Mouse device node assignment for Seat 2 (or 'auto').",
		Type:        "string",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
	},
	{
		Key:         "multiseat.seat2_force_grab_cursor",
		Label:       "Seat 2 Force Grab Cursor",
		Description: "Confine Seat 2 mouse cursor within Seat 2 window boundaries.",
		Type:        "boolean",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     false,
	},
	{
		Key:         "multiseat.seat2_grab_keyboard",
		Label:       "Seat 2 Grab Keyboard",
		Description: "Exclusively grab evdev keyboard device for Seat 2 input isolation.",
		Type:        "boolean",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     false,
	},
	{
		Key:         "multiseat.seat2_compositor",
		Label:       "Seat 2 Compositor",
		Description: "Nested Wayland compositor engine used to isolate Seat 2.",
		Type:        "select",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     "auto",
		Options: []PropertyOption{
			{Value: "auto", Label: "Auto (Best Available)"},
			{Value: "gamescope", Label: "Gamescope"},
			{Value: "cage", Label: "Cage Kiosk"},
			{Value: "kwin", Label: "Nested KWin"},
		},
	},
	{
		Key:         "multiseat.seat2_nest_cage",
		Label:       "Nest in Cage",
		Description: "Run Seat 2 inside an isolated Cage kiosk instance.",
		Type:        "boolean",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     false,
	},
	{
		Key:         "multiseat.seat2_show_cursor",
		Label:       "Seat 2 Show Cursor",
		Description: "Display visible cursor overlay inside Seat 2 compositor.",
		Type:        "boolean",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     true,
	},
	{
		Key:         "multiseat.seat1_display_index",
		Label:       "Seat 1 Display Index",
		Description: "Target physical monitor index for Player 1 (0-indexed).",
		Type:        "number",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     0,
		Min:         floatPtr(0),
		Max:         floatPtr(16),
		Step:        floatPtr(1),
	},
	{
		Key:         "multiseat.seat2_display_index",
		Label:       "Seat 2 Display Index",
		Description: "Target physical monitor index for Player 2 (0-indexed).",
		Type:        "number",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     1,
		Min:         floatPtr(0),
		Max:         floatPtr(16),
		Step:        floatPtr(1),
	},
	{
		Key:         "multiseat.sunshine_port_p1",
		Label:       "Sunshine Port P1",
		Description: "Base web & streaming port for Player 1 Sunshine server.",
		Type:        "number",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     47989,
		Min:         floatPtr(1024),
		Max:         floatPtr(65535),
		Step:        floatPtr(1),
	},
	{
		Key:         "multiseat.sunshine_port_p2",
		Label:       "Sunshine Port P2",
		Description: "Base web & streaming port for Player 2 Sunshine server.",
		Type:        "number",
		Category:    "Multi-Seat",
		Scope:       ScopeGlobal,
		Default:     48010,
		Min:         floatPtr(1024),
		Max:         floatPtr(65535),
		Step:        floatPtr(1),
	},
}

// GetConfigSchema returns the server-driven property descriptor schema for all global configuration fields.
func GetConfigSchema() []PropertyDescriptor {
	var result []PropertyDescriptor
	for _, prop := range masterSchema {
		if prop.Scope == ScopeGlobal || prop.Scope == ScopeBoth {
			result = append(result, prop)
		}
	}
	return result
}

// GetGameConfigSchema returns the server-driven property descriptor schema for game-level launch profile overrides.
func GetGameConfigSchema() []PropertyDescriptor {
	var result []PropertyDescriptor
	for _, prop := range masterSchema {
		if prop.Scope == ScopeGame || prop.Scope == ScopeBoth {
			result = append(result, prop)
		}
	}
	return result
}
