package multiseat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SeatPlacement maps a seat process ID to its layout positioning requirements.
type SeatPlacement struct {
	PID          int    `json:"pid"`
	SeatIndex    int    `json:"seat_index"`
	SeatID       string `json:"seat_id"`
	DisplayIndex int    `json:"display_index"`
}

// KWinSplitCoordinator manages transient, pixel-perfect window placement and border removal
// for multi-seat Gamescope sessions on KDE Plasma (Wayland / X11).
// It registers an ephemeral KWin script via DBus without mutating user configuration files (~/.config/kwinrulesrc).
type KWinSplitCoordinator struct {
	mu         sync.Mutex
	qdbusBin   string
	pluginName string
	scriptPath string
	scriptID   int
	placements []SeatPlacement
	layout     DisplayLayout
	isLoaded   bool
	cancelFn   context.CancelFunc
}

// ResolveQDBusBinary finds qdbus or qdbus6 in system PATH.
func ResolveQDBusBinary() (string, error) {
	if bin, err := exec.LookPath("qdbus"); err == nil {
		return bin, nil
	}
	if bin, err := exec.LookPath("qdbus6"); err == nil {
		return bin, nil
	}
	return "", fmt.Errorf("qdbus binary not found")
}

// IsKWinAvailable checks whether KWin scripting DBus service is actively responding.
func IsKWinAvailable() bool {
	bin, err := ResolveQDBusBinary()
	if err != nil {
		return false
	}
	out, err := exec.Command(bin, "org.kde.KWin", "/Scripting").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "org.kde.kwin.Scripting")
}

// GenerateKWinSplitScript generates the ECMAScript content for positioning Gamescope windows
// with 0 gaps, 0 borders, and raised above desktop panels.
func GenerateKWinSplitScript(seat1PID, seat2PID int, layout DisplayLayout) string {
	return GenerateKWinMultiScript([]SeatPlacement{
		{PID: seat1PID, SeatIndex: 0, SeatID: "p1"},
		{PID: seat2PID, SeatIndex: 1, SeatID: "p2"},
	}, layout)
}

