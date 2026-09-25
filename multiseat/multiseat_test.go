package multiseat

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/staernid/gamux/util"
)

func TestCalculateSplitGeometry(t *testing.T) {
	totalW, totalH := 1920, 1080

	// Side-by-Side (left/right vertical cut)
	w1, h1, x1, y1 := CalculateSplitGeometry(LayoutSideBySide, "p1", totalW, totalH)
	if w1 != 960 || h1 != 1080 || x1 != 0 || y1 != 0 {
		t.Fatalf("p1 side-by-side split mismatch: w=%d, h=%d, x=%d, y=%d", w1, h1, x1, y1)
	}

	w2, h2, x2, y2 := CalculateSplitGeometry(LayoutSideBySide, "p2", totalW, totalH)
	if w2 != 960 || h2 != 1080 || x2 != 960 || y2 != 0 {
		t.Fatalf("p2 side-by-side split mismatch: w=%d, h=%d, x=%d, y=%d", w2, h2, x2, y2)
	}

	// Top/Bottom (horizontal cut)
	vw1, vh1, vx1, vy1 := CalculateSplitGeometry(LayoutTopBottom, "p1", totalW, totalH)
	if vw1 != 1920 || vh1 != 540 || vx1 != 0 || vy1 != 0 {
		t.Fatalf("p1 top/bottom split mismatch: w=%d, h=%d, x=%d, y=%d", vw1, vh1, vx1, vy1)
	}

	vw2, vh2, vx2, vy2 := CalculateSplitGeometry(LayoutTopBottom, "p2", totalW, totalH)
	if vw2 != 1920 || vh2 != 540 || vx2 != 0 || vy2 != 540 {
		t.Fatalf("p2 top/bottom split mismatch: w=%d, h=%d, x=%d, y=%d", vw2, vh2, vx2, vy2)
	}
}

func TestBuildGamescopeArgs(t *testing.T) {
	opts := DisplayOptions{
		Mode:            SeatModeLocal,
		Layout:          LayoutDualMonitor,
		SeatID:          "p2",
		DisplayIndex:    1,
		TargetWidth:     1920,
		TargetHeight:    1080,
		TargetFPS:       100,
		Fullscreen:      true,
		ForceGrabCursor: true,
		GrabKeyboard:    true,
		CursorPath:      "/tmp/cursor.png",
		Command:         []string{"wine", "game.exe"},
	}

	args, err := BuildGamescopeArgs(opts)
	if err != nil {
		t.Fatalf("BuildGamescopeArgs failed: %v", err)
	}

	str := strings.Join(args, " ")
	if !strings.Contains(str, "--display-index 1") {
		t.Errorf("expected --display-index 1 in %s", str)
	}
	if !strings.Contains(str, "-r 100") {
		t.Errorf("expected -r 100 in %s", str)
	}
	if !strings.Contains(str, "-f") {
		t.Errorf("expected -f in %s", str)
	}
	if !strings.Contains(str, "--force-grab-cursor") {
		t.Errorf("expected --force-grab-cursor in %s", str)
	}
	if !strings.Contains(str, "-g") {
		t.Errorf("expected -g (grab keyboard) in %s", str)
	}
	if !strings.Contains(str, "--cursor /tmp/cursor.png") {
		t.Errorf("expected --cursor in %s", str)
	}
	if !strings.Contains(str, "-- wine game.exe") {
		t.Errorf("expected command terminator and inner command in %s", str)
	}
}

func TestBuildCageArgs(t *testing.T) {
	cmd := []string{"gamescope", "-W", "1920", "--", "game.exe"}
	args, err := BuildCageArgs("wayland-gamux-p1", cmd)
	if err != nil {
		t.Fatalf("BuildCageArgs failed: %v", err)
	}
	str := strings.Join(args, " ")
	if !strings.Contains(str, "-s -- gamescope") {
		t.Errorf("expected cage args with inner gamescope, got: %s", str)
	}
}

func TestBuildPastaArgs(t *testing.T) {
	opts := PastaOptions{
		EnableBroadcastRelay: true,
		BroadcastPort:        47584,
		ForwardTCPPorts:      []int{7777},
		ForwardUDPPorts:      []int{27015},
		Command:              []string{"bwrap", "--version"},
	}

	args, err := BuildPastaArgs(opts)
	if err != nil {
		t.Fatalf("BuildPastaArgs failed: %v", err)
	}

	str := strings.Join(args, " ")
	if !strings.Contains(str, "--config-net") {
		t.Errorf("expected --config-net in %s", str)
	}
	if !strings.Contains(str, "-t 7777") {
		t.Errorf("expected -t 7777 in %s", str)
	}
	if !strings.Contains(str, "-u 27015") {
		t.Errorf("expected -u 27015 in %s", str)
	}
	if !strings.Contains(str, "-u 47584") {
		t.Errorf("expected -u 47584 in %s", str)
	}
	if !strings.Contains(str, "-- bwrap --version") {
		t.Errorf("expected command terminator in %s", str)
	}
}

func TestEnsureSeatProfile(t *testing.T) {
	tmpDir := t.TempDir()
	prof := PlayerProfile{
		AccountName:    "CoopBuddy",
		AccountSteamID: "76561198000000002",
		Language:       "german",
		IPCountry:      "DE",
	}

	if err := EnsureSeatProfile(tmpDir, prof); err != nil {
		t.Fatalf("EnsureSeatProfile failed: %v", err)
	}

	iniPath := filepath.Join(tmpDir, "home", ".local", "share", "GSE Saves", "settings", "configs.user.ini")
	data, err := os.ReadFile(iniPath)
	if err != nil {
		t.Fatalf("read configs.user.ini: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "account_name=CoopBuddy") {
		t.Errorf("missing account_name in ini: %s", content)
	}
	if !strings.Contains(content, "account_steamid=76561198000000002") {
		t.Errorf("missing account_steamid in ini: %s", content)
	}
	if !strings.Contains(content, "language=german") {
		t.Errorf("missing language in ini: %s", content)
	}
	if !strings.Contains(content, "ip_country=DE") {
		t.Errorf("missing ip_country in ini: %s", content)
	}

	broadcastsFile := filepath.Join(tmpDir, "home", ".local", "share", "GSE Saves", "settings", "custom_broadcasts.txt")
	bData, err := os.ReadFile(broadcastsFile)
	if err != nil {
		t.Fatalf("read custom_broadcasts.txt: %v", err)
	}
	if !strings.Contains(string(bData), "127.0.0.1") {
		t.Errorf("expected 127.0.0.1 in custom_broadcasts.txt, got: %s", string(bData))
	}
}

func TestBuildBwrapArgs(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "games", "MyGame")
	_ = os.MkdirAll(gameDir, 0755)

	profileDir := filepath.Join(tmpDir, "profiles", "player2")
	_ = os.MkdirAll(profileDir, 0755)

	prefixDir := filepath.Join(profileDir, "prefix")
	opts := BwrapOptions{
		SeatID:        "p2",
		ProfileDir:    profileDir,
		GameDir:       gameDir,
		WinePrefix:    prefixDir,
		AudioSink:     "gamux-p2",
		Command:       []string{"wine", "game.exe"},
		DieWithParent: true,
	}

	args, err := BuildBwrapArgs(opts)
	if err != nil {
		t.Fatalf("BuildBwrapArgs failed: %v", err)
	}

	str := strings.Join(args, " ")
	if !strings.Contains(str, "--setenv WINEPREFIX "+prefixDir) {
		t.Errorf("expected WINEPREFIX setenv in %s", str)
	}
	if !strings.Contains(str, "--setenv STEAM_COMPAT_DATA_PATH "+prefixDir) {
		t.Errorf("expected STEAM_COMPAT_DATA_PATH setenv in %s", str)
	}
	if !strings.Contains(str, "--die-with-parent") {
		t.Errorf("expected --die-with-parent in %s", str)
	}
	if !strings.Contains(str, "--ro-bind "+gameDir) {
		t.Errorf("expected read-only game binding in %s", str)
	}
	if !strings.Contains(str, "--bind "+profileDir+" "+profileDir) {
		t.Errorf("expected read-write profile directory binding in %s", str)
	}
	if !strings.Contains(str, "--setenv PULSE_SINK gamux-p2") {
		t.Errorf("expected PULSE_SINK setenv in %s", str)
	}
	expectedHome := filepath.Join(profileDir, "home")
	if !strings.Contains(str, "--setenv HOME "+expectedHome) {
		t.Errorf("expected HOME setenv in %s", str)
	}
	if os.Getuid() != 0 {
		if !strings.Contains(str, "--unshare-user") {
			t.Errorf("expected --unshare-user in %s", str)
		}
		if !strings.Contains(str, fmt.Sprintf("--uid %d", os.Getuid())) {
			t.Errorf("expected --uid %d in %s", os.Getuid(), str)
		}
	}
	if !strings.Contains(str, "--setenv XDG_RUNTIME_DIR") {
		t.Errorf("expected XDG_RUNTIME_DIR in %s", str)
	}
	realHome, _ := os.UserHomeDir()
	if realHome != "" {
		if util.FileExists(filepath.Join(realHome, ".cache")) {
			expectedCache := filepath.Join(profileDir, "home", ".cache")
			if !strings.Contains(str, "--bind "+expectedCache+" "+filepath.Join(realHome, ".cache")) {
				t.Errorf("expected .cache bind in %s", str)
			}
		}
		if util.FileExists(filepath.Join(realHome, ".config")) {
			expectedConfig := filepath.Join(profileDir, "home", ".config")
			if !strings.Contains(str, "--bind "+expectedConfig+" "+filepath.Join(realHome, ".config")) {
				t.Errorf("expected .config bind in %s", str)
			}
		}
	}
	if !strings.Contains(str, "-- wine game.exe") {
		t.Errorf("expected command terminator in %s", str)
	}
}

