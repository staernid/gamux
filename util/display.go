package util

import (
	"encoding/json"
	"math"
	"os/exec"
	"regexp"
	"strconv"
)

// DisplayInfo holds detected screen metrics.
type DisplayInfo struct {
	Name          string  `json:"name,omitempty"`           // e.g. "DP-1"
	Index         int     `json:"index"`                    // Output index (0, 1, ...)
	Width         int     `json:"width"`                    // Physical pixel width
	Height        int     `json:"height"`                   // Physical pixel height
	LogicalWidth  int     `json:"logical_width,omitempty"`  // Scaled logical width
	LogicalHeight int     `json:"logical_height,omitempty"` // Scaled logical height
	X             int     `json:"x"`                        // Offset X
	Y             int     `json:"y"`                        // Offset Y
	Scale         float64 `json:"scale,omitempty"`          // e.g. 1.0, 1.15, 1.25
	RefreshRate   int     `json:"refresh_rate"`             // In Hz (e.g. 60, 100, 144)
	Primary       bool    `json:"primary"`
}

// DetectDisplays returns all connected and active physical monitors.
func DetectDisplays() []DisplayInfo {
	if list := detectKScreenDoctorDisplays(); len(list) > 0 {
		return list
	}
	if list := detectXrandrDisplays(); len(list) > 0 {
		return list
	}
	if info := DetectPrimaryDisplay(); info.Width > 0 && info.Height > 0 {
		return []DisplayInfo{info}
	}
	return []DisplayInfo{
		{
			Name:        "Default",
			Index:       0,
			Width:       1920,
			Height:      1080,
			RefreshRate: 60,
			Scale:       1.0,
			Primary:     true,
		},
	}
}

// DetectPrimaryDisplay inspects the host environment to detect the primary monitor resolution and refresh rate.
// It queries DRM sysfs, kscreen-doctor, and xrandr in order of efficiency and accuracy.
// If all detection methods fail, it returns safe standard defaults (1920x1080 @ 60Hz).
func DetectPrimaryDisplay() DisplayInfo {
	displays := DetectDisplays()
	for _, d := range displays {
		if d.Primary {
			return d
		}
	}
	if len(displays) > 0 {
		return displays[0]
	}
	return DisplayInfo{
		Name:        "Default",
		Index:       0,
		Width:       1920,
		Height:      1080,
		RefreshRate: 60,
		Scale:       1.0,
		Primary:     true,
	}
}

type kscreenOutput struct {
	Name          string  `json:"name"`
	Connected     bool    `json:"connected"`
	Enabled       bool    `json:"enabled"`
	Priority      int     `json:"priority"`
	Scale         float64 `json:"scale"`
	CurrentModeID string  `json:"currentModeId"`
	Pos           struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"pos"`
	Size struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"size"`
	Modes []struct {
		ID          string  `json:"id"`
		RefreshRate float64 `json:"refreshRate"`
		Size        struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"size"`
	} `json:"modes"`
}

type kscreenRoot struct {
	Outputs []kscreenOutput `json:"outputs"`
}

func detectKScreenDoctorDisplays() []DisplayInfo {
	cmd := exec.Command("kscreen-doctor", "-j")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var root kscreenRoot
	if err := json.Unmarshal(out, &root); err != nil {
		return nil
	}

	var res []DisplayInfo
	idx := 0
	for _, out := range root.Outputs {
		if !out.Connected || !out.Enabled {
			continue
		}
		var width, height int
		rate := 60
		for _, m := range out.Modes {
			if m.ID == out.CurrentModeID && m.Size.Width > 0 && m.Size.Height > 0 {
				width = m.Size.Width
				height = m.Size.Height
				r := int(math.Round(m.RefreshRate))
				if r > 0 {
					rate = r
				}
				break
			}
		}
		if width <= 0 || height <= 0 {
			width = out.Size.Width
			height = out.Size.Height
		}
		if width <= 0 || height <= 0 {
			continue
		}

		scale := out.Scale
		if scale <= 0 {
			scale = 1.0
		}

		logW := int(math.Round(float64(width) / scale))
		logH := int(math.Round(float64(height) / scale))

		res = append(res, DisplayInfo{
			Name:          out.Name,
			Index:         idx,
			Width:         width,
			Height:        height,
			LogicalWidth:  logW,
			LogicalHeight: logH,
			X:             out.Pos.X,
			Y:             out.Pos.Y,
			Scale:         scale,
			RefreshRate:   rate,
			Primary:       out.Priority == 1,
		})
		idx++
	}
	return res
}

func detectXrandrDisplays() []DisplayInfo {
	cmd := exec.Command("xrandr", "--current")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Match output headers like: "DP-1 connected primary 3440x1440+0+0 (normal left inverted right x axis y axis) 797mm x 334mm"
	// or "HDMI-1 connected 1920x1080+3440+0 ..."
	reHeader := regexp.MustCompile(`(?m)^([A-Za-z0-9_-]+)\s+connected\s+(primary\s+)?(\d+)x(\d+)\+(\d+)\+(\d+)`)
	matches := reHeader.FindAllSubmatch(out, -1)
	if len(matches) == 0 {
		return nil
	}

	var res []DisplayInfo
	for i, m := range matches {
		name := string(m[1])
		isPrimary := len(m[2]) > 0
		w, _ := strconv.Atoi(string(m[3]))
		h, _ := strconv.Atoi(string(m[4]))
		x, _ := strconv.Atoi(string(m[5]))
		y, _ := strconv.Atoi(string(m[6]))

		res = append(res, DisplayInfo{
			Name:          name,
			Index:         i,
			Width:         w,
			Height:        h,
			LogicalWidth:  w,
			LogicalHeight: h,
			X:             x,
			Y:             y,
			Scale:         1.0,
			RefreshRate:   60,
			Primary:       isPrimary || i == 0,
		})
	}
	return res
}
