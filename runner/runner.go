package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/events"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
)

// Operating Mode Constants
const (
	ModeStandalone = string(domain.ModeStandalone)
	ModeLoader     = string(domain.ModeLoader)
	ModePortable   = string(domain.ModePortable)
	ModeNative     = string(domain.ModeNative)
)

// Runner Type Constants
const (
	RunnerAuto   = domain.RunnerAuto
	RunnerUMU    = domain.RunnerUMU
	RunnerWine   = domain.RunnerWine
	RunnerProton = domain.RunnerProton
	RunnerLinux  = domain.RunnerLinux
)

// LaunchProfile is the canonical launch configuration model from domain.
type LaunchProfile = domain.LaunchProfile

// LaunchOptions holds comprehensive execution options for a game (alias for LaunchProfile).
type LaunchOptions = domain.LaunchProfile

// isValidExecutable checks if a file exists, is non-empty, and is not a corrupted ar archive (!<arch>).
func isValidExecutable(path string) bool {
	if !util.FileExists(path) {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, _ := f.Read(buf)
	if n >= 7 && strings.HasPrefix(string(buf[:n]), "!<arch>") {
		return false
	}
	return true
}

// FindUMUBinary searches PATH and standard runtime paths for the umu-run binary.
func FindUMUBinary() (string, error) {
	if bin, err := exec.LookPath("umu-run"); err == nil && isValidExecutable(bin) {
		return bin, nil
	}
	home, _ := os.UserHomeDir()
	var candidates []string
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "umu", "umu-run"),
			filepath.Join(home, ".local", "share", "umu", "umu", "umu-run"),
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
		if isValidExecutable(c) {
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
	runners = append(runners, RunnerLinux)
	return runners
}

// detectBinaryPlatform inspects executable magic bytes to determine platform (ELF vs PE).
func detectBinaryPlatform(exePath string) string {
	if !util.FileExists(exePath) {
		return ""
	}
	f, err := os.Open(exePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	header := make([]byte, 4)
	if n, err := f.Read(header); err != nil || n < 2 {
		return ""
	}

	if len(header) >= 4 && string(header[:4]) == "\x7fELF" {
		return "linux"
	}
	if string(header[:2]) == "MZ" {
		return "win64"
	}
	return ""
}

// findCandidateExecutable searches gameDir for the most likely primary executable.
func findCandidateExecutable(gameDir string) string {
	if !util.FileExists(gameDir) {
		return ""
	}
	var candidates []string
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		nameLower := strings.ToLower(d.Name())
		if strings.HasSuffix(nameLower, ".exe") || (d.Type().IsRegular() && !strings.Contains(d.Name(), ".")) {
			if strings.Contains(nameLower, "steamclient_loader") ||
				strings.Contains(nameLower, "coldclientloader") ||
				strings.Contains(nameLower, "unitycrashhandler") ||
				strings.Contains(nameLower, "crashreport") ||
				strings.Contains(nameLower, "unins000") ||
				strings.Contains(nameLower, "setup") ||
				strings.Contains(nameLower, "dxsetup") {
				return nil
			}
			candidates = append(candidates, path)
		}
		return nil
	})
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

// ResolveEffectiveOptions merges global user configuration with game-specific launch options/profile.
func ResolveEffectiveOptions(cfg *config.Config, opts LaunchOptions) LaunchOptions {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	res := opts

	// 1. Cascading Profile Load: if GameDir is provided, load on-disk profile for missing fields
	if res.GameDir != "" {
		if profile, err := LoadLaunchProfile(res.GameDir); err == nil && profile != nil {
			if res.Name == "" && profile.Name != "" {
				res.Name = profile.Name
			}
			if res.AppID == "" && profile.AppID != "" {
				res.AppID = profile.AppID
			}
			if res.Platform == "" && profile.Platform != "" {
				res.Platform = profile.Platform
			}
			if res.ExePath == "" && profile.ExePath != "" {
				res.ExePath = profile.ExePath
			}
			if res.ExeArgs == "" && profile.ExeArgs != "" {
				res.ExeArgs = profile.ExeArgs
			}
			if res.OperatingMode == "" && profile.OperatingMode != "" {
				res.OperatingMode = profile.OperatingMode
			}
			if res.Runner == "" && profile.Runner != "" {
				res.Runner = profile.Runner
			}
			if res.WinePrefix == "" && profile.WinePrefix != "" {
				res.WinePrefix = profile.WinePrefix
			}
			if res.MangoHud == nil && profile.MangoHud != nil {
				res.MangoHud = profile.MangoHud
			}
			if res.WineFullscreenFSR == nil && profile.WineFullscreenFSR != nil {
				res.WineFullscreenFSR = profile.WineFullscreenFSR
			}
			if res.ProtonNVAPI == nil && profile.ProtonNVAPI != nil {
				res.ProtonNVAPI = profile.ProtonNVAPI
			}
			if res.DXVKHud == "" && profile.DXVKHud != "" {
				res.DXVKHud = profile.DXVKHud
			}
			if res.DllOverrides == "" && profile.DllOverrides != "" {
				res.DllOverrides = profile.DllOverrides
			}
			if profile.CustomEnv != nil {
				if res.CustomEnv == nil {
					res.CustomEnv = make(map[string]string)
				}
				for k, v := range profile.CustomEnv {
					if _, exists := res.CustomEnv[k]; !exists {
						res.CustomEnv[k] = v
					}
				}
			}
		}
	}

	// 2. Candidate Discovery Fallback: if ExePath is still empty, scan GameDir
	if res.ExePath == "" && res.GameDir != "" {
		if cand := findCandidateExecutable(res.GameDir); cand != "" {
			res.ExePath = cand
		}
	}

	// 3. Platform Detection Fallback: if Platform is unset, inspect binary header
	if res.Platform == "" && res.ExePath != "" {
		targetPath := res.ExePath
		if !filepath.IsAbs(targetPath) && res.GameDir != "" {
			targetPath = filepath.Join(res.GameDir, targetPath)
		}
		if detected := detectBinaryPlatform(targetPath); detected != "" {
			res.Platform = detected
		}
	}

	if res.OperatingMode == "" {
		if cfg.OperatingMode != "" {
			res.OperatingMode = cfg.OperatingMode
		} else {
			res.OperatingMode = ModeStandalone
		}
	}

	if res.Runner == "" || res.Runner == RunnerAuto {
		if res.Platform == "linux" {
			res.Runner = RunnerLinux
		} else if cfg.DefaultRunner != "" {
			res.Runner = cfg.DefaultRunner
		} else {
			res.Runner = RunnerUMU
		}
	}

	if res.WinePrefix == "" && res.Runner != RunnerLinux {
		if cfg.WinePrefix != "" {
			res.WinePrefix = cfg.WinePrefix
		}
	}

	if res.MangoHud == nil && cfg.MangoHud {
		res.MangoHud = util.Ptr(cfg.MangoHud)
	}
	if res.WineFullscreenFSR == nil && cfg.WineFullscreenFSR {
		res.WineFullscreenFSR = util.Ptr(cfg.WineFullscreenFSR)
	}
	if res.ProtonNVAPI == nil && cfg.ProtonNVAPI {
		res.ProtonNVAPI = util.Ptr(cfg.ProtonNVAPI)
	}
	if res.DXVKHud == "" && cfg.DXVKHud != "" {
		res.DXVKHud = cfg.DXVKHud
	}

	if res.WorkingDir == "" && res.GameDir != "" {
		res.WorkingDir = res.GameDir
	}
	if res.ProtonPath == "" && cfg.ProtonPath != "" {
		res.ProtonPath = cfg.ProtonPath
	}

	// Merge Environment Variables (Global < Custom Profile)
	mergedEnv := make(map[string]string)
	if cfg.GlobalEnv != nil {
		for k, v := range cfg.GlobalEnv {
			mergedEnv[k] = v
		}
	}
	if res.CustomEnv != nil {
		for k, v := range res.CustomEnv {
			mergedEnv[k] = v
		}
	}
	res.CustomEnv = mergedEnv

	return res
}

// BuildCommand constructs an *exec.Cmd configured for the specified operating mode and runner.
func BuildCommand(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	resolved := ResolveEffectiveOptions(cfg, opts)

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

		// Construct and merge WINEDLLOVERRIDES
		baseOverrides := "d3d10core,d3d11,d3d12,d3d12core,d3d8,d3d9,dxgi,nvapi,nvapi64,nvofapi64=n;winemenubuilder="
		var overridesParts []string

		if resolved.DllOverrides != "" {
			overridesParts = append(overridesParts, strings.TrimSpace(resolved.DllOverrides))
		}

		// If in loader or emulator mode, automatically redirect Steam client libraries through Wine DLL overrides
		if resolved.OperatingMode == ModeLoader || resolved.OperatingMode == "loader" {
			overridesParts = append(overridesParts, "steamclient,steamclient64=n,b")
		}

		if customOverrides, ok := envMap["WINEDLLOVERRIDES"]; ok && customOverrides != "" {
			overridesParts = append(overridesParts, customOverrides)
		}

		if len(overridesParts) == 0 {
			envMap["WINEDLLOVERRIDES"] = baseOverrides
		} else {
			merged := strings.Join(overridesParts, ";")
			if !strings.Contains(merged, "winemenubuilder=") {
				merged = baseOverrides + ";" + merged
			}
			envMap["WINEDLLOVERRIDES"] = merged
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
		if targetExe != "" && util.FileExists(targetExe) {
			workingDir = filepath.Dir(targetExe)
		} else {
			workingDir = absGameDir
		}
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

// ResolveLogPath determines the log file location for a game.
func ResolveLogPath(gameDir, gameName, appID string) string {
	slug := util.Slugify(gameName)
	if slug == "" {
		if appID != "" {
			slug = appID
		} else if gameDir != "" {
			slug = util.Slugify(filepath.Base(gameDir))
		}
	}
	if slug == "" {
		slug = "game"
	}

	if gameDir != "" {
		settingsDir := filepath.Join(gameDir, "steam_settings")
		_ = os.MkdirAll(settingsDir, 0755)
		return filepath.Join(settingsDir, "last_run.log")
	}

	home, _ := os.UserHomeDir()
	logsDir := filepath.Join(home, ".local", "share", "gamux", "logs")
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, slug+".log")
}

func createLogFile(logPath string, resolved LaunchOptions, cmd *exec.Cmd) (*os.File, error) {
	if logPath == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	header := fmt.Sprintf("=== Gamux Launch Log ===\nTime: %s\nGame: %s\nAppID: %s\nPlatform: %s\nMode: %s\nRunner: %s\nExecutable: %s\nWorkingDir: %s\nCommand: %s\n========================\n\n",
		time.Now().Format(time.RFC1123),
		resolved.Name,
		resolved.AppID,
		resolved.Platform,
		resolved.OperatingMode,
		resolved.Runner,
		resolved.ExePath,
		cmd.Dir,
		cmd.String(),
	)
	_, _ = f.WriteString(header)
	_ = f.Sync()
	return f, nil
}


// GetLog reads the last or current execution log for a game.
func GetLog(gameDir, gameName, appID string) (string, error) {
	logPath := ResolveLogPath(gameDir, gameName, appID)
	if !util.FileExists(logPath) {
		slug := util.Slugify(gameName)
		if slug == "" && appID != "" {
			slug = appID
		}
		if slug != "" {
			home, _ := os.UserHomeDir()
			altPath := filepath.Join(home, ".local", "share", "gamux", "logs", slug+".log")
			if util.FileExists(altPath) {
				logPath = altPath
			}
		}
	}

	if !util.FileExists(logPath) {
		return "", fmt.Errorf("no launch log available for this game yet")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", fmt.Errorf("read game log: %w", err)
	}

	if len(data) > 512*1024 {
		data = data[len(data)-512*1024:]
	}

	return string(data), nil
}

// Launch starts a game asynchronously and returns the active *exec.Cmd process handle.
func Launch(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, *os.File, error) {
	resolved := ResolveEffectiveOptions(cfg, opts)
	cmd, err := BuildCommand(ctx, cfg, resolved)
	if err != nil {
		return nil, nil, fmt.Errorf("build launch command: %w", err)
	}

	logPath := ResolveLogPath(resolved.GameDir, resolved.Name, resolved.AppID)
	logFile, lErr := createLogFile(logPath, resolved, cmd)
	if lErr == nil && logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_, _ = logFile.WriteString(fmt.Sprintf("\n[Failed to start process: %v]\n", err))
			_ = logFile.Close()
		}
		return nil, nil, fmt.Errorf("start game process: %w", err)
	}

	slog.Info("Game launched successfully", "pid", cmd.Process.Pid, "name", opts.Name, "log", logPath)
	return cmd, logFile, nil
}

