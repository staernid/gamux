package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/util"
)

// ProcessEvent communicates status changes when a game starts or terminates.
type ProcessEvent struct {
	GameDir   string `json:"game_dir"`
	IsRunning bool   `json:"is_running"`
	PID       int    `json:"pid"`
}

type trackedProcess struct {
	cmd    *exec.Cmd
	pgid   int
	cancel context.CancelFunc
}

// Manager tracks and supervises running game processes.
type Manager struct {
	mu        sync.Mutex
	processes map[string]*trackedProcess
	bus       *events.Bus
	listeners []func(ProcessEvent)
}

// NewManager creates an initialized process Manager with an optional event broker.
func NewManager(bus ...*events.Bus) *Manager {
	var b *events.Bus
	if len(bus) > 0 {
		b = bus[0]
	}
	return &Manager{
		processes: make(map[string]*trackedProcess),
		bus:       b,
	}
}

// SetBus configures the event broker for process lifecycle events.
func (m *Manager) SetBus(bus *events.Bus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bus = bus
}

var defaultManager = NewManager()

// DefaultManager returns the shared singleton process manager.
func DefaultManager() *Manager {
	return defaultManager
}

// Start launches a game, tracks the process, and dispatches status events upon start and termination.
func (m *Manager) Start(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, error) {
	resolved := ResolveEffectiveOptions(cfg, opts)

	var steamCleanup func()
	if resolved.Platform == "linux" && domain.NormalizeOperatingMode(resolved.OperatingMode) == ModeLoader {
		var pErr error
		steamCleanup, pErr = PrepareLinuxSteamEnvironment(cfg, resolved)
		if pErr != nil {
			slog.Warn("Failed to prepare Linux Steam environment for loader mode", "error", pErr)
		}
	}

	cmd, logFile, err := Launch(ctx, cfg, opts)
	if err != nil {
		if steamCleanup != nil {
			steamCleanup()
		}
		return nil, err
	}

	gameDir := util.CanonicalPath(opts.GameDir)
	if gameDir != "" && cmd != nil && cmd.Process != nil {
		pgid := 0
		if p, err := syscall.Getpgid(cmd.Process.Pid); err == nil && p > 1 && p != os.Getpid() {
			pgid = p
		}

		supCtx, supCancel := context.WithCancel(context.Background())

		m.mu.Lock()
		m.processes[gameDir] = &trackedProcess{cmd: cmd, pgid: pgid, cancel: supCancel}
		b := m.bus
		m.mu.Unlock()

		m.emit(ProcessEvent{
			GameDir:   gameDir,
			IsRunning: true,
			PID:       cmd.Process.Pid,
		})

		if b != nil {
			b.Publish(events.NewGameLaunched(gameDir, opts.Name, opts.AppID, cmd.Process.Pid))
		}

		isGamescope := isGamescopeCmd(cfg, opts.Gamescope, cmd)

		go func() {
			_ = superviseProcess(supCtx, cmd, SuperviseOptions{
				GameDir:     gameDir,
				IsGamescope: isGamescope,
				LogFile:     logFile,
				OnExit: func(waitErr error, sessionSec int64) {
					if steamCleanup != nil {
						steamCleanup()
					}

					var totalPlaytime int64
					var lastPlayed int64
					if prof, err := compat.LoadProfile(gameDir); err == nil && prof != nil {
						totalPlaytime = prof.PlaytimeSeconds
						lastPlayed = prof.LastPlayed
					}

					m.mu.Lock()
					delete(m.processes, gameDir)
					m.mu.Unlock()

					m.emit(ProcessEvent{
						GameDir:   gameDir,
						IsRunning: false,
						PID:       0,
					})

					if b != nil {
						ev := events.NewGameExited(gameDir, opts.Name, opts.AppID, waitErr)
						if ev.Metadata == nil {
							ev.Metadata = make(map[string]string)
						}
						ev.Metadata["session_seconds"] = fmt.Sprintf("%d", sessionSec)
						if totalPlaytime > 0 {
							ev.Metadata["playtime_seconds"] = fmt.Sprintf("%d", totalPlaytime)
						}
						if lastPlayed > 0 {
							ev.Metadata["last_played"] = fmt.Sprintf("%d", lastPlayed)
						}
						b.Publish(ev)
					}
				},
			})
		}()
	}

	return cmd, nil
}

