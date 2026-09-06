package multiseat

import (
	"fmt"

	"github.com/staernid/gamux/util"
)

// DisplayOptions configures compositor and window layout for gamescope or cage.
type DisplayOptions struct {
	Mode            SeatMode      // "local" | "stream"
	Layout          DisplayLayout // "dual_monitor" | "side_by_side" | "top_bottom" | "windowed"
	SeatID          string        // "p1" | "p2"
	DisplayIndex    int           // Physical monitor output index (0, 1, etc.)
	TargetWidth     int           // Internal render width (e.g. 1920)
	TargetHeight    int           // Internal render height (e.g. 1080)
	TargetFPS       int           // Target refresh rate / FPS (e.g. 60, 100)
	WindowWidth     int           // Window width on host
	WindowHeight    int           // Window height on host
	WindowX         int           // Window position X
	WindowY         int           // Window position Y
	Fullscreen      bool
	Borderless      bool
	ForceGrabCursor bool     // Tell gamescope to force relative mouse mode (--force-grab-cursor)
	GrabKeyboard    bool     // Tell gamescope to grab the keyboard (-g)
	CursorPath      string   // Custom cursor image path (--cursor)
	Command         []string // Executable command to run inside gamescope
}

// DefaultDisplayDimensions returns standard baseline screen dimensions if unset.
func DefaultDisplayDimensions() (width, height int) {
	info := util.DetectPrimaryDisplay()
	if info.Width > 0 && info.Height > 0 {
		return info.Width, info.Height
	}
	return 1920, 1080
}

// CalculateSplitGeometry calculates window dimensions and offsets for split-screen layouts.
func CalculateSplitGeometry(layout DisplayLayout, seatID string, totalWidth, totalHeight int) (winW, winH, winX, winY int) {
	if totalWidth <= 0 || totalHeight <= 0 {
		totalWidth, totalHeight = DefaultDisplayDimensions()
	}

	switch layout {
	case LayoutSideBySide:
		// Left/Right side-by-side (Vertical split cut line)
		winW = totalWidth / 2
		winH = totalHeight
		winY = 0
		if seatID == "p2" {
			winX = winW
		} else {
			winX = 0
		}
	case LayoutTopBottom:
		// Top/Bottom (Horizontal split cut line)
		winW = totalWidth
		winH = totalHeight / 2
		winX = 0
		if seatID == "p2" {
			winY = winH
		} else {
			winY = 0
		}
	default:
		// Full window / single monitor
		winW = totalWidth
		winH = totalHeight
		winX = 0
		winY = 0
	}

	return winW, winH, winX, winY
}

// BuildGamescopeArgs creates command line arguments for gamescope.
func BuildGamescopeArgs(opts DisplayOptions) ([]string, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("inner command is required for gamescope")
	}

	tw := opts.TargetWidth
	th := opts.TargetHeight

	var args []string

	if opts.Mode == SeatModeStream {
		if tw <= 0 || th <= 0 {
			tw, th = DefaultDisplayDimensions()
		}
		// Headless / stream mode: match window resolution to target resolution
		args = append(args, "-w", fmt.Sprintf("%d", tw), "-h", fmt.Sprintf("%d", th))
		args = append(args, "-W", fmt.Sprintf("%d", tw), "-H", fmt.Sprintf("%d", th))
	} else {
		// Local mode with layout
		switch opts.Layout {
		case LayoutDualMonitor:
			if tw <= 0 || th <= 0 {
				tw, th = DefaultDisplayDimensions()
			}
			args = append(args, "-w", fmt.Sprintf("%d", tw), "-h", fmt.Sprintf("%d", th))
			args = append(args, "--display-index", fmt.Sprintf("%d", opts.DisplayIndex))
			if opts.Fullscreen {
				args = append(args, "-f")
			}
		case LayoutSideBySide, LayoutTopBottom:
			ww := opts.WindowWidth
			wh := opts.WindowHeight
			if ww <= 0 || wh <= 0 {
				ww, wh, _, _ = CalculateSplitGeometry(opts.Layout, opts.SeatID, opts.WindowWidth, opts.WindowHeight)
			}
			// For split layouts, default render resolution to window resolution for 1:1 pixel mapping
			if tw <= 0 || th <= 0 {
				tw = ww
				th = wh
			}
			args = append(args, "-w", fmt.Sprintf("%d", tw), "-h", fmt.Sprintf("%d", th))
			if ww > 0 && wh > 0 {
				args = append(args, "-W", fmt.Sprintf("%d", ww), "-H", fmt.Sprintf("%d", wh))
			}
			args = append(args, "-b") // Borderless window
		default: // LayoutWindowed
			ww := opts.WindowWidth
			wh := opts.WindowHeight
			if ww <= 0 || wh <= 0 {
				if tw > 0 && th > 0 {
					ww = tw
					wh = th
				} else {
					ww, wh = DefaultDisplayDimensions()
				}
			}
			if tw <= 0 || th <= 0 {
				tw = ww
				th = wh
			}
			args = append(args, "-w", fmt.Sprintf("%d", tw), "-h", fmt.Sprintf("%d", th))
			if ww > 0 && wh > 0 {
				args = append(args, "-W", fmt.Sprintf("%d", ww), "-H", fmt.Sprintf("%d", wh))
			}
			if opts.Fullscreen {
				args = append(args, "-f")
			} else if opts.Borderless {
				args = append(args, "-b")
			}
		}
	}

	if opts.TargetFPS > 0 {
		args = append(args, "-r", fmt.Sprintf("%d", opts.TargetFPS))
	}

	if opts.ForceGrabCursor {
		args = append(args, "--force-grab-cursor")
	}
	if opts.GrabKeyboard {
		args = append(args, "-g")
	}
	if opts.CursorPath != "" {
		args = append(args, "--cursor", opts.CursorPath)
	}

	// Inner command terminator
	args = append(args, "--")
	args = append(args, opts.Command...)

	return args, nil
}

// BuildCageArgs wraps a command in cage for headless Wayland streaming sessions.
func BuildCageArgs(waylandSocket string, command []string) ([]string, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("command is required for cage")
	}

	args := []string{"-s", "--"}
	args = append(args, command...)
	return args, nil
}
