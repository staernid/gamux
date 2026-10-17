package multiseat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
)

// normalizeLayout downgrades a layout that needs more seats than are present.
func normalizeLayout(layout DisplayLayout, seatCount int) DisplayLayout {
	switch {
	case seatCount == 2 && (layout == LayoutThreePlayer || layout == LayoutQuadGrid):
		return LayoutSideBySide
	case seatCount == 3 && layout == LayoutQuadGrid:
		return LayoutThreePlayer
	}
	return layout
}

// clampDisplayIndices forces each seat onto a connected display. Split layouts
// on a single display always use display 0.
func clampDisplayIndices(seats []SeatConfig, numDisplays int, layout DisplayLayout) {
	if numDisplays == 0 {
		numDisplays = 1
	}
	isSplit := layout == LayoutSideBySide || layout == LayoutTopBottom || layout == LayoutQuadGrid || layout == LayoutThreePlayer
	for i := range seats {
		if seats[i].DisplayIndex >= numDisplays || seats[i].DisplayIndex < 0 {
			seats[i].DisplayIndex = 0
		}
		if numDisplays == 1 && isSplit {
			seats[i].DisplayIndex = 0
		}
	}
}

// seatSlot returns the screen slot for a seat: its explicit ScreenSlot when
// valid, otherwise its position in the session.
func seatSlot(cfg SeatConfig, idx, total int) int {
	if cfg.ScreenSlot != nil && *cfg.ScreenSlot >= 0 && *cfg.ScreenSlot < total {
		return *cfg.ScreenSlot
	}
	return idx
}

// sunshinePortFor resolves the streaming port for a seat.
func sunshinePortFor(seat SeatConfig, sessionCfg MultiseatSessionConfig, idx int) int {
	if seat.SunshinePort > 0 {
		return seat.SunshinePort
	}
	if idx == 0 && sessionCfg.SunshinePortP1 > 0 {
		return sessionCfg.SunshinePortP1
	}
	if idx == 1 && sessionCfg.SunshinePortP2 > 0 {
		return sessionCfg.SunshinePortP2
	}
	return 47989 + idx*10
}

