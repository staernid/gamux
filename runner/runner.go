package runner

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

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/gamecfg"
	"github.com/staernid/gamux/lutris"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
)

// Operating Mode Constants
const (
	ModeLutris     = gamecfg.ModeLutris
	ModeStandalone = gamecfg.ModeStandalone
)

// Runner Type Constants
const (
	RunnerAuto   = gamecfg.RunnerAuto
	RunnerUMU    = gamecfg.RunnerUMU
	RunnerWine   = gamecfg.RunnerWine
	RunnerProton = gamecfg.RunnerProton
	RunnerLinux  = gamecfg.RunnerLinux
	RunnerLutris = gamecfg.RunnerLutris
)

// LaunchOptions holds comprehensive execution options for a game.
type LaunchOptions = gamecfg.Profile

// FindUMUBinary searches PATH and standard runtime paths for the umu-run binary.
func FindUMUBinary() (string, error) {
	if bin, err := exec.LookPath("umu-run"); err == nil {
		return bin, nil
	}
	home, _ := os.UserHomeDir()
	var candidates []string
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "umu", "umu-run"),
			filepath.Join(home, ".local", "share", "lutris", "runtime", "umu", "umu-run"),
			filepath.Join(home, ".local", "bin", "umu-run"),
			filepath.Join(home, ".cargo", "bin", "umu-run"),
		)
	}
	candidates = append(candidates,
		"/usr/local/bin/umu-run",
		"/usr/bin/umu-run",
		"/var/lib/flatpak/exports/bin/umu-run",
	)
	for _, c := range candidates {
		if util.FileExists(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("umu-run binary not found in PATH or standard runtime directories (~/.local/share/umu/umu-run)")
}

// GetAvailableRunners inspects system PATH to detect available execution tools.
func GetAvailableRunners() []string {
	var runners []string
	if _, err := FindUMUBinary(); err == nil {
		runners = append(runners, RunnerUMU)
	}
	if _, err := exec.LookPath("wine"); err == nil {
		runners = append(runners, RunnerWine)
	}
	if _, err := exec.LookPath("proton"); err == nil {
		runners = append(runners, RunnerProton)
	}
	if _, err := exec.LookPath("lutris"); err == nil {
		runners = append(runners, RunnerLutris)
	}
	runners = append(runners, RunnerLinux)
	return runners
}

// ResolveEffectiveOptions merges global configuration with specific launch options.
func ResolveEffectiveOptions(cfg *config.Config, opts LaunchOptions) LaunchOptions {
	return gamecfg.ResolveEffective(cfg, &opts)
}

// BuildCommand constructs an *exec.Cmd configured for the specified operating mode and runner.
func BuildCommand(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, error) {
	resolved := ResolveEffectiveOptions(cfg, opts)

	if resolved.OperatingMode == ModeLutris || resolved.Runner == RunnerLutris {
		lutrisBin, err := exec.LookPath("lutris")
		if err != nil {
			return nil, fmt.Errorf("lutris binary not found in PATH: %w", err)
		}
		slug := lutris.Slugify(resolved.Name)
		if slug == "" && resolved.AppID != "" {
			slug = resolved.AppID
		}
		if slug == "" {
			return nil, fmt.Errorf("cannot launch via Lutris: game name or AppID is required to resolve slug")
		}

		slog.Info("Prepared Lutris launch command", "slug", slug)
		cmd := exec.CommandContext(ctx, lutrisBin, fmt.Sprintf("lutris:rungame/%s", slug))
		return cmd, nil
	}

	// Standalone Launcher Mode
	if resolved.ExePath == "" {
		return nil, fmt.Errorf("executable path is required for standalone launch")
	}

	absGameDir, err := filepath.Abs(resolved.GameDir)
	if err != nil {
		return nil, fmt.Errorf("resolve game dir: %w", err)
	}

	targetExe := resolved.ExePath
	if !filepath.IsAbs(targetExe) {
		targetExe = filepath.Join(absGameDir, targetExe)
	}

	// Check if Goldberg loader mode is present for Windows games
	if resolved.Platform != "linux" {
		loaderX64 := filepath.Join(absGameDir, "steamclient_loader_x64.exe")
		loaderX86 := filepath.Join(absGameDir, "steamclient_loader_x86.exe")
		coldIni := filepath.Join(absGameDir, "ColdClientLoader.ini")

		if util.FileExists(coldIni) {
			if resolved.Platform == "win32" && util.FileExists(loaderX86) {
				if !strings.HasSuffix(targetExe, "steamclient_loader_x86.exe") {
					slog.Info("Goldberg loader mode detected; delegating execution to steamclient_loader_x86.exe", "dir", absGameDir)
					targetExe = loaderX86
				}
			} else if util.FileExists(loaderX64) {
				if !strings.HasSuffix(targetExe, "steamclient_loader_x64.exe") {
					slog.Info("Goldberg loader mode detected; delegating execution to steamclient_loader_x64.exe", "dir", absGameDir)
					targetExe = loaderX64
				}
			}
		}
	}

	// Assemble Environment Variables
	envMap := make(map[string]string)
	for _, envStr := range os.Environ() {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Steam / UMU / AppID identification
	if resolved.AppID != "" && resolved.AppID != "0" {
		envMap["SteamAppId"] = resolved.AppID
		envMap["SteamGameId"] = resolved.AppID
	}

	if resolved.Runner == RunnerUMU || resolved.Runner == RunnerAuto || resolved.Runner == "" {
		if _, exists := envMap["GAMEID"]; !exists {
			envMap["GAMEID"] = "umu-default"
		}
		if _, exists := envMap["UMU_ID"]; !exists {
			envMap["UMU_ID"] = "umu-default"
		}
	} else if resolved.AppID != "" && resolved.AppID != "0" {
		if _, exists := envMap["GAMEID"]; !exists {
			envMap["GAMEID"] = resolved.AppID
		}
		if _, exists := envMap["UMU_ID"]; !exists {
			envMap["UMU_ID"] = resolved.AppID
		}
	}

	// Wine prefix
	if resolved.WinePrefix != "" {
		expandedPrefix := util.ExpandPath(resolved.WinePrefix)
		envMap["WINEPREFIX"] = expandedPrefix
		envMap["STEAM_COMPAT_DATA_PATH"] = expandedPrefix
	}

	// Proton path override
	if resolved.ProtonPath != "" {
		envMap["PROTONPATH"] = resolved.ProtonPath
	}

	// Gaming environment toggles
	if resolved.MangoHud != nil && *resolved.MangoHud {
		envMap["MANGOHUD"] = "1"
	}
	if resolved.WineFullscreenFSR != nil && *resolved.WineFullscreenFSR {
		envMap["WINE_FULLSCREEN_FSR"] = "1"
	}
	if resolved.ProtonNVAPI != nil && *resolved.ProtonNVAPI {
		envMap["PROTON_ENABLE_NVAPI"] = "1"
	}
	if resolved.DXVKHud != "" {
		envMap["DXVK_HUD"] = resolved.DXVKHud
	}

	// Wine / Proton / Graphics performance & stability defaults for Windows games
	if resolved.Platform != "linux" || resolved.Runner == RunnerWine || resolved.Runner == RunnerProton || resolved.Runner == RunnerUMU {
		envMap["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
		envMap["WINE_LARGE_ADDRESS_AWARE"] = "1"
		envMap["PROTON_FORCE_LARGE_ADDRESS_AWARE"] = "1"
		envMap["SDL_VIDEO_MINIMIZE_ON_FOCUS_LOSS"] = "0"
		envMap["WINEESYNC"] = "1"
		envMap["WINEFSYNC"] = "1"
		envMap["__GL_SHADER_DISK_CACHE"] = "1"

		if _, exists := envMap["WINEDEBUG"]; !exists {
			envMap["WINEDEBUG"] = "-all"
		}
		if _, exists := envMap["WINEDLLOVERRIDES"]; !exists {
			envMap["WINEDLLOVERRIDES"] = "d3d10core,d3d11,d3d12,d3d12core,d3d8,d3d9,dxgi,nvapi,nvapi64,nvofapi64=n;winemenubuilder="
		}
		if resolved.Runner == RunnerUMU || resolved.Runner == RunnerAuto || resolved.Runner == "" {
			if _, exists := envMap["PROTONPATH"]; !exists && resolved.ProtonPath == "" {
				envMap["PROTONPATH"] = "GE-Proton"
			}
			if _, exists := envMap["PROTON_VERB"]; !exists {
				envMap["PROTON_VERB"] = "waitforexitandrun"
			}
		}
	} else {
		// Linux native loader support
		gbeBase := cfg.GbeDir
		if !filepath.IsAbs(gbeBase) {
			if home, err := os.UserHomeDir(); err == nil {
				gbeBase = filepath.Join(home, gbeBase)
			}
		}
		gbeSO := filepath.Join(gbeBase, "linux_release", "experimental", "x64", "steamclient.so")
		if util.FileExists(gbeSO) {
			existingPreload := envMap["LD_PRELOAD"]
			if existingPreload != "" && !strings.Contains(existingPreload, "steamclient.so") {
				envMap["LD_PRELOAD"] = gbeSO + ":" + existingPreload
			} else if existingPreload == "" {
				envMap["LD_PRELOAD"] = gbeSO
			}
		}
	}

	// Custom environment overrides
	for k, v := range resolved.CustomEnv {
		envMap[k] = v
	}

	var envList []string
	for k, v := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	var exeArgs []string
	if resolved.ExeArgs != "" {
		exeArgs = strings.Fields(resolved.ExeArgs)
	}

	workingDir := resolved.WorkingDir
	if workingDir == "" {
		workingDir = absGameDir
	}

	var cmd *exec.Cmd
	var cmdBinary string
	var cmdArgs []string

	switch resolved.Runner {
	case RunnerLinux:
		cmdBinary = targetExe
		cmdArgs = exeArgs

	case RunnerUMU:
		umuBin, err := FindUMUBinary()
		if err != nil {
			slog.Info("umu-run not found locally, auto-bootstrapping from GitHub release")
			if booted, bErr := tools.NewUMUTool(cfg).EnsureAvailable(ctx); bErr == nil && util.FileExists(booted) {
				umuBin = booted
			} else {
				return nil, fmt.Errorf("runner 'umu' selected: %w (auto-bootstrap failed: %v)", err, bErr)
			}
		}
		cmdBinary = umuBin
		cmdArgs = append([]string{targetExe}, exeArgs...)

	case RunnerProton:
		protonBin, err := exec.LookPath("proton")
		if err != nil {
			if resolved.ProtonPath != "" {
				customProton := filepath.Join(resolved.ProtonPath, "proton")
				if util.FileExists(customProton) {
					protonBin = customProton
				}
			}
		}
		if protonBin == "" {
			return nil, fmt.Errorf("proton binary not found in PATH or configured proton path")
		}
		cmdBinary = protonBin
		cmdArgs = append([]string{"run", targetExe}, exeArgs...)

	case RunnerWine:
		wineBin, err := exec.LookPath("wine")
		if err != nil {
			return nil, fmt.Errorf("wine binary not found in PATH: %w", err)
		}
		cmdBinary = wineBin
		cmdArgs = append([]string{targetExe}, exeArgs...)

	default:
		// Fallback based on platform
		if resolved.Platform == "linux" {
			cmdBinary = targetExe
			cmdArgs = exeArgs
		} else {
			umuBin, err := FindUMUBinary()
			if err != nil {
				slog.Info("umu-run not found locally, auto-bootstrapping from GitHub release")
				if booted, bErr := tools.NewUMUTool(cfg).EnsureAvailable(ctx); bErr == nil && util.FileExists(booted) {
					umuBin = booted
				} else {
					return nil, fmt.Errorf("no suitable runner found for platform %s: %w", resolved.Platform, err)
				}
			}
			cmdBinary = umuBin
			cmdArgs = append([]string{targetExe}, exeArgs...)
		}
	}

	// For Windows runners, apply gamemoderun and mangohud wrappers when available
	if resolved.Platform != "linux" || resolved.Runner != RunnerLinux {
		var wrappers []string
		if resolved.MangoHud != nil && *resolved.MangoHud {
			if mangohudBin, err := exec.LookPath("mangohud"); err == nil {
				wrappers = append(wrappers, mangohudBin)
			}
		}
		if gamemodeBin, err := exec.LookPath("gamemoderun"); err == nil {
			wrappers = append([]string{gamemodeBin}, wrappers...)
		}

		if len(wrappers) > 0 {
			finalArgs := append(wrappers[1:], cmdBinary)
			finalArgs = append(finalArgs, cmdArgs...)
			cmd = exec.CommandContext(ctx, wrappers[0], finalArgs...)
		} else {
			cmd = exec.CommandContext(ctx, cmdBinary, cmdArgs...)
		}
	} else {
		cmd = exec.CommandContext(ctx, cmdBinary, cmdArgs...)
	}

	cmd.Dir = workingDir
	cmd.Env = envList
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	slog.Info("Constructed launch command",
		"mode", resolved.OperatingMode,
		"runner", resolved.Runner,
		"targetExe", targetExe,
		"workingDir", workingDir,
	)

	return cmd, nil
}

// Launch starts a game asynchronously and returns the active *exec.Cmd process handle.
func Launch(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, error) {
	cmd, err := BuildCommand(ctx, cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("build launch command: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start game process: %w", err)
	}

	slog.Info("Game launched successfully", "pid", cmd.Process.Pid, "name", opts.Name)
	return cmd, nil
}

// Run executes a game synchronously, waiting for it to complete.
func Run(ctx context.Context, cfg *config.Config, opts LaunchOptions) error {
	cmd, err := BuildCommand(ctx, cfg, opts)
	if err != nil {
		return fmt.Errorf("build launch command: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("game execution failed: %w", err)
	}

	return nil
}

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
	listeners []func(ProcessEvent)
}

// NewManager creates an initialized process Manager.
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*exec.Cmd),
	}
}

