package multiseat

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
)

// CheckCapabilities probes the host system for multi-seat support dependencies.
func CheckCapabilities() SystemCapabilities {
	_, bwrapErr := exec.LookPath("bwrap")
	_, gamescopeErr := exec.LookPath("gamescope")
	_, cageErr := exec.LookPath("cage")
	_, sunshineErr := exec.LookPath("sunshine")
	_, pactlErr := exec.LookPath("pactl")

	return SystemCapabilities{
		HasBwrap:     bwrapErr == nil,
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
