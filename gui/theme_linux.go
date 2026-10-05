//go:build linux

package gui

import (
	"context"
	"log/slog"

	"github.com/godbus/dbus/v5"
)

// monitorSystemThemeChanges subscribes to XDG Portal and GNOME SettingsChanged signals
// to detect runtime desktop dark/light mode toggles across KDE Plasma, GNOME, and Freedesktop portals.
func (a *App) monitorSystemThemeChanges(ctx context.Context) {
	go func() {
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			slog.Debug("Failed to connect to session bus for theme monitoring", "error", err)
			return
		}
		defer conn.Close()

		call := conn.BusObject().Call(
			"org.freedesktop.DBus.AddMatch",
			0,
			"type='signal',interface='org.freedesktop.portal.Settings',member='SettingChanged'",
		)
		if call.Err != nil {
			slog.Debug("Failed to subscribe to portal SettingChanged", "error", call.Err)
			return
		}

		ch := make(chan *dbus.Signal, 10)
		conn.Signal(ch)
		defer conn.RemoveSignal(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case sig, ok := <-ch:
				if !ok {
					return
				}
				if len(sig.Body) < 3 {
					continue
				}
				namespace, _ := sig.Body[0].(string)
				key, _ := sig.Body[1].(string)
				if key != "color-scheme" {
					continue
				}

				isDark := true
				handled := false

				// FreeDesktop Portal: namespace "org.freedesktop.appearance", uint32 (1 = dark, 2 = light)
				if namespace == "org.freedesktop.appearance" {
					if variant, ok := sig.Body[2].(dbus.Variant); ok {
						if val, ok := variant.Value().(uint32); ok {
							isDark = (val != 2) // 1 = dark, 2 = light
							handled = true
						}
					}
				}

				// GNOME Interface: namespace "org.gnome.desktop.interface", string ("prefer-dark" vs "default")
				if namespace == "org.gnome.desktop.interface" {
					if variant, ok := sig.Body[2].(dbus.Variant); ok {
						if val, ok := variant.Value().(string); ok {
							isDark = (val == "prefer-dark")
							handled = true
						}
					}
				}

				if handled {
					slog.Debug("Detected desktop color-scheme change via DBus", "namespace", namespace, "isDark", isDark)
					safeEmit(a.ctx, "gamux:system-theme-changed", map[string]any{
						"isDark": isDark,
					})
				}
			}
		}
	}()
}
