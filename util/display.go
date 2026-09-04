package util

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// DisplayInfo holds detected screen metrics.
type DisplayInfo struct {
	Width       int
	Height      int
	RefreshRate int
}

// DetectPrimaryDisplay inspects the host environment to detect the primary monitor resolution and refresh rate.
// It queries DRM sysfs, kscreen-doctor, and xrandr in order of efficiency and accuracy.
// If all detection methods fail, it returns safe standard defaults (1920x1080 @ 60Hz).
func DetectPrimaryDisplay() DisplayInfo {
	// 1. Try kscreen-doctor (KDE Plasma Wayland / X11)
	if info, ok := detectKScreenDoctor(); ok && info.Width > 0 && info.Height > 0 {
		return info
	}

	// 2. Try xrandr (X11 / Xwayland)
	if info, ok := detectXrandr(); ok && info.Width > 0 && info.Height > 0 {
		return info
	}

	// 3. Try Linux DRM sysfs (/sys/class/drm)
	if info, ok := detectDrmSysfs(); ok && info.Width > 0 && info.Height > 0 {
		return info
	}

	// 4. Fallback baseline
	return DisplayInfo{
		Width:       1920,
		Height:      1080,
		RefreshRate: 60,
	}
}

type kscreenOutput struct {
	Connected     bool   `json:"connected"`
	Enabled       bool   `json:"enabled"`
	CurrentModeID string `json:"currentModeId"`
	Modes         []struct {
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

func detectKScreenDoctor() (DisplayInfo, bool) {
	cmd := exec.Command("kscreen-doctor", "-j")
	out, err := cmd.Output()
	if err != nil {
		return DisplayInfo{}, false
	}

	var root kscreenRoot
	if err := json.Unmarshal(out, &root); err != nil {
		return DisplayInfo{}, false
	}

	for _, out := range root.Outputs {
		if out.Connected && out.Enabled {
			for _, m := range out.Modes {
				if m.ID == out.CurrentModeID && m.Size.Width > 0 && m.Size.Height > 0 {
					rate := int(math.Round(m.RefreshRate))
					if rate <= 0 {
						rate = 60
					}
					return DisplayInfo{
						Width:       m.Size.Width,
						Height:      m.Size.Height,
						RefreshRate: rate,
					}, true
				}
			}
		}
	}

	return DisplayInfo{}, false
}

func detectXrandr() (DisplayInfo, bool) {
	cmd := exec.Command("xrandr", "--current")
	out, err := cmd.Output()
	if err != nil {
		return DisplayInfo{}, false
	}

	// Match lines like: "   3440x1440     99.96*+"
	re := regexp.MustCompile(`(?m)^\s+(\d+)x(\d+)\s+([0-9.]+)\*`)
	matches := re.FindSubmatch(out)
	if len(matches) >= 4 {
		w, _ := strconv.Atoi(string(matches[1]))
		h, _ := strconv.Atoi(string(matches[2]))
		rateFloat, _ := strconv.ParseFloat(string(matches[3]), 64)
		rate := int(math.Round(rateFloat))
		if rate <= 0 {
			rate = 60
		}
		if w > 0 && h > 0 {
			return DisplayInfo{
				Width:       w,
				Height:      h,
				RefreshRate: rate,
			}, true
		}
	}

	return DisplayInfo{}, false
}

func detectDrmSysfs() (DisplayInfo, bool) {
	matches, err := filepath.Glob("/sys/class/drm/card*-*/status")
	if err != nil || len(matches) == 0 {
		return DisplayInfo{}, false
	}

	for _, statusPath := range matches {
		data, err := os.ReadFile(statusPath)
		if err != nil || strings.TrimSpace(string(data)) != "connected" {
			continue
		}

		modesPath := filepath.Join(filepath.Dir(statusPath), "modes")
		modesData, err := os.ReadFile(modesPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(modesData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var w, h int
			if n, _ := fmt.Sscanf(line, "%dx%d", &w, &h); n == 2 && w > 0 && h > 0 {
				return DisplayInfo{
					Width:       w,
					Height:      h,
					RefreshRate: 60,
				}, true
			}
		}
	}

	return DisplayInfo{}, false
}