func TestBuildBwrapArgs_CompatibilityToolsSymlinkProtection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	realCompat := filepath.Join(tmpDir, ".local", "share", "Steam", "compatibilitytools.d")
	_ = os.MkdirAll(realCompat, 0755)

	profileDir := filepath.Join(tmpDir, "profile_p2")
	seatHome := filepath.Join(profileDir, "home")
	seatCompat := filepath.Join(seatHome, ".local", "share", "Steam", "compatibilitytools.d")
	_ = os.MkdirAll(filepath.Dir(seatCompat), 0755)
	_ = os.Symlink(realCompat, seatCompat)

	opts := BwrapOptions{
		SeatID:        "p2",
		ProfileDir:    profileDir,
		GameDir:       tmpDir,
		WinePrefix:    filepath.Join(tmpDir, "prefix"),
		Command:       []string{"wine", "game.exe"},
		DieWithParent: true,
	}

	_, err := BuildBwrapArgs(opts)
	if err != nil {
		t.Fatalf("BuildBwrapArgs failed: %v", err)
	}

	// Verify that the symlink was removed and is now a directory so bwrap doesn't fail
	fi, err := os.Lstat(seatCompat)
	if err != nil {
		t.Fatalf("stat seatCompat failed: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Errorf("expected seatCompat to NOT be a symlink")
	}
	if !fi.IsDir() {
		t.Errorf("expected seatCompat to be a directory")
	}
}

func TestBuildBwrapArgs_Player1_RealHome(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "games", "MyGame")
	_ = os.MkdirAll(gameDir, 0755)

	profileDir := filepath.Join(tmpDir, "profiles", "player1")
	_ = os.MkdirAll(profileDir, 0755)

	realHome, _ := os.UserHomeDir()
	if realHome == "" {
		t.Skip("UserHomeDir not available")
	}

	opts := BwrapOptions{
		SeatID:        "p1",
		ProfileDir:    profileDir,
		GameDir:       gameDir,
		WinePrefix:    filepath.Join(realHome, "Games", "umu", "12345"),
		AudioSink:     "gamux-p1",
		Command:       []string{"wine", "game.exe"},
		DieWithParent: true,
	}

	args, err := BuildBwrapArgs(opts)
	if err != nil {
		t.Fatalf("BuildBwrapArgs failed: %v", err)
	}

	str := strings.Join(args, " ")
	if !strings.Contains(str, "--bind "+realHome+" "+realHome) {
		t.Errorf("expected read-write real home bind for p1, got: %s", str)
	}
	if strings.Contains(str, "--ro-bind "+realHome+" "+realHome) {
		t.Errorf("did not expect read-only real home bind for p1, got: %s", str)
	}
	if !strings.Contains(str, "--setenv HOME "+realHome) {
		t.Errorf("expected HOME to be real home for p1, got: %s", str)
	}
	if strings.Contains(str, filepath.Join(profileDir, "home")) {
		t.Errorf("did not expect any reference to fake profile home for p1, got: %s", str)
	}
}

func TestResolvePastaBinary(t *testing.T) {
	caps := CheckCapabilities()
	if !caps.HasPasta {
		t.Skip("pasta not installed on host")
	}

	bin, err := ResolvePastaBinary()
	if err != nil {
		t.Fatalf("ResolvePastaBinary failed: %v", err)
	}
	if bin == "" {
		t.Fatal("expected non-empty pasta binary path")
	}
	if fi, err := os.Stat(bin); err != nil || fi.Mode()&0111 == 0 {
		t.Fatalf("resolved pasta %s is not executable: %v", bin, err)
	}
}

func TestGenerateSunshineConfig(t *testing.T) {
	tmpDir := t.TempDir()
	opts := SunshineConfig{
		Port:          48010,
		WebUIPort:     48012,
		AudioSink:     "gamux-p2",
		WaylandSocket: "wayland-gamux-p2",
		ConfigDir:     tmpDir,
	}

	confPath, err := GenerateSunshineConfig(opts)
	if err != nil {
		t.Fatalf("GenerateSunshineConfig failed: %v", err)
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read sunshine.conf: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "port = 48010") {
		t.Errorf("missing port in sunshine.conf: %s", content)
	}
	if !strings.Contains(content, "capture = wlr") {
		t.Errorf("missing capture = wlr in sunshine.conf: %s", content)
	}
	if !strings.Contains(content, "audio_sink = gamux-p2") {
		t.Errorf("missing audio_sink in sunshine.conf: %s", content)
	}
	if !strings.Contains(content, "web_ui_port = 48012") {
		t.Errorf("missing web_ui_port in sunshine.conf: %s", content)
	}
}

func TestInputDeviceFiltering(t *testing.T) {
	devices := []InputDevice{
		{DevNode: "/dev/input/event0", Name: "Power Button", IsGamepad: false, IsKeyboard: false, IsMouse: false},
		{DevNode: "/dev/input/event1", Name: "Logitech USB Optical Mouse", IsGamepad: false, IsKeyboard: false, IsMouse: true},
		{DevNode: "/dev/input/event2", Name: "Microsoft Xbox One Controller", IsGamepad: true, IsKeyboard: false, IsMouse: false},
		{DevNode: "/dev/input/event3", Name: "Sony Interactive Entertainment Wireless Controller", IsGamepad: true, IsKeyboard: false, IsMouse: false},
		{DevNode: "/dev/input/event4", Name: "Keychron K2 Mechanical Keyboard", IsGamepad: false, IsKeyboard: true, IsMouse: false},
	}

	pads := FilterGamepads(devices)
	if len(pads) != 2 {
		t.Fatalf("expected 2 gamepads, got %d", len(pads))
	}
	if pads[0].Name != "Microsoft Xbox One Controller" || pads[1].Name != "Sony Interactive Entertainment Wireless Controller" {
		t.Errorf("unexpected gamepad filtering result: %+v", pads)
	}

	kbds := FilterKeyboards(devices)
	if len(kbds) != 1 || kbds[0].Name != "Keychron K2 Mechanical Keyboard" {
		t.Errorf("unexpected keyboard filtering result: %+v", kbds)
	}

	mice := FilterMice(devices)
	if len(mice) != 1 || mice[0].Name != "Logitech USB Optical Mouse" {
		t.Errorf("unexpected mice filtering result: %+v", mice)
	}
}

func TestInputGrabber(t *testing.T) {
	// Test grabber initialization with empty/invalid nodes doesn't panic
	g := NewInputGrabber("", "none", "/dev/null/nonexistent")
	if g == nil {
		t.Fatal("expected non-nil grabber")
	}
	if g.IsActive() {
		t.Error("expected grabber not to be active")
	}

	// Grab attempt on non-existent devices should gracefully handle error
	_ = g.Grab()
	if g.IsActive() {
		t.Error("grabber should not be active for invalid nodes")
	}

	g.Release()
	if g.IsActive() {
		t.Error("grabber should be inactive after release")
	}
}

func TestSupervisor_Status(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	st := sup.GetStatus("/some/game/dir")
	if st == nil {
		t.Fatal("expected status object")
	}
	if st.IsRunning {
		t.Error("expected session not to be running")
	}

	err := sup.StopSeat("/some/game/dir", "p1")
	if err == nil {
		t.Error("expected error stopping non-existent session")
	}
}

func TestCheckCapabilities(t *testing.T) {
	caps := CheckCapabilities()
	t.Logf("Caps: %+v", caps)
}

func setupMockUMU(t *testing.T, homeDir string) {
	t.Helper()
	umuDir := filepath.Join(homeDir, ".local", "share", "umu")
	if err := os.MkdirAll(umuDir, 0755); err != nil {
		t.Fatalf("failed to create mock umu dir: %v", err)
	}
	umuBin := filepath.Join(umuDir, "umu-run")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(umuBin, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write mock umu-run: %v", err)
	}
}

