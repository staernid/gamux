package multiseat

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/util"
)

// snapshotWaylandSockets captures all existing Wayland socket filenames in XDG_RUNTIME_DIR.
func snapshotWaylandSockets() map[string]bool {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	sockets := make(map[string]bool)
	entries, err := os.ReadDir(runtimeDir)
	if err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "wayland-") && !strings.HasSuffix(e.Name(), ".lock") {
				sockets[e.Name()] = true
			}
		}
	}
	return sockets
}

// waitForNewWaylandSocket polls XDG_RUNTIME_DIR for a newly created, connectable Wayland socket.
func waitForNewWaylandSocket(existing map[string]bool, timeout time.Duration) (string, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(runtimeDir)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, "wayland-") && !strings.HasSuffix(name, ".lock") && !strings.Contains(name, "gamux") {
					if !existing[name] {
						sockPath := filepath.Join(runtimeDir, name)
						conn, err := net.DialTimeout("unix", sockPath, 100*time.Millisecond)
						if err == nil {
							_ = conn.Close()
							return sockPath, nil
						}
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "", fmt.Errorf("timeout waiting for new Wayland socket in %s", runtimeDir)
}

// CheckCapabilities probes the host system for multi-seat support dependencies.
func CheckCapabilities() SystemCapabilities {
	_, bwrapErr := exec.LookPath("bwrap")
	_, pastaErr := exec.LookPath("pasta")
	_, gamescopeErr := exec.LookPath("gamescope")
	_, cageErr := exec.LookPath("cage")
	_, sunshineErr := exec.LookPath("sunshine")
	_, pactlErr := exec.LookPath("pactl")

	return SystemCapabilities{
		HasBwrap:     bwrapErr == nil,
		HasPasta:     pastaErr == nil,
		HasGamescope: gamescopeErr == nil,
		HasCage:      cageErr == nil,
		HasSunshine:  sunshineErr == nil,
		HasPipewire:  pactlErr == nil,
	}
}

type runningSeat struct {
	cmd          *exec.Cmd
	pgid         int
	sink         *VirtualSink
	sunshineCmd  *exec.Cmd
	sunshinePgid int
	grabber      *InputGrabber
	status       SeatStatus
	logFile      *os.File
	logPath      string
	waylandLink  string
}

type runningSession struct {
	config    MultiseatSessionConfig
	seat1     runningSeat
	seat2     runningSeat
	startTime int64
}

// Supervisor manages concurrent multi-seat sessions and their lifecycle.
type Supervisor struct {
	mu       sync.Mutex
	sessions map[string]*runningSession // keyed by canonical gameDir
	cfg      *config.Config
	bus      *events.Bus
}

var defaultSupervisor = NewSupervisor(nil, nil)

// DefaultSupervisor returns the global supervisor singleton.
func DefaultSupervisor() *Supervisor {
	return defaultSupervisor
}

// NewSupervisor instantiates a multi-seat session supervisor.
func NewSupervisor(cfg *config.Config, bus *events.Bus) *Supervisor {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Supervisor{
		sessions: make(map[string]*runningSession),
		cfg:      cfg,
		bus:      bus,
	}
}

// SetBus sets the event broker for notifications.
func (s *Supervisor) SetBus(bus *events.Bus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bus = bus
}

// SetConfig updates the configuration reference.
func (s *Supervisor) SetConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}