// GenerateKWinMultiScript generates the ECMAScript content for N Gamescope windows.
func GenerateKWinMultiScript(placements []SeatPlacement, layout DisplayLayout) string {
	dispTotals := make(map[int]int)
	dispCounts := make(map[int]int)
	for _, p := range placements {
		if p.PID > 0 {
			dispTotals[p.DisplayIndex]++
		}
	}

	var pidMapEntries []string
	for _, p := range placements {
		if p.PID > 0 {
			localIdx := dispCounts[p.DisplayIndex]
			dispCounts[p.DisplayIndex]++
			totalOnDisp := dispTotals[p.DisplayIndex]
			pidMapEntries = append(pidMapEntries, fmt.Sprintf("%d: { index: %d, id: %q, disp: %d, localIndex: %d, dispTotal: %d }",
				p.PID, p.SeatIndex, p.SeatID, p.DisplayIndex, localIdx, totalOnDisp))
		}
	}
	pidMapJS := strings.Join(pidMapEntries, ",\n        ")
	totalSeats := len(placements)
	if totalSeats == 0 {
		totalSeats = 2
	}

	return fmt.Sprintf(`// Gamux transient splitscreen coordinator for KWin
(function() {
    var pidMap = {
        %s
    };
    var layout = %q;
    var totalSeats = %d;

    function getScreenGeometry(w, dispIndex) {
        if (workspace.screens && workspace.screens.length > 0) {
            var idx = Math.min(Math.max(0, dispIndex), workspace.screens.length - 1);
            var s = workspace.screens[idx] || workspace.screens[0];
            if (s && s.geometry) return s.geometry;
        }
        var sW = workspace.virtualScreenSize ? workspace.virtualScreenSize.width : 1920;
        var sH = workspace.virtualScreenSize ? workspace.virtualScreenSize.height : 1080;
        return { x: 0, y: 0, width: sW, height: sH };
    }

    function placeSeat(w, info) {
        if (!w || !info) return;
        w.noBorder = true;
        w.keepAbove = true;
        w.skipTaskbar = true;
        w.skipPager = true;

        var numScreens = (workspace.screens && workspace.screens.length > 0) ? workspace.screens.length : 1;
        var dispIndex = (numScreens > 1) ? (info.disp || 0) : 0;
        var geom = getScreenGeometry(w, dispIndex);
        var sX = geom.x || 0;
        var sY = geom.y || 0;
        var sW = geom.width;
        var sH = geom.height;

        var dispTotal = (numScreens === 1) ? totalSeats : (info.dispTotal || 1);
        var idx = (numScreens === 1) ? (info.index !== undefined ? info.index : 0) : (info.localIndex !== undefined ? info.localIndex : info.index);

        if (layout === "dual_monitor" || layout === "multi_monitor") {
            w.frameGeometry = { x: sX, y: sY, width: sW, height: sH };
            return;
        }
        if (dispTotal === 1 && totalSeats === 1) {
            w.frameGeometry = { x: sX, y: sY, width: sW, height: sH };
            return;
        }

        var effLayout = layout;
        if (dispTotal < 3 && (effLayout === "three_player" || effLayout === "quad_grid")) {
            effLayout = "side_by_side";
        } else if (dispTotal === 3 && effLayout === "quad_grid") {
            effLayout = "three_player";
        }

        if (effLayout === "side_by_side") {
            var colW = Math.floor(sW / dispTotal);
            var x = sX + (idx * colW);
            var wW = (idx === dispTotal - 1) ? (sX + sW - x) : colW;
            w.frameGeometry = { x: x, y: sY, width: wW, height: sH };
        } else if (effLayout === "top_bottom") {
            var rowH = Math.floor(sH / dispTotal);
            var y = sY + (idx * rowH);
            var hH = (idx === dispTotal - 1) ? (sY + sH - y) : rowH;
            w.frameGeometry = { x: sX, y: y, width: sW, height: hH };
        } else if (effLayout === "quad_grid") {
            var halfW = Math.floor(sW * 0.5);
            var halfH = Math.floor(sH * 0.5);
            if (idx === 0) {
                w.frameGeometry = { x: sX, y: sY, width: halfW, height: halfH };
            } else if (idx === 1) {
                w.frameGeometry = { x: sX + halfW, y: sY, width: sW - halfW, height: halfH };
            } else if (idx === 2) {
                w.frameGeometry = { x: sX, y: sY + halfH, width: halfW, height: sH - halfH };
            } else {
                w.frameGeometry = { x: sX + halfW, y: sY + halfH, width: sW - halfW, height: sH - halfH };
            }
        } else if (effLayout === "three_player") {
            var halfH = Math.floor(sH * 0.5);
            var halfW = Math.floor(sW * 0.5);
            if (idx === 0) {
                w.frameGeometry = { x: sX, y: sY, width: sW, height: halfH };
            } else if (idx === 1) {
                w.frameGeometry = { x: sX, y: sY + halfH, width: halfW, height: sH - halfH };
            } else {
                w.frameGeometry = { x: sX + halfW, y: sY + halfH, width: sW - halfW, height: sH - halfH };
            }
        }
    }

    function checkAndPlace(w) {
        if (!w) return;
        var info = pidMap[w.pid];
        if (info) {
            placeSeat(w, info);
        }
    }

    function applyPositions() {
        var list = workspace.windowList ? workspace.windowList() : workspace.clientList();
        if (!list) return;
        for (var i = 0; i < list.length; i++) {
            checkAndPlace(list[i]);
        }
    }

    if (typeof workspace.windowAdded !== 'undefined') {
        workspace.windowAdded.connect(checkAndPlace);
    } else if (typeof workspace.clientAdded !== 'undefined') {
        workspace.clientAdded.connect(checkAndPlace);
    }

    applyPositions();
})();
`, pidMapJS, string(layout), totalSeats)
}