func TestBuildSeatCommand_DefaultPrefixes(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "Half-Life"),
		Name:    "Half-Life",
		AppID:   "70",
		ExePath: filepath.Join(tmpDir, "Half-Life", "hl.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID: "p2",
			Mode:   SeatModeLocal,
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
	}

	// 1. Verify Player 1 seat command uses ~/Games/umu/70
	cmdP1, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat1, filepath.Join(tmpDir, "p1"), nil, caps, 0, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p1) failed: %v", err)
	}
	argsP1 := strings.Join(cmdP1.Args, " ")
	expectedP1Pfx := filepath.Join(tmpDir, "Games", "umu", "70")
	if !strings.Contains(argsP1, "--bind "+expectedP1Pfx+" "+expectedP1Pfx) {
		t.Errorf("expected P1 args to bind prefix %s, got: %s", expectedP1Pfx, argsP1)
	}
	if !strings.Contains(argsP1, "--setenv WINEPREFIX "+expectedP1Pfx) {
		t.Errorf("expected P1 args to setenv WINEPREFIX to %s, got: %s", expectedP1Pfx, argsP1)
	}

	// 2. Verify Player 2 seat command uses ~/Games/umu-p2/70
	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}
	argsP2 := strings.Join(cmdP2.Args, " ")
	expectedP2Pfx := filepath.Join(tmpDir, "Games", "umu-p2", "70")
	if !strings.Contains(argsP2, "--bind "+expectedP2Pfx+" "+expectedP2Pfx) {
		t.Errorf("expected P2 args to bind prefix %s, got: %s", expectedP2Pfx, argsP2)
	}
	if !strings.Contains(argsP2, "--setenv WINEPREFIX "+expectedP2Pfx) {
		t.Errorf("expected P2 args to setenv WINEPREFIX to %s, got: %s", expectedP2Pfx, argsP2)
	}
}

func TestFindGamepadCompanionNodes_Empty(t *testing.T) {
	js, hid, sib := FindGamepadCompanionNodes("")
	if len(js) != 0 || len(hid) != 0 || len(sib) != 0 {
		t.Errorf("expected empty companion nodes for empty input, got js=%v, hid=%v, sib=%v", js, hid, sib)
	}
}

func TestBuildBwrapArgs_InputIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	gameDir := filepath.Join(tmpDir, "games", "TestGame")
	_ = os.MkdirAll(gameDir, 0755)
	profileDir := filepath.Join(tmpDir, "profiles", "player1")
	_ = os.MkdirAll(profileDir, 0755)

	pads := FindAllHostGamepadNodes()

	// Case 1: GamepadNode is "none" (Keyboard/Mouse seat)
	optsNone := BwrapOptions{
		SeatID:      "p1",
		ProfileDir:  profileDir,
		GameDir:     gameDir,
		WinePrefix:  filepath.Join(profileDir, "prefix"),
		GamepadNode: "none",
		Command:     []string{"wine", "game.exe"},
	}
	argsNone, err := BuildBwrapArgs(optsNone)
	if err != nil {
		t.Fatalf("BuildBwrapArgs(optsNone) failed: %v", err)
	}
	strNone := strings.Join(argsNone, " ")

	for _, node := range pads {
		expectedMask := "--ro-bind /dev/null " + node
		if !strings.Contains(strNone, expectedMask) {
			t.Errorf("expected seat with no gamepad to mask %s with /dev/null", node)
		}
	}

	// Case 2: GamepadNode is assigned to a specific gamepad
	if len(pads) > 0 {
		targetPad := pads[0]
		optsAssigned := BwrapOptions{
			SeatID:      "p2",
			ProfileDir:  profileDir,
			GameDir:     gameDir,
			WinePrefix:  filepath.Join(profileDir, "prefix"),
			GamepadNode: targetPad,
			Command:     []string{"wine", "game.exe"},
		}
		argsAssigned, err := BuildBwrapArgs(optsAssigned)
		if err != nil {
			t.Fatalf("BuildBwrapArgs(optsAssigned) failed: %v", err)
		}
		strAssigned := strings.Join(argsAssigned, " ")

		targetMask := "--ro-bind /dev/null " + targetPad
		if strings.Contains(strAssigned, targetMask) {
			t.Errorf("assigned gamepad %s should NOT be masked with /dev/null", targetPad)
		}
	}

	// Case 3: MaskInputNodes explicitly configured
	fakeNode := filepath.Join(tmpDir, "fake_event99")
	_ = os.WriteFile(fakeNode, []byte("fake"), 0644)
	optsMasked := BwrapOptions{
		SeatID:         "p1",
		ProfileDir:     profileDir,
		GameDir:        gameDir,
		WinePrefix:     filepath.Join(profileDir, "prefix"),
		MaskInputNodes: []string{fakeNode},
		Command:        []string{"wine", "game.exe"},
	}
	argsMasked, err := BuildBwrapArgs(optsMasked)
	if err != nil {
		t.Fatalf("BuildBwrapArgs(optsMasked) failed: %v", err)
	}
	strMasked := strings.Join(argsMasked, " ")
	if !strings.Contains(strMasked, "--ro-bind /dev/null "+fakeNode) {
		t.Errorf("expected explicit node %s to be masked with /dev/null in %s", fakeNode, strMasked)
	}
}

func TestBuildSeatCommand_WindowPositioning(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "Half-Life"),
		Name:    "Half-Life",
		AppID:   "70",
		ExePath: filepath.Join(tmpDir, "Half-Life", "hl.exe"),
		Runner:  "umu",
		Layout:  LayoutSideBySide, // Side by Side split layout
		Seat1: SeatConfig{
			SeatID:       "p1",
			Mode:         SeatModeLocal,
			TargetWidth:  1920,
			TargetHeight: 1080,
		},
		Seat2: SeatConfig{
			SeatID:       "p2",
			Mode:         SeatModeLocal,
			TargetWidth:  1920,
			TargetHeight: 1080,
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
	}

	// Seat 1 should have SDL_VIDEO_WINDOW_POS=0,0
	cmdP1, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat1, filepath.Join(tmpDir, "p1"), nil, caps, 0, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p1) failed: %v", err)
	}
	var p1PosFound bool
	for _, env := range cmdP1.Env {
		if env == "SDL_VIDEO_WINDOW_POS=0,0" {
			p1PosFound = true
			break
		}
	}
	if !p1PosFound {
		t.Errorf("expected SDL_VIDEO_WINDOW_POS=0,0 in P1 Env: %v", cmdP1.Env)
	}

	// Seat 2 should have SDL_VIDEO_WINDOW_POS=960,0 (halfway across 1920)
	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}
	var p2PosFound bool
	for _, env := range cmdP2.Env {
		if env == "SDL_VIDEO_WINDOW_POS=960,0" {
			p2PosFound = true
			break
		}
	}
	if !p2PosFound {
		t.Errorf("expected SDL_VIDEO_WINDOW_POS=960,0 in P2 Env: %v", cmdP2.Env)
	}
}

func TestDecodeLinuxInputEvent(t *testing.T) {
	buf := make([]byte, 24)
	binary.LittleEndian.PutUint64(buf[0:8], 1700000000) // sec
	binary.LittleEndian.PutUint64(buf[8:16], 500000)    // usec
	binary.LittleEndian.PutUint16(buf[16:18], EV_KEY)   // type
	binary.LittleEndian.PutUint16(buf[18:20], 30)       // code (KEY_A)
	binary.LittleEndian.PutUint32(buf[20:24], 1)        // value (pressed)

	ev, err := DecodeLinuxInputEvent(buf)
	if err != nil {
		t.Fatalf("DecodeLinuxInputEvent failed: %v", err)
	}
	if ev.TimeSec != 1700000000 || ev.TimeUsec != 500000 {
		t.Errorf("unexpected time: sec=%d, usec=%d", ev.TimeSec, ev.TimeUsec)
	}
	if ev.Type != EV_KEY || ev.Code != 30 || ev.Value != 1 {
		t.Errorf("unexpected event: type=%d, code=%d, val=%d", ev.Type, ev.Code, ev.Value)
	}

	// Test short buffer error handling
	_, err = DecodeLinuxInputEvent(buf[:10])
	if err == nil {
		t.Error("expected error for buffer shorter than 24 bytes")
	}
}

func TestEncodeString(t *testing.T) {
	s := "wl_seat" // 7 chars + 1 NUL = 8 bytes (already aligned to 4)
	enc := encodeString(s)
	// 4 bytes length prefix + 8 bytes payload = 12 bytes
	if len(enc) != 12 {
		t.Fatalf("expected encoded string len 12, got %d", len(enc))
	}
	l := binary.LittleEndian.Uint32(enc[0:4])
	if l != 8 {
		t.Errorf("expected length prefix 8, got %d", l)
	}
	if string(enc[4:11]) != "wl_seat" {
		t.Errorf("unexpected string content: %s", string(enc[4:11]))
	}
	if enc[11] != 0 {
		t.Errorf("expected null byte at index 11")
	}
}

func TestSnapshotWaylandSockets(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "wayland-0"), []byte(""), 0600)
	_ = os.WriteFile(filepath.Join(tmpDir, "wayland-0.lock"), []byte(""), 0600)
	_ = os.WriteFile(filepath.Join(tmpDir, "wayland-1"), []byte(""), 0600)
	_ = os.WriteFile(filepath.Join(tmpDir, "other-file"), []byte(""), 0600)

	snaps := snapshotWaylandSockets()
	if !snaps["wayland-0"] {
		t.Errorf("expected wayland-0 in snapshot")
	}
	if !snaps["wayland-1"] {
		t.Errorf("expected wayland-1 in snapshot")
	}
	if snaps["wayland-0.lock"] {
		t.Errorf("did not expect lock file in snapshot")
	}
	if snaps["other-file"] {
		t.Errorf("did not expect other-file in snapshot")
	}
}

