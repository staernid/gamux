package multiseat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SunshineConfig holds options for launching an ephemeral Sunshine streaming instance.
type SunshineConfig struct {
	Port          int    // Streaming base port (default 47989, P2 48010)
	WebUIPort     int    // Management web UI port (default 47990, P2 47992)
	AudioSink     string // PipeWire sink to capture (e.g. "gamux-p1")
	WaylandSocket string // Target Wayland socket for wlr screencopy
	ConfigDir     string // Working directory for ephemeral config
}

// GenerateSunshineConfig creates an isolated sunshine.conf file.
func GenerateSunshineConfig(opts SunshineConfig) (string, error) {
	if opts.ConfigDir == "" {
		return "", fmt.Errorf("config dir cannot be empty")
	}

	if err := os.MkdirAll(opts.ConfigDir, 0755); err != nil {
		return "", fmt.Errorf("create sunshine config dir: %w", err)
	}

	port := opts.Port
	if port <= 0 {
		port = 47989
	}

	confPath := filepath.Join(opts.ConfigDir, "sunshine.conf")
	content := fmt.Sprintf(`port = %d
capture = wlr
origin_pin_allowed = pc
min_log_level = info
`, port)

	if opts.AudioSink != "" {
		content += fmt.Sprintf("audio_sink = %s\n", opts.AudioSink)
	}
	if opts.WebUIPort > 0 {
		content += fmt.Sprintf("web_ui_port = %d\n", opts.WebUIPort)
	}

	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write sunshine config: %w", err)
	}

	return confPath, nil
}

// BuildSunshineCmd constructs the exec.Cmd for starting an ephemeral Sunshine streaming server.
func BuildSunshineCmd(ctx context.Context, configPath string, waylandSocket string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "sunshine", configPath)
	if waylandSocket != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("WAYLAND_DISPLAY=%s", waylandSocket))
	}
	return cmd
}
