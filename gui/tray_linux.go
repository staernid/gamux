//go:build linux

package gui

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// KDbusImage represents an image in ARGB32 network byte order.
type KDbusImage struct {
	Width  int32
	Height int32
	Data   []byte
}

// KDbusToolTip represents the StatusNotifierItem tooltip structure.
type KDbusToolTip struct {
	IconName    string
	Pixmaps     []KDbusImage
	Title       string
	Description string
}

// DBusMenuLayoutItem represents a menu layout item for com.canonical.dbusmenu.
type DBusMenuLayoutItem struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

// DBusMenuGroupProp represents properties for a list of items.
type DBusMenuGroupProp struct {
	ID    int32
	Props map[string]dbus.Variant
}

// TrayManager manages the StatusNotifierItem (SNI) and DBusMenu session bus registration.
type TrayManager struct {
	conn           *dbus.Conn
	busName        string
	onActivate     func()
	onQuit         func()
	onLaunchGame   func(gameDir string)
	getRecentGames func() []TrayGame

	mu          sync.RWMutex
	recentGames []TrayGame
	revision    uint32
	closed      bool
}

type sniReceiver struct {
	tray *TrayManager
}

func (s *sniReceiver) ContextMenu(x int32, y int32) *dbus.Error {
	slog.Debug("SNI ContextMenu activated", "x", x, "y", y)
	return nil
}

func (s *sniReceiver) Activate(x int32, y int32) *dbus.Error {
	slog.Debug("SNI Activate clicked; restoring window", "x", x, "y", y)
	if s.tray != nil && s.tray.onActivate != nil {
		s.tray.onActivate()
	}
	return nil
}

func (s *sniReceiver) SecondaryActivate(x int32, y int32) *dbus.Error {
	slog.Debug("SNI SecondaryActivate clicked", "x", x, "y", y)
	if s.tray != nil && s.tray.onActivate != nil {
		s.tray.onActivate()
	}
	return nil
}

func (s *sniReceiver) Scroll(delta int32, orientation string) *dbus.Error {
	return nil
}

func (s *sniReceiver) ProvideXdgActivationToken(token string) *dbus.Error {
	return nil
}

type dbusMenuReceiver struct {
	tray *TrayManager
}

func (m *dbusMenuReceiver) AboutToShow(id int32) (bool, *dbus.Error) {
	if id == 0 && m.tray != nil {
		updated := m.tray.refreshRecentGames()
		return updated, nil
	}
	return false, nil
}

func (m *dbusMenuReceiver) Event(id int32, eventId string, data dbus.Variant, timestamp uint32) *dbus.Error {
	slog.Debug("DBusMenu Event", "id", id, "eventId", eventId)
	if eventId == "clicked" {
		switch id {
		case 1:
			if m.tray != nil && m.tray.onActivate != nil {
				m.tray.onActivate()
			}
		case 3:
			if m.tray != nil && m.tray.onQuit != nil {
				go m.tray.onQuit()
			}
		default:
			if id >= 10 && m.tray != nil {
				m.tray.mu.RLock()
				idx := int(id - 10)
				var targetDir string
				if idx >= 0 && idx < len(m.tray.recentGames) {
					targetDir = m.tray.recentGames[idx].GameDir
				}
				m.tray.mu.RUnlock()

				if targetDir != "" && m.tray.onLaunchGame != nil {
					go m.tray.onLaunchGame(targetDir)
				}
			}
		}
	}
	return nil
}

func (m *dbusMenuReceiver) GetProperty(id int32, property string) (dbus.Variant, *dbus.Error) {
	items := m.tray.menuItems()
	for _, it := range items {
		if it.ID == id {
			if val, ok := it.Props[property]; ok {
				return val, nil
			}
			break
		}
	}
	return dbus.MakeVariant(""), dbus.NewError("com.canonical.dbusmenu.Error.NoSuchProperty", nil)
}

