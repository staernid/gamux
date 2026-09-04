package multiseat

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestBuildSeatCommand_DefaultPrefixes(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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
	cmdP1, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p1", filepath.Join(tmpDir, "p1"), nil, caps)
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
	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p2", filepath.Join(tmpDir, "p2"), nil, caps)
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
	cmdP1, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p1", filepath.Join(tmpDir, "p1"), nil, caps)
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
	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p2", filepath.Join(tmpDir, "p2"), nil, caps)
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

func TestBuildSeatCommand_P2_KBM_UsesCage(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p2", filepath.Join(tmpDir, "p2"), nil, caps)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	if !strings.Contains(argsStr, "cage -s --") {
		t.Errorf("expected Seat 2 with dedicated KBM to wrap in cage, got: %s", argsStr)
	}

	var hasSoftwareCursor bool
	for _, env := range cmdP2.Env {
		if env == "WLR_NO_HARDWARE_CURSORS=1" {
			hasSoftwareCursor = true
			break
		}
	}
	if !hasSoftwareCursor {
		t.Errorf("expected WLR_NO_HARDWARE_CURSORS=1 in Seat 2 Env: %v", cmdP2.Env)
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

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p2", filepath.Join(tmpDir, "p2"), nil, caps)
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

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p2", filepath.Join(tmpDir, "p2"), nil, caps)
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

	cmdP2, _, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, "p2", filepath.Join(tmpDir, "p2"), nil, caps)
	if err != nil {
		t.Fatalf("buildSeatCommand(p2) failed: %v", err)
	}

	argsStr := strings.Join(cmdP2.Args, " ")
	if !strings.Contains(argsStr, "cage -s --") {
		t.Errorf("expected Seat 2 to wrap in cage, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "gamescope") {
		t.Errorf("expected Seat 2 to nest gamescope inside cage, got: %s", argsStr)
	}
	cageIdx := strings.Index(argsStr, "cage")
	gamescopeIdx := strings.Index(argsStr, "gamescope")
	if cageIdx == -1 || gamescopeIdx == -1 || cageIdx >= gamescopeIdx {
		t.Errorf("expected cage to wrap gamescope (cageIdx=%d, gamescopeIdx=%d), got: %s", cageIdx, gamescopeIdx, argsStr)
	}
}