// Run executes a game synchronously, waiting for it to complete.
func Run(ctx context.Context, cfg *config.Config, opts LaunchOptions) error {
	resolved := ResolveEffectiveOptions(cfg, opts)
	cmd, err := BuildCommand(ctx, cfg, resolved)
	if err != nil {
		return fmt.Errorf("build launch command: %w", err)
	}

	logPath := ResolveLogPath(resolved.GameDir, resolved.Name, resolved.AppID)
	logFile, _ := createLogFile(logPath, resolved, cmd)
	if logFile != nil {
		defer logFile.Close()
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

		if b != nil {
			b.Publish(events.NewGameLaunched(gameDir, opts.Name, opts.AppID, cmd.Process.Pid))
		}

		go func(gDir string, pCmd *exec.Cmd, lFile *os.File, eb *events.Bus, name, appID string) {
			waitErr := pCmd.Wait()
			if lFile != nil {
				exitMsg := fmt.Sprintf("\n\n=== Process Finished ===\nTime: %s\nExit State: %v\n", time.Now().Format(time.RFC1123), waitErr)
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
				eb.Publish(events.NewGameExited(gDir, name, appID, waitErr))
			}
		}(gameDir, cmd, logFile, b, opts.Name, opts.AppID)
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
		ExePath:       game.ExePath,
		ExeArgs:       game.ExeArgs,
		Platform:      game.Platform,
		OperatingMode: game.OperatingMode,
		Runner:        game.Runner,
		WinePrefix:    game.WinePrefix,
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
		ExePath:       game.ExePath,
		ExeArgs:       game.ExeArgs,
		Platform:      game.Platform,
		OperatingMode: game.OperatingMode,
		Runner:        game.Runner,
		WinePrefix:    game.WinePrefix,
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

// LoadLaunchProfile retrieves the active launch profile from <gameDir>/steam_settings/launch_profile.json,
// falling back to inspecting on-disk emulator metadata and directory heuristics if absent.
func LoadLaunchProfile(gameDir string) (*LaunchProfile, error) {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return nil, fmt.Errorf("game directory cannot be empty")
	}

	profilePath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(profilePath) {
		data, err := os.ReadFile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("read launch profile: %w", err)
		}
		var p LaunchProfile
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("unmarshal launch profile: %w", err)
		}
		if p.GameDir == "" {
			p.GameDir = gameDir
		}
		hydrateFallbacks(&p, gameDir)
		return &p, nil
	}

	// Fallback to checking nested steam_settings directory
	var foundPath string
	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "launch_profile.json" && strings.Contains(path, "steam_settings") {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})

	if foundPath != "" {
		data, err := os.ReadFile(foundPath)
		if err == nil {
			var p LaunchProfile
			if err := json.Unmarshal(data, &p); err == nil {
				if p.GameDir == "" {
					p.GameDir = gameDir
				}
				hydrateFallbacks(&p, gameDir)
				return &p, nil
			}
		}
	}

	// No launch_profile.json on disk: build initial baseline from on-disk artifacts
	p := &LaunchProfile{
		GameDir: gameDir,
	}
	hydrateFallbacks(p, gameDir)
	return p, nil
}

