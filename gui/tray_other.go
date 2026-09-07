//go:build !linux

package gui

// TrayManager stub for non-Linux platforms.
type TrayManager struct{}

// StartTray stub for non-Linux platforms.
func StartTray(cb TrayCallbacks) (*TrayManager, error) {
	return nil, nil
}

// SetRecentGames stub for non-Linux platforms.
func (t *TrayManager) SetRecentGames(games []TrayGame) bool {
	return false
}

// Close stub for non-Linux platforms.
func (t *TrayManager) Close() {}