func (m *dbusMenuReceiver) GetGroupProperties(ids []int32, propertyNames []string) ([]DBusMenuGroupProp, *dbus.Error) {
	items := m.tray.menuItems()
	idSet := make(map[int32]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	var res []DBusMenuGroupProp
	for _, it := range items {
		if len(ids) == 0 || idSet[it.ID] {
			props := make(map[string]dbus.Variant)
			if len(propertyNames) == 0 {
				props = it.Props
			} else {
				for _, name := range propertyNames {
					if v, ok := it.Props[name]; ok {
						props[name] = v
					}
				}
			}
			res = append(res, DBusMenuGroupProp{
				ID:    it.ID,
				Props: props,
			})
		}
	}
	return res, nil
}

func (m *dbusMenuReceiver) GetLayout(parentId int32, recursionDepth int32, propertyNames []string) (uint32, DBusMenuLayoutItem, *dbus.Error) {
	if m.tray != nil {
		m.tray.refreshRecentGames()
	}
	childItems := m.tray.menuItems()
	var childVariants []dbus.Variant
	for _, ch := range childItems {
		childVariants = append(childVariants, dbus.MakeVariant(ch))
	}

	root := DBusMenuLayoutItem{
		ID: 0,
		Props: map[string]dbus.Variant{
			"children-display": dbus.MakeVariant("submenu"),
		},
		Children: childVariants,
	}

	rev := uint32(1)
	if m.tray != nil {
		m.tray.mu.RLock()
		rev = m.tray.revision
		if rev == 0 {
			rev = 1
		}
		m.tray.mu.RUnlock()
	}

	return rev, root, nil
}

// SetRecentGames updates the top played games list displayed in the tray menu.
// Returns true if the menu layout was updated.
func (t *TrayManager) SetRecentGames(games []TrayGame) bool {
	if t == nil {
		return false
	}
	sorted := SortTrayGames(games, 3)

	t.mu.Lock()
	defer t.mu.Unlock()

	if len(sorted) == len(t.recentGames) {
		equal := true
		for i := range sorted {
			if sorted[i] != t.recentGames[i] {
				equal = false
				break
			}
		}
		if equal {
			return false
		}
	}

	t.recentGames = sorted
	t.revision++
	if t.revision == 0 {
		t.revision = 1
	}

	if t.conn != nil && !t.closed {
		_ = t.conn.Emit("/MenuBar", "com.canonical.dbusmenu.LayoutUpdated", t.revision, int32(0))
	}
	return true
}

func (t *TrayManager) refreshRecentGames() bool {
	if t == nil || t.getRecentGames == nil {
		return false
	}
	games := t.getRecentGames()
	return t.SetRecentGames(games)
}

func (t *TrayManager) menuItems() []DBusMenuLayoutItem {
	t.mu.RLock()
	games := make([]TrayGame, len(t.recentGames))
	copy(games, t.recentGames)
	t.mu.RUnlock()

	var items []DBusMenuLayoutItem

	// 1. Last three played games (topmost items)
	for i, g := range games {
		id := int32(10 + i)
		label := strings.TrimSpace(g.Name)
		if label == "" {
			label = filepath.Base(g.GameDir)
		}
		if label == "" {
			label = "Unknown Game"
		}
		items = append(items, DBusMenuLayoutItem{
			ID: id,
			Props: map[string]dbus.Variant{
				"label":   dbus.MakeVariant(strings.ReplaceAll(label, "_", "__")),
				"enabled": dbus.MakeVariant(true),
				"visible": dbus.MakeVariant(true),
			},
			Children: []dbus.Variant{},
		})
	}

	// Separator after recent games
	if len(games) > 0 {
		items = append(items, DBusMenuLayoutItem{
			ID: 4,
			Props: map[string]dbus.Variant{
				"type":    dbus.MakeVariant("separator"),
				"enabled": dbus.MakeVariant(true),
				"visible": dbus.MakeVariant(true),
			},
			Children: []dbus.Variant{},
		})
	}

	// Show Gamux
	items = append(items, DBusMenuLayoutItem{
		ID: 1,
		Props: map[string]dbus.Variant{
			"label":   dbus.MakeVariant("Show Gamux"),
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		},
		Children: []dbus.Variant{},
	})

	// Separator before Quit
	items = append(items, DBusMenuLayoutItem{
		ID: 2,
		Props: map[string]dbus.Variant{
			"type":    dbus.MakeVariant("separator"),
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		},
		Children: []dbus.Variant{},
	})

	// Quit
	items = append(items, DBusMenuLayoutItem{
		ID: 3,
		Props: map[string]dbus.Variant{
			"label":   dbus.MakeVariant("Quit"),
			"enabled": dbus.MakeVariant(true),
			"visible": dbus.MakeVariant(true),
		},
		Children: []dbus.Variant{},
	})

	return items
}

// ensureUserIconInstalled ensures the scalable icon is cached in the user's icon theme directory.
func ensureUserIconInstalled() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	targetDir := filepath.Join(homeDir, ".local", "share", "icons", "hicolor", "scalable", "apps")
	targetFile := filepath.Join(targetDir, "io.github.staernid.gamux.svg")

	if _, err := os.Stat(targetFile); err == nil {
		return targetDir
	}

	// Try copy from local assets if found
	candidates := []string{
		"assets/io.github.staernid.gamux.svg",
		"../assets/io.github.staernid.gamux.svg",
	}
	for _, c := range candidates {
		data, err := os.ReadFile(c)
		if err == nil {
			_ = os.MkdirAll(targetDir, 0755)
			_ = os.WriteFile(targetFile, data, 0644)
			return targetDir
		}
	}
	return targetDir
}