var defaultManager = NewManager()

// DefaultManager returns the shared singleton process manager.
func DefaultManager() *Manager {
	return defaultManager
}

// Start launches a game, tracks the process, and dispatches status events upon start and termination.
func (m *Manager) Start(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, error) {
	cmd, err := Launch(ctx, cfg, opts)
	if err != nil {
		return nil, err
	}

	gameDir := opts.GameDir
	if gameDir != "" && cmd != nil && cmd.Process != nil {
		m.mu.Lock()
		m.processes[gameDir] = cmd
		m.mu.Unlock()

		m.emit(ProcessEvent{
			GameDir:   gameDir,
			IsRunning: true,
			PID:       cmd.Process.Pid,
		})

		go func(gDir string, pCmd *exec.Cmd) {
			_ = pCmd.Wait()
			m.mu.Lock()
			delete(m.processes, gDir)
			m.mu.Unlock()

			m.emit(ProcessEvent{
				GameDir:   gDir,
				IsRunning: false,
				PID:       0,
			})
		}(gameDir, cmd)
	}

	return cmd, nil
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

// GetLaunchProfile reads custom launch configurations from <gameDir>/steam_settings/launch_profile.json.
func GetLaunchProfile(gameDir string) (*LaunchOptions, error) {
	return gamecfg.Load(gameDir)
}

// SaveLaunchProfile writes custom launch configurations to <gameDir>/steam_settings/launch_profile.json.
func SaveLaunchProfile(gameDir string, opts LaunchOptions) error {
	opts.GameDir = gameDir
	return gamecfg.Save(&opts)
}

// ResetLaunchProfile removes custom launch configurations from <gameDir>/steam_settings/launch_profile.json.
func ResetLaunchProfile(gameDir string) error {
	return gamecfg.Reset(gameDir)
}
