package multiseat

import (
	"context"
	"os"
	"os/exec"
	"sync"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/events"
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
	seatID      string
	profileID   string
	cmd         *exec.Cmd
	pgid        int
	sink        *VirtualSink
	sunshineCmd *exec.Cmd
	grabber     *InputGrabber
	status      SeatStatus
	logFile     *os.File
	logPath     string
	waylandLink string
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
	seatDeps   seatDeps // nil means host implementation; tests inject fakes
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
