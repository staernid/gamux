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

func (a *App) multiseatSupervisor() *multiseat.Supervisor {
	if a.Engine != nil && a.Engine.Multiseat != nil {
		return a.Engine.Multiseat
	}
	return multiseat.DefaultSupervisor()
}

// LaunchMultiseatSession starts a paired split-screen or streaming session.
func (a *App) LaunchMultiseatSession(sessionCfg multiseat.MultiseatSessionConfig) error {
	return a.multiseatSupervisor().LaunchSession(a.getContext(), sessionCfg)
}

// StopMultiseatSession stops an entire 2-player multi-seat session.
func (a *App) StopMultiseatSession(gameDir string) error {
	return a.multiseatSupervisor().StopSession(gameDir)
}

// StopMultiseatSeat stops only one player's seat in an active session.
func (a *App) StopMultiseatSeat(gameDir string, seatID string) error {
	return a.multiseatSupervisor().StopSeat(gameDir, seatID)
}

// GetMultiseatSessionStatus returns real-time telemetry for a multi-seat session.
func (a *App) GetMultiseatSessionStatus(gameDir string) *multiseat.MultiseatSessionStatus {
	return a.multiseatSupervisor().GetStatus(gameDir)
}

// IsMultiseatSessionRunning checks if a multi-seat session is active for the game.
func (a *App) IsMultiseatSessionRunning(gameDir string) bool {
	return a.multiseatSupervisor().IsSessionRunning(gameDir)
}