// LaunchSession starts an isolated two-player split-screen or multi-seat session.
func (s *Supervisor) LaunchSession(ctx context.Context, sessionCfg MultiseatSessionConfig) error {
	gameDir := util.CanonicalPath(sessionCfg.GameDir)
	if gameDir == "" {
		return fmt.Errorf("game dir cannot be empty")
	}

	s.mu.Lock()
	if _, exists := s.sessions[gameDir]; exists {
		s.mu.Unlock()
		return fmt.Errorf("a multi-seat session is already running for %s", gameDir)
	}
	s.mu.Unlock()

	homeDir, _ := os.UserHomeDir()
	profilesBase := filepath.Join(homeDir, ".local", "share", "gamux", "profiles")

	// 1. Prepare Seat Profiles (Player 1 uses real host home; Player 2 uses isolated profile)
	p1Dir := filepath.Join(profilesBase, "player1")
	p2Dir := filepath.Join(profilesBase, "player2")

	_ = os.MkdirAll(filepath.Join(p1Dir, "logs"), 0755)
	if err := EnsureSeatProfile(p2Dir, sessionCfg.Seat2.Player); err != nil {
		return fmt.Errorf("prepare seat 2 profile: %w", err)
	}

	// 2. Audio Sinks (PipeWire null sinks)
	var sinkP1, sinkP2 *VirtualSink
	caps := CheckCapabilities()
	if caps.HasPipewire {
		var err error
		sinkP1, err = CreateVirtualSink(ctx, "gamux-p1", fmt.Sprintf("Gamux P1 - %s", sessionCfg.Name))
		if err != nil {
			slog.Warn("Could not create PipeWire sink for P1", "error", err)
		} else if sessionCfg.Seat1.PhysicalSink != "" {
			_ = RouteSinkToPhysical(ctx, sinkP1, sessionCfg.Seat1.PhysicalSink)
		}

		sinkP2, err = CreateVirtualSink(ctx, "gamux-p2", fmt.Sprintf("Gamux P2 - %s", sessionCfg.Name))
		if err != nil {
			slog.Warn("Could not create PipeWire sink for P2", "error", err)
		} else if sessionCfg.Seat2.PhysicalSink != "" {
			_ = RouteSinkToPhysical(ctx, sinkP2, sessionCfg.Seat2.PhysicalSink)
		}
	}

	// 3. Hardware Input Grabbers (for seats with dedicated keyboard/mouse devices)
	var p1Grabber, p2Grabber *InputGrabber
	if sessionCfg.Seat1.ExclusiveGrab && (isExplicitDeviceNode(sessionCfg.Seat1.KeyboardNode) || isExplicitDeviceNode(sessionCfg.Seat1.MouseNode)) {
		p1Grabber = NewInputGrabber(sessionCfg.Seat1.KeyboardNode, sessionCfg.Seat1.MouseNode)
		if err := p1Grabber.Grab(); err != nil {
			slog.Warn("Failed to grab P1 input devices", "error", err)
		}
	}
	p2HasKBM := isExplicitDeviceNode(sessionCfg.Seat2.KeyboardNode) || isExplicitDeviceNode(sessionCfg.Seat2.MouseNode)
	p2UsesCage := (sessionCfg.Seat2.Compositor == "cage" || sessionCfg.Seat2.Compositor == "cage_gamescope" || sessionCfg.Seat2.NestCage ||
		((sessionCfg.Seat2.Compositor == "" || sessionCfg.Seat2.Compositor == "auto") && p2HasKBM) ||
		(sessionCfg.Seat2.Mode == SeatModeStream && !caps.HasGamescope)) && caps.HasCage
	if sessionCfg.Seat2.Compositor == "gamescope" {
		p2UsesCage = false
	}

	if (sessionCfg.Seat2.ExclusiveGrab || (p2HasKBM && p2UsesCage)) && p2HasKBM {
		p2Grabber = NewInputGrabber(sessionCfg.Seat2.KeyboardNode, sessionCfg.Seat2.MouseNode)
		if err := p2Grabber.Grab(); err != nil {
			slog.Warn("Failed to grab P2 input devices", "error", err)
		}
	}

	// 4. Resolve Base Inner Game Launch Commands
	p1Cmd, p1Log, p1LogPath, err := s.buildSeatCommand(ctx, sessionCfg, "p1", p1Dir, sinkP1, caps)
	if err != nil {
		s.cleanupSinks(ctx, sinkP1, sinkP2)
		if p1Grabber != nil {
			p1Grabber.Release()
		}
		if p2Grabber != nil {
			p2Grabber.Release()
		}
		return fmt.Errorf("build seat 1 command: %w", err)
	}

	p2Cmd, p2Log, p2LogPath, err := s.buildSeatCommand(ctx, sessionCfg, "p2", p2Dir, sinkP2, caps)
	if err != nil {
		s.cleanupSinks(ctx, sinkP1, sinkP2)
		if p1Grabber != nil {
			p1Grabber.Release()
		}
		if p2Grabber != nil {
			p2Grabber.Release()
		}
		return fmt.Errorf("build seat 2 command: %w", err)
	}

	// 5. Start Sunshine instances if streaming
	var sunP1Cmd, sunP2Cmd *exec.Cmd
	if sessionCfg.Seat1.Mode == SeatModeStream && caps.HasSunshine {
		sunPort := sessionCfg.SunshinePortP1
		if sunPort <= 0 {
			sunPort = 47989
		}
		sunDir := filepath.Join(p1Dir, "sunshine")
		confPath, cErr := GenerateSunshineConfig(SunshineConfig{
			Port:          sunPort,
			WebUIPort:     sunPort + 1,
			AudioSink:     "gamux-p1",
			WaylandSocket: "wayland-gamux-p1",
			ConfigDir:     sunDir,
		})
		if cErr == nil {
			sunP1Cmd = BuildSunshineCmd(ctx, confPath, "wayland-gamux-p1")
			sunP1Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := sunP1Cmd.Start(); err != nil {
				slog.Warn("Failed to start Sunshine for P1", "error", err)
			}
		}
	}

	if sessionCfg.Seat2.Mode == SeatModeStream && caps.HasSunshine {
		sunPort := sessionCfg.SunshinePortP2
		if sunPort <= 0 {
			sunPort = 48010
		}
		sunDir := filepath.Join(p2Dir, "sunshine")
		confPath, cErr := GenerateSunshineConfig(SunshineConfig{
			Port:          sunPort,
			WebUIPort:     sunPort + 1,
			AudioSink:     "gamux-p2",
			WaylandSocket: "wayland-gamux-p2",
			ConfigDir:     sunDir,
		})
		if cErr == nil {
			sunP2Cmd = BuildSunshineCmd(ctx, confPath, "wayland-gamux-p2")
			sunP2Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := sunP2Cmd.Start(); err != nil {
				slog.Warn("Failed to start Sunshine for P2", "error", err)
			}
		}
	}

	// 6. Launch Player 1 Process
	p1Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := p1Cmd.Start(); err != nil {
		s.cleanupSinks(ctx, sinkP1, sinkP2)
		if p1Grabber != nil {
			p1Grabber.Release()
		}
		if p2Grabber != nil {
			p2Grabber.Release()
		}
		return fmt.Errorf("start seat 1 process: %w", err)
	}

	// 7. Launch Player 2 Process
	var initialSockets map[string]bool
	if p2UsesCage {
		initialSockets = snapshotWaylandSockets()
	}

	p2Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := p2Cmd.Start(); err != nil {
		_ = syscall.Kill(-p1Cmd.Process.Pid, syscall.SIGTERM)
		s.cleanupSinks(ctx, sinkP1, sinkP2)
		if p1Grabber != nil {
			p1Grabber.Release()
		}
		if p2Grabber != nil {
			p2Grabber.Release()
		}
		return fmt.Errorf("start seat 2 process: %w", err)
	}

	var p2WaylandLink string
	if p2UsesCage {
		sockPath, err := waitForNewWaylandSocket(initialSockets, 4*time.Second)
		if err != nil {
			slog.Warn("Failed to detect Cage Wayland socket for Seat 2", "error", err)
		} else {
			runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
			if runtimeDir == "" {
				runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
			}
			linkPath := filepath.Join(runtimeDir, "wayland-gamux-p2")
			_ = os.Remove(linkPath)
			if err := os.Symlink(sockPath, linkPath); err == nil {
				p2WaylandLink = linkPath
				slog.Info("Symlinked Seat 2 Wayland socket", "target", sockPath, "link", linkPath)
			}

			// If p2Grabber is active, initialize Wayland virtual input injection
			if p2Grabber != nil {
				wlClient := NewWaylandClient()
				if err := wlClient.Connect(sockPath); err != nil {
					slog.Warn("Failed to initialize Wayland virtual input for Seat 2", "socket", sockPath, "error", err)
				} else {
					p2Grabber.SetWaylandClient(wlClient)
					slog.Info("Wayland virtual input injector connected for Seat 2", "socket", sockPath)
				}
			}
		}
	}

	p1Pgid, _ := syscall.Getpgid(p1Cmd.Process.Pid)
	p2Pgid, _ := syscall.Getpgid(p2Cmd.Process.Pid)

	session := &runningSession{
		config:    sessionCfg,
		startTime: time.Now().Unix(),
		seat1: runningSeat{
			cmd:     p1Cmd,
			pgid:    p1Pgid,
			sink:    sinkP1,
			grabber: p1Grabber,
			logFile: p1Log,
			logPath: p1LogPath,
			status: SeatStatus{
				PID:          p1Cmd.Process.Pid,
				Mode:         sessionCfg.Seat1.Mode,
				IsRunning:    true,
				AudioSink:    "gamux-p1",
				GamepadNode:  sessionCfg.Seat1.GamepadNode,
				KeyboardNode: sessionCfg.Seat1.KeyboardNode,
				MouseNode:    sessionCfg.Seat1.MouseNode,
				ProcessGroup: p1Pgid,
			},
		},
		seat2: runningSeat{
			cmd:         p2Cmd,
			pgid:        p2Pgid,
			sink:        sinkP2,
			grabber:     p2Grabber,
			logFile:     p2Log,
			logPath:     p2LogPath,
			waylandLink: p2WaylandLink,
			status: SeatStatus{
				PID:          p2Cmd.Process.Pid,
				Mode:         sessionCfg.Seat2.Mode,
				IsRunning:    true,
				AudioSink:    "gamux-p2",
				GamepadNode:  sessionCfg.Seat2.GamepadNode,
				KeyboardNode: sessionCfg.Seat2.KeyboardNode,
				MouseNode:    sessionCfg.Seat2.MouseNode,
				ProcessGroup: p2Pgid,
			},
		},
	}

	if sunP1Cmd != nil && sunP1Cmd.Process != nil {
		session.seat1.sunshineCmd = sunP1Cmd
		session.seat1.status.StreamURL = fmt.Sprintf("https://localhost:%d", sessionCfg.SunshinePortP1)
		session.seat1.status.WebUIURL = fmt.Sprintf("https://localhost:%d", sessionCfg.SunshinePortP1+1)
	}
	if sunP2Cmd != nil && sunP2Cmd.Process != nil {
		session.seat2.sunshineCmd = sunP2Cmd
		session.seat2.status.StreamURL = fmt.Sprintf("https://localhost:%d", sessionCfg.SunshinePortP2)
		session.seat2.status.WebUIURL = fmt.Sprintf("https://localhost:%d", sessionCfg.SunshinePortP2+1)
	}

	s.mu.Lock()
	s.sessions[gameDir] = session
	eb := s.bus
	s.mu.Unlock()

	if eb != nil {
		eb.Publish(events.NewGameLaunched(gameDir, sessionCfg.Name, sessionCfg.AppID, p1Cmd.Process.Pid))
	}

	slog.Info("Multi-seat session started", "dir", gameDir, "p1_pid", p1Cmd.Process.Pid, "p2_pid", p2Cmd.Process.Pid)

	// Background supervisor goroutine to monitor exit
	go s.superviseSession(gameDir, session)

	return nil
}

