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

// KWinSplitCoordinator manages transient, pixel-perfect window placement and border removal
// for multi-seat Gamescope sessions on KDE Plasma (Wayland / X11).
// It registers an ephemeral KWin script via DBus without mutating user configuration files (~/.config/kwinrulesrc).
type KWinSplitCoordinator struct {
	mu         sync.Mutex
	qdbusBin   string
	pluginName string
	scriptPath string
	scriptID   int
	seat1PID   int
	seat2PID   int
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

// GenerateKWinSplitScript generates the ECMAScript content for positioning two Gamescope windows
// side-by-side or top-and-bottom with 0 gaps, 0 borders, and raised above desktop panels.
func GenerateKWinSplitScript(seat1PID, seat2PID int, layout DisplayLayout) string {
	isSideBySide := layout != LayoutTopBottom

	return fmt.Sprintf(`// Gamux transient splitscreen coordinator for KWin
(function() {
    var p1PID = %d;
    var p2PID = %d;
    var isSideBySide = %t;

    function placeSeat(w, seatIndex) {
        if (!w) return;
        w.noBorder = true;
        w.keepAbove = true;
        w.skipTaskbar = true;
        w.skipPager = true;

        var sW = workspace.virtualScreenSize.width;
        var sH = workspace.virtualScreenSize.height;

        if (isSideBySide) {
            var halfW = Math.round(sW * 0.5);
            if (seatIndex === 1) {
                w.frameGeometry = { x: 0, y: 0, width: halfW, height: sH };
            } else {
                w.frameGeometry = { x: halfW, y: 0, width: sW - halfW, height: sH };
            }
        } else {
            var halfH = Math.round(sH * 0.5);
            if (seatIndex === 1) {
                w.frameGeometry = { x: 0, y: 0, width: sW, height: halfH };
            } else {
                w.frameGeometry = { x: 0, y: halfH, width: sW, height: sH - halfH };
            }
        }
    }

    function applyPositions() {
        var list = workspace.windowList ? workspace.windowList() : workspace.clientList();
        if (!list) return;
        for (var i = 0; i < list.length; i++) {
            var w = list[i];
            if (!w) continue;
            if (w.pid === p1PID) {
                placeSeat(w, 1);
            } else if (w.pid === p2PID) {
                placeSeat(w, 2);
            }
        }
    }

    if (typeof workspace.windowAdded !== 'undefined') {
        workspace.windowAdded.connect(function(w) {
            if (!w) return;
            if (w.pid === p1PID) {
                placeSeat(w, 1);
            } else if (w.pid === p2PID) {
                placeSeat(w, 2);
            }
        });
    } else if (typeof workspace.clientAdded !== 'undefined') {
        workspace.clientAdded.connect(function(w) {
            if (!w) return;
            if (w.pid === p1PID) {
                placeSeat(w, 1);
            } else if (w.pid === p2PID) {
                placeSeat(w, 2);
            }
        });
    }

    applyPositions();
})();
`, seat1PID, seat2PID, isSideBySide)
}

// NewKWinSplitCoordinator initializes a new coordinator for the given seat PIDs and layout.
func NewKWinSplitCoordinator(seat1PID, seat2PID int, layout DisplayLayout) (*KWinSplitCoordinator, error) {
	if seat1PID <= 0 || seat2PID <= 0 {
		return nil, fmt.Errorf("valid PIDs required for both seats (p1=%d, p2=%d)", seat1PID, seat2PID)
	}

	qdbus, err := ResolveQDBusBinary()
	if err != nil {
		return nil, err
	}

	pluginName := fmt.Sprintf("gamux_split_%d_%d_%d", time.Now().UnixNano(), seat1PID, seat2PID)

	return &KWinSplitCoordinator{
		qdbusBin:   qdbus,
		pluginName: pluginName,
		seat1PID:   seat1PID,
		seat2PID:   seat2PID,
		layout:     layout,
		scriptID:   -1,
	}, nil
}

// Start writes the transient script, loads it into KWin via DBus, and begins polling to ensure
// delayed window initialization is caught and positioned.
func (c *KWinSplitCoordinator) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isLoaded {
		return nil
	}

	scriptContent := GenerateKWinSplitScript(c.seat1PID, c.seat2PID, c.layout)

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

	slog.Info("KWin splitscreen coordinator initialized", "plugin", c.pluginName, "script_id", c.scriptID, "seat1_pid", c.seat1PID, "seat2_pid", c.seat2PID)

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
