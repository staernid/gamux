package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
)

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
			res.OperatingMode = ModeLoader
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
	} else if resolved.OperatingMode == ModeLoader || resolved.OperatingMode == "loader" {
		// Linux native loader support (only when ModeLoader is active)
		gbeBase := cfg.GbeDir
		if !filepath.IsAbs(gbeBase) {
			if home, err := os.UserHomeDir(); err == nil {
				gbeBase = filepath.Join(home, gbeBase)
			}
		}
		archSubdir := "x64"
		if resolved.Platform == "linux32" {
			archSubdir = "x86"
		}
		gbeSO := filepath.Join(gbeBase, "linux_release", "experimental", archSubdir, "steamclient.so")
		if !util.FileExists(gbeSO) && archSubdir == "x86" {
			fallbackSO := filepath.Join(gbeBase, "linux_release", "experimental", "x32", "steamclient.so")
			if util.FileExists(fallbackSO) {
				gbeSO = fallbackSO
			}
		}
		if !util.FileExists(gbeSO) {
			gbeSO = filepath.Join(gbeBase, "linux_release", "experimental", "x64", "steamclient.so")
		}
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
