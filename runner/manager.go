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

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
)

// ProcessEvent communicates status changes when a game starts or terminates.
type ProcessEvent struct {
	GameDir   string `json:"game_dir"`
	IsRunning bool   `json:"is_running"`
	PID       int    `json:"pid"`
}

// Manager tracks and supervises running game processes.
type Manager struct {
	mu        sync.Mutex
	processes map[string]*exec.Cmd
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
		processes: make(map[string]*exec.Cmd),
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
	cmd, logFile, err := Launch(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}

	gameDir := opts.GameDir
	if gameDir != "" && cmd != nil && cmd.Process != nil {
		m.mu.Lock()
		m.processes[gameDir] = cmd
		b := m.bus
		m.mu.Unlock()

		m.emit(ProcessEvent{
			GameDir:   gameDir,
			IsRunning: true,
			PID:       cmd.Process.Pid,
		})

		startTime := time.Now()

		if b != nil {
			b.Publish(events.NewGameLaunched(gameDir, opts.Name, opts.AppID, cmd.Process.Pid))
		}

		go func(gDir string, pCmd *exec.Cmd, lFile *os.File, eb *events.Bus, name, appID string, started time.Time) {
			waitErr := pCmd.Wait()
			elapsed := time.Since(started)
			sessionSec := int64(elapsed.Seconds())
			var totalPlaytime int64
			var lastPlayed int64

			if gDir != "" && sessionSec > 0 {
				if prof, err := LoadLaunchProfile(gDir); err == nil && prof != nil {
					prof.PlaytimeSeconds += sessionSec
					prof.LastPlayed = time.Now().Unix()
					totalPlaytime = prof.PlaytimeSeconds
					lastPlayed = prof.LastPlayed
					_ = SaveLaunchProfile(gDir, *prof)
				}
			}

			if lFile != nil {
				exitMsg := fmt.Sprintf("\n\n=== Process Finished ===\nTime: %s\nSession Duration: %v\nExit State: %v\n", time.Now().Format(time.RFC1123), elapsed.Round(time.Second), waitErr)
				_, _ = lFile.WriteString(exitMsg)
				_ = lFile.Close()
			}

			m.mu.Lock()
			delete(m.processes, gDir)
			m.mu.Unlock()

			m.emit(ProcessEvent{
				GameDir:   gDir,
				IsRunning: false,
				PID:       0,
			})

			if eb != nil {
				ev := events.NewGameExited(gDir, name, appID, waitErr)
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
				eb.Publish(ev)
			}
		}(gameDir, cmd, logFile, b, opts.Name, opts.AppID, startTime)
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
		opts.WineFullscreenFSR = game.LaunchProfile.WineFullscreenFSR
		opts.ProtonNVAPI = game.LaunchProfile.ProtonNVAPI
		opts.DXVKHud = game.LaunchProfile.DXVKHud
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
		if ov.WineFullscreenFSR != nil {
			opts.WineFullscreenFSR = ov.WineFullscreenFSR
		}
		if ov.ProtonNVAPI != nil {
			opts.ProtonNVAPI = ov.ProtonNVAPI
		}
		if ov.DXVKHud != "" {
			opts.DXVKHud = ov.DXVKHud
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
		opts.WineFullscreenFSR = game.LaunchProfile.WineFullscreenFSR
		opts.ProtonNVAPI = game.LaunchProfile.ProtonNVAPI
		opts.DXVKHud = game.LaunchProfile.DXVKHud
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
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	m.mu.Lock()
	cmd, ok := m.processes[gameDir]
	m.mu.Unlock()

	if !ok || cmd == nil || cmd.Process == nil {
		return nil
	}

	slog.Info("Terminating running game process", "dir", gameDir, "pid", cmd.Process.Pid)

	// Send SIGTERM to process group, fallback to SIGKILL
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		time.Sleep(150 * time.Millisecond)
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		_ = cmd.Process.Kill()
	}

	return nil
}

// IsRunning checks if a game process in the given game directory is active.
func (m *Manager) IsRunning(gameDir string) bool {
	if gameDir == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd, ok := m.processes[gameDir]
	return ok && cmd != nil && cmd.Process != nil
}

// RunningGames returns a slice of game directories that are currently running.
func (m *Manager) RunningGames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []string
	for dir := range m.processes {
		list = append(list, dir)
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