// GetLaunchProfile reads custom launch configurations from <gameDir>/steam_settings/launch_profile.json (alias for LoadLaunchProfile).
func GetLaunchProfile(gameDir string) (*LaunchProfile, error) {
	return LoadLaunchProfile(gameDir)
}

// hydrateFallbacks checks on-disk GBE/ACF metadata files if fields are missing.
func hydrateFallbacks(p *LaunchProfile, gameDir string) {
	settingsDir := filepath.Join(gameDir, "steam_settings")

	// Read steam_appid.txt
	if p.AppID == "" {
		appIDFile := filepath.Join(settingsDir, "steam_appid.txt")
		if !util.FileExists(appIDFile) {
			appIDFile = filepath.Join(gameDir, "steam_appid.txt")
		}
		if data, err := os.ReadFile(appIDFile); err == nil {
			p.AppID = strings.TrimSpace(string(data))
		}
	}

	// Read force_language.txt
	if p.Language == "" {
		langFile := filepath.Join(settingsDir, "force_language.txt")
		if data, err := os.ReadFile(langFile); err == nil {
			p.Language = strings.TrimSpace(string(data))
		}
	}

	// Read ColdClientLoader.ini Exe target if ExePath is still empty
	if p.ExePath == "" {
		iniPath := filepath.Join(gameDir, "ColdClientLoader.ini")
		if data, err := os.ReadFile(iniPath); err == nil {
			reIniExe := regexp.MustCompile(`(?i)^\s*Exe\s*=\s*(.+)`)
			for _, line := range strings.Split(string(data), "\n") {
				if m := reIniExe.FindStringSubmatch(strings.TrimSpace(line)); len(m) > 1 {
					realExe := strings.TrimSpace(m[1])
					if !filepath.IsAbs(realExe) {
						realExe = filepath.Join(gameDir, realExe)
					}
					if util.FileExists(realExe) {
						p.ExePath = realExe
						break
					}
				}
			}
		}
	}

	if p.Store == "" {
		if p.AppID != "" && p.AppID != "0" {
			p.Store = domain.StoreSteam
		} else if p.GOGID != "" {
			p.Store = domain.StoreGOG
		}
	}
}