func TestBuildSeatCommand_P2_KBM_UsesGamescope(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID:       "p2",
			Mode:         SeatModeLocal,
			KeyboardNode: "/dev/input/event10",
			MouseNode:    "/dev/input/event11",
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	if !strings.Contains(argsStr, "gamescope") {
		t.Errorf("expected Seat 2 with dedicated KBM to wrap in gamescope, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "cage -s --") {
		t.Errorf("expected Seat 2 with dedicated KBM to NOT wrap in cage, got: %s", argsStr)
	}
}

func TestWaylandClient_And_Grabber_MockCompositor(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "wayland-mock")

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("mock listen failed: %v", err)
	}
	defer l.Close()

	doneCh := make(chan struct{})

	// Run mock compositor server
	go func() {
		defer close(doneCh)
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read client get_registry and sync requests (total 16 bytes: 2x 8-byte messages)
		reqBuf := make([]byte, 16)
		if _, err := io.ReadFull(conn, reqBuf); err != nil {
			return
		}

		sendEvent := func(objID uint32, opcode uint16, payload []byte) {
			sz := uint16(8 + len(payload))
			h := make([]byte, sz)
			binary.LittleEndian.PutUint32(h[0:4], objID)
			binary.LittleEndian.PutUint16(h[4:6], opcode)
			binary.LittleEndian.PutUint16(h[6:8], sz)
			copy(h[8:], payload)
			_, _ = conn.Write(h)
		}

		sendGlobal := func(regID uint32, name uint32, iface string, ver uint32) {
			p := make([]byte, 4)
			binary.LittleEndian.PutUint32(p[0:4], name)
			p = append(p, encodeString(iface)...)
			var v [4]byte
			binary.LittleEndian.PutUint32(v[0:4], ver)
			p = append(p, v[:]...)
			sendEvent(regID, 0, p)
		}

		// Announce globals on registry (id=2)
		sendGlobal(2, 1, "wl_seat", 1)
		sendGlobal(2, 2, "zwlr_virtual_pointer_manager_v1", 2)
		sendGlobal(2, 3, "zwp_virtual_keyboard_manager_v1", 1)

		// Send wl_callback.done on callback (id=3, op=0)
		var doneData [4]byte
		sendEvent(3, 0, doneData[:])

		// Read requests from client until client closes connection
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	client := NewWaylandClient()
	if err := client.Connect(sockPath); err != nil {
		t.Fatalf("client Connect failed: %v", err)
	}

	// Test pointer motion
	if err := client.SendMotion(10, 20); err != nil {
		t.Errorf("SendMotion failed: %v", err)
	}

	// Test pointer button
	if err := client.SendButton(272, 1); err != nil {
		t.Errorf("SendButton failed: %v", err)
	}

	// Test pointer scroll axis
	if err := client.SendAxis(0, 3840); err != nil {
		t.Errorf("SendAxis failed: %v", err)
	}

	// Test pointer frame
	if err := client.SendFrame(); err != nil {
		t.Errorf("SendFrame failed: %v", err)
	}

	// Test keyboard key
	if err := client.SendKey(30, 1); err != nil {
		t.Errorf("SendKey failed: %v", err)
	}

	_ = client.Close()
	<-doneCh
}

func TestBuildSeatCommand_ExplicitGamescope_NoCage(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID:       "p2",
			Mode:         SeatModeLocal,
			Compositor:   "gamescope",
			KeyboardNode: "/dev/input/event10",
			MouseNode:    "/dev/input/event11",
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	if strings.Contains(argsStr, "cage -s --") {
		t.Errorf("did not expect Seat 2 with explicit compositor=gamescope to wrap in cage, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "gamescope") {
		t.Errorf("expected Seat 2 to use gamescope, got: %s", argsStr)
	}
}

func TestBuildSeatCommand_AutoCompositor_NoDedicatedKBM_NoCage(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID:       "p2",
			Mode:         SeatModeLocal,
			Compositor:   "auto",
			KeyboardNode: "none",
			MouseNode:    "none",
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	if strings.Contains(argsStr, "cage -s --") {
		t.Errorf("did not expect Seat 2 with compositor=auto and no dedicated KBM to wrap in cage, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "gamescope") {
		t.Errorf("expected Seat 2 to use gamescope, got: %s", argsStr)
	}
}

func TestBuildSeatCommand_NestedCageGamescope(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	// compositor="cage_gamescope" should nest gamescope inside cage
	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID:       "p2",
			Mode:         SeatModeLocal,
			Compositor:   "auto",
			KeyboardNode: "/dev/input/event10",
			MouseNode:    "/dev/input/event11",
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	// P2 with dedicated KBM and Gamescope available should use Gamescope directly (not Cage)
	if strings.Contains(argsStr, "cage -s --") {
		t.Errorf("Seat 2 with Gamescope should NOT wrap in cage, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "gamescope") {
		t.Errorf("Seat 2 should use gamescope, got: %s", argsStr)
	}
}

func TestBuildSeatCommand_CageOnly_NoGamescope(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	// compositor="cage" should wrap in cage only — no gamescope at all
	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID:     "p2",
			Mode:       SeatModeLocal,
			Compositor: "cage",
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	if !strings.Contains(argsStr, "cage -s --") {
		t.Errorf("expected cage wrapper, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "gamescope") {
		t.Errorf("compositor=cage should NOT include gamescope, got: %s", argsStr)
	}
}

func TestBuildSeatCommand_Gamescope_SplitWindowPos(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	// P2 with dedicated KBM + auto compositor uses standalone Gamescope with split window pos
	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutSideBySide,
		Seat1: SeatConfig{
			SeatID:       "p1",
			Mode:         SeatModeLocal,
			TargetWidth:  1920,
			TargetHeight: 1080,
		},
		Seat2: SeatConfig{
			SeatID:       "p2",
			Mode:         SeatModeLocal,
			TargetWidth:  1920,
			TargetHeight: 1080,
			KeyboardNode: "/dev/input/event10",
			MouseNode:    "/dev/input/event11",
			Compositor:   "auto",
		},
	}
	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	var p2HasPos bool
	for _, env := range cmdP2.Env {
		if strings.HasPrefix(env, "SDL_VIDEO_WINDOW_POS=960,0") {
			p2HasPos = true
			break
		}
	}
	if !p2HasPos {
		t.Errorf("expected SDL_VIDEO_WINDOW_POS=960,0 for P2 side-by-side gamescope")
	}

	// Seat 1 (standalone gamescope) should also have SDL_VIDEO_WINDOW_POS set to 0,0
	cmdP1, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat1, filepath.Join(tmpDir, "p1"), nil, caps, 0, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p1) failed: %v", err)
	}
	var p1HasPos bool
	for _, env := range cmdP1.Env {
		if strings.HasPrefix(env, "SDL_VIDEO_WINDOW_POS=") {
			p1HasPos = true
			break
		}
	}
	if !p1HasPos {
		t.Errorf("P1 (standalone gamescope, side-by-side) should have SDL_VIDEO_WINDOW_POS set")
	}

	// Verify explicit ScreenSlot swapping: P2 explicitly placed on left half (slot 0)
	slot0 := 0
	sessionCfg.Seat2.ScreenSlot = &slot0
	cmdP2Swapped, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2_swapped"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2 swapped) failed: %v", err)
	}
	var p2SwappedHasPos bool
	for _, env := range cmdP2Swapped.Env {
		if strings.HasPrefix(env, "SDL_VIDEO_WINDOW_POS=0,0") {
			p2SwappedHasPos = true
			break
		}
	}
	if !p2SwappedHasPos {
		t.Errorf("expected SDL_VIDEO_WINDOW_POS=0,0 for P2 swapped to slot 0")
	}
}

func TestResolveCageDecision(t *testing.T) {
	capsAll := SystemCapabilities{HasGamescope: true, HasCage: true}
	capsNoCage := SystemCapabilities{HasGamescope: true, HasCage: false}
	capsNoGS := SystemCapabilities{HasGamescope: false, HasCage: true}

	tests := []struct {
		name     string
		seatCfg  SeatConfig
		seatID   string
		caps     SystemCapabilities
		wantCage bool
	}{
		{
			name:     "compositor=gamescope → no cage",
			seatCfg:  SeatConfig{Compositor: "gamescope"},
			seatID:   "p2",
			caps:     capsAll,
			wantCage: false,
		},
		{
			name:     "compositor=cage → cage",
			seatCfg:  SeatConfig{Compositor: "cage"},
			seatID:   "p2",
			caps:     capsAll,
			wantCage: true,
		},

		{
			name:     "NestCage=true → cage",
			seatCfg:  SeatConfig{NestCage: true},
			seatID:   "p2",
			caps:     capsAll,
			wantCage: true,
		},
		{
			name:     "auto + P2 KBM with gamescope available → gamescope (no cage)",
			seatCfg:  SeatConfig{Compositor: "auto", KeyboardNode: "/dev/input/event10", MouseNode: "/dev/input/event11"},
			seatID:   "p2",
			caps:     capsAll,
			wantCage: false,
		},
		{
			name:     "auto + P2 KBM without gamescope → cage fallback",
			seatCfg:  SeatConfig{Compositor: "auto", KeyboardNode: "/dev/input/event10", MouseNode: "/dev/input/event11"},
			seatID:   "p2",
			caps:     capsNoGS,
			wantCage: true,
		},
		{
			name:     "auto + P2 no KBM → no cage",
			seatCfg:  SeatConfig{Compositor: "auto", KeyboardNode: "none", MouseNode: "none"},
			seatID:   "p2",
			caps:     capsAll,
			wantCage: false,
		},
		{
			name:     "auto + P1 KBM → no cage (only P2 triggers KBM cage)",
			seatCfg:  SeatConfig{Compositor: "auto", KeyboardNode: "/dev/input/event10"},
			seatID:   "p1",
			caps:     capsAll,
			wantCage: false,
		},
		{
			name:     "stream mode without gamescope → cage fallback",
			seatCfg:  SeatConfig{Mode: SeatModeStream},
			seatID:   "p2",
			caps:     capsNoGS,
			wantCage: true,
		},
		{
			name:     "cage requested but not installed → no cage",
			seatCfg:  SeatConfig{Compositor: "cage"},
			seatID:   "p2",
			caps:     capsNoCage,
			wantCage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCageDecision(tt.seatCfg, tt.seatID, tt.caps)
			if got != tt.wantCage {
				t.Errorf("resolveCageDecision() = %v, want %v", got, tt.wantCage)
			}
		})
	}
}