func (s *Supervisor) buildSeatCommand(ctx context.Context, sessionCfg MultiseatSessionConfig, seatID string, profileDir string, sink *VirtualSink, caps SystemCapabilities) (*exec.Cmd, *os.File, string, error) {
	seatCfg := sessionCfg.Seat1
	if seatID == "p2" {
		seatCfg = sessionCfg.Seat2
	}

	numAppID, _ := util.ParseAppID(sessionCfg.AppID)
	gameStub := &domain.Game{
		GameRecord: domain.GameRecord{
			Name:         sessionCfg.Name,
			AppID:        sessionCfg.AppID,
			NumericAppID: numAppID,
			GameDir:      sessionCfg.GameDir,
		},
	}

	pfxPath, pfxErr := compat.EnsureSeatPrefix(ctx, s.cfg, gameStub, seatID, seatCfg.WinePrefix)
	if pfxErr != nil || pfxPath == "" {
		pfxPath = filepath.Join(profileDir, "prefix")
		_ = os.MkdirAll(pfxPath, 0755)
	}

	// 1. Build base game launch specification via runner
	launchOpts := runner.LaunchOptions{
		GameDir:       sessionCfg.GameDir,
		Name:          sessionCfg.Name,
		AppID:         sessionCfg.AppID,
		ExePath:       sessionCfg.ExePath,
		ExeArgs:       sessionCfg.ExeArgs,
		Runner:        sessionCfg.Runner,
		OperatingMode: "loader",
		WinePrefix:    pfxPath,
		CustomEnv:     make(map[string]string),
	}

	if seatCfg.Player.CustomEnv != nil {
		for k, v := range seatCfg.Player.CustomEnv {
			launchOpts.CustomEnv[k] = v
		}
	}

	spec, err := runner.BuildLaunchSpec(ctx, s.cfg, launchOpts)
	if err != nil {
		return nil, nil, "", fmt.Errorf("build launch spec: %w", err)
	}

	cmdArgs := spec.Args

	// 2. Wrap in Bubblewrap sandbox (bwrap)
	// Bwrap provides filesystem isolation (read-only game dir, isolated Wine prefix, isolated GSE saves)
	// and input isolation (gamepad and KBM masking).
	if caps.HasBwrap {
		sinkName := ""
		if sink != nil {
			sinkName = sink.SinkName
		}

		bwrapOpts := BwrapOptions{
			SeatID:        seatID,
			ProfileDir:    profileDir,
			GameDir:       sessionCfg.GameDir,
			WinePrefix:    pfxPath,
			GamepadNode:   seatCfg.GamepadNode,
			KeyboardNode:  seatCfg.KeyboardNode,
			MouseNode:     seatCfg.MouseNode,
			AudioSink:     sinkName,
			CustomEnv:     spec.Env,
			Command:       cmdArgs,
			DieWithParent: true,
		}

		bwArgs, bErr := BuildBwrapArgs(bwrapOpts)
		if bErr != nil {
			return nil, nil, "", fmt.Errorf("build bwrap args: %w", bErr)
		}
		bwrapBin, err := exec.LookPath("bwrap")
		if err != nil {
			bwrapBin = "bwrap"
		}
		cmdArgs = append([]string{bwrapBin}, bwArgs...)
	}

	// Calculate display geometry
	dispWidth := seatCfg.TargetWidth
	dispHeight := seatCfg.TargetHeight
	dispFPS := seatCfg.TargetFPS

	if s.cfg != nil {
		if dispWidth <= 0 && s.cfg.GamescopeWidth > 0 {
			dispWidth = s.cfg.GamescopeWidth
		}
		if dispHeight <= 0 && s.cfg.GamescopeHeight > 0 {
			dispHeight = s.cfg.GamescopeHeight
		}
		if dispFPS <= 0 && s.cfg.GamescopeFPS > 0 {
			dispFPS = s.cfg.GamescopeFPS
		}
	}

	if dispWidth <= 0 || dispHeight <= 0 {
		info := util.DetectPrimaryDisplay()
		dispWidth = info.Width
		dispHeight = info.Height
		if dispFPS <= 0 {
			dispFPS = info.RefreshRate
		}
	}

	winW, winH, winX, winY := CalculateSplitGeometry(sessionCfg.Layout, seatID, dispWidth, dispHeight)

	// 3. Wrap in Pasta network namespace (for Seat 2 to prevent port collisions; Seat 1 if requested)
	if caps.HasPasta && seatID == "p2" {
		pastaOpts := PastaOptions{
			EnableBroadcastRelay: sessionCfg.EnableBroadcastRelay,
			BroadcastPort:        47584,
			Command:              cmdArgs,
		}
		pArgs, pErr := BuildPastaArgs(pastaOpts)
		if pErr != nil {
			return nil, nil, "", fmt.Errorf("build pasta args: %w", pErr)
		}
		pastaBin, pErr := ResolvePastaBinary()
		if pErr != nil || pastaBin == "" {
			pastaBin = "pasta"
		}
		cmdArgs = append([]string{pastaBin}, pArgs...)
	}

	// 4. Wrap in Cage (if requested, dedicated KBM, or streaming fallback) and/or Gamescope (for window composition on host)
	p2HasKBM := seatID == "p2" && (isExplicitDeviceNode(seatCfg.KeyboardNode) || isExplicitDeviceNode(seatCfg.MouseNode))
	useCage := (seatCfg.Compositor == "cage" || seatCfg.Compositor == "cage_gamescope" || seatCfg.NestCage ||
		((seatCfg.Compositor == "" || seatCfg.Compositor == "auto") && p2HasKBM) ||
		(seatCfg.Mode == SeatModeStream && !caps.HasGamescope)) && caps.HasCage
	if seatCfg.Compositor == "gamescope" {
		useCage = false
	}

	if caps.HasGamescope {
		renderW := seatCfg.TargetWidth
		renderH := seatCfg.TargetHeight
		if renderW <= 0 || renderH <= 0 {
			renderW = winW
			renderH = winH
		}

		dispOpts := DisplayOptions{
			Mode:            seatCfg.Mode,
			Layout:          sessionCfg.Layout,
			SeatID:          seatID,
			DisplayIndex:    seatCfg.DisplayIndex,
			TargetWidth:     renderW,
			TargetHeight:    renderH,
			TargetFPS:       dispFPS,
			WindowWidth:     winW,
			WindowHeight:    winH,
			WindowX:         winX,
			WindowY:         winY,
			Fullscreen:      seatCfg.Mode == SeatModeLocal && sessionCfg.Layout == LayoutDualMonitor,
			Borderless:      sessionCfg.Layout == LayoutSideBySide || sessionCfg.Layout == LayoutTopBottom,
			ForceGrabCursor: seatCfg.ForceGrabCursor,
			GrabKeyboard:    seatCfg.GrabKeyboard,
			CursorPath:      seatCfg.CursorPath,
			Command:         cmdArgs,
		}

		gsArgs, gErr := BuildGamescopeArgs(dispOpts)
		if gErr != nil {
			return nil, nil, "", fmt.Errorf("build gamescope args: %w", gErr)
		}
		gsBin, err := exec.LookPath("gamescope")
		if err != nil {
			gsBin = "gamescope"
		}
		cmdArgs = append([]string{gsBin}, gsArgs...)
	}

	if useCage && caps.HasCage {
		cageBin, err := exec.LookPath("cage")
		if err != nil {
			cageBin = "cage"
		}
		cageArgs, cErr := BuildCageArgs("", cmdArgs)
		if cErr != nil {
			return nil, nil, "", fmt.Errorf("build cage args: %w", cErr)
		}
		cmdArgs = append([]string{cageBin}, cageArgs...)
	}

	finalCmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	finalCmd.Dir = sessionCfg.GameDir
	finalCmd.Env = os.Environ()

	// Ensure cursor theme and sizing environment variables are set so Gamescope/Xwayland/Wine
	// always renders a visible mouse cursor sprite.
	cursorTheme := os.Getenv("XCURSOR_THEME")
	if cursorTheme == "" {
		cursorTheme = "default"
	}
	cursorSize := os.Getenv("XCURSOR_SIZE")
	if cursorSize == "" {
		cursorSize = "24"
	}
	cursorPath := os.Getenv("XCURSOR_PATH")
	if cursorPath == "" {
		cursorPath = "/usr/share/icons:/usr/share/pixmaps:~/.local/share/icons:~/.icons"
	}

	finalCmd.Env = append(finalCmd.Env,
		"XCURSOR_THEME="+cursorTheme,
		"XCURSOR_SIZE="+cursorSize,
		"XCURSOR_PATH="+cursorPath,
	)

	// If software cursor is requested (ShowCursor) or running under Cage/wlroots
	if seatCfg.ShowCursor || useCage {
		finalCmd.Env = append(finalCmd.Env, "WLR_NO_HARDWARE_CURSORS=1")
	}

	// If split layout is requested or coordinates are nonzero, set SDL window position environment variables
	// so SDL/Gamescope opens at the calculated (winX, winY) half of the screen rather than centered.
	if winX > 0 || winY > 0 || sessionCfg.Layout == LayoutSideBySide || sessionCfg.Layout == LayoutTopBottom {
		posStr := fmt.Sprintf("%d,%d", winX, winY)
		finalCmd.Env = append(finalCmd.Env,
			"SDL_VIDEO_WINDOW_POS="+posStr,
			"SDL_WINDOW_POS="+posStr,
		)
	}

	// Prepare log file
	logDir := filepath.Join(profileDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", seatID))
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if logFile != nil {
		finalCmd.Stdout = logFile
		finalCmd.Stderr = logFile
	}

	return finalCmd, logFile, logPath, nil
}

func readLogTail(path string, maxLines int) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func (s *Supervisor) superviseSession(gameDir string, session *runningSession) {
	// Wait for processes to exit
	p1Done := make(chan error, 1)
	p2Done := make(chan error, 1)

	go func() {
		p1Done <- session.seat1.cmd.Wait()
	}()
	go func() {
		p2Done <- session.seat2.cmd.Wait()
	}()

	var firstErr error
	failedSeat := ""
	failedLogPath := ""

	select {
	case err := <-p1Done:
		session.seat1.status.IsRunning = false
		firstErr = err
		failedSeat = "Player 1"
		failedLogPath = session.seat1.logPath
		slog.Info("Multi-seat Seat 1 terminated", "dir", gameDir, "err", err)
	case err := <-p2Done:
		session.seat2.status.IsRunning = false
		firstErr = err
		failedSeat = "Player 2"
		failedLogPath = session.seat2.logPath
		slog.Info("Multi-seat Seat 2 terminated", "dir", gameDir, "err", err)
	}

	duration := time.Now().Unix() - session.startTime

	s.mu.Lock()
	eb := s.bus
	s.mu.Unlock()

	if eb != nil {
		if firstErr != nil || duration < 10 {
			msg := fmt.Sprintf("%s exited unexpectedly after %ds.", failedSeat, duration)
			if snippet := readLogTail(failedLogPath, 15); snippet != "" {
				msg += "\n\nLog tail:\n" + snippet
			}
			eb.Publish(events.NewNotification("Multi-Seat Session Error", msg))
		}
		eb.Publish(events.NewGameExited(gameDir, session.config.Name, session.config.AppID, firstErr))
	}

	// Check if both have finished or user initiates teardown
	s.mu.Lock()
	activeSession := s.sessions[gameDir]
	s.mu.Unlock()

	if activeSession != nil {
		// Stop session cleans up both processes, sinks, relay, and removes from map
		_ = s.StopSession(gameDir)
	}
}

// StopSession cleanly terminates all running seats, sinks, relays, and sunshine servers for a game.
func (s *Supervisor) StopSession(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	session, exists := s.sessions[gameDir]
	if !exists {
		s.mu.Unlock()
		return nil
	}
	delete(s.sessions, gameDir)
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Kill Player 1 process group and release grabber
	if session.seat1.grabber != nil {
		session.seat1.grabber.Release()
	}
	if session.seat1.pgid > 1 {
		_ = syscall.Kill(-session.seat1.pgid, syscall.SIGTERM)
	} else if session.seat1.cmd != nil && session.seat1.cmd.Process != nil {
		_ = session.seat1.cmd.Process.Kill()
	}

	// 2. Kill Player 2 process group and release grabber
	if session.seat2.grabber != nil {
		session.seat2.grabber.Release()
	}
	if session.seat2.pgid > 1 {
		_ = syscall.Kill(-session.seat2.pgid, syscall.SIGTERM)
	} else if session.seat2.cmd != nil && session.seat2.cmd.Process != nil {
		_ = session.seat2.cmd.Process.Kill()
	}

	// 3. Kill Sunshine processes
	if session.seat1.sunshineCmd != nil && session.seat1.sunshineCmd.Process != nil {
		_ = session.seat1.sunshineCmd.Process.Kill()
	}
	if session.seat2.sunshineCmd != nil && session.seat2.sunshineCmd.Process != nil {
		_ = session.seat2.sunshineCmd.Process.Kill()
	}

	// 4. Cleanup PipeWire virtual sinks
	s.cleanupSinks(ctx, session.seat1.sink, session.seat2.sink)

	// 5. Close log files
	if session.seat1.logFile != nil {
		_ = session.seat1.logFile.Close()
	}
	if session.seat2.logFile != nil {
		_ = session.seat2.logFile.Close()
	}

	// 6. Cleanup Wayland symlink if created
	if session.seat2.waylandLink != "" {
		_ = os.Remove(session.seat2.waylandLink)
	}

	slog.Info("Multi-seat session cleanly terminated", "dir", gameDir)
	return nil
}

// StopSeat terminates only a single player seat in the session without killing the other.
func (s *Supervisor) StopSeat(gameDir string, seatID string) error {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	session, exists := s.sessions[gameDir]
	s.mu.Unlock()

	if !exists || session == nil {
		return fmt.Errorf("no active multi-seat session for %s", gameDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if seatID == "p1" {
		if session.seat1.grabber != nil {
			session.seat1.grabber.Release()
		}
		if session.seat1.pgid > 1 {
			_ = syscall.Kill(-session.seat1.pgid, syscall.SIGTERM)
		} else if session.seat1.cmd != nil && session.seat1.cmd.Process != nil {
			_ = session.seat1.cmd.Process.Kill()
		}
		if session.seat1.sink != nil {
			session.seat1.sink.Destroy(ctx)
		}
		session.seat1.status.IsRunning = false
	} else if seatID == "p2" {
		if session.seat2.grabber != nil {
			session.seat2.grabber.Release()
		}
		if session.seat2.pgid > 1 {
			_ = syscall.Kill(-session.seat2.pgid, syscall.SIGTERM)
		} else if session.seat2.cmd != nil && session.seat2.cmd.Process != nil {
			_ = session.seat2.cmd.Process.Kill()
		}
		if session.seat2.sink != nil {
			session.seat2.sink.Destroy(ctx)
		}
		if session.seat2.waylandLink != "" {
			_ = os.Remove(session.seat2.waylandLink)
			session.seat2.waylandLink = ""
		}
		session.seat2.status.IsRunning = false
	} else {
		return s.StopSession(gameDir)
	}

	return nil
}

// GetStatus returns the live multi-seat session status for a game.
func (s *Supervisor) GetStatus(gameDir string) *MultiseatSessionStatus {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	session, exists := s.sessions[gameDir]
	s.mu.Unlock()

	if !exists || session == nil {
		return &MultiseatSessionStatus{
			GameDir:   gameDir,
			IsRunning: false,
		}
	}

	return &MultiseatSessionStatus{
		GameDir:   gameDir,
		IsRunning: session.seat1.status.IsRunning || session.seat2.status.IsRunning,
		Seat1:     session.seat1.status,
		Seat2:     session.seat2.status,
		StartTime: session.startTime,
	}
}

// IsSessionRunning checks if a multi-seat session is active for the game directory.
func (s *Supervisor) IsSessionRunning(gameDir string) bool {
	st := s.GetStatus(gameDir)
	return st != nil && st.IsRunning
}

func (s *Supervisor) cleanupSinks(ctx context.Context, s1, s2 *VirtualSink) {
	if s1 != nil {
		s1.Destroy(ctx)
	}
	if s2 != nil {
		s2.Destroy(ctx)
	}
}
