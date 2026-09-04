package multiseat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnumerateInputDevices scans /sys/class/input for connected gamepads and controllers.
func EnumerateInputDevices() ([]InputDevice, error) {
	const inputClassDir = "/sys/class/input"
	entries, err := os.ReadDir(inputClassDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", inputClassDir, err)
	}

	var devices []InputDevice
	seenNodes := make(map[string]bool)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "event") {
			continue
		}

		eventPath := filepath.Join(inputClassDir, name)
		devNode := filepath.Join("/dev", "input", name)
		if seenNodes[devNode] {
			continue
		}

		// Read device name from /sys/class/input/eventX/device/name
		devNameFile := filepath.Join(eventPath, "device", "name")
		devNameBytes, err := os.ReadFile(devNameFile)
		if err != nil {
			continue
		}
		devName := strings.TrimSpace(string(devNameBytes))
		if devName == "" {
			continue
		}

		// Check bus type
		bus := "unknown"
		busFile := filepath.Join(eventPath, "device", "id", "bustype")
		if busBytes, err := os.ReadFile(busFile); err == nil {
			busCode := strings.TrimSpace(string(busBytes))
			switch busCode {
			case "0003":
				bus = "usb"
			case "0005":
				bus = "bluetooth"
			default:
				bus = busCode
			}
		}

		// Determine device roles
		isGamepad := isLikelyGamepad(eventPath, devName)
		isKbd := isLikelyKeyboard(eventPath, devName)
		isMouse := isLikelyMouse(eventPath, devName)

		devices = append(devices, InputDevice{
			DevNode:    devNode,
			SysPath:    eventPath,
			Name:       devName,
			ID:         name,
			IsGamepad:  isGamepad,
			IsKeyboard: isKbd,
			IsMouse:    isMouse,
			Bus:        bus,
		})
		seenNodes[devNode] = true
	}

	return devices, nil
}

// isLikelyGamepad heuristically detects whether an input event node is a gamepad or joystick.
func isLikelyGamepad(eventSysPath, devName string) bool {
	lowerName := strings.ToLower(devName)

	// Filter out keyboards, mice, audio headsets, power buttons, and composite HID devices
	nonGamepadKeywords := []string{
		"keyboard", "mouse", "touchpad", "trackball", "headset", "audio",
		"power button", "sleep button", "video bus", "webcam", "camera",
		" kb", "-kb", "_kb", "system multi axis", "consumer control", "system control",
		"mystic light", "wireless radio", "input accessory", "gaming kb",
	}
	for _, kw := range nonGamepadKeywords {
		if strings.Contains(lowerName, kw) {
			return false
		}
	}

	// Check /dev/input/by-id for explicit -event-kbd or -event-mouse symlinks
	baseEvent := filepath.Base(eventSysPath) // e.g. "event12"
	if byIdEntries, err := os.ReadDir("/dev/input/by-id"); err == nil {
		for _, e := range byIdEntries {
			target, err := os.Readlink(filepath.Join("/dev/input/by-id", e.Name()))
			if err == nil && (target == "../"+baseEvent || target == baseEvent) {
				symName := strings.ToLower(e.Name())
				if strings.Contains(symName, "-event-kbd") || strings.Contains(symName, "-event-mouse") || strings.Contains(symName, "mouse") {
					return false
				}
				if strings.Contains(symName, "-event-joystick") || strings.Contains(symName, "joystick") {
					return true
				}
			}
		}
	}

	// Positive gamepad keyword indicators
	gamepadKeywords := []string{
		"controller", "gamepad", "joystick", "xbox", "dualshock", "dualsense",
		"nintendo", "switch pro", "steam deck", "steam controller", "flydigi",
		"8bitdo", "stadia", "shield", "logitech gamepad", "razer raiju", "wolverine",
		"green ghost", "thrustmaster", "hori", "gamesir",
	}
	for _, kw := range gamepadKeywords {
		if strings.Contains(lowerName, kw) {
			return true
		}
	}

	// Check if parent device directory has any "js*" child (indicating Linux joystick subsystem bound)
	parentDev := filepath.Join(eventSysPath, "device")
	if children, err := os.ReadDir(parentDev); err == nil {
		for _, child := range children {
			if strings.HasPrefix(child.Name(), "js") {
				return true
			}
		}
	}

	return false
}