type mockInjector struct {
	mu      sync.Mutex
	motions [][2]int32
	buttons [][2]uint32
	axes    [][2]int32
	keys    [][2]uint32
	frames  int
	closed  bool
}

func (m *mockInjector) SendMotion(dx, dy int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.motions = append(m.motions, [2]int32{dx, dy})
	return nil
}

func (m *mockInjector) SendButton(button uint32, state uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buttons = append(m.buttons, [2]uint32{button, state})
	return nil
}

func (m *mockInjector) SendAxis(axis uint32, value int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.axes = append(m.axes, [2]int32{int32(axis), value})
	return nil
}

func (m *mockInjector) SendFrame() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.frames++
	return nil
}

func (m *mockInjector) SendKey(key uint32, state uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys = append(m.keys, [2]uint32{key, state})
	return nil
}

func (m *mockInjector) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func TestInputGrabber_SetInjector(t *testing.T) {
	grabber := NewInputGrabber()
	mock := &mockInjector{}

	grabber.SetInjector(mock)
	if grabber.injector != mock {
		t.Errorf("expected injector to be set")
	}

	grabber.Release()
	if !mock.closed {
		t.Errorf("expected mock injector to be closed on release")
	}
}

func TestEIClient_InvalidSocket(t *testing.T) {
	client := NewEIClient()
	err := client.Connect("/non/existent/path/gamescope-99-ei")
	if err == nil {
		t.Fatalf("expected connect to non-existent socket to fail")
	}
	_ = client.Close()
}

func TestEISocketHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	initial := snapshotEISockets()
	if len(initial) != 0 {
		t.Errorf("expected empty initial EIS sockets in temp dir, got %v", initial)
	}

	// Create dummy gamescope-5-ei socket listener
	sockPath := filepath.Join(tmpDir, "gamescope-5-ei")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer l.Close()

	found, err := waitForNewEISocket(initial, 1*time.Second)
	if err != nil {
		t.Fatalf("expected to find new socket: %v", err)
	}
	if found != sockPath {
		t.Errorf("expected found socket %s, got %s", sockPath, found)
	}
}

func TestWaitForEISocketForPID(t *testing.T) {
	tmpDir := t.TempDir()
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	sockPath := filepath.Join(tmpDir, "gamescope-1-ei")
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
	}
	defer l.Close()

	// Accept in background so dial completes
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			time.Sleep(50 * time.Millisecond)
			_ = conn.Close()
		}
	}()

	// 1. Should match our current PID
	found, err := waitForEISocketForPID(os.Getpid(), nil, 1*time.Second)
	if err != nil {
		t.Fatalf("expected to match current PID %d: %v", os.Getpid(), err)
	}
	if found != sockPath {
		t.Errorf("expected sockPath %s, got %s", sockPath, found)
	}

	// 2. Should match our current PID even if socket was present in existing snapshot
	existingMap := map[string]bool{"gamescope-1-ei": true}
	foundExisting, err := waitForEISocketForPID(os.Getpid(), existingMap, 1*time.Second)
	if err != nil {
		t.Fatalf("expected to match PID even when in existing snapshot: %v", err)
	}
	if foundExisting != sockPath {
		t.Errorf("expected sockPath %s, got %s", sockPath, foundExisting)
	}

	// 3. Should timeout when looking for an unrelated PID
	_, err = waitForEISocketForPID(99999999, nil, 200*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout when searching for mismatched PID")
	}

	// 4. For generic search (targetPID <= 0), it should skip sockets in existing snapshot
	_, err = waitForEISocketForPID(0, existingMap, 200*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout when generic search skips existing socket")
	}
}

