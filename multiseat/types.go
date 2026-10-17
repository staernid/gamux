package multiseat

// SeatMode defines the display/compositor target for a player seat.
type SeatMode string

const (
	SeatModeLocal  SeatMode = "local"
	SeatModeStream SeatMode = "stream"
)

// DisplayLayout defines the screen partitioning layout for local play.
type DisplayLayout string

const (
	LayoutDualMonitor  DisplayLayout = "dual_monitor"
	LayoutMultiMonitor DisplayLayout = "multi_monitor" // Alias/generalized multi-monitor
	LayoutSideBySide   DisplayLayout = "side_by_side"  // Vertical cut lines (N columns)
	LayoutTopBottom    DisplayLayout = "top_bottom"    // Horizontal cut lines (N rows)
	LayoutQuadGrid     DisplayLayout = "quad_grid"     // 2x2 quadrant grid (up to 4 players)
	LayoutThreePlayer  DisplayLayout = "three_player"  // 1 top / 2 bottom or 1 left / 2 right
	LayoutCustom       DisplayLayout = "custom"        // Explicit WindowX, WindowY, WindowWidth, WindowHeight
	LayoutWindowed     DisplayLayout = "windowed"
)

// InputDevice represents an enumerated host input device (gamepad/controller, keyboard, mouse).
type InputDevice struct {
	DevNode    string `json:"dev_node"`    // e.g. "/dev/input/event12"
	SysPath    string `json:"sys_path"`    // e.g. "/sys/class/input/event12"
	Name       string `json:"name"`        // e.g. "Xbox Wireless Controller"
	ID         string `json:"id"`          // Unique device identifier
	IsGamepad  bool   `json:"is_gamepad"`  // True if identified as a joystick/gamepad
	IsKeyboard bool   `json:"is_keyboard"` // True if identified as a keyboard
	IsMouse    bool   `json:"is_mouse"`    // True if identified as a mouse/pointer
	Bus        string `json:"bus"`         // "usb", "bluetooth", etc.
}

// PlayerProfile represents the emulated persona and identity for a seat.
type PlayerProfile struct {
	AccountName    string            `json:"account_name"`
	AccountSteamID string            `json:"account_steamid"`
	Language       string            `json:"language"`
	IPCountry      string            `json:"ip_country"`
	CustomEnv      map[string]string `json:"custom_env,omitempty"`
}

// SeatConfig defines the execution parameters for a single player seat.
type SeatConfig struct {
	SeatID          string        `json:"seat_id"` // "p1", "p2", "p3", "p4", etc.
	Player          PlayerProfile `json:"player"`
	ProfileID       string        `json:"profile_id,omitempty"` // Bound Gamux profile ID (e.g. "main", "p2", "p3")
	Mode            SeatMode      `json:"mode"`
	DisplayIndex    int           `json:"display_index"`          // Physical monitor index for gamescope
	DisplayName     string        `json:"display_name,omitempty"` // Physical output connector (e.g. "DP-1", "HDMI-A-1")
	TargetWidth     int           `json:"target_width,omitempty"`
	TargetHeight    int           `json:"target_height,omitempty"`
	TargetFPS       int           `json:"target_fps,omitempty"`
	WindowWidth     int           `json:"window_width,omitempty"`
	WindowHeight    int           `json:"window_height,omitempty"`
	WindowX         int           `json:"window_x,omitempty"`
	WindowY         int           `json:"window_y,omitempty"`
	ScreenSlot      *int          `json:"screen_slot,omitempty"`       // Explicit slot index for screen partitioning (0, 1, 2, 3...)
	GamepadNode     string        `json:"gamepad_node,omitempty"`      // Bound /dev/input/eventX node
	KeyboardNode    string        `json:"keyboard_node,omitempty"`     // Bound /dev/input/eventX keyboard node
	MouseNode       string        `json:"mouse_node,omitempty"`        // Bound /dev/input/eventX mouse node
	ForceGrabCursor bool          `json:"force_grab_cursor,omitempty"` // Tell gamescope to force grab cursor / relative mouse
	GrabKeyboard    bool          `json:"grab_keyboard,omitempty"`     // Tell gamescope to grab keyboard (-g)
	ExclusiveGrab   bool          `json:"exclusive_grab,omitempty"`    // Acquire exclusive kernel EVIOCGRAB on input nodes
	CursorPath      string        `json:"cursor_path,omitempty"`       // Custom cursor icon image path
	Compositor      string        `json:"compositor,omitempty"`        // "auto" | "gamescope" | "cage" | "cage_gamescope"
	NestCage        bool          `json:"nest_cage,omitempty"`         // Run gamescope nested inside cage compositor
	ShowCursor      bool          `json:"show_cursor,omitempty"`       // Force visible cursor via software cursor env (WLR_NO_HARDWARE_CURSORS=1)
	AudioSink       string        `json:"audio_sink,omitempty"`        // Assigned PipeWire sink name
	PhysicalSink    string        `json:"physical_sink,omitempty"`     // Physical output sink to route to
	WinePrefix      string        `json:"wine_prefix,omitempty"`
	HomeDir         string        `json:"home_dir,omitempty"`
	Disabled        bool          `json:"disabled,omitempty"`      // If true, this seat is temporarily benched/inactive
	SunshinePort    int           `json:"sunshine_port,omitempty"` // Dedicated streaming port for this seat
}

