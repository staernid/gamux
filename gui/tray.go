package gui

import (
	"log/slog"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// TrayManager manages the unified cross-platform system tray using Wails v3.
type TrayManager struct {
	systemTray  *application.SystemTray
	cb          TrayCallbacks
	mu          sync.RWMutex
	recentGames []TrayGame
	closed      bool
}

// StartTray initializes the system tray for the application using Wails v3.
func StartTray(cb TrayCallbacks) (*TrayManager, error) {
	var initialRecent []TrayGame
	if cb.GetRecentGames != nil {
		initialRecent = SortTrayGames(cb.GetRecentGames(), 3)
	}

	wailsApp := application.Get()
	if wailsApp == nil {
		slog.Debug("Wails application not initialized, system tray disabled")
		return &TrayManager{cb: cb, recentGames: initialRecent}, nil
	}

	st := wailsApp.SystemTray.New()
	st.SetLabel("gamux")
	st.SetTooltip("gamux - Steam Game Manager")

	tm := &TrayManager{
		systemTray:  st,
		cb:          cb,
		recentGames: initialRecent,
	}

	if cb.OnActivate != nil {
		st.OnClick(func() {
			cb.OnActivate()
		})
	}

	tm.rebuildMenu()
	st.Run()
	return tm, nil
}

// rebuildMenu constructs the tray context menu with recent games, activate, and quit items.
func (t *TrayManager) rebuildMenu() {
	if t.systemTray == nil {
		return
	}
	wailsApp := application.Get()
	if wailsApp == nil {
		return
	}

	menu := wailsApp.NewMenu()

	// Title / Open item
	menu.Add("Open Gamux").OnClick(func(ctx *application.Context) {
		if t.cb.OnActivate != nil {
			t.cb.OnActivate()
		}
	})

	menu.AddSeparator()

	// Recent games section
	t.mu.RLock()
	games := t.recentGames
	t.mu.RUnlock()

	if len(games) > 0 {
		recentSubmenu := menu.AddSubmenu("Recent Games")
		for _, g := range games {
			gameDir := g.GameDir
			recentSubmenu.Add(g.Name).OnClick(func(ctx *application.Context) {
				if t.cb.OnLaunchGame != nil {
					t.cb.OnLaunchGame(gameDir)
				}
			})
		}
		menu.AddSeparator()
	}

	// Quit item
	menu.Add("Quit Gamux").OnClick(func(ctx *application.Context) {
		if t.cb.OnQuit != nil {
			t.cb.OnQuit()
		}
	})

	t.systemTray.SetMenu(menu)
}

// SetRecentGames updates the list of recent games displayed in the tray.
func (t *TrayManager) SetRecentGames(games []TrayGame) bool {
	t.mu.Lock()
	t.recentGames = SortTrayGames(games, 3)
	t.mu.Unlock()

	t.rebuildMenu()
	return true
}

// Close destroys the system tray instance.
func (t *TrayManager) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	if t.systemTray != nil {
		t.systemTray.Destroy()
	}
}