func TestBuildGamescopeArgs_SplitGeometry_CorrectDimensions(t *testing.T) {
	// Full screen is 1920x1080.
	// Side-by-Side split should give 960x1080 per seat.
	winW, winH, winX, winY := CalculateSplitGeometry(LayoutSideBySide, "p2", 1920, 1080)
	if winW != 960 || winH != 1080 || winX != 960 || winY != 0 {
		t.Fatalf("unexpected split geometry: winW=%d, winH=%d, winX=%d, winY=%d", winW, winH, winX, winY)
	}

	dispOpts := DisplayOptions{
		Mode:         SeatModeLocal,
		Layout:       LayoutSideBySide,
		SeatID:       "p2",
		TargetWidth:  winW,
		TargetHeight: winH,
		WindowWidth:  winW,
		WindowHeight: winH,
		WindowX:      winX,
		WindowY:      winY,
		Command:      []string{"game.exe"},
	}

	args, err := BuildGamescopeArgs(dispOpts)
	if err != nil {
		t.Fatalf("BuildGamescopeArgs failed: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-W 960") {
		t.Errorf("expected -W 960 in gamescope args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-H 1080") {
		t.Errorf("expected -H 1080 in gamescope args, got: %s", argsStr)
	}
	if strings.Contains(argsStr, "-W 480") {
		t.Errorf("detected double-halved width (-W 480) in gamescope args: %s", argsStr)
	}
}

func TestEIClient_LiveGamescope(t *testing.T) {
	gsBin, err := exec.LookPath("gamescope")
	if err != nil {
		t.Skip("gamescope binary not available on host, skipping live test")
	}

	// Start gamescope with headless backend in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, gsBin, "-W", "640", "-H", "480", "--", "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start gamescope: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	// Detect gamescope's EIS socket by its exact PID
	sockPath, err := waitForEISocketForPID(cmd.Process.Pid, nil, 3*time.Second)
	if err != nil {
		t.Fatalf("failed to detect gamescope EIS socket: %v", err)
	}

	client := NewEIClient()
	if err := client.Connect(sockPath); err != nil {
		t.Fatalf("failed to connect EIClient: %v", err)
	}

	client.SetDimensions(640, 480)
	startX, startY := client.GetCursorPos()
	if startX != 320 || startY != 240 {
		t.Errorf("expected initial centered cursor at (320, 240), got (%f, %f)", startX, startY)
	}

	// Verify sending pointer motion, button, scroll, frame, and key
	if err := client.SendMotion(10, -5); err != nil {
		t.Errorf("SendMotion failed: %v", err)
	}
	curX, curY := client.GetCursorPos()
	if curX != 330 || curY != 235 {
		t.Errorf("expected cursor after motion at (330, 235), got (%f, %f)", curX, curY)
	}

	// Verify clamping within bounds
	if err := client.SendMotion(2000, 2000); err != nil {
		t.Errorf("SendMotion failed: %v", err)
	}
	maxX, maxY := client.GetCursorPos()
	if maxX != 640 || maxY != 480 {
		t.Errorf("expected cursor clamped to (640, 480), got (%f, %f)", maxX, maxY)
	}

	if err := client.SendMotion(-5000, -5000); err != nil {
		t.Errorf("SendMotion failed: %v", err)
	}
	minX, minY := client.GetCursorPos()
	if minX != 0 || minY != 0 {
		t.Errorf("expected cursor clamped to (0, 0), got (%f, %f)", minX, minY)
	}

	if err := client.SendMotionAbsolute(200, 150); err != nil {
		t.Errorf("SendMotionAbsolute failed: %v", err)
	}
	absX, absY := client.GetCursorPos()
	if absX != 200 || absY != 150 {
		t.Errorf("expected cursor at (200, 150), got (%f, %f)", absX, absY)
	}

	if err := client.SendButton(0x110, 1); err != nil { // BTN_LEFT press
		t.Errorf("SendButton press failed: %v", err)
	}
	if err := client.SendButton(0x110, 0); err != nil { // BTN_LEFT release
		t.Errorf("SendButton release failed: %v", err)
	}
	if err := client.SendAxis(0, 3840); err != nil { // vertical scroll
		t.Errorf("SendAxis failed: %v", err)
	}
	if err := client.SendFrame(); err != nil {
		t.Errorf("SendFrame failed: %v", err)
	}
	if err := client.SendKey(30, 1); err != nil { // KEY_A press
		t.Errorf("SendKey press failed: %v", err)
	}
	if err := client.SendKey(30, 0); err != nil { // KEY_A release
		t.Errorf("SendKey release failed: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestEIClient_CoordinatesAndDimensions(t *testing.T) {
	client := NewEIClient()
	x, y := client.GetCursorPos()
	if x != 960 || y != 540 {
		t.Fatalf("expected default cursor (960, 540), got (%f, %f)", x, y)
	}

	client.SetDimensions(2560, 1440)
	x2, y2 := client.GetCursorPos()
	if x2 != 1280 || y2 != 720 {
		t.Fatalf("expected cursor at (1280, 720), got (%f, %f)", x2, y2)
	}

	// Negative/zero fallback to default 1920x1080
	client.SetDimensions(0, -100)
	x3, y3 := client.GetCursorPos()
	if x3 != 960 || y3 != 540 {
		t.Fatalf("expected cursor fallback to (960, 540), got (%f, %f)", x3, y3)
	}

	// Unopened client should return errors on sends
	if err := client.SendMotion(10, 10); err == nil {
		t.Fatalf("expected error sending motion on unopened client")
	}
	if err := client.SendMotionAbsolute(10, 10); err == nil {
		t.Fatalf("expected error sending absolute motion on unopened client")
	}
	if err := client.SendButton(0x110, 1); err == nil {
		t.Fatalf("expected error sending button on unopened client")
	}
	if err := client.SendAxis(0, 100); err == nil {
		t.Fatalf("expected error sending axis on unopened client")
	}
	if err := client.SendKey(30, 1); err == nil {
		t.Fatalf("expected error sending key on unopened client")
	}
	if err := client.SendFrame(); err == nil {
		t.Fatalf("expected error sending frame on unopened client")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("expected nil on closing unopened client: %v", err)
	}
}

func TestFindDeviceCompanionNodes_SyntheticSysfs(t *testing.T) {
	tmpDir := t.TempDir()

	origSysClass := sysClassInputDir
	origDevInput := devInputDir
	origDev := devDir
	defer func() {
		sysClassInputDir = origSysClass
		devInputDir = origDevInput
		devDir = origDev
	}()

	mockSys := filepath.Join(tmpDir, "sys", "class", "input")
	mockDevIn := filepath.Join(tmpDir, "dev", "input")
	mockDevBase := filepath.Join(tmpDir, "dev")
	_ = os.MkdirAll(mockSys, 0755)
	_ = os.MkdirAll(mockDevIn, 0755)

	sysClassInputDir = mockSys
	devInputDir = mockDevIn
	devDir = mockDevBase

	// Create a mock USB composite hardware root directory
	usbRoot := filepath.Join(tmpDir, "sys", "devices", "pci0000:00", "0000:00:14.0", "usb1", "1-1")
	_ = os.MkdirAll(usbRoot, 0755)
	_ = os.WriteFile(filepath.Join(usbRoot, "idVendor"), []byte("1234\n"), 0644)

	// Create interface 1 (event4)
	iface1 := filepath.Join(usbRoot, "1-1:1.0", "0003:1234:5678.0001", "input", "input40", "event4")
	_ = os.MkdirAll(iface1, 0755)
	// Create interface 2 (event10)
	iface2 := filepath.Join(usbRoot, "1-1:1.1", "0003:1234:5678.0002", "input", "input41", "event10")
	_ = os.MkdirAll(iface2, 0755)
	// Create interface 3 (event11)
	iface3 := filepath.Join(usbRoot, "1-1:1.2", "0003:1234:5678.0003", "input", "input42", "event11")
	_ = os.MkdirAll(iface3, 0755)

	// In /sys/class/input:
	// event4/device points to input40
	_ = os.MkdirAll(filepath.Join(mockSys, "event4"), 0755)
	_ = os.Symlink(filepath.Join(usbRoot, "1-1:1.0", "0003:1234:5678.0001", "input", "input40"), filepath.Join(mockSys, "event4", "device"))

	_ = os.MkdirAll(filepath.Join(mockSys, "event10"), 0755)
	_ = os.Symlink(filepath.Join(usbRoot, "1-1:1.1", "0003:1234:5678.0002", "input", "input41"), filepath.Join(mockSys, "event10", "device"))

	_ = os.MkdirAll(filepath.Join(mockSys, "event11"), 0755)
	_ = os.Symlink(filepath.Join(usbRoot, "1-1:1.2", "0003:1234:5678.0003", "input", "input42"), filepath.Join(mockSys, "event11", "device"))

	// Create device nodes in mockDevInput
	ev4Dev := filepath.Join(mockDevIn, "event4")
	ev10Dev := filepath.Join(mockDevIn, "event10")
	ev11Dev := filepath.Join(mockDevIn, "event11")
	_ = os.WriteFile(ev4Dev, []byte{}, 0644)
	_ = os.WriteFile(ev10Dev, []byte{}, 0644)
	_ = os.WriteFile(ev11Dev, []byte{}, 0644)

	// Test FindDeviceCompanionNodes on event4
	siblings := FindDeviceCompanionNodes(ev4Dev)
	if len(siblings) != 2 {
		t.Fatalf("expected 2 companion sibling nodes, got %d: %v", len(siblings), siblings)
	}
	if !containsString(siblings, ev10Dev) || !containsString(siblings, ev11Dev) {
		t.Errorf("expected siblings to contain %s and %s, got: %v", ev10Dev, ev11Dev, siblings)
	}
	if containsString(siblings, ev4Dev) {
		t.Errorf("did not expect base event node %s to be listed in siblings", ev4Dev)
	}

	// Test NewInputGrabber automatically includes companion nodes
	grabber := NewInputGrabber(ev4Dev)
	if len(grabber.nodes) != 3 {
		t.Fatalf("expected grabber to contain 3 nodes (base + 2 companions), got %d: %v", len(grabber.nodes), grabber.nodes)
	}
	if !containsString(grabber.nodes, ev4Dev) || !containsString(grabber.nodes, ev10Dev) || !containsString(grabber.nodes, ev11Dev) {
		t.Errorf("grabber nodes missing expected entries: %v", grabber.nodes)
	}

	// Add capability bitmasks to test FilterKeyboards scoring
	_ = os.MkdirAll(filepath.Join(usbRoot, "1-1:1.0", "0003:1234:5678.0001", "input", "input40", "capabilities"), 0755)
	_ = os.WriteFile(filepath.Join(usbRoot, "1-1:1.0", "0003:1234:5678.0001", "input", "input40", "capabilities", "key"), []byte("10000 0 0\n"), 0644)

	_ = os.MkdirAll(filepath.Join(usbRoot, "1-1:1.1", "0003:1234:5678.0002", "input", "input41", "capabilities"), 0755)
	_ = os.WriteFile(filepath.Join(usbRoot, "1-1:1.1", "0003:1234:5678.0002", "input", "input41", "capabilities", "key"), []byte("ffffffff ffffffff ffffffff ffffffff\n"), 0644)

	kbds := []InputDevice{
		{DevNode: ev4Dev, IsKeyboard: true, Name: "Gaming Keyboard Boot"},
		{DevNode: ev10Dev, IsKeyboard: true, Name: "Gaming Keyboard NKRO"},
	}
	filteredKbds := FilterKeyboards(kbds)
	if len(filteredKbds) != 1 {
		t.Fatalf("expected 1 deduplicated keyboard, got %d: %v", len(filteredKbds), filteredKbds)
	}
	if filteredKbds[0].DevNode != ev10Dev {
		t.Errorf("expected FilterKeyboards to pick richest interface %s, got: %s", ev10Dev, filteredKbds[0].DevNode)
	}

	// Test FilterMice deduplication
	mice := []InputDevice{
		{DevNode: ev4Dev, IsMouse: true, Name: "Gaming Mouse Pointer"},
		{DevNode: ev10Dev, IsMouse: true, Name: "Gaming Mouse Extra"},
	}
	filteredMice := FilterMice(mice)
	if len(filteredMice) != 1 {
		t.Fatalf("expected 1 deduplicated mouse, got %d: %v", len(filteredMice), filteredMice)
	}
	if filteredMice[0].DevNode != ev4Dev {
		t.Errorf("expected FilterMice to keep primary pointer %s, got: %s", ev4Dev, filteredMice[0].DevNode)
	}
}

func TestBuildSeatCommand_DedicatedKBM_NoHostGrab(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupMockUMU(t, tmpDir)

	caps := SystemCapabilities{
		HasBwrap:     true,
		HasGamescope: true,
		HasCage:      true,
	}

	// 1. Seat 2 has dedicated KBM and user specified ForceGrabCursor=true, GrabKeyboard=true
	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmpDir, "MyGame"),
		Name:    "MyGame",
		AppID:   "100",
		ExePath: filepath.Join(tmpDir, "MyGame", "game.exe"),
		Runner:  "umu",
		Layout:  LayoutDualMonitor,
		Seat1: SeatConfig{
			SeatID: "p1",
			Mode:   SeatModeLocal,
		},
		Seat2: SeatConfig{
			SeatID:          "p2",
			Mode:            SeatModeLocal,
			Compositor:      "gamescope",
			KeyboardNode:    "/dev/input/event10",
			MouseNode:       "/dev/input/event11",
			ForceGrabCursor: true,
			GrabKeyboard:    true,
		},
	}

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, sessionCfg.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsP2 := strings.Join(cmdP2.Args, " ")
	if strings.Contains(argsP2, "--force-grab-cursor") {
		t.Errorf("expected --force-grab-cursor to be suppressed for Seat 2 with dedicated KBM, got: %s", argsP2)
	}
	if strings.Contains(argsP2, " -g ") || strings.HasSuffix(argsP2, " -g") {
		t.Errorf("expected -g (grab keyboard) to be suppressed for Seat 2 with dedicated KBM, got: %s", argsP2)
	}

	// Verify that gamescope_cursor_composite=2 is enabled in environment for dedicated KBM
	envP2 := strings.Join(cmdP2.Env, "\n")
	if !strings.Contains(envP2, "gamescope_cursor_composite=2") {
		t.Errorf("expected gamescope_cursor_composite=2 in P2 env with dedicated KBM, got: %v", cmdP2.Env)
	}

	// 2. Seat 2 with NO dedicated KBM preserves ForceGrabCursor=true and GrabKeyboard=true
	sessionCfgNoKBM := sessionCfg
	sessionCfgNoKBM.Seat2.KeyboardNode = "none"
	sessionCfgNoKBM.Seat2.MouseNode = "none"

	cmdP2NoKBM, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfgNoKBM, sessionCfgNoKBM.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2 no kbm) failed: %v", err)
	}

	argsP2NoKBM := strings.Join(cmdP2NoKBM.Args, " ")
	if !strings.Contains(argsP2NoKBM, "--force-grab-cursor") {
		t.Errorf("expected --force-grab-cursor to be present when no dedicated KBM, got: %s", argsP2NoKBM)
	}
	if !strings.Contains(argsP2NoKBM, "-g") {
		t.Errorf("expected -g to be present when no dedicated KBM, got: %s", argsP2NoKBM)
	}

	// Verify that gamescope_cursor_composite=2 is NOT present when no dedicated KBM and no ShowCursor
	envP2NoKBM := strings.Join(cmdP2NoKBM.Env, "\n")
	if strings.Contains(envP2NoKBM, "gamescope_cursor_composite=2") {
		t.Errorf("expected gamescope_cursor_composite=2 to be absent when no dedicated KBM, got: %v", cmdP2NoKBM.Env)
	}

	// 3. Seat 2 with NO dedicated KBM, but ShowCursor=true -> gamescope_cursor_composite=2 IS present
	sessionCfgShowCursor := sessionCfgNoKBM
	sessionCfgShowCursor.Seat2.ShowCursor = true
	cmdP2ShowCursor, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfgShowCursor, sessionCfgShowCursor.Seat2, filepath.Join(tmpDir, "p2"), nil, caps, 1, 2)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2 show cursor) failed: %v", err)
	}
	envP2ShowCursor := strings.Join(cmdP2ShowCursor.Env, "\n")
	if !strings.Contains(envP2ShowCursor, "gamescope_cursor_composite=2") {
		t.Errorf("expected gamescope_cursor_composite=2 to be present when ShowCursor=true, got: %v", cmdP2ShowCursor.Env)
	}
}