// isLikelyKeyboard heuristically detects whether an input event node is a keyboard.
func isLikelyKeyboard(eventSysPath, devName string) bool {
	lowerName := strings.ToLower(devName)

	excludeKeywords := []string{
		"mouse", "touchpad", "trackball", "headset", "audio",
		"power button", "sleep button", "video bus", "webcam", "camera",
		"controller", "gamepad", "joystick",
	}
	for _, kw := range excludeKeywords {
		if strings.Contains(lowerName, kw) {
			return false
		}
	}

	// Check /dev/input/by-id for explicit -event-kbd symlink
	baseEvent := filepath.Base(eventSysPath)
	if byIdEntries, err := os.ReadDir("/dev/input/by-id"); err == nil {
		for _, e := range byIdEntries {
			target, err := os.Readlink(filepath.Join("/dev/input/by-id", e.Name()))
			if err == nil && (target == "../"+baseEvent || target == baseEvent) {
				symName := strings.ToLower(e.Name())
				if strings.Contains(symName, "-event-kbd") {
					return true
				}
				if strings.Contains(symName, "-event-mouse") || strings.Contains(symName, "mouse") {
					return false
				}
			}
		}
	}

	// Positive keyboard keywords
	keyboardKeywords := []string{
		"keyboard", "keychron", "ducky", "filco", "wooting", "ergodox", "planck",
		"anne pro", "corsair gaming keyboard", "logitech keyboard", "huntsman",
		"apex pro", "deathstalker", "blackwidow", "alloy", "k95", "k70", "k65",
	}
	for _, kw := range keyboardKeywords {
		if strings.Contains(lowerName, kw) {
			return true
		}
	}

	// Check capabilities/key bitmask if available
	keyCapFile := filepath.Join(eventSysPath, "device", "capabilities", "key")
	if keyBytes, err := os.ReadFile(keyCapFile); err == nil {
		keyHex := strings.TrimSpace(string(keyBytes))
		if len(keyHex) >= 20 {
			return true
		}
	}

	return false
}

// isLikelyMouse heuristically detects whether an input event node is a mouse or pointer device.
func isLikelyMouse(eventSysPath, devName string) bool {
	lowerName := strings.ToLower(devName)

	excludeKeywords := []string{
		"power button", "sleep button", "video bus", "webcam", "camera",
		"controller", "gamepad", "joystick", "headset",
	}
	for _, kw := range excludeKeywords {
		if strings.Contains(lowerName, kw) {
			return false
		}
	}

	// Check /dev/input/by-id for explicit -event-mouse symlink
	baseEvent := filepath.Base(eventSysPath)
	if byIdEntries, err := os.ReadDir("/dev/input/by-id"); err == nil {
		for _, e := range byIdEntries {
			target, err := os.Readlink(filepath.Join("/dev/input/by-id", e.Name()))
			if err == nil && (target == "../"+baseEvent || target == baseEvent) {
				symName := strings.ToLower(e.Name())
				if strings.Contains(symName, "-event-mouse") || strings.Contains(symName, "mouse") {
					return true
				}
				if strings.Contains(symName, "-event-kbd") {
					return false
				}
			}
		}
	}

	// Positive mouse keywords
	mouseKeywords := []string{
		"mouse", "trackball", "touchpad", "pointer", "superlight", "deathadder",
		"viper", "basilisk", "g502", "g pro", "rival", "aerox", "pulsefire", "kone",
	}
	for _, kw := range mouseKeywords {
		if strings.Contains(lowerName, kw) {
			return true
		}
	}

	// Check capabilities/rel (relative axes for mice: REL_X / REL_Y)
	relCapFile := filepath.Join(eventSysPath, "device", "capabilities", "rel")
	if relBytes, err := os.ReadFile(relCapFile); err == nil {
		relHex := strings.TrimSpace(string(relBytes))
		if relHex != "" && relHex != "0" {
			return true
		}
	}

	return false
}

// FilterGamepads returns only devices flagged as gamepads.
func FilterGamepads(devices []InputDevice) []InputDevice {
	var pads []InputDevice
	for _, d := range devices {
		if d.IsGamepad {
			pads = append(pads, d)
		}
	}
	return pads
}

// FilterKeyboards returns only devices flagged as keyboards.
func FilterKeyboards(devices []InputDevice) []InputDevice {
	var kbds []InputDevice
	for _, d := range devices {
		if d.IsKeyboard {
			kbds = append(kbds, d)
		}
	}
	return kbds
}

// FilterMice returns only devices flagged as mice/pointers.
func FilterMice(devices []InputDevice) []InputDevice {
	var mice []InputDevice
	for _, d := range devices {
		if d.IsMouse {
			mice = append(mice, d)
		}
	}
	return mice
}

