package multiseat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	sysClassInputDir = "/sys/class/input"
	devInputDir      = "/dev/input"
	devDir           = "/dev"
)

// EnumerateInputDevices scans /sys/class/input for connected gamepads and controllers.
func EnumerateInputDevices() ([]InputDevice, error) {
	entries, err := os.ReadDir(sysClassInputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", sysClassInputDir, err)
	}

	var devices []InputDevice
	seenNodes := make(map[string]bool)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "event") {
			continue
		}

		eventPath := filepath.Join(sysClassInputDir, name)
		devNode := filepath.Join(devInputDir, name)
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
// For composite devices with multiple event interfaces, it groups them by physical
// device and keeps the primary keyboard interface (with richest key capabilities).
func FilterKeyboards(devices []InputDevice) []InputDevice {
	var kbds []InputDevice
	for _, d := range devices {
		if d.IsKeyboard {
			kbds = append(kbds, d)
		}
	}
	return deduplicateKeyboards(kbds)
}

func deduplicateKeyboards(kbds []InputDevice) []InputDevice {
	if len(kbds) <= 1 {
		return kbds
	}
	type kbdScore struct {
		dev      InputDevice
		capScore int
	}
	byRoot := make(map[string]kbdScore)
	var order []string

	for _, k := range kbds {
		_, root := findDeviceRootSysPath(k.DevNode)
		if root == "" {
			root = k.DevNode
		}
		score := 0
		base := filepath.Base(k.DevNode)
		if realNode, err := filepath.EvalSymlinks(k.DevNode); err == nil && realNode != "" {
			base = filepath.Base(realNode)
		}
		keyCapFile := filepath.Join(sysClassInputDir, base, "device", "capabilities", "key")
		if data, err := os.ReadFile(keyCapFile); err == nil {
			score = len(strings.TrimSpace(string(data)))
		}
		if existing, exists := byRoot[root]; exists {
			if score > existing.capScore {
				byRoot[root] = kbdScore{dev: k, capScore: score}
			}
		} else {
			byRoot[root] = kbdScore{dev: k, capScore: score}
			order = append(order, root)
		}
	}

	var result []InputDevice
	for _, root := range order {
		result = append(result, byRoot[root].dev)
	}
	return result
}

// FilterMice returns only devices flagged as mice/pointers.
// For composite devices with multiple event interfaces, it groups them by physical
// device and keeps the primary pointer interface.
func FilterMice(devices []InputDevice) []InputDevice {
	var mice []InputDevice
	for _, d := range devices {
		if d.IsMouse {
			mice = append(mice, d)
		}
	}
	return deduplicateMice(mice)
}

func deduplicateMice(mice []InputDevice) []InputDevice {
	if len(mice) <= 1 {
		return mice
	}
	byRoot := make(map[string]InputDevice)
	var order []string

	for _, m := range mice {
		_, root := findDeviceRootSysPath(m.DevNode)
		if root == "" {
			root = m.DevNode
		}
		if _, exists := byRoot[root]; !exists {
			byRoot[root] = m
			order = append(order, root)
		}
	}

	var result []InputDevice
	for _, root := range order {
		result = append(result, byRoot[root])
	}
	return result
}

func containsString(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// findDeviceRootSysPath resolves the physical hardware root directory (e.g. parent USB or Bluetooth device)
// for a given /dev/input/eventX node or symlink by inspecting sysfs.
func findDeviceRootSysPath(devNode string) (baseEvent, rootSysPath string) {
	if devNode == "" {
		return "", ""
	}
	realNode, err := filepath.EvalSymlinks(devNode)
	if err != nil || realNode == "" {
		realNode = devNode
	}
	base := filepath.Base(realNode) // e.g. "event4"
	if !strings.HasPrefix(base, "event") {
		return "", ""
	}
	sysInput := filepath.Join(sysClassInputDir, base)
	sysDev, err := filepath.EvalSymlinks(filepath.Join(sysInput, "device"))
	if err != nil {
		sysDev = filepath.Join(sysInput, "device")
	}

	curr := sysDev
	var rootPath string
	for i := 0; i < 8; i++ {
		curr = filepath.Dir(curr)
		if curr == "/" || curr == "." {
			break
		}
		// In Linux sysfs, a physical USB device root contains idVendor,
		// and Bluetooth devices contain address or uevent.
		if fi, err := os.Stat(filepath.Join(curr, "idVendor")); err == nil && !fi.IsDir() {
			rootPath = curr
			break
		}
		if rootPath == "" {
			if fi, err := os.Stat(filepath.Join(curr, "uevent")); err == nil && !fi.IsDir() {
				rootPath = curr
			}
		}
	}
	if rootPath == "" {
		rootPath = sysDev
	}
	return base, rootPath
}

// FindDeviceCompanionNodes discovers all companion sibling /dev/input/event* device nodes
// that belong to the same physical hardware device as devNode (e.g. companion interfaces on composite keyboards/mice).
func FindDeviceCompanionNodes(devNode string) []string {
	base, rootPath := findDeviceRootSysPath(devNode)
	if base == "" || rootPath == "" {
		return nil
	}

	var siblings []string
	seen := make(map[string]bool)

	_ = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "event") && name != "event" && name != base {
			p := filepath.Join(devInputDir, name)
			if _, err := os.Stat(p); err == nil && !seen[p] {
				siblings = append(siblings, p)
				seen[p] = true
			}
		}
		return nil
	})

	return siblings
}

// FindGamepadCompanionNodes discovers all companion js*, hidraw*, and sibling event*
// device nodes that belong to the same hardware device as devNode (e.g. /dev/input/event29).
func FindGamepadCompanionNodes(devNode string) (jsNodes, hidrawNodes, siblingEventNodes []string) {
	base, rootPath := findDeviceRootSysPath(devNode)
	if base == "" || rootPath == "" {
		return nil, nil, nil
	}

	seenJs := make(map[string]bool)
	seenHid := make(map[string]bool)
	seenEvent := make(map[string]bool)

	_ = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, "hidraw") && name != "hidraw" {
			p := filepath.Join(devDir, name)
			if _, err := os.Stat(p); err == nil && !seenHid[p] {
				hidrawNodes = append(hidrawNodes, p)
				seenHid[p] = true
			}
		} else if strings.HasPrefix(name, "js") && name != "js" {
			p := filepath.Join(devInputDir, name)
			if _, err := os.Stat(p); err == nil && !seenJs[p] {
				jsNodes = append(jsNodes, p)
				seenJs[p] = true
			}
		} else if strings.HasPrefix(name, "event") && name != "event" && name != base {
			p := filepath.Join(devInputDir, name)
			if _, err := os.Stat(p); err == nil && !seenEvent[p] {
				siblingEventNodes = append(siblingEventNodes, p)
				seenEvent[p] = true
			}
		}
		return nil
	})

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