// SaveLaunchProfile writes custom launch configurations to <gameDir>/steam_settings/launch_profile.json atomically.
func SaveLaunchProfile(gameDir string, opts LaunchProfile) error {
	opts.GameDir = gameDir
	return SaveProfile(&opts)
}

// SaveProfile persists the profile to <gameDir>/steam_settings/launch_profile.json atomically.
func SaveProfile(p *LaunchProfile) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	p.GameDir = util.CanonicalPath(p.GameDir)
	if p.GameDir == "" {
		return fmt.Errorf("profile GameDir cannot be empty")
	}

	settingsDir := filepath.Join(p.GameDir, "steam_settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("create steam_settings directory: %w", err)
	}

	profilePath := filepath.Join(settingsDir, "launch_profile.json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal launch profile: %w", err)
	}

	return os.WriteFile(profilePath, data, 0644)
}

// ResetLaunchProfile removes custom launch configurations from <gameDir>/steam_settings/launch_profile.json.
func ResetLaunchProfile(gameDir string) error {
	gameDir = util.CanonicalPath(gameDir)
	if gameDir == "" {
		return fmt.Errorf("game directory cannot be empty")
	}

	profilePath := filepath.Join(gameDir, "steam_settings", "launch_profile.json")
	if util.FileExists(profilePath) {
		if err := os.Remove(profilePath); err != nil {
			return fmt.Errorf("remove launch profile: %w", err)
		}
	}

	_ = filepath.WalkDir(gameDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "launch_profile.json" && strings.Contains(path, "steam_settings") {
			_ = os.Remove(path)
		}
		return nil
	})

	return nil
}