func containsString(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// FindGamepadCompanionNodes discovers all companion js*, hidraw*, and sibling event*
// device nodes that belong to the same hardware device as devNode (e.g. /dev/input/event29).
func FindGamepadCompanionNodes(devNode string) (jsNodes, hidrawNodes, siblingEventNodes []string) {
	if devNode == "" {
		return nil, nil, nil
	}

	base := filepath.Base(devNode) // e.g. "event29"
	sysInput := filepath.Join("/sys/class/input", base)

	sysDev, err := filepath.EvalSymlinks(filepath.Join(sysInput, "device"))
	if err != nil {
		sysDev = filepath.Join(sysInput, "device")
	}

	// 1. Sibling js* and event* nodes directly under input device
	if entries, err := os.ReadDir(sysDev); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "js") {
				p := filepath.Join("/dev/input", name)
				if _, err := os.Stat(p); err == nil {
					jsNodes = append(jsNodes, p)
				}
			} else if strings.HasPrefix(name, "event") && name != base {
				p := filepath.Join("/dev/input", name)
				if _, err := os.Stat(p); err == nil {
					siblingEventNodes = append(siblingEventNodes, p)
				}
			}
		}
	}

	// 2. Parent USB or composite device to find companion interfaces (e.g. hidraw4, hidraw9, sibling event30)
	curr := sysDev
	var usbParent string
	for i := 0; i < 4; i++ {
		curr = filepath.Dir(curr)
		if curr == "/" || curr == "." {
			break
		}
		if entries, err := os.ReadDir(curr); err == nil {
			for _, e := range entries {
				if strings.Contains(e.Name(), ":") {
					usbParent = curr
					break
				}
			}
		}
		if usbParent != "" {
			break
		}
	}

	if usbParent != "" {
		_ = filepath.Walk(usbParent, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}
			name := info.Name()
			if strings.HasPrefix(name, "hidraw") && name != "hidraw" {
				p := filepath.Join("/dev", name)
				if _, err := os.Stat(p); err == nil && !containsString(hidrawNodes, p) {
					hidrawNodes = append(hidrawNodes, p)
				}
			} else if strings.HasPrefix(name, "js") && name != "js" {
				p := filepath.Join("/dev/input", name)
				if _, err := os.Stat(p); err == nil && !containsString(jsNodes, p) {
					jsNodes = append(jsNodes, p)
				}
			} else if strings.HasPrefix(name, "event") && name != "event" && name != base {
				p := filepath.Join("/dev/input", name)
				if _, err := os.Stat(p); err == nil && !containsString(siblingEventNodes, p) {
					siblingEventNodes = append(siblingEventNodes, p)
				}
			}
			return nil
		})
	}

	return jsNodes, hidrawNodes, siblingEventNodes
}

// FindAllHostGamepadNodes discovers all dev nodes (event, js, hidraw) across all detected host gamepads.
func FindAllHostGamepadNodes() []string {
	devices, err := EnumerateInputDevices()
	if err != nil {
		return nil
	}
	pads := FilterGamepads(devices)
	var allNodes []string
	seen := make(map[string]bool)

	for _, pad := range pads {
		if !seen[pad.DevNode] {
			allNodes = append(allNodes, pad.DevNode)
			seen[pad.DevNode] = true
		}
		js, hid, sib := FindGamepadCompanionNodes(pad.DevNode)
		for _, n := range append(append(js, hid...), sib...) {
			if !seen[n] {
				allNodes = append(allNodes, n)
				seen[n] = true
			}
		}
	}

	return allNodes
}

// FindAllHostKBMNodes discovers all event and companion nodes across all detected keyboards and mice.
func FindAllHostKBMNodes() (keyboards, mice []string) {
	devices, err := EnumerateInputDevices()
	if err != nil {
		return nil, nil
	}

	kbds := FilterKeyboards(devices)
	seenKbd := make(map[string]bool)
	for _, k := range kbds {
		if !seenKbd[k.DevNode] {
			keyboards = append(keyboards, k.DevNode)
			seenKbd[k.DevNode] = true
		}
		_, _, sib := FindGamepadCompanionNodes(k.DevNode)
		for _, s := range sib {
			if !seenKbd[s] {
				keyboards = append(keyboards, s)
				seenKbd[s] = true
			}
		}
	}

	ms := FilterMice(devices)
	seenMouse := make(map[string]bool)
	for _, m := range ms {
		if !seenMouse[m.DevNode] {
			mice = append(mice, m.DevNode)
			seenMouse[m.DevNode] = true
		}
		_, _, sib := FindGamepadCompanionNodes(m.DevNode)
		for _, s := range sib {
			if !seenMouse[s] {
				mice = append(mice, s)
				seenMouse[s] = true
			}
		}
	}

	return keyboards, mice
}