func TestSupervisor_RunningSessions(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	gameDir1 := "/games/Portal 2"
	gameDir2 := "/games/Overcooked/"

	sup.sessions[util.CanonicalPath(gameDir1)] = &runningSession{
		config: MultiseatSessionConfig{GameDir: gameDir1},
		seats: map[string]*runningSeat{
			"p1": {seatID: "p1", status: SeatStatus{IsRunning: true}},
			"p2": {seatID: "p2", status: SeatStatus{IsRunning: false}},
		},
	}
	sup.sessions[util.CanonicalPath(gameDir2)] = &runningSession{
		config: MultiseatSessionConfig{GameDir: gameDir2},
		seats: map[string]*runningSeat{
			"p1": {seatID: "p1", status: SeatStatus{IsRunning: false}},
			"p2": {seatID: "p2", status: SeatStatus{IsRunning: true}},
		},
	}

	running := sup.RunningSessions()
	if len(running) != 2 {
		t.Fatalf("expected 2 running sessions, got %d: %v", len(running), running)
	}

	clean1 := util.CanonicalPath(gameDir1)
	clean2 := util.CanonicalPath(gameDir2)
	has1 := false
	has2 := false
	for _, r := range running {
		if r == clean1 {
			has1 = true
		}
		if r == clean2 {
			has2 = true
		}
	}
	if !has1 || !has2 {
		t.Errorf("RunningSessions() missing expected paths: got %v", running)
	}
}

func TestSupervisor_IndependentSeatDrop(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	gameDir := "/games/CoopGame"
	cleanDir := util.CanonicalPath(gameDir)

	cmdP1 := exec.Command("sleep", "3")
	cmdP2 := exec.Command("sleep", "3")
	if err := cmdP1.Start(); err != nil {
		t.Skip("could not start sleep process")
	}
	if err := cmdP2.Start(); err != nil {
		_ = cmdP1.Process.Kill()
		t.Skip("could not start sleep process")
	}

	sess := &runningSession{
		config:    MultiseatSessionConfig{GameDir: gameDir, Name: "CoopGame", AppID: "999"},
		startTime: time.Now().Unix(),
		seats: map[string]*runningSeat{
			"p1": {
				seatID:    "p1",
				profileID: "main",
				cmd:       cmdP1,
				status:    SeatStatus{SeatID: "p1", ProfileID: "main", IsRunning: true, PID: cmdP1.Process.Pid},
			},
			"p2": {
				seatID:    "p2",
				profileID: "p2",
				cmd:       cmdP2,
				status:    SeatStatus{SeatID: "p2", ProfileID: "p2", IsRunning: true, PID: cmdP2.Process.Pid},
			},
		},
	}

	sup.mu.Lock()
	sup.sessions[cleanDir] = sess
	sup.mu.Unlock()

	go sup.superviseSession(cleanDir, sess)

	// Verify session is initially running
	if !sup.IsSessionRunning(cleanDir) {
		t.Errorf("expected session to be running initially")
	}

	// Drop Seat 2 via StopSeat
	if err := sup.StopSeat(cleanDir, "p2"); err != nil {
		t.Fatalf("StopSeat(p2) failed: %v", err)
	}

	// Give a brief moment for p2 to exit and superviseSession to process p2Done
	time.Sleep(100 * time.Millisecond)

	// Player 1 MUST STILL BE RUNNING!
	st := sup.GetStatus(cleanDir)
	if st == nil || !st.IsRunning {
		t.Errorf("expected session to still be running after P2 dropped")
	}
	if st.Seat1.IsRunning != true {
		t.Errorf("expected Seat 1 to still be running")
	}
	if st.Seat2.IsRunning != false {
		t.Errorf("expected Seat 2 to be marked stopped")
	}

	// Now stop the whole session
	if err := sup.StopSession(cleanDir); err != nil {
		t.Fatalf("StopSession failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if sup.IsSessionRunning(cleanDir) {
		t.Errorf("expected session to be completely stopped after StopSession")
	}
}

func TestVirtualSink_LifecycleAndIdempotence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	caps := CheckCapabilities()
	if !caps.HasPipewire {
		t.Skip("PipeWire/pactl not available on host")
	}

	sinkName := "gamux-unit-test-sink"
	vs1, err := CreateVirtualSink(ctx, sinkName, "Unit Test Virtual Sink 1")
	if err != nil {
		t.Fatalf("CreateVirtualSink 1 failed: %v", err)
	}
	if vs1.ModuleID == "" {
		t.Errorf("expected non-empty ModuleID")
	}

	// Idempotent re-creation: should clean up previous sink and create anew without error
	vs2, err := CreateVirtualSink(ctx, sinkName, "Unit Test Virtual Sink 2")
	if err != nil {
		t.Fatalf("CreateVirtualSink 2 (idempotent overwrite) failed: %v", err)
	}
	if vs2.ModuleID == "" {
		t.Errorf("expected non-empty ModuleID on second creation")
	}

	// Clean up
	vs2.Destroy(ctx)
	CleanupGamuxSinks(ctx)
}

func TestSupervisor_StopAll(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	sup.StopAll()
}

func TestGenerateKWinSplitScript_SideBySide(t *testing.T) {
	script := GenerateKWinSplitScript(1234, 5678, LayoutSideBySide)

	if !strings.Contains(script, "1234: { index: 0, id: \"p1\"") {
		t.Errorf("expected p1 placement for PID 1234 in script")
	}
	if !strings.Contains(script, "5678: { index: 1, id: \"p2\"") {
		t.Errorf("expected p2 placement for PID 5678 in script")
	}
	if !strings.Contains(script, "var layout = \"side_by_side\";") {
		t.Errorf("expected layout side_by_side in script")
	}
	if !strings.Contains(script, "w.noBorder = true;") {
		t.Errorf("expected w.noBorder = true in script")
	}
	if !strings.Contains(script, "w.keepAbove = true;") {
		t.Errorf("expected w.keepAbove = true in script")
	}
	if !strings.Contains(script, "workspace.virtualScreenSize") {
		t.Errorf("expected workspace.virtualScreenSize in script")
	}
	if !strings.Contains(script, "workspace.windowAdded") {
		t.Errorf("expected workspace.windowAdded in script")
	}
	if !strings.Contains(script, "colW = Math.floor(sW / dispTotal)") {
		t.Errorf("expected colW calculation in script")
	}
}