// LaunchSession starts an isolated split-screen or multi-seat session for 1 to N players.
func (s *Supervisor) LaunchSession(ctx context.Context, sessionCfg MultiseatSessionConfig) error {
	gameDir := util.CanonicalPath(sessionCfg.GameDir)
	if gameDir == "" {
		return fmt.Errorf("game dir cannot be empty")
	}

	s.mu.Lock()
	_, exists := s.sessions[gameDir]
	s.mu.Unlock()
	if exists {
		return fmt.Errorf("a multi-seat session is already running for %s", gameDir)
	}

	seats := sessionCfg.EffectiveSeats()
	if len(seats) == 0 {
		return fmt.Errorf("at least one seat required")
	}

	sessionCfg.Layout = normalizeLayout(sessionCfg.Layout, len(seats))
	clampDisplayIndices(seats, len(util.DetectDisplays()), sessionCfg.Layout)

	// Lock participating display scaling to 1.0 (100%) to prevent fractional sub-pixel rounding gaps
	if s.scaleGuard != nil {
		var dispIndices []int
		for _, sCfg := range seats {
			dispIndices = append(dispIndices, sCfg.DisplayIndex)
		}
		if err := s.scaleGuard.LockDisplays(ctx, dispIndices); err != nil {
			slog.Warn("ScaleGuard encountered error locking displays", "error", err)
		}
	}

	caps := CheckCapabilities()
	homeDir, _ := os.UserHomeDir()
	profilesBase := filepath.Join(homeDir, ".local", "share", "gamux", "profiles")

	var prepSeats []*preparedSeat
	for idx, sCfg := range seats {
		ps, err := s.prepareSeat(ctx, sessionCfg, sCfg, idx, len(seats), caps, profilesBase)
		if err != nil {
			s.cleanupPrepared(ctx, prepSeats)
			return err
		}
		prepSeats = append(prepSeats, ps)
	}

	// Socket snapshots let us identify the sockets created by each seat's compositor.
	var initialWaylandSockets, initialEISockets map[string]bool
	needWaylandSnapshot, needEISnapshot := false, false
	for _, ps := range prepSeats {
		useCage := resolveCageDecision(ps.seatCfg, ps.seatID, caps)
		if useCage {
			needWaylandSnapshot = true
		}
		if caps.HasGamescope && !useCage && ps.grabber != nil {
			needEISnapshot = true
		}
	}
	if needWaylandSnapshot {
		initialWaylandSockets = snapshotWaylandSockets()
	}
	if needEISnapshot {
		initialEISockets = snapshotEISockets()
	}

	sessionSeats := make(map[string]*runningSeat)
	var placements []SeatPlacement
	for idx, ps := range prepSeats {
		rs, err := s.startSeat(sessionCfg, ps, idx, len(prepSeats), caps, initialWaylandSockets, initialEISockets)
		if err != nil {
			s.cleanupPrepared(ctx, prepSeats)
			return err
		}
		sessionSeats[ps.seatID] = rs
		placements = append(placements, SeatPlacement{
			PID:          ps.cmd.Process.Pid,
			SeatIndex:    seatSlot(ps.seatCfg, idx, len(prepSeats)),
			SeatID:       ps.seatID,
			DisplayIndex: ps.seatCfg.DisplayIndex,
		})
	}

	session := &runningSession{
		config:    sessionCfg,
		seats:     sessionSeats,
		startTime: time.Now().Unix(),
	}

	// For local splitscreen sessions on KDE Plasma, initialize transient pixel-perfect window placement
	if IsKWinAvailable() && len(placements) > 0 {
		coord, err := NewKWinMultiCoordinator(placements, sessionCfg.Layout)
		if err != nil {
			slog.Warn("Failed to initialize KWin splitscreen coordinator", "error", err)
		} else if err := coord.Start(ctx); err != nil {
			slog.Warn("Failed to start KWin splitscreen coordinator", "error", err)
		} else {
			session.kwinCoord = coord
		}
	}

	s.mu.Lock()
	s.sessions[gameDir] = session
	eb := s.bus
	s.mu.Unlock()

	if eb != nil {
		for _, rs := range sessionSeats {
			eb.Publish(events.NewGameInstanceLaunched(gameDir, sessionCfg.Name, sessionCfg.AppID, rs.cmd.Process.Pid, rs.profileID, rs.seatID))
		}
	}

	slog.Info("Multi-seat session started", "dir", gameDir, "seats_count", len(sessionSeats))

	// Background supervisor goroutine to monitor exit independently
	go s.superviseSession(gameDir, session)

	return nil
}