// SyncLaunchProfile updates on-disk Goldberg settings to match the LaunchProfile.
func SyncLaunchProfile(ctx context.Context, p *LaunchProfile, cfg *config.Config) error {
	if p == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// 1. Persist the launch profile JSON
	if err := SaveProfile(p); err != nil {
		return fmt.Errorf("save profile: %w", err)
	}

	settingsDir := filepath.Join(p.GameDir, "steam_settings")
	_ = os.MkdirAll(settingsDir, 0755)

	// 2. Sync AppID to steam_appid.txt if set
	if p.AppID != "" && p.AppID != "0" {
		_ = os.WriteFile(filepath.Join(settingsDir, "steam_appid.txt"), []byte(p.AppID), 0644)
		_ = os.WriteFile(filepath.Join(p.GameDir, "steam_appid.txt"), []byte(p.AppID), 0644)
	}

	// 3. Sync Language to force_language.txt
	langFile := filepath.Join(settingsDir, "force_language.txt")
	if p.Language != "" && p.Language != "auto" {
		_ = os.WriteFile(langFile, []byte(strings.ToLower(p.Language)+"\n"), 0644)
	} else {
		_ = os.Remove(langFile)
	}

	// 4. Sync ColdClientLoader.ini Exe line if loader mode is used
	coldClientIni := filepath.Join(p.GameDir, "ColdClientLoader.ini")
	if util.FileExists(coldClientIni) && p.ExePath != "" {
		relExe := p.ExePath
		if filepath.IsAbs(relExe) {
			if rel, err := filepath.Rel(p.GameDir, relExe); err == nil {
				relExe = rel
			}
		}
		if iniData, err := os.ReadFile(coldClientIni); err == nil {
			reExe := regexp.MustCompile(`(?i)(Exe\s*=\s*)(.+)`)
			var updated string
			if reExe.Match(iniData) {
				updated = reExe.ReplaceAllString(string(iniData), fmt.Sprintf("${1}%s", relExe))
			} else {
				updated = string(iniData) + fmt.Sprintf("\nExe = %s\n", relExe)
			}
			_ = os.WriteFile(coldClientIni, []byte(updated), 0644)
		}
	}

	return nil
}