// NewKWinMultiCoordinator initializes a coordinator for N seats.
func NewKWinMultiCoordinator(placements []SeatPlacement, layout DisplayLayout) (*KWinSplitCoordinator, error) {
	if len(placements) == 0 {
		return nil, fmt.Errorf("at least one seat placement required")
	}

	qdbus, err := ResolveQDBusBinary()
	if err != nil {
		return nil, err
	}

	pluginName := fmt.Sprintf("gamux_split_%d", time.Now().UnixNano())

	return &KWinSplitCoordinator{
		qdbusBin:   qdbus,
		pluginName: pluginName,
		placements: placements,
		layout:     layout,
		scriptID:   -1,
	}, nil
}

// NewKWinSplitCoordinator initializes a new coordinator for the given seat PIDs and layout (2 seats).
func NewKWinSplitCoordinator(seat1PID, seat2PID int, layout DisplayLayout) (*KWinSplitCoordinator, error) {
	if seat1PID <= 0 || seat2PID <= 0 {
		return nil, fmt.Errorf("valid PIDs required for both seats (p1=%d, p2=%d)", seat1PID, seat2PID)
	}

	return NewKWinMultiCoordinator([]SeatPlacement{
		{PID: seat1PID, SeatIndex: 0, SeatID: "p1"},
		{PID: seat2PID, SeatIndex: 1, SeatID: "p2"},
	}, layout)
}

// Start writes the transient script, loads it into KWin via DBus, and begins polling to ensure
// delayed window initialization is caught and positioned.
func (c *KWinSplitCoordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isLoaded {
		return nil
	}

	scriptContent := GenerateKWinMultiScript(c.placements, c.layout)

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.js", c.pluginName))
	if err := os.WriteFile(tmpFile, []byte(scriptContent), 0600); err != nil {
		return fmt.Errorf("write kwin script: %w", err)
	}
	c.scriptPath = tmpFile

	// Load script into KWin
	loadCmd := exec.CommandContext(ctx, c.qdbusBin, "org.kde.KWin", "/Scripting", "org.kde.kwin.Scripting.loadScript", tmpFile, c.pluginName)
	out, err := loadCmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("load kwin script via qdbus: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}

	idStr := strings.TrimSpace(string(out))
	id, pErr := strconv.Atoi(idStr)
	if pErr != nil {
		id = 0
	}
	c.scriptID = id
	c.isLoaded = true

	// Run initial execution
	scriptObjPath := fmt.Sprintf("/Scripting/Script%d", c.scriptID)
	_ = exec.CommandContext(ctx, c.qdbusBin, "org.kde.KWin", scriptObjPath, "org.kde.kwin.Script.run").Run()

	slog.Info("KWin splitscreen coordinator initialized", "plugin", c.pluginName, "script_id", c.scriptID, "seats_count", len(c.placements))

	// Launch background re-application ticker to catch late window creation (e.g. game startup taking a few seconds)
	pollCtx, cancel := context.WithCancel(ctx)
	c.cancelFn = cancel

	go func() {
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()
		timeout := time.After(8 * time.Second)

		for {
			select {
			case <-pollCtx.Done():
				return
			case <-timeout:
				return
			case <-ticker.C:
				_ = exec.Command(c.qdbusBin, "org.kde.KWin", scriptObjPath, "org.kde.kwin.Script.run").Run()
			}
		}
	}()

	return nil
}

// Stop cleanly unloads the script from KWin and removes temporary script files.
func (c *KWinSplitCoordinator) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFn != nil {
		c.cancelFn()
		c.cancelFn = nil
	}

	if !c.isLoaded {
		if c.scriptPath != "" {
			_ = os.Remove(c.scriptPath)
		}
		return nil
	}

	// Unload script from KWin
	unloadCmd := exec.Command(c.qdbusBin, "org.kde.KWin", "/Scripting", "org.kde.kwin.Scripting.unloadScript", c.pluginName)
	_ = unloadCmd.Run()

	if c.scriptPath != "" {
		_ = os.Remove(c.scriptPath)
		c.scriptPath = ""
	}

	c.isLoaded = false
	slog.Info("KWin splitscreen coordinator unloaded", "plugin", c.pluginName)
	return nil
}