// prepareSeat provisions everything a seat needs (profile, audio sink, input
// grab, launch command, optional Sunshine) without starting the game process.
// On error it releases the resources it created itself; previously prepared
// seats remain the caller's responsibility.
func (s *Supervisor) prepareSeat(ctx context.Context, sessionCfg MultiseatSessionConfig, sCfg SeatConfig, idx, total int, caps SystemCapabilities, profilesBase string) (*preparedSeat, error) {
	seatID := sCfg.SeatID
	if seatID == "" {
		seatID = fmt.Sprintf("p%d", idx+1)
	}
	profileID := sCfg.ProfileID
	if profileID == "" {
		profileID = seatID
	}
	sCfg.SeatID = seatID
	sCfg.ProfileID = profileID
	sCfg.KeyboardNode = ResolveDeviceNode(sCfg.KeyboardNode)
	sCfg.MouseNode = ResolveDeviceNode(sCfg.MouseNode)
	sCfg.GamepadNode = ResolveDeviceNode(sCfg.GamepadNode)

	// Profile dir & GSE Saves
	var pDir string
	if seatID == "p1" {
		pDir = filepath.Join(profilesBase, "player1")
	} else {
		pDir = filepath.Join(profilesBase, profileID)
		if err := EnsureSeatProfile(pDir, sCfg.Player); err != nil {
			return nil, fmt.Errorf("prepare seat %s profile: %w", seatID, err)
		}
		seatGSEDir := filepath.Join(pDir, "home", ".local", "share", "GSE Saves")
		if sCfg.WinePrefix != "" {
			_ = compat.EnsurePrefixGSESymlink(sCfg.WinePrefix, seatGSEDir)
		}
		_ = compat.EnsurePrefixGSESymlink(compat.DefaultPrefixesDirForSeat(profileID), seatGSEDir)
	}

	// Virtual PipeWire Sink
	var sink *VirtualSink
	if caps.HasPipewire {
		sinkName := fmt.Sprintf("gamux-%s", seatID)
		sinkDesc := fmt.Sprintf("Gamux %s - %s", strings.ToUpper(seatID), sessionCfg.Name)
		var err error
		sink, err = CreateVirtualSink(ctx, sinkName, sinkDesc)
		if err != nil {
			slog.Warn("Could not create PipeWire sink", "seat", seatID, "error", err)
		} else if sCfg.PhysicalSink != "" {
			_ = RouteSinkToPhysical(ctx, sink, sCfg.PhysicalSink)
		}
	}

	// Hardware Input Grabber
	var grabber *InputGrabber
	hasKBM := isExplicitDeviceNode(sCfg.KeyboardNode) || isExplicitDeviceNode(sCfg.MouseNode)
	if hasKBM && (seatID != "p1" || sCfg.ExclusiveGrab) {
		grabber = NewInputGrabber(sCfg.KeyboardNode, sCfg.MouseNode)
		if err := grabber.Grab(); err != nil {
			slog.Warn("Failed to grab input devices", "seat", seatID, "error", err)
		}
	}

	cmd, logFile, logPath, err := s.buildSeatCommand(ctx, sessionCfg, sCfg, pDir, sink, caps, idx, total)
	if err != nil {
		if grabber != nil {
			grabber.Release()
		}
		if sink != nil {
			sink.Destroy(ctx)
		}
		return nil, fmt.Errorf("build seat %s command: %w", seatID, err)
	}

	sunPort := sunshinePortFor(sCfg, sessionCfg, idx)
	var sunCmd *exec.Cmd
	if sCfg.Mode == SeatModeStream && caps.HasSunshine {
		sunCmd = startSunshine(ctx, seatID, pDir, sunPort)
	}

	return &preparedSeat{
		seatCfg:      sCfg,
		seatID:       seatID,
		profileID:    profileID,
		profileDir:   pDir,
		sink:         sink,
		grabber:      grabber,
		cmd:          cmd,
		logFile:      logFile,
		logPath:      logPath,
		sunshineCmd:  sunCmd,
		sunshinePort: sunPort,
	}, nil
}

// startSunshine launches a Sunshine streaming instance for a seat. It returns
// nil if the config could not be generated; a failed Start is logged and the
// command is still returned, matching prior behavior.
func startSunshine(ctx context.Context, seatID, profileDir string, port int) *exec.Cmd {
	waylandSocket := fmt.Sprintf("wayland-gamux-%s", seatID)
	confPath, err := GenerateSunshineConfig(SunshineConfig{
		Port:          port,
		WebUIPort:     port + 1,
		AudioSink:     fmt.Sprintf("gamux-%s", seatID),
		WaylandSocket: waylandSocket,
		ConfigDir:     filepath.Join(profileDir, "sunshine"),
	})
	if err != nil {
		return nil
	}
	cmd := BuildSunshineCmd(ctx, confPath, waylandSocket)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		slog.Warn("Failed to start Sunshine for seat", "seat", seatID, "error", err)
	}
	return cmd
}