// Launch starts a domain.Game entity asynchronously using the configured operating mode and runner.
func (m *Manager) Launch(ctx context.Context, cfg *config.Config, game *domain.Game, overrides ...LaunchOptions) (*exec.Cmd, error) {
	if game == nil {
		return nil, fmt.Errorf("game cannot be nil")
	}
	opts := LaunchOptions{
		Name:          game.Name,
		AppID:         game.AppID,
		GameDir:       game.GameDir,
		ExePath:       game.EffectiveExePath(),
		ExeArgs:       game.EffectiveExeArgs(),
		Platform:      game.Platform,
		OperatingMode: game.EffectiveOperatingMode(),
		Runner:        game.EffectiveRunner(),
		WinePrefix:    game.EffectiveWinePrefix(),
	}
	if game.LaunchProfile != nil {
		opts.MangoHud = game.LaunchProfile.MangoHud
		opts.DisableOverlay = game.LaunchProfile.DisableOverlay
		opts.OfflineMode = game.LaunchProfile.OfflineMode
		opts.CustomEnv = game.LaunchProfile.CustomEnv
	}
	if len(overrides) > 0 {
		ov := overrides[0]
		if ov.ExePath != "" {
			opts.ExePath = ov.ExePath
		}
		if ov.ExeArgs != "" {
			opts.ExeArgs = ov.ExeArgs
		}
		if ov.Runner != "" {
			opts.Runner = ov.Runner
		}
		if ov.OperatingMode != "" {
			opts.OperatingMode = ov.OperatingMode
		}
		if ov.WinePrefix != "" {
			opts.WinePrefix = ov.WinePrefix
		}
		if ov.MangoHud != nil {
			opts.MangoHud = ov.MangoHud
		}
		if ov.DisableOverlay != nil {
			opts.DisableOverlay = ov.DisableOverlay
		}
		if ov.OfflineMode != nil {
			opts.OfflineMode = ov.OfflineMode
		}
		if ov.CustomEnv != nil {
			if opts.CustomEnv == nil {
				opts.CustomEnv = make(map[string]string)
			}
			for k, v := range ov.CustomEnv {
				opts.CustomEnv[k] = v
			}
		}
	}
	return m.Start(ctx, cfg, opts)
}

// RunGame executes a domain.Game entity synchronously, streaming output to stdout/stderr.
func (m *Manager) RunGame(ctx context.Context, cfg *config.Config, game *domain.Game, overrides ...LaunchOptions) error {
	if game == nil {
		return fmt.Errorf("game cannot be nil")
	}
	opts := LaunchOptions{
		Name:          game.Name,
		AppID:         game.AppID,
		GameDir:       game.GameDir,
		ExePath:       game.EffectiveExePath(),
		ExeArgs:       game.EffectiveExeArgs(),
		Platform:      game.Platform,
		OperatingMode: game.EffectiveOperatingMode(),
		Runner:        game.EffectiveRunner(),
		WinePrefix:    game.EffectiveWinePrefix(),
	}
	if game.LaunchProfile != nil {
		opts.MangoHud = game.LaunchProfile.MangoHud
		opts.DisableOverlay = game.LaunchProfile.DisableOverlay
		opts.OfflineMode = game.LaunchProfile.OfflineMode
		opts.CustomEnv = game.LaunchProfile.CustomEnv
	}
	if len(overrides) > 0 {
		ov := overrides[0]
		if ov.ExePath != "" {
			opts.ExePath = ov.ExePath
		}
		if ov.ExeArgs != "" {
			opts.ExeArgs = ov.ExeArgs
		}
		if ov.Runner != "" {
			opts.Runner = ov.Runner
		}
	}
	return Run(ctx, cfg, opts)
}

// Stop terminates an active game process and its child process group.
func (m *Manager) Stop(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	m.mu.Lock()
	proc, ok := m.processes[gameDir]
	m.mu.Unlock()

	if !ok || proc == nil {
		return nil
	}

	cmd := proc.cmd
	pgid := proc.pgid
	if pgid <= 1 && cmd != nil && cmd.Process != nil {
		if p, err := syscall.Getpgid(cmd.Process.Pid); err == nil && p > 1 && p != os.Getpid() {
			pgid = p
		}
	}

	slog.Info("Terminating running game process", "dir", gameDir, "pgid", pgid)

	if proc.cancel != nil {
		proc.cancel()
	}

	if pgid > 1 {
		terminateProcessGroup(pgid, 2500*time.Millisecond)
	} else if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		deadline := time.Now().Add(2500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		_ = cmd.Process.Kill()
	}

	return nil
}

// IsRunning checks if a game process in the given game directory is active.
func (m *Manager) IsRunning(gameDir string) bool {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	proc, ok := m.processes[gameDir]
	return ok && proc != nil
}

// RunningGames returns a slice of game directories that are currently running.
func (m *Manager) RunningGames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []string
	for dir := range m.processes {
		list = append(list, util.CanonicalPath(dir))
	}
	return list
}

// Subscribe registers a listener callback for process events.
func (m *Manager) Subscribe(fn func(ProcessEvent)) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, fn)
}

func (m *Manager) emit(ev ProcessEvent) {
	m.mu.Lock()
	listeners := make([]func(ProcessEvent), len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.Unlock()

	for _, fn := range listeners {
		go fn(ev)
	}
}