func TestGenerateKWinSplitScript_TopBottom(t *testing.T) {
	script := GenerateKWinSplitScript(1111, 2222, LayoutTopBottom)

	if !strings.Contains(script, "1111: { index: 0, id: \"p1\"") {
		t.Errorf("expected p1 placement for PID 1111 in script")
	}
	if !strings.Contains(script, "2222: { index: 1, id: \"p2\"") {
		t.Errorf("expected p2 placement for PID 2222 in script")
	}
	if !strings.Contains(script, "var layout = \"top_bottom\";") {
		t.Errorf("expected layout top_bottom in script")
	}
	if !strings.Contains(script, "rowH = Math.floor(sH / dispTotal)") {
		t.Errorf("expected rowH calculation in script")
	}
}

func TestEffectiveSeats_DisabledFiltering(t *testing.T) {
	cfg := MultiseatSessionConfig{
		Seats: []SeatConfig{
			{SeatID: "p1", Disabled: false},
			{SeatID: "p2", Disabled: true},
			{SeatID: "p3", Disabled: false},
		},
	}
	eff := cfg.EffectiveSeats()
	if len(eff) != 2 {
		t.Fatalf("expected 2 active seats, got %d", len(eff))
	}
	if eff[0].SeatID != "p1" || eff[1].SeatID != "p3" {
		t.Errorf("expected p1 and p3, got %s and %s", eff[0].SeatID, eff[1].SeatID)
	}
}

func TestScaleGuard_StatePersistenceAndRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "scaleguard.json")

	sg := &ScaleGuard{
		kscreenBin:     "", // Mocked absent
		originalScales: map[string]float64{"DP-1": 1.15},
		stateFilePath:  stateFile,
	}

	sg.persistLockedScales()

	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatalf("expected scaleguard.json to be created")
	}

	// Read content
	data, _ := os.ReadFile(stateFile)
	var state ScaleLockState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if state.OriginalScales["DP-1"] != 1.15 {
		t.Errorf("expected DP-1 scale 1.15, got %f", state.OriginalScales["DP-1"])
	}

	sg.removeStateFile()
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Errorf("expected scaleguard.json to be removed")
	}
}

func TestNewKWinSplitCoordinator_Validation(t *testing.T) {
	_, err := NewKWinSplitCoordinator(0, 100, LayoutSideBySide)
	if err == nil {
		t.Errorf("expected error for seat1PID <= 0")
	}

	_, err = NewKWinSplitCoordinator(100, 0, LayoutSideBySide)
	if err == nil {
		t.Errorf("expected error for seat2PID <= 0")
	}
}

func TestBuildGamescopeArgs_NestedUnfocusedRefresh(t *testing.T) {
	opts := DisplayOptions{
		Mode:         SeatModeLocal,
		Layout:       LayoutSideBySide,
		SeatID:       "p1",
		WindowWidth:  960,
		WindowHeight: 1080,
		TargetFPS:    100,
		Command:      []string{"game.exe"},
	}

	args, err := BuildGamescopeArgs(opts)
	if err != nil {
		t.Fatalf("BuildGamescopeArgs failed: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "-r 100") {
		t.Errorf("expected -r 100 in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-o 100") {
		t.Errorf("expected -o 100 (unfocused refresh rate) in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-W 960") || !strings.Contains(argsStr, "-H 1080") {
		t.Errorf("expected window resolution 960x1080 in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "-w 960") || !strings.Contains(argsStr, "-h 1080") {
		t.Errorf("expected 1:1 render resolution 960x1080 in args, got: %s", argsStr)
	}
}

func TestKWinSplitCoordinator_LiveLifecycle(t *testing.T) {
	if !IsKWinAvailable() {
		t.Skip("KWin DBus scripting not available on host")
	}

	coord, err := NewKWinSplitCoordinator(os.Getpid(), os.Getpid()+1, LayoutSideBySide)
	if err != nil {
		t.Fatalf("NewKWinSplitCoordinator failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := coord.Start(ctx); err != nil {
		t.Fatalf("coord.Start failed: %v", err)
	}

	if err := coord.Stop(); err != nil {
		t.Fatalf("coord.Stop failed: %v", err)
	}
}

func TestCalculateSeatGeometry_QuadGrid(t *testing.T) {
	totalW, totalH := 3840, 2160

	// 4 seats in quad grid
	// Seat 0: top-left
	w0, h0, x0, y0 := CalculateSeatGeometry(LayoutQuadGrid, "p1", 0, 4, totalW, totalH)
	if w0 != 1920 || h0 != 1080 || x0 != 0 || y0 != 0 {
		t.Errorf("Seat 0 quad_grid unexpected: %dx%d @ (%d,%d)", w0, h0, x0, y0)
	}

	// Seat 1: top-right
	w1, h1, x1, y1 := CalculateSeatGeometry(LayoutQuadGrid, "p2", 1, 4, totalW, totalH)
	if w1 != 1920 || h1 != 1080 || x1 != 1920 || y1 != 0 {
		t.Errorf("Seat 1 quad_grid unexpected: %dx%d @ (%d,%d)", w1, h1, x1, y1)
	}

	// Seat 2: bottom-left
	w2, h2, x2, y2 := CalculateSeatGeometry(LayoutQuadGrid, "p3", 2, 4, totalW, totalH)
	if w2 != 1920 || h2 != 1080 || x2 != 0 || y2 != 1080 {
		t.Errorf("Seat 2 quad_grid unexpected: %dx%d @ (%d,%d)", w2, h2, x2, y2)
	}

	// Seat 3: bottom-right
	w3, h3, x3, y3 := CalculateSeatGeometry(LayoutQuadGrid, "p4", 3, 4, totalW, totalH)
	if w3 != 1920 || h3 != 1080 || x3 != 1920 || y3 != 1080 {
		t.Errorf("Seat 3 quad_grid unexpected: %dx%d @ (%d,%d)", w3, h3, x3, y3)
	}
}

func TestCalculateSeatGeometry_ThreePlayer(t *testing.T) {
	totalW, totalH := 1920, 1080

	// Seat 0: top half
	w0, h0, x0, y0 := CalculateSeatGeometry(LayoutThreePlayer, "p1", 0, 3, totalW, totalH)
	if w0 != 1920 || h0 != 540 || x0 != 0 || y0 != 0 {
		t.Errorf("Seat 0 three_player unexpected: %dx%d @ (%d,%d)", w0, h0, x0, y0)
	}

	// Seat 1: bottom-left
	w1, h1, x1, y1 := CalculateSeatGeometry(LayoutThreePlayer, "p2", 1, 3, totalW, totalH)
	if w1 != 960 || h1 != 540 || x1 != 0 || y1 != 540 {
		t.Errorf("Seat 1 three_player unexpected: %dx%d @ (%d,%d)", w1, h1, x1, y1)
	}

	// Seat 2: bottom-right
	w2, h2, x2, y2 := CalculateSeatGeometry(LayoutThreePlayer, "p3", 2, 3, totalW, totalH)
	if w2 != 960 || h2 != 540 || x2 != 960 || y2 != 540 {
		t.Errorf("Seat 2 three_player unexpected: %dx%d @ (%d,%d)", w2, h2, x2, y2)
	}
}

func TestSupervisor_GetRunningInstances(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	gameDir := "/games/Overcooked"
	cleanDir := util.CanonicalPath(gameDir)

	cmdP1 := exec.Command("sleep", "1")
	cmdP2 := exec.Command("sleep", "1")
	_ = cmdP1.Start()
	_ = cmdP2.Start()
	defer func() {
		if cmdP1.Process != nil {
			_ = cmdP1.Process.Kill()
		}
		if cmdP2.Process != nil {
			_ = cmdP2.Process.Kill()
		}
	}()

	sess := &runningSession{
		config:    MultiseatSessionConfig{GameDir: gameDir, Name: "Overcooked", AppID: "448510"},
		startTime: 1234567,
		seats: map[string]*runningSeat{
			"p1": {
				seatID:    "p1",
				profileID: "main",
				cmd:       cmdP1,
				status:    SeatStatus{SeatID: "p1", ProfileID: "main", IsRunning: true, Mode: SeatModeLocal},
			},
			"p2": {
				seatID:    "p2",
				profileID: "p2",
				cmd:       cmdP2,
				status:    SeatStatus{SeatID: "p2", ProfileID: "p2", IsRunning: true, Mode: SeatModeLocal},
			},
		},
	}

	sup.sessions[cleanDir] = sess

	instances := sup.GetRunningInstances()
	if len(instances) != 2 {
		t.Fatalf("expected 2 running instances, got %d", len(instances))
	}

	p1Inst := instances[0]
	p2Inst := instances[1]
	if p1Inst.SeatID != "p1" {
		p1Inst, p2Inst = p2Inst, p1Inst
	}

	if p1Inst.GameName != "Overcooked" || p1Inst.AppID != "448510" || !p1Inst.IsMultiseat {
		t.Errorf("unexpected p1 instance: %+v", p1Inst)
	}
	if p1Inst.InstanceID != cleanDir {
		t.Errorf("expected p1 instance ID %s, got %s", cleanDir, p1Inst.InstanceID)
	}
	if p2Inst.InstanceID != fmt.Sprintf("%s:p2", cleanDir) {
		t.Errorf("expected p2 instance ID %s:p2, got %s", cleanDir, p2Inst.InstanceID)
	}
}


