package gui

import (
	"fmt"

	"github.com/staernid/gamux/multiseat"
)

// GetMultiseatCapabilities returns available host tools for multi-seat support.
func (a *App) GetMultiseatCapabilities() multiseat.SystemCapabilities {
	return multiseat.CheckCapabilities()
}

// GetConnectedGamepads scans the host system and returns detected gamepads.
func (a *App) GetConnectedGamepads() ([]multiseat.InputDevice, error) {
	devices, err := multiseat.EnumerateInputDevices()
	if err != nil {
		return nil, fmt.Errorf("enumerate input devices: %w", err)
	}
	return multiseat.FilterGamepads(devices), nil
}

// GetConnectedKeyboards scans the host system and returns detected keyboards.
func (a *App) GetConnectedKeyboards() ([]multiseat.InputDevice, error) {
	devices, err := multiseat.EnumerateInputDevices()
	if err != nil {
		return nil, fmt.Errorf("enumerate keyboards: %w", err)
	}
	return multiseat.FilterKeyboards(devices), nil
}

// GetConnectedMice scans the host system and returns detected mice/pointers.
func (a *App) GetConnectedMice() ([]multiseat.InputDevice, error) {
	devices, err := multiseat.EnumerateInputDevices()
	if err != nil {
		return nil, fmt.Errorf("enumerate mice: %w", err)
	}
	return multiseat.FilterMice(devices), nil
}

// LaunchMultiseatSession starts a paired split-screen or streaming session.
func (a *App) LaunchMultiseatSession(sessionCfg multiseat.MultiseatSessionConfig) error {
	sup := multiseat.DefaultSupervisor()
	if a.Config != nil {
		sup.SetConfig(a.Config)
	}
	if a.Engine != nil && a.Engine.EventBus != nil {
		sup.SetBus(a.Engine.EventBus)
	}
	return sup.LaunchSession(a.getContext(), sessionCfg)
}

// StopMultiseatSession stops an entire 2-player multi-seat session.
func (a *App) StopMultiseatSession(gameDir string) error {
	return multiseat.DefaultSupervisor().StopSession(gameDir)
}

// StopMultiseatSeat stops only one player's seat in an active session.
func (a *App) StopMultiseatSeat(gameDir string, seatID string) error {
	return multiseat.DefaultSupervisor().StopSeat(gameDir, seatID)
}

// GetMultiseatSessionStatus returns real-time telemetry for a multi-seat session.
func (a *App) GetMultiseatSessionStatus(gameDir string) *multiseat.MultiseatSessionStatus {
	return multiseat.DefaultSupervisor().GetStatus(gameDir)
}

// IsMultiseatSessionRunning checks if a multi-seat session is active for the game.
func (a *App) IsMultiseatSessionRunning(gameDir string) bool {
	return multiseat.DefaultSupervisor().IsSessionRunning(gameDir)
}