// MultiseatSessionConfig holds the full pairing configuration for launching an N-player session.
type MultiseatSessionConfig struct {
	GameDir        string        `json:"game_dir"`
	Name           string        `json:"name"`
	AppID          string        `json:"app_id,omitempty"`
	ExePath        string        `json:"exe_path"`
	ExeArgs        string        `json:"exe_args,omitempty"`
	Runner         string        `json:"runner,omitempty"` // "umu", "wine", "proton", "native"
	Layout         DisplayLayout `json:"layout"`
	Seats          []SeatConfig  `json:"seats,omitempty"`
	Seat1          SeatConfig    `json:"seat1,omitempty"`            // Backwards compatibility for 2-seat callers
	Seat2          SeatConfig    `json:"seat2,omitempty"`            // Backwards compatibility for 2-seat callers
	SunshinePortP1 int           `json:"sunshine_port_p1,omitempty"` // Base 47989
	SunshinePortP2 int           `json:"sunshine_port_p2,omitempty"` // Base 48010
}

// EffectiveSeats returns the normalized slice of active (non-disabled) SeatConfigs for this session.
func (c *MultiseatSessionConfig) EffectiveSeats() []SeatConfig {
	var raw []SeatConfig
	if len(c.Seats) > 0 {
		raw = c.Seats
	} else {
		if c.Seat1.SeatID != "" || c.Seat1.Player.AccountName != "" || c.Seat1.Player.AccountSteamID != "" {
			s1 := c.Seat1
			if s1.SeatID == "" {
				s1.SeatID = "p1"
			}
			raw = append(raw, s1)
		}
		if c.Seat2.SeatID != "" || c.Seat2.Player.AccountName != "" || c.Seat2.Player.AccountSteamID != "" {
			s2 := c.Seat2
			if s2.SeatID == "" {
				s2.SeatID = "p2"
			}
			raw = append(raw, s2)
		}
	}

	var active []SeatConfig
	for _, s := range raw {
		if !s.Disabled {
			active = append(active, s)
		}
	}
	return active
}

// SeatStatus reflects the runtime state of a single seat.
type SeatStatus struct {
	SeatID       string   `json:"seat_id"`
	ProfileID    string   `json:"profile_id,omitempty"`
	PID          int      `json:"pid"`
	Mode         SeatMode `json:"mode"`
	IsRunning    bool     `json:"is_running"`
	AudioSink    string   `json:"audio_sink,omitempty"`
	StreamURL    string   `json:"stream_url,omitempty"`
	WebUIURL     string   `json:"web_ui_url,omitempty"`
	GamepadNode  string   `json:"gamepad_node,omitempty"`
	KeyboardNode string   `json:"keyboard_node,omitempty"`
	MouseNode    string   `json:"mouse_node,omitempty"`
	ProcessGroup int      `json:"process_group,omitempty"`
}

// MultiseatSessionStatus holds active multi-seat telemetry.
type MultiseatSessionStatus struct {
	GameDir   string                `json:"game_dir"`
	IsRunning bool                  `json:"is_running"`
	Seats     map[string]SeatStatus `json:"seats,omitempty"`
	Seat1     SeatStatus            `json:"seat1"` // Backwards compatibility
	Seat2     SeatStatus            `json:"seat2"` // Backwards compatibility
	StartTime int64                 `json:"start_time"`
}

// SystemCapabilities reports available host tools for multi-seat support.
type SystemCapabilities struct {
	HasBwrap     bool `json:"has_bwrap"`
	HasGamescope bool `json:"has_gamescope"`
	HasCage      bool `json:"has_cage"`
	HasSunshine  bool `json:"has_sunshine"`
	HasPipewire  bool `json:"has_pipewire"`
}

// isExplicitDeviceNode reports whether a device node string points to a real device node path,
// rather than an empty value, "none", or "auto".
func isExplicitDeviceNode(node string) bool {
	node = ResolveDeviceNode(node)
	return node != "" && node != "none" && node != "auto"
}
