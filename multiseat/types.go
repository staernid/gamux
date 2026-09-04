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
	LayoutDualMonitor DisplayLayout = "dual_monitor"
	LayoutSideBySide  DisplayLayout = "side_by_side" // Left / Right (Vertical cut line)
	LayoutTopBottom   DisplayLayout = "top_bottom"   // Top / Bottom (Horizontal cut line)
	LayoutWindowed    DisplayLayout = "windowed"
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
	SeatID          string        `json:"seat_id"` // "p1" | "p2"
	Player          PlayerProfile `json:"player"`
	Mode            SeatMode      `json:"mode"`
	DisplayIndex    int           `json:"display_index"` // Physical monitor index for gamescope
	TargetWidth     int           `json:"target_width,omitempty"`
	TargetHeight    int           `json:"target_height,omitempty"`
	TargetFPS       int           `json:"target_fps,omitempty"`
	WindowWidth     int           `json:"window_width,omitempty"`
	WindowHeight    int           `json:"window_height,omitempty"`
	WindowX         int           `json:"window_x,omitempty"`
	WindowY         int           `json:"window_y,omitempty"`
	GamepadNode     string        `json:"gamepad_node,omitempty"`     // Bound /dev/input/eventX node
	KeyboardNode    string        `json:"keyboard_node,omitempty"`    // Bound /dev/input/eventX keyboard node
	MouseNode       string        `json:"mouse_node,omitempty"`       // Bound /dev/input/eventX mouse node
	ForceGrabCursor bool          `json:"force_grab_cursor,omitempty"`// Tell gamescope to force grab cursor / relative mouse
	GrabKeyboard    bool          `json:"grab_keyboard,omitempty"`    // Tell gamescope to grab keyboard (-g)
	ExclusiveGrab   bool          `json:"exclusive_grab,omitempty"`   // Acquire exclusive kernel EVIOCGRAB on input nodes
	CursorPath      string        `json:"cursor_path,omitempty"`      // Custom cursor icon image path
	Compositor      string        `json:"compositor,omitempty"`       // "auto" | "gamescope" | "cage" | "cage_gamescope"
	NestCage        bool          `json:"nest_cage,omitempty"`        // Run gamescope nested inside cage compositor
	ShowCursor      bool          `json:"show_cursor,omitempty"`      // Force visible cursor via software cursor env (WLR_NO_HARDWARE_CURSORS=1)
	AudioSink       string        `json:"audio_sink,omitempty"`       // Assigned PipeWire sink name
	PhysicalSink    string        `json:"physical_sink,omitempty"`    // Physical output sink to route to
	WinePrefix      string        `json:"wine_prefix,omitempty"`
	HomeDir         string        `json:"home_dir,omitempty"`
}

// MultiseatSessionConfig holds the full pairing configuration for launching a 2-player session.
type MultiseatSessionConfig struct {
	GameDir              string        `json:"game_dir"`
	Name                 string        `json:"name"`
	AppID                string        `json:"app_id,omitempty"`
	ExePath              string        `json:"exe_path"`
	ExeArgs              string        `json:"exe_args,omitempty"`
	Runner               string        `json:"runner,omitempty"` // "umu", "wine", "proton", "native"
	Layout               DisplayLayout `json:"layout"`
	Seat1                SeatConfig    `json:"seat1"`
	Seat2                SeatConfig    `json:"seat2"`
	EnableBroadcastRelay bool          `json:"enable_broadcast_relay"`
	SunshinePortP1       int           `json:"sunshine_port_p1,omitempty"` // Base 47989
	SunshinePortP2       int           `json:"sunshine_port_p2,omitempty"` // Base 48010
}

// SeatStatus reflects the runtime state of a single seat.
type SeatStatus struct {
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
	GameDir   string     `json:"game_dir"`
	IsRunning bool       `json:"is_running"`
	Seat1     SeatStatus `json:"seat1"`
	Seat2     SeatStatus `json:"seat2"`
	StartTime int64      `json:"start_time"`
}

// SystemCapabilities reports available host tools for multi-seat support.
type SystemCapabilities struct {
	HasBwrap     bool `json:"has_bwrap"`
	HasPasta     bool `json:"has_pasta"`
	HasGamescope bool `json:"has_gamescope"`
	HasCage      bool `json:"has_cage"`
	HasSunshine  bool `json:"has_sunshine"`
	HasPipewire  bool `json:"has_pipewire"`
}

// isExplicitDeviceNode reports whether a device node string points to a real device node path,
// rather than an empty value, "none", or "auto".
func isExplicitDeviceNode(node string) bool {
	return node != "" && node != "none" && node != "auto"
}