// startSeat launches the prepared seat's game process, wires up virtual input
// injection, and returns its running state.
func (s *Supervisor) startSeat(sessionCfg MultiseatSessionConfig, ps *preparedSeat, idx, total int, caps SystemCapabilities, initialWayland, initialEI map[string]bool) (*runningSeat, error) {
	ps.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := ps.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start seat %s process: %w", ps.seatID, err)
	}
	pgid, _ := syscall.Getpgid(ps.cmd.Process.Pid)

	waylandLink := s.connectSeatInput(sessionCfg, ps, idx, total, caps, initialWayland, initialEI)

	st := SeatStatus{
		SeatID:       ps.seatID,
		ProfileID:    ps.profileID,
		PID:          ps.cmd.Process.Pid,
		Mode:         ps.seatCfg.Mode,
		IsRunning:    true,
		AudioSink:    fmt.Sprintf("gamux-%s", ps.seatID),
		GamepadNode:  ps.seatCfg.GamepadNode,
		KeyboardNode: ps.seatCfg.KeyboardNode,
		MouseNode:    ps.seatCfg.MouseNode,
		ProcessGroup: pgid,
	}
	if ps.sunshineCmd != nil && ps.sunshineCmd.Process != nil {
		st.StreamURL = fmt.Sprintf("https://localhost:%d", ps.sunshinePort)
		st.WebUIURL = fmt.Sprintf("https://localhost:%d", ps.sunshinePort+1)
	}

	return &runningSeat{
		seatID:      ps.seatID,
		profileID:   ps.profileID,
		cmd:         ps.cmd,
		pgid:        pgid,
		sink:        ps.sink,
		sunshineCmd: ps.sunshineCmd,
		grabber:     ps.grabber,
		status:      st,
		logFile:     ps.logFile,
		logPath:     ps.logPath,
		waylandLink: waylandLink,
	}, nil
}

// connectSeatInput attaches a virtual input injector to the seat's compositor
// (Cage via Wayland, Gamescope via EIS) when the seat has a grabber. It returns
// the Wayland socket symlink path, if one was created.
func (s *Supervisor) connectSeatInput(sessionCfg MultiseatSessionConfig, ps *preparedSeat, idx, total int, caps SystemCapabilities, initialWayland, initialEI map[string]bool) string {
	useCage := resolveCageDecision(ps.seatCfg, ps.seatID, caps)
	useGamescope := caps.HasGamescope && !useCage

	if useCage {
		sockPath, err := waitForNewWaylandSocket(initialWayland, 4*time.Second)
		if err != nil {
			slog.Warn("Failed to detect Cage Wayland socket", "seat", ps.seatID, "error", err)
			return ""
		}
		var waylandLink string
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
		}
		linkPath := filepath.Join(runtimeDir, fmt.Sprintf("wayland-gamux-%s", ps.seatID))
		_ = os.Remove(linkPath)
		if err := os.Symlink(sockPath, linkPath); err == nil {
			waylandLink = linkPath
			slog.Info("Symlinked Wayland socket", "seat", ps.seatID, "target", sockPath, "link", linkPath)
		}
		if ps.grabber != nil {
			wlClient := NewWaylandClient()
			if err := wlClient.Connect(sockPath); err != nil {
				slog.Warn("Failed to initialize Wayland virtual input", "seat", ps.seatID, "error", err)
			} else {
				ps.grabber.SetInjector(wlClient)
				slog.Info("Wayland virtual input injector connected", "seat", ps.seatID)
			}
		}
		return waylandLink
	}

	if useGamescope && ps.grabber != nil {
		pid := ps.cmd.Process.Pid
		sockPath, err := waitForEISocketForPID(pid, initialEI, 4*time.Second)
		if err != nil {
			slog.Warn("Failed to detect Gamescope EIS socket", "seat", ps.seatID, "pid", pid, "error", err)
			return ""
		}
		eiClient := NewEIClient()
		dispW, dispH, _ := s.resolveDisplayDimensions(ps.seatCfg)
		winW, winH, _, _ := CalculateSeatGeometry(sessionCfg.Layout, ps.seatID, seatSlot(ps.seatCfg, idx, total), total, dispW, dispH)
		renderW, renderH := ps.seatCfg.TargetWidth, ps.seatCfg.TargetHeight
		if renderW <= 0 || renderH <= 0 {
			renderW, renderH = winW, winH
		}
		eiClient.SetDimensions(renderW, renderH)

		if err := eiClient.Connect(sockPath); err != nil {
			slog.Warn("Failed to initialize Gamescope EIS virtual input", "seat", ps.seatID, "error", err)
		} else {
			ps.grabber.SetInjector(eiClient)
			slog.Info("Gamescope EIS virtual input injector connected", "seat", ps.seatID, "pid", pid)
		}
	}
	return ""
}