// StartTray initializes the StatusNotifierItem and registers it with the session bus.
func StartTray(cb TrayCallbacks) (*TrayManager, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect session bus: %w", err)
	}

	iconThemePath := ensureUserIconInstalled()

	tray := &TrayManager{
		conn:           conn,
		onActivate:     cb.OnActivate,
		onQuit:         cb.OnQuit,
		onLaunchGame:   cb.OnLaunchGame,
		getRecentGames: cb.GetRecentGames,
		revision:       1,
	}

	if cb.GetRecentGames != nil {
		tray.recentGames = SortTrayGames(cb.GetRecentGames(), 3)
	}

	busName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	tray.busName = busName
	_, err = conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		slog.Warn("Failed to acquire StatusNotifierItem bus name, proceeding with connection name", "busName", busName, "error", err)
	}

	// 1. Export StatusNotifierItem on /StatusNotifierItem
	sniPath := dbus.ObjectPath("/StatusNotifierItem")
	sniReceiverObj := &sniReceiver{tray: tray}
	if err := conn.Export(sniReceiverObj, sniPath, "org.kde.StatusNotifierItem"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("export StatusNotifierItem: %w", err)
	}

	sniProps := prop.Map{
		"org.kde.StatusNotifierItem": {
			"Category":            &prop.Prop{Value: "ApplicationStatus", Emit: prop.EmitTrue},
			"Id":                  &prop.Prop{Value: "io.github.staernid.gamux", Emit: prop.EmitTrue},
			"Title":               &prop.Prop{Value: "gamux", Emit: prop.EmitTrue},
			"Status":              &prop.Prop{Value: "Active", Emit: prop.EmitTrue},
			"WindowId":            &prop.Prop{Value: int32(0), Emit: prop.EmitFalse},
			"IconThemePath":       &prop.Prop{Value: iconThemePath, Emit: prop.EmitFalse},
			"Menu":                &prop.Prop{Value: dbus.ObjectPath("/MenuBar"), Emit: prop.EmitFalse},
			"ItemIsMenu":          &prop.Prop{Value: false, Emit: prop.EmitFalse},
			"IconName":            &prop.Prop{Value: "io.github.staernid.gamux", Emit: prop.EmitTrue},
			"IconPixmap":          &prop.Prop{Value: []KDbusImage{}, Emit: prop.EmitFalse},
			"OverlayIconName":     &prop.Prop{Value: "", Emit: prop.EmitFalse},
			"OverlayIconPixmap":   &prop.Prop{Value: []KDbusImage{}, Emit: prop.EmitFalse},
			"AttentionIconName":   &prop.Prop{Value: "", Emit: prop.EmitFalse},
			"AttentionIconPixmap": &prop.Prop{Value: []KDbusImage{}, Emit: prop.EmitFalse},
			"AttentionMovieName":  &prop.Prop{Value: "", Emit: prop.EmitFalse},
			"ToolTip": &prop.Prop{Value: KDbusToolTip{
				IconName:    "io.github.staernid.gamux",
				Pixmaps:     []KDbusImage{},
				Title:       "gamux",
				Description: "Steam Game Manager",
			}, Emit: prop.EmitTrue},
		},
	}
	if _, err := prop.Export(conn, sniPath, sniProps); err != nil {
		slog.Warn("Failed to export SNI properties", "error", err)
	}

	sniIntrospectNode := &introspect.Node{
		Name: string(sniPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: "org.kde.StatusNotifierItem",
				Methods: []introspect.Method{
					{
						Name: "ContextMenu",
						Args: []introspect.Arg{
							{Name: "x", Type: "i", Direction: "in"},
							{Name: "y", Type: "i", Direction: "in"},
						},
					},
					{
						Name: "Activate",
						Args: []introspect.Arg{
							{Name: "x", Type: "i", Direction: "in"},
							{Name: "y", Type: "i", Direction: "in"},
						},
					},
					{
						Name: "SecondaryActivate",
						Args: []introspect.Arg{
							{Name: "x", Type: "i", Direction: "in"},
							{Name: "y", Type: "i", Direction: "in"},
						},
					},
					{
						Name: "Scroll",
						Args: []introspect.Arg{
							{Name: "delta", Type: "i", Direction: "in"},
							{Name: "orientation", Type: "s", Direction: "in"},
						},
					},
					{
						Name: "ProvideXdgActivationToken",
						Args: []introspect.Arg{
							{Name: "token", Type: "s", Direction: "in"},
						},
					},
				},
				Signals: []introspect.Signal{
					{Name: "NewTitle"},
					{Name: "NewIcon"},
					{Name: "NewAttentionIcon"},
					{Name: "NewOverlayIcon"},
					{Name: "NewMenu"},
					{Name: "NewToolTip"},
					{
						Name: "NewStatus",
						Args: []introspect.Arg{
							{Name: "status", Type: "s"},
						},
					},
				},
				Properties: []introspect.Property{
					{Name: "Category", Type: "s", Access: "read"},
					{Name: "Id", Type: "s", Access: "read"},
					{Name: "Title", Type: "s", Access: "read"},
					{Name: "Status", Type: "s", Access: "read"},
					{Name: "WindowId", Type: "i", Access: "read"},
					{Name: "IconThemePath", Type: "s", Access: "read"},
					{Name: "Menu", Type: "o", Access: "read"},
					{Name: "ItemIsMenu", Type: "b", Access: "read"},
					{Name: "IconName", Type: "s", Access: "read"},
					{Name: "IconPixmap", Type: "a(iiay)", Access: "read"},
					{Name: "OverlayIconName", Type: "s", Access: "read"},
					{Name: "OverlayIconPixmap", Type: "a(iiay)", Access: "read"},
					{Name: "AttentionIconName", Type: "s", Access: "read"},
					{Name: "AttentionIconPixmap", Type: "a(iiay)", Access: "read"},
					{Name: "AttentionMovieName", Type: "s", Access: "read"},
					{Name: "ToolTip", Type: "(sa(iiay)ss)", Access: "read"},
				},
			},
		},
	}
	_ = conn.Export(introspect.NewIntrospectable(sniIntrospectNode), sniPath, "org.freedesktop.DBus.Introspectable")

	// 2. Export DBusMenu on /MenuBar
	menuPath := dbus.ObjectPath("/MenuBar")
	menuReceiverObj := &dbusMenuReceiver{tray: tray}
	if err := conn.Export(menuReceiverObj, menuPath, "com.canonical.dbusmenu"); err != nil {
		slog.Warn("Failed to export DBusMenu", "error", err)
	}

	menuProps := prop.Map{
		"com.canonical.dbusmenu": {
			"Version": &prop.Prop{Value: uint32(3), Emit: prop.EmitFalse},
			"Status":  &prop.Prop{Value: "normal", Emit: prop.EmitFalse},
		},
	}
	if _, err := prop.Export(conn, menuPath, menuProps); err != nil {
		slog.Warn("Failed to export DBusMenu properties", "error", err)
	}

	menuIntrospectNode := &introspect.Node{
		Name: string(menuPath),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: "com.canonical.dbusmenu",
				Methods: []introspect.Method{
					{
						Name: "GetLayout",
						Args: []introspect.Arg{
							{Name: "parentId", Type: "i", Direction: "in"},
							{Name: "recursionDepth", Type: "i", Direction: "in"},
							{Name: "propertyNames", Type: "as", Direction: "in"},
							{Name: "revision", Type: "u", Direction: "out"},
							{Name: "item", Type: "(ia{sv}av)", Direction: "out"},
						},
					},
					{
						Name: "GetGroupProperties",
						Args: []introspect.Arg{
							{Name: "ids", Type: "ai", Direction: "in"},
							{Name: "propertyNames", Type: "as", Direction: "in"},
							{Name: "props", Type: "a(ia{sv})", Direction: "out"},
						},
					},
					{
						Name: "GetProperty",
						Args: []introspect.Arg{
							{Name: "id", Type: "i", Direction: "in"},
							{Name: "property", Type: "s", Direction: "in"},
							{Name: "value", Type: "v", Direction: "out"},
						},
					},
					{
						Name: "Event",
						Args: []introspect.Arg{
							{Name: "id", Type: "i", Direction: "in"},
							{Name: "eventId", Type: "s", Direction: "in"},
							{Name: "data", Type: "v", Direction: "in"},
							{Name: "timestamp", Type: "u", Direction: "in"},
						},
					},
					{
						Name: "AboutToShow",
						Args: []introspect.Arg{
							{Name: "id", Type: "i", Direction: "in"},
							{Name: "needUpdate", Type: "b", Direction: "out"},
						},
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "LayoutUpdated",
						Args: []introspect.Arg{
							{Name: "revision", Type: "u"},
							{Name: "parent", Type: "i"},
						},
					},
				},
				Properties: []introspect.Property{
					{Name: "Version", Type: "u", Access: "read"},
					{Name: "Status", Type: "s", Access: "read"},
				},
			},
		},
	}
	_ = conn.Export(introspect.NewIntrospectable(menuIntrospectNode), menuPath, "org.freedesktop.DBus.Introspectable")

	// 3. Register with StatusNotifierWatcher
	watcherObj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	registerTarget := busName
	call := watcherObj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, registerTarget)
	if call.Err != nil {
		slog.Debug("RegisterStatusNotifierItem with bus name had notice, attempting /StatusNotifierItem fallback", "error", call.Err)
		callFallback := watcherObj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, "/StatusNotifierItem")
		if callFallback.Err != nil {
			slog.Warn("StatusNotifierWatcher registration failed (no active tray host running)", "error", callFallback.Err)
		} else {
			slog.Info("Registered system tray icon with StatusNotifierWatcher using /StatusNotifierItem path")
		}
	} else {
		slog.Info("Registered system tray icon with StatusNotifierWatcher", "service", registerTarget)
	}

	return tray, nil
}

// Close unregisters the tray item and closes the DBus connection.
func (t *TrayManager) Close() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	if t.conn != nil {
		_ = t.conn.Close()
	}
}
