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

// snapshotEISockets captures all existing gamescope-*-ei socket filenames in XDG_RUNTIME_DIR.
func snapshotEISockets() map[string]bool {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	sockets := make(map[string]bool)
	entries, err := os.ReadDir(runtimeDir)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "gamescope-") && strings.HasSuffix(name, "-ei") {
				sockets[name] = true
			}
		}
	}
	return sockets
}

// getSocketPeerPID retrieves the server PID from a connected unix domain socket via SO_PEERCRED.
func getSocketPeerPID(conn net.Conn) (int, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, fmt.Errorf("not a unix connection")
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("get syscall conn: %w", err)
	}
	var cred *syscall.Ucred
	var credErr error
	err = raw.Control(func(fd uintptr) {
		cred, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil {
		return 0, err
	}
	if credErr != nil {
		return 0, credErr
	}
	if cred == nil {
		return 0, fmt.Errorf("nil credentials returned")
	}
	return int(cred.Pid), nil
}

// waitForEISocketForPID polls XDG_RUNTIME_DIR for the Gamescope EIS socket corresponding to targetPID.
// If targetPID <= 0, it matches any new socket not present in the existing snapshot.
func waitForEISocketForPID(targetPID int, existing map[string]bool, timeout time.Duration) (string, error) {
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
				if strings.HasPrefix(name, "gamescope-") && strings.HasSuffix(name, "-ei") {
					// Only exclude snapshot sockets when blindly waiting for a generic new socket (targetPID <= 0).
					// If targetPID > 0, we can deterministically match the socket's owner via SO_PEERCRED,
					// even if the socket file already existed on disk from a previous launch.
					if targetPID <= 0 && existing != nil && existing[name] {
						continue
					}
					sockPath := filepath.Join(runtimeDir, name)
					if targetPID > 0 {
						conn, err := net.DialTimeout("unix", sockPath, 150*time.Millisecond)
						if err == nil {
							pid, pidErr := getSocketPeerPID(conn)
							_ = conn.Close()
							if pidErr == nil && pid == targetPID {
								return sockPath, nil
							}
						}
					} else {
						conn, err := net.DialTimeout("unix", sockPath, 150*time.Millisecond)
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
	if targetPID > 0 {
		return "", fmt.Errorf("timeout waiting for Gamescope EIS socket for PID %d in %s", targetPID, runtimeDir)
	}
	return "", fmt.Errorf("timeout waiting for new Gamescope EIS socket in %s", runtimeDir)
}

// waitForNewEISocket polls XDG_RUNTIME_DIR for a newly created, connectable Gamescope EIS socket.
func waitForNewEISocket(existing map[string]bool, timeout time.Duration) (string, error) {
	return waitForEISocketForPID(0, existing, timeout)
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

// resolveCageDecision determines whether a seat should use cage as its compositor.
// Cage is used when:
//   - compositor is explicitly "cage" or "cage_gamescope"
//   - NestCage is set
//   - gamescope is not available but cage is (streaming or fallback)
//
// Cage is NOT used when compositor is explicitly "gamescope" or when gamescope is available.
func resolveCageDecision(seatCfg SeatConfig, seatID string, caps SystemCapabilities) bool {
	if seatCfg.Compositor == "gamescope" || !caps.HasCage {
		return false
	}
	if seatCfg.Compositor == "cage" || seatCfg.NestCage {
		return true
	}
	// Fallback to cage only if gamescope is not available
	return !caps.HasGamescope
}

type runningSeat struct {
	seatID       string
	profileID    string
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
	seats     map[string]*runningSeat // keyed by seatID ("p1", "p2", ...)
	startTime int64
	kwinCoord *KWinSplitCoordinator
}

// Supervisor manages concurrent multi-seat sessions and their lifecycle.
type Supervisor struct {
	mu         sync.Mutex
	sessions   map[string]*runningSession // keyed by canonical gameDir
	cfg        *config.Config
	bus        *events.Bus
	scaleGuard *ScaleGuard
}

var defaultSupervisor = NewSupervisor(nil, nil)

// DefaultSupervisor returns the global supervisor singleton.
//
// Deprecated: Supervisor should be injected via Engine.Multiseat. DefaultSupervisor
// remains strictly for CLI or tests requiring a fallback singleton.
func DefaultSupervisor() *Supervisor {
	return defaultSupervisor
}

// NewSupervisor instantiates a multi-seat session supervisor.
func NewSupervisor(cfg *config.Config, bus *events.Bus) *Supervisor {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	CleanupGamuxSinks(context.Background())
	scaleGuard := NewScaleGuard()
	scaleGuard.RecoverOrphanedLocks(context.Background())
	return &Supervisor{
		sessions:   make(map[string]*runningSession),
		cfg:        cfg,
		bus:        bus,
		scaleGuard: scaleGuard,
	}
}

// StopAll terminates all active multi-seat sessions and unloads any virtual audio sinks.
func (s *Supervisor) StopAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	dirs := make([]string, 0, len(s.sessions))
	for dir := range s.sessions {
		dirs = append(dirs, dir)
	}
	s.mu.Unlock()

	for _, dir := range dirs {
		_ = s.StopSession(dir)
	}
	if s.scaleGuard != nil {
		_ = s.scaleGuard.Restore(context.Background())
	}
	CleanupGamuxSinks(context.Background())
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

type preparedSeat struct {
	seatCfg      SeatConfig
	seatID       string
	profileID    string
	profileDir   string
	sink         *VirtualSink
	grabber      *InputGrabber
	cmd          *exec.Cmd
	logFile      *os.File
	logPath      string
	sunshineCmd  *exec.Cmd
	sunshinePort int
}

func (s *Supervisor) cleanupPrepared(ctx context.Context, prepSeats []*preparedSeat) {
	for _, ps := range prepSeats {
		if ps.grabber != nil {
			ps.grabber.Release()
		}
		if ps.sink != nil {
			ps.sink.Destroy(ctx)
		}
		if ps.sunshineCmd != nil && ps.sunshineCmd.Process != nil {
			_ = ps.sunshineCmd.Process.Kill()
		}
		if ps.cmd != nil && ps.cmd.Process != nil {
			_ = ps.cmd.Process.Kill()
		}
		if ps.logFile != nil {
			_ = ps.logFile.Close()
		}
	}
	s.mu.Lock()
	sessionsLeft := len(s.sessions)
	s.mu.Unlock()
	if sessionsLeft == 0 && s.scaleGuard != nil {
		_ = s.scaleGuard.Restore(ctx)
	}
}

// LaunchSession starts an isolated split-screen or multi-seat session for 1 to N players.
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

	seats := sessionCfg.EffectiveSeats()
	if len(seats) == 0 {
		return fmt.Errorf("at least one seat required")
	}

	// Normalize layout if seat count is smaller than requested layout requires
	if len(seats) == 2 && (sessionCfg.Layout == LayoutThreePlayer || sessionCfg.Layout == LayoutQuadGrid) {
		sessionCfg.Layout = LayoutSideBySide
	} else if len(seats) == 3 && sessionCfg.Layout == LayoutQuadGrid {
		sessionCfg.Layout = LayoutThreePlayer
	}

	// Clamp display index to valid connected displays
	detectedDisplays := util.DetectDisplays()
	numDisplays := len(detectedDisplays)
	if numDisplays == 0 {
		numDisplays = 1
	}
	isSplitLayout := sessionCfg.Layout == LayoutSideBySide || sessionCfg.Layout == LayoutTopBottom || sessionCfg.Layout == LayoutQuadGrid || sessionCfg.Layout == LayoutThreePlayer
	for i := range seats {
		if seats[i].DisplayIndex >= numDisplays || seats[i].DisplayIndex < 0 {
			seats[i].DisplayIndex = 0
		}
		if numDisplays == 1 && isSplitLayout {
			seats[i].DisplayIndex = 0
		}
	}

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
				s.cleanupPrepared(ctx, prepSeats)
				return fmt.Errorf("prepare seat %s profile: %w", seatID, err)
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

		// Base Inner Game Launch Command
		cmd, logFile, logPath, err := s.buildSeatCommand(ctx, sessionCfg, sCfg, pDir, sink, caps, idx, len(seats))
		if err != nil {
			s.cleanupPrepared(ctx, prepSeats)
			if grabber != nil {
				grabber.Release()
			}
			if sink != nil {
				sink.Destroy(ctx)
			}
			return fmt.Errorf("build seat %s command: %w", seatID, err)
		}

		// Sunshine instance if streaming
		var sunCmd *exec.Cmd
		sunPort := sCfg.SunshinePort
		if sunPort <= 0 {
			if idx == 0 && sessionCfg.SunshinePortP1 > 0 {
				sunPort = sessionCfg.SunshinePortP1
			} else if idx == 1 && sessionCfg.SunshinePortP2 > 0 {
				sunPort = sessionCfg.SunshinePortP2
			} else {
				sunPort = 47989 + (idx * 10)
			}
		}
		if sCfg.Mode == SeatModeStream && caps.HasSunshine {
			sunDir := filepath.Join(pDir, "sunshine")
			confPath, cErr := GenerateSunshineConfig(SunshineConfig{
				Port:          sunPort,
				WebUIPort:     sunPort + 1,
				AudioSink:     fmt.Sprintf("gamux-%s", seatID),
				WaylandSocket: fmt.Sprintf("wayland-gamux-%s", seatID),
				ConfigDir:     sunDir,
			})
			if cErr == nil {
				sunCmd = BuildSunshineCmd(ctx, confPath, fmt.Sprintf("wayland-gamux-%s", seatID))
				sunCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				if err := sunCmd.Start(); err != nil {
					slog.Warn("Failed to start Sunshine for seat", "seat", seatID, "error", err)
				}
			}
		}

		prepSeats = append(prepSeats, &preparedSeat{
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
		})
	}

	// Sockets snapshots for injection
	var initialWaylandSockets map[string]bool
	var initialEISockets map[string]bool
	needWaylandSnapshot := false
	needEISnapshot := false

	for _, ps := range prepSeats {
		useCage := resolveCageDecision(ps.seatCfg, ps.seatID, caps)
		useGamescope := caps.HasGamescope && !useCage
		if useCage {
			needWaylandSnapshot = true
		}
		if useGamescope && ps.grabber != nil {
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
		ps.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := ps.cmd.Start(); err != nil {
			s.cleanupPrepared(ctx, prepSeats)
			return fmt.Errorf("start seat %s process: %w", ps.seatID, err)
		}

		pgid, _ := syscall.Getpgid(ps.cmd.Process.Pid)
		var waylandLink string
		useCage := resolveCageDecision(ps.seatCfg, ps.seatID, caps)
		useGamescope := caps.HasGamescope && !useCage

		if useCage {
			sockPath, err := waitForNewWaylandSocket(initialWaylandSockets, 4*time.Second)
			if err != nil {
				slog.Warn("Failed to detect Cage Wayland socket", "seat", ps.seatID, "error", err)
			} else {
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
			}
		} else if useGamescope && ps.grabber != nil {
			sockPath, err := waitForEISocketForPID(ps.cmd.Process.Pid, initialEISockets, 4*time.Second)
			if err != nil {
				slog.Warn("Failed to detect Gamescope EIS socket", "seat", ps.seatID, "pid", ps.cmd.Process.Pid, "error", err)
			} else {
				eiClient := NewEIClient()
				dispW, dispH, _ := s.resolveDisplayDimensions(ps.seatCfg)
				slot := idx
				if ps.seatCfg.ScreenSlot != nil && *ps.seatCfg.ScreenSlot >= 0 && *ps.seatCfg.ScreenSlot < len(prepSeats) {
					slot = *ps.seatCfg.ScreenSlot
				}
				winW, winH, _, _ := CalculateSeatGeometry(sessionCfg.Layout, ps.seatID, slot, len(prepSeats), dispW, dispH)
				renderW := ps.seatCfg.TargetWidth
				renderH := ps.seatCfg.TargetHeight
				if renderW <= 0 || renderH <= 0 {
					renderW, renderH = winW, winH
				}
				eiClient.SetDimensions(renderW, renderH)

				if err := eiClient.Connect(sockPath); err != nil {
					slog.Warn("Failed to initialize Gamescope EIS virtual input", "seat", ps.seatID, "error", err)
				} else {
					ps.grabber.SetInjector(eiClient)
					slog.Info("Gamescope EIS virtual input injector connected", "seat", ps.seatID, "pid", ps.cmd.Process.Pid)
				}
			}
		}

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

		sessionSeats[ps.seatID] = &runningSeat{
			seatID:       ps.seatID,
			profileID:    ps.profileID,
			cmd:          ps.cmd,
			pgid:         pgid,
			sink:         ps.sink,
			sunshineCmd:  ps.sunshineCmd,
			sunshinePgid: 0,
			grabber:      ps.grabber,
			status:       st,
			logFile:      ps.logFile,
			logPath:      ps.logPath,
			waylandLink:  waylandLink,
		}

		placementSlot := idx
		if ps.seatCfg.ScreenSlot != nil && *ps.seatCfg.ScreenSlot >= 0 && *ps.seatCfg.ScreenSlot < len(prepSeats) {
			placementSlot = *ps.seatCfg.ScreenSlot
		}
		placements = append(placements, SeatPlacement{
			PID:          ps.cmd.Process.Pid,
			SeatIndex:    placementSlot,
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
		} else {
			if err := coord.Start(ctx); err != nil {
				slog.Warn("Failed to start KWin splitscreen coordinator", "error", err)
			} else {
				session.kwinCoord = coord
			}
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

func (s *Supervisor) resolveDisplayDimensions(seatCfg SeatConfig) (dispWidth, dispHeight, dispFPS int) {
	dispWidth = seatCfg.TargetWidth
	dispHeight = seatCfg.TargetHeight
	dispFPS = seatCfg.TargetFPS

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
	return dispWidth, dispHeight, dispFPS
}

func (s *Supervisor) buildSeatCommand(ctx context.Context, sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, profileDir string, sink *VirtualSink, caps SystemCapabilities, seatIndex, totalSeats int) (*exec.Cmd, *os.File, string, error) {
	seatID := seatCfg.SeatID
	if seatID == "" {
		seatID = fmt.Sprintf("p%d", seatIndex+1)
	}
	seatCfg.SeatID = seatID
	seatCfg.KeyboardNode = ResolveDeviceNode(seatCfg.KeyboardNode)
	seatCfg.MouseNode = ResolveDeviceNode(seatCfg.MouseNode)
	seatCfg.GamepadNode = ResolveDeviceNode(seatCfg.GamepadNode)

	numAppID, _ := util.ParseAppID(sessionCfg.AppID)
	operatingMode := domain.ModeLoader
	if profile, err := compat.LoadProfile(sessionCfg.GameDir); err == nil && profile != nil && profile.OperatingMode != "" {
		operatingMode = domain.NormalizeOperatingMode(profile.OperatingMode)
	} else if s.cfg != nil && s.cfg.OperatingMode != "" {
		operatingMode = domain.NormalizeOperatingMode(s.cfg.OperatingMode)
	}

	gameStub := &domain.Game{
		GameRecord: domain.GameRecord{
			Name:         sessionCfg.Name,
			AppID:        sessionCfg.AppID,
			NumericAppID: numAppID,
			GameDir:      sessionCfg.GameDir,
		},
		LaunchProfile: &domain.LaunchProfile{
			OperatingMode: operatingMode,
		},
	}

	pfxPath, pfxErr := compat.EnsureSeatPrefix(ctx, s.cfg, gameStub, seatID, seatCfg.WinePrefix)
	if pfxErr != nil || pfxPath == "" {
		pfxPath = filepath.Join(profileDir, "prefix")
		_ = os.MkdirAll(pfxPath, 0755)
	}

	// 1. Build base game launch specification via runner
	effectiveProfileID := seatID
	if seatCfg.ProfileID != "" {
		effectiveProfileID = seatCfg.ProfileID
	}

	launchOpts := runner.LaunchOptions{
		GameDir:       sessionCfg.GameDir,
		Name:          sessionCfg.Name,
		AppID:         sessionCfg.AppID,
		ExePath:       sessionCfg.ExePath,
		ExeArgs:       sessionCfg.ExeArgs,
		Runner:        sessionCfg.Runner,
		OperatingMode: operatingMode,
		ProfileID:     effectiveProfileID,
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
	dispWidth, dispHeight, dispFPS := s.resolveDisplayDimensions(seatCfg)
	calcSlot := seatIndex
	if seatCfg.ScreenSlot != nil && *seatCfg.ScreenSlot >= 0 && *seatCfg.ScreenSlot < totalSeats {
		calcSlot = *seatCfg.ScreenSlot
	}
	winW, winH, winX, winY := CalculateSeatGeometry(sessionCfg.Layout, seatID, calcSlot, totalSeats, dispWidth, dispHeight)

	// 3. Wrap in Pasta network namespace (for seats > 0 to prevent port collisions; or if explicitly requested)
	if caps.HasPasta && (seatIndex > 0 || sessionCfg.EnableBroadcastRelay) {
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

	// 4. Wrap in compositors:
	//   Path 1: gamescope (scaling/resolution/FPS limit, split-screen or dual monitor, EIS input redirection)
	//   Path 2: cage (standalone kiosk Wayland compositor, fallback or explicit)
	useCage := resolveCageDecision(seatCfg, seatID, caps)
	useGamescope := caps.HasGamescope && !useCage

	hasDedicatedKBM := isExplicitDeviceNode(seatCfg.KeyboardNode) || isExplicitDeviceNode(seatCfg.MouseNode)
	if seatIndex == 0 && !seatCfg.ExclusiveGrab {
		hasDedicatedKBM = false
	}

	if useGamescope {
		renderW := seatCfg.TargetWidth
		renderH := seatCfg.TargetHeight
		if renderW <= 0 || renderH <= 0 {
			renderW = winW
			renderH = winH
		}

		gsBin, err := exec.LookPath("gamescope")
		if err != nil {
			gsBin = "gamescope"
		}

		forceGrab := seatCfg.ForceGrabCursor
		grabKbd := seatCfg.GrabKeyboard
		if hasDedicatedKBM {
			// In multi-seat sessions with exclusive hardware input grabbers, input is injected
			// directly via EIS. Gamescope's nested host window must NEVER grab the host's
			// desktop keyboard (-g) or mouse cursor (--force-grab-cursor), preventing conflict with Player 1.
			forceGrab = false
			grabKbd = false
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
			ForceGrabCursor: forceGrab,
			GrabKeyboard:    grabKbd,
			CursorPath:      seatCfg.CursorPath,
			Command:         cmdArgs,
		}
		gsArgs, gErr := BuildGamescopeArgs(dispOpts)
		if gErr != nil {
			return nil, nil, "", fmt.Errorf("build gamescope args: %w", gErr)
		}
		cmdArgs = append([]string{gsBin}, gsArgs...)
	} else if useCage {
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

	// In nested mode, Gamescope defaults to cv_cursor_composite=1 ("composite when not nested"),
	// expecting the host window manager to draw the cursor over the window.
	// However, for multi-seat sessions with dedicated hardware grabbers (or when ShowCursor is toggled),
	// physical input is redirected via EIS and the host desktop never renders a cursor for the seat.
	// Setting gamescope_cursor_composite=2 ("Always composite a cursor manually") forces Gamescope's
	// internal Vulkan compositor to render the mouse cursor plane directly into its frame output.
	if useGamescope && (seatCfg.ShowCursor || hasDedicatedKBM) {
		finalCmd.Env = append(finalCmd.Env, "gamescope_cursor_composite=2")
	}

	// If split layout is requested or coordinates are nonzero, set SDL window position environment
	// variables so SDL/Gamescope opens at the calculated (winX, winY) half of the screen.
	if useGamescope && (winX > 0 || winY > 0 || sessionCfg.Layout == LayoutSideBySide || sessionCfg.Layout == LayoutTopBottom) {
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
	s.mu.Lock()
	seats := make([]*runningSeat, 0, len(session.seats))
	for _, seat := range session.seats {
		seats = append(seats, seat)
	}
	s.mu.Unlock()

	type seatExit struct {
		seatID string
		seat   *runningSeat
		err    error
	}

	exitCh := make(chan seatExit, len(seats))
	for _, seat := range seats {
		st := seat
		go func() {
			err := st.cmd.Wait()
			exitCh <- seatExit{seatID: st.seatID, seat: st, err: err}
		}()
	}

	activeCount := len(seats)
	var lastExitErr error

	for activeCount > 0 {
		ex := <-exitCh
		activeCount--
		if ex.err != nil {
			lastExitErr = ex.err
		}

		s.mu.Lock()
		ex.seat.status.IsRunning = false
		link := ex.seat.waylandLink
		ex.seat.waylandLink = ""
		logF := ex.seat.logFile
		ex.seat.logFile = nil
		eb := s.bus
		s.mu.Unlock()

		slog.Info("Multi-seat Seat terminated", "dir", gameDir, "seat", ex.seatID, "profile", ex.seat.profileID, "err", ex.err)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if ex.seat.grabber != nil {
			ex.seat.grabber.Release()
		}
		if ex.seat.sink != nil {
			ex.seat.sink.Destroy(ctx)
		}
		if link != "" {
			_ = os.Remove(link)
		}
		if ex.seat.sunshineCmd != nil && ex.seat.sunshineCmd.Process != nil {
			_ = ex.seat.sunshineCmd.Process.Kill()
		}
		if logF != nil {
			_ = logF.Close()
		}
		cancel()

		if eb != nil {
			eb.Publish(events.NewGameInstanceExited(gameDir, session.config.Name, session.config.AppID, ex.err, ex.seat.profileID, ex.seatID))
		}

		// If a seat exited unexpectedly while other seats are still running
		if (ex.err != nil || time.Now().Unix()-session.startTime < 10) && activeCount > 0 {
			if eb != nil {
				msg := fmt.Sprintf("Player seat %s exited unexpectedly.", ex.seatID)
				if snippet := readLogTail(ex.seat.logPath, 15); snippet != "" {
					msg += "\n\nLog tail:\n" + snippet
				}
				eb.Publish(events.NewNotification(fmt.Sprintf("Multi-Seat %s Error", strings.ToUpper(ex.seatID)), msg))
			}
		}
	}

	duration := time.Now().Unix() - session.startTime

	s.mu.Lock()
	eb := s.bus
	s.mu.Unlock()

	if eb != nil {
		if lastExitErr != nil || duration < 10 {
			msg := fmt.Sprintf("Multi-Seat Session terminated after %ds.", duration)
			eb.Publish(events.NewNotification("Multi-Seat Session Finished", msg))
		}
		eb.Publish(events.NewGameExited(gameDir, session.config.Name, session.config.AppID, lastExitErr))
	}

	// Clean up remaining session state
	s.mu.Lock()
	activeSession := s.sessions[gameDir]
	s.mu.Unlock()

	if activeSession != nil {
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

	s.mu.Lock()
	for _, seat := range session.seats {
		seat.status.IsRunning = false
	}
	s.mu.Unlock()

	for _, seat := range session.seats {
		if seat.grabber != nil {
			seat.grabber.Release()
		}
		if seat.pgid > 1 {
			_ = syscall.Kill(-seat.pgid, syscall.SIGTERM)
		} else if seat.cmd != nil && seat.cmd.Process != nil {
			_ = seat.cmd.Process.Kill()
		}
		if seat.sunshineCmd != nil && seat.sunshineCmd.Process != nil {
			_ = seat.sunshineCmd.Process.Kill()
		}
		if seat.sink != nil {
			seat.sink.Destroy(ctx)
		}
		s.mu.Lock()
		logF := seat.logFile
		seat.logFile = nil
		link := seat.waylandLink
		seat.waylandLink = ""
		s.mu.Unlock()
		if logF != nil {
			_ = logF.Close()
		}
		if link != "" {
			_ = os.Remove(link)
		}
	}

	// Cleanup KWin splitscreen coordinator if active
	if session.kwinCoord != nil {
		_ = session.kwinCoord.Stop()
	}

	// Restore display scales if no other sessions are active
	s.mu.Lock()
	sessionsLeft := len(s.sessions)
	s.mu.Unlock()
	if sessionsLeft == 0 && s.scaleGuard != nil {
		_ = s.scaleGuard.Restore(ctx)
	}

	slog.Info("Multi-seat session cleanly terminated", "dir", gameDir)
	return nil
}

// StopSeat terminates only a single player seat in the session without killing others.
func (s *Supervisor) StopSeat(gameDir string, seatID string) error {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	session, exists := s.sessions[gameDir]
	s.mu.Unlock()

	if !exists || session == nil {
		return fmt.Errorf("no active multi-seat session for %s", gameDir)
	}

	s.mu.Lock()
	seat, found := session.seats[seatID]
	s.mu.Unlock()

	if !found || seat == nil {
		return fmt.Errorf("seat %s not found in session for %s", seatID, gameDir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if seat.grabber != nil {
		seat.grabber.Release()
	}
	if seat.pgid > 1 {
		_ = syscall.Kill(-seat.pgid, syscall.SIGTERM)
	} else if seat.cmd != nil && seat.cmd.Process != nil {
		_ = seat.cmd.Process.Kill()
	}
	if seat.sunshineCmd != nil && seat.sunshineCmd.Process != nil {
		_ = seat.sunshineCmd.Process.Kill()
	}
	if seat.sink != nil {
		seat.sink.Destroy(ctx)
	}
	s.mu.Lock()
	seat.status.IsRunning = false
	logF := seat.logFile
	seat.logFile = nil
	link := seat.waylandLink
	seat.waylandLink = ""
	s.mu.Unlock()

	if logF != nil {
		_ = logF.Close()
	}
	if link != "" {
		_ = os.Remove(link)
	}

	return nil
}

// GetStatus returns the live multi-seat session status for a game.
func (s *Supervisor) GetStatus(gameDir string) *MultiseatSessionStatus {
	gameDir = util.CanonicalPath(gameDir)

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[gameDir]
	if !exists || session == nil {
		return &MultiseatSessionStatus{
			GameDir:   gameDir,
			IsRunning: false,
		}
	}

	seatsStatus := make(map[string]SeatStatus, len(session.seats))
	isRunning := false
	for id, st := range session.seats {
		seatsStatus[id] = st.status
		if st.status.IsRunning {
			isRunning = true
		}
	}

	status := &MultiseatSessionStatus{
		GameDir:   gameDir,
		IsRunning: isRunning,
		Seats:     seatsStatus,
		StartTime: session.startTime,
	}

	// Populate backward-compatible Seat1/Seat2
	if s1, ok := seatsStatus["p1"]; ok {
		status.Seat1 = s1
	}
	if s2, ok := seatsStatus["p2"]; ok {
		status.Seat2 = s2
	}

	return status
}

// IsSessionRunning checks if a multi-seat session is active for the game directory.
func (s *Supervisor) IsSessionRunning(gameDir string) bool {
	st := s.GetStatus(gameDir)
	return st != nil && st.IsRunning
}

// RunningSessions returns canonical paths of all active multi-seat sessions.
func (s *Supervisor) RunningSessions() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := make([]string, 0, len(s.sessions))
	for dir, sess := range s.sessions {
		if sess != nil {
			for _, seat := range sess.seats {
				if seat.status.IsRunning {
					list = append(list, util.CanonicalPath(dir))
					break
				}
			}
		}
	}
	return list
}

// GetRunningInstances returns all active seat instances across all multiseat sessions.
func (s *Supervisor) GetRunningInstances() []domain.RunningInstance {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []domain.RunningInstance
	for gameDir, sess := range s.sessions {
		if sess == nil {
			continue
		}
		for seatID, seat := range sess.seats {
			if seat.status.IsRunning && seat.cmd != nil && seat.cmd.Process != nil {
				inst := domain.RunningInstance{
					InstanceID:  domain.MakeInstanceID(gameDir, seat.profileID),
					GameDir:     gameDir,
					GameName:    sess.config.Name,
					AppID:       sess.config.AppID,
					ProfileID:   seat.profileID,
					SeatID:      seatID,
					PID:         seat.cmd.Process.Pid,
					IsMultiseat: true,
					Mode:        string(seat.status.Mode),
					StartTime:   sess.startTime,
				}
				result = append(result, inst)
			}
		}
	}
	return result
}

func (s *Supervisor) cleanupSinks(ctx context.Context, sinks ...*VirtualSink) {
	for _, sink := range sinks {
		if sink != nil {
			sink.Destroy(ctx)
		}
	}
}
