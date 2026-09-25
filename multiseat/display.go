package multiseat

import (
	"fmt"

	"github.com/staernid/gamux/util"
)

// DisplayOptions configures compositor and window layout for gamescope or cage.
type DisplayOptions struct {
	Mode            SeatMode      // "local" | "stream"
	Layout          DisplayLayout // "dual_monitor" | "multi_monitor" | "side_by_side" | "top_bottom" | "quad_grid" | "three_player" | "custom" | "windowed"
	SeatID          string        // "p1", "p2", "p3", "p4", etc.
	SeatIndex       int           // 0-indexed position among active seats
	TotalSeats      int           // Total active seats in session
	DisplayIndex    int           // Physical monitor output index (0, 1, etc.)
	DisplayName     string        // Physical connector name (e.g. "DP-1")
	Backend         string        // Compositor backend: "wayland", "sdl", "auto"
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

// SeatIndexFromID parses standard seat identifiers ("p1"->0, "p2"->1, "p3"->2, "p4"->3).
func SeatIndexFromID(seatID string) int {
	switch seatID {
	case "p1", "1", "seat1":
		return 0
	case "p2", "2", "seat2":
		return 1
	case "p3", "3", "seat3":
		return 2
	case "p4", "4", "seat4":
		return 3
	default:
		var idx int
		if n, _ := fmt.Sscanf(seatID, "p%d", &idx); n == 1 && idx > 0 {
			return idx - 1
		}
		return 0
	}
}

// DefaultDisplayDimensions returns standard baseline screen dimensions if unset.
func DefaultDisplayDimensions() (width, height int) {
	info := util.DetectPrimaryDisplay()
	if info.Width > 0 && info.Height > 0 {
		return info.Width, info.Height
	}
	return 1920, 1080
}

// CalculateSplitGeometry calculates window dimensions and offsets for 2-player split-screen layouts.
func CalculateSplitGeometry(layout DisplayLayout, seatID string, totalWidth, totalHeight int) (winW, winH, winX, winY int) {
	seatIdx := SeatIndexFromID(seatID)
	return CalculateSeatGeometry(layout, seatID, seatIdx, 2, totalWidth, totalHeight)
}

// CalculateSeatGeometry calculates window dimensions and offsets for N-player split-screen layouts.
func CalculateSeatGeometry(layout DisplayLayout, seatID string, seatIndex, totalSeats int, totalWidth, totalHeight int) (winW, winH, winX, winY int) {
	if totalWidth <= 0 || totalHeight <= 0 {
		totalWidth, totalHeight = DefaultDisplayDimensions()
	}
	if totalSeats <= 0 {
		totalSeats = 2
	}
	if seatIndex < 0 {
		seatIndex = 0
	}
	if seatIndex >= totalSeats {
		seatIndex = totalSeats - 1
	}

	switch layout {
	case LayoutSideBySide:
		// Vertical columns (Left to Right)
		colW := totalWidth / totalSeats
		winH = totalHeight
		winY = 0
		winX = seatIndex * colW
		if seatIndex == totalSeats-1 {
			winW = totalWidth - winX
		} else {
			winW = colW
		}

	case LayoutTopBottom:
		// Horizontal rows (Top to Bottom)
		rowH := totalHeight / totalSeats
		winW = totalWidth
		winX = 0
		winY = seatIndex * rowH
		if seatIndex == totalSeats-1 {
			winH = totalHeight - winY
		} else {
			winH = rowH
		}

	case LayoutQuadGrid:
		if totalSeats < 4 {
			if totalSeats == 3 {
				return CalculateSeatGeometry(LayoutThreePlayer, seatID, seatIndex, totalSeats, totalWidth, totalHeight)
			}
			return CalculateSeatGeometry(LayoutSideBySide, seatID, seatIndex, totalSeats, totalWidth, totalHeight)
		}
		// 2x2 grid:
		// [0: Top-Left ] [1: Top-Right ]
		// [2: Bottom-L ] [3: Bottom-R  ]
		halfW := totalWidth / 2
		halfH := totalHeight / 2
		switch seatIndex {
		case 0:
			winX, winY = 0, 0
			winW, winH = halfW, halfH
		case 1:
			winX, winY = halfW, 0
			winW, winH = totalWidth-halfW, halfH
		case 2:
			winX, winY = 0, halfH
			winW, winH = halfW, totalHeight-halfH
		default: // 3 or beyond
			winX, winY = halfW, halfH
			winW, winH = totalWidth-halfW, totalHeight-halfH
		}

	case LayoutThreePlayer:
		if totalSeats < 3 {
			return CalculateSeatGeometry(LayoutSideBySide, seatID, seatIndex, totalSeats, totalWidth, totalHeight)
		}
		// 3-Player layout: Seat 0 gets top full width, Seats 1 & 2 share bottom half
		halfH := totalHeight / 2
		switch seatIndex {
		case 0:
			winX, winY = 0, 0
			winW, winH = totalWidth, halfH
		case 1:
			halfW := totalWidth / 2
			winX, winY = 0, halfH
			winW, winH = halfW, totalHeight-halfH
		default:
			halfW := totalWidth / 2
			winX, winY = halfW, halfH
			winW, winH = totalWidth-halfW, totalHeight-halfH
		}

	default:
		// Full window / single monitor / dual monitor
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

	if opts.Backend != "" {
		args = append(args, "--backend", opts.Backend)
	}

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
		case LayoutDualMonitor, LayoutMultiMonitor:
			if tw <= 0 || th <= 0 {
				tw, th = DefaultDisplayDimensions()
			}
			args = append(args, "-w", fmt.Sprintf("%d", tw), "-h", fmt.Sprintf("%d", th))
			args = append(args, "--display-index", fmt.Sprintf("%d", opts.DisplayIndex))
			if opts.Fullscreen {
				args = append(args, "-f")
			}
		case LayoutSideBySide, LayoutTopBottom, LayoutQuadGrid, LayoutThreePlayer:
			ww := opts.WindowWidth
			wh := opts.WindowHeight
			if ww <= 0 || wh <= 0 {
				ww, wh, _, _ = CalculateSeatGeometry(opts.Layout, opts.SeatID, opts.SeatIndex, opts.TotalSeats, 0, 0)
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
		default: // LayoutWindowed, LayoutCustom
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
			} else if opts.Borderless || opts.Layout == LayoutCustom {
				args = append(args, "-b")
			}
		}
	}

	if opts.TargetFPS > 0 {
		args = append(args, "-r", fmt.Sprintf("%d", opts.TargetFPS))
		if opts.Mode != SeatModeStream {
			args = append(args, "-o", fmt.Sprintf("%d", opts.TargetFPS))
		}
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
