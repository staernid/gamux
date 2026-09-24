package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/profiles"
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
		if profile, err := compat.LoadProfile(res.GameDir); err == nil && profile != nil {
			res.MergeFrom(profile)
		}
	}

	// 2. Candidate Discovery Fallback: if ExePath is still empty, scan GameDir
	if res.ExePath == "" && res.GameDir != "" {
		if cand := findCandidateExecutable(res.GameDir, res.Name); cand != "" {
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

	if res.Gamescope == nil && cfg.Gamescope {
		res.Gamescope = util.Ptr(cfg.Gamescope)
	}
	if res.GamescopeWidth <= 0 && cfg.GamescopeWidth > 0 {
		res.GamescopeWidth = cfg.GamescopeWidth
	}
	if res.GamescopeHeight <= 0 && cfg.GamescopeHeight > 0 {
		res.GamescopeHeight = cfg.GamescopeHeight
	}
	if res.GamescopeFPS <= 0 && cfg.GamescopeFPS > 0 {
		res.GamescopeFPS = cfg.GamescopeFPS
	}

	if res.DisableOverlay == nil && cfg.DisableOverlay {
		res.DisableOverlay = util.Ptr(cfg.DisableOverlay)
	}

	if res.OfflineMode == nil && cfg.OfflineMode {
		res.OfflineMode = util.Ptr(cfg.OfflineMode)
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

	if res.WorkingDir == "" {
		if res.ExePath != "" {
			if filepath.IsAbs(res.ExePath) {
				res.WorkingDir = filepath.Dir(res.ExePath)
			} else if res.GameDir != "" {
				res.WorkingDir = filepath.Dir(filepath.Join(res.GameDir, res.ExePath))
			}
		}
		if res.WorkingDir == "" && res.GameDir != "" {
			res.WorkingDir = res.GameDir
		}
	}

	return res
}

// LaunchSpec contains the complete resolved execution specification for a game.
type LaunchSpec struct {
	GameDir       string            // Absolute path to game directory
	WorkingDir    string            // Working directory for execution
	TargetExe     string            // Absolute path to game executable
	Env           map[string]string // Explicit Gamux/Game environment variables to set
	Args          []string          // Command line arguments including executable: [binary/wrapper, ...args]
	OperatingMode string            // Active operating mode (loader, portable, none, etc.)
	Platform      string            // Platform identifier (linux, win64, etc.)
	GbeDir        string            // GBE directory path
}

// BuildLaunchSpec resolves all configuration, environment variables, runners, and wrappers into a LaunchSpec.
func BuildLaunchSpec(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*LaunchSpec, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	resolved := ResolveEffectiveOptions(cfg, opts)

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

	// Assemble explicit Gamux/Game environment variables
	gameEnv := make(map[string]string)

	// Steam / UMU / AppID identification & Protonfixes mapping
	if resolved.AppID != "" && resolved.AppID != "0" {
		gameEnv["SteamAppId"] = resolved.AppID
		gameEnv["SteamGameId"] = resolved.AppID
		gameEnv["GAMEID"] = resolved.AppID
		gameEnv["UMU_ID"] = resolved.AppID
		gameEnv["STORE"] = "steam"
	} else {
		gameEnv["GAMEID"] = "umu-default"
		gameEnv["UMU_ID"] = "umu-default"
		gameEnv["STORE"] = "none"
	}
	gameEnv["SteamAppUser"] = "client_player"
	gameEnv["SteamUser"] = "client_player"
	gameEnv["SteamClientLaunch"] = "1"
	gameEnv["SteamEnv"] = "1"

	// Profile sandbox isolation (secondary profiles use isolated home & XDG directories)
	for k, v := range profiles.ResolveProfileEnv(resolved.ProfileID) {
		gameEnv[k] = v
	}
	seatID := compat.SeatIDFromProfileID(resolved.ProfileID)

	// Wine / Proton prefix isolation with central Syncthing user data symlinking
	if resolved.Platform != "linux" || resolved.Runner == RunnerWine || resolved.Runner == RunnerProton || resolved.Runner == RunnerUMU {
		numAppID, _ := util.ParseAppID(resolved.AppID)
		gameStub := &domain.Game{
			GameRecord: domain.GameRecord{
				Name:         resolved.Name,
				AppID:        resolved.AppID,
				NumericAppID: numAppID,
				Platform:     resolved.Platform,
				GameDir:      resolved.GameDir,
			},
			LaunchProfile: &domain.LaunchProfile{
				OperatingMode: resolved.OperatingMode,
			},
		}
		prefixDir, pErr := compat.EnsureSeatPrefix(ctx, cfg, gameStub, seatID, resolved.WinePrefix)
		if pErr == nil && prefixDir != "" {
			gameEnv["WINEPREFIX"] = prefixDir
			gameEnv["STEAM_COMPAT_DATA_PATH"] = prefixDir
		}
	}

	// Proton path override
	if resolved.ProtonPath != "" {
		gameEnv["PROTONPATH"] = resolved.ProtonPath
	}

	// Gaming environment toggles
	if resolved.MangoHud != nil && *resolved.MangoHud {
		gameEnv["MANGOHUD"] = "1"
	}

	// Wine / Proton / Graphics performance & stability defaults for Windows games
	if resolved.Platform != "linux" || resolved.Runner == RunnerWine || resolved.Runner == RunnerProton || resolved.Runner == RunnerUMU {
		gameEnv["PROTON_DISABLE_LSTEAMCLIENT"] = "1"
		gameEnv["WINE_LARGE_ADDRESS_AWARE"] = "1"
		gameEnv["PROTON_FORCE_LARGE_ADDRESS_AWARE"] = "1"
		gameEnv["SDL_VIDEO_MINIMIZE_ON_FOCUS_LOSS"] = "0"
		gameEnv["WINEESYNC"] = "1"
		gameEnv["WINEFSYNC"] = "1"
		gameEnv["__GL_SHADER_DISK_CACHE"] = "1"
		gameEnv["WINEDEBUG"] = "-all"

		// Construct and merge WINEDLLOVERRIDES
		baseOverrides := "d3d10core,d3d11,d3d12,d3d12core,d3d8,d3d9,dxgi,nvapi,nvapi64,nvofapi64=n;winemenubuilder="
		var overridesParts []string

		if resolved.DllOverrides != "" {
			overridesParts = append(overridesParts, strings.TrimSpace(resolved.DllOverrides))
		}

		// If in loader or emulator mode, automatically redirect Steam client libraries through Wine DLL overrides
		resolvedMode := domain.NormalizeOperatingMode(resolved.OperatingMode)
		if resolvedMode == ModeLoader {
			overridesParts = append(overridesParts, "steamclient,steamclient64,GameOverlayRenderer,GameOverlayRenderer64=n,b")
		}

		if len(overridesParts) == 0 {
			gameEnv["WINEDLLOVERRIDES"] = baseOverrides
		} else {
			merged := strings.Join(overridesParts, ";")
			if !strings.Contains(merged, "winemenubuilder=") {
				merged = baseOverrides + ";" + merged
			}
			gameEnv["WINEDLLOVERRIDES"] = merged
		}
		if resolved.Runner == RunnerUMU || resolved.Runner == RunnerAuto || resolved.Runner == "" {
			if resolved.ProtonPath == "" {
				if latestProton := FindLatestInstalledProton(); latestProton != "" {
					gameEnv["PROTONPATH"] = latestProton
				} else {
					gameEnv["PROTONPATH"] = "GE-Proton"
				}
			}
			gameEnv["PROTON_VERB"] = "waitforexitandrun"
		}
	} else if domain.NormalizeOperatingMode(resolved.OperatingMode) == ModeLoader {
		// Linux native loader support
		// steamclient.so requires libX11.so.6 preloaded so X11 symbols resolve at runtime for the overlay
		if existingPreload, ok := gameEnv["LD_PRELOAD"]; ok && existingPreload != "" {
			if !strings.Contains(existingPreload, "libX11.so.6") {
				gameEnv["LD_PRELOAD"] = "libX11.so.6:" + existingPreload
			}
		} else {
			gameEnv["LD_PRELOAD"] = "libX11.so.6"
		}

		if resolved.AppID != "" && resolved.AppID != "0" {
			gameEnv["SteamAppId"] = resolved.AppID
			gameEnv["SteamGameId"] = resolved.AppID
		}
		gameEnv["SteamAppUser"] = "client_player"
		gameEnv["SteamUser"] = "client_player"
		gameEnv["SteamClientLaunch"] = "1"
		gameEnv["SteamEnv"] = "1"
	}

	// Custom environment overrides (global + profile CustomEnv)
	for k, v := range resolved.CustomEnv {
		if k == "LD_PRELOAD" && domain.NormalizeOperatingMode(resolved.OperatingMode) == ModeLoader {
			if !strings.Contains(v, "libX11.so.6") {
				v = "libX11.so.6:" + v
			}
		}
		gameEnv[k] = v
	}

	var exeArgs []string
	if resolved.ExeArgs != "" {
		exeArgs = util.SplitArgs(resolved.ExeArgs)
	}

	workingDir := resolved.WorkingDir
	if workingDir == "" {
		if targetExe != "" && util.FileExists(targetExe) {
			workingDir = filepath.Dir(targetExe)
		} else {
			workingDir = absGameDir
		}
	}

	// Use polymorphic Runner adapter to resolve binary and args
	runnerAdapter := ResolveRunner(resolved.Runner, resolved.Platform)
	cmdBinary, cmdArgs, err := runnerAdapter.BuildArgs(ctx, cfg, targetExe, exeArgs, resolved)
	if err != nil {
		return nil, fmt.Errorf("build runner args (%s): %w", runnerAdapter.Name(), err)
	}

	finalArgs := append([]string{cmdBinary}, cmdArgs...)

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
			finalArgs = append(wrappers, finalArgs...)
		}
	}

	// Apply gamescope wrapper if explicitly enabled (default is off)
	if resolved.Gamescope != nil && *resolved.Gamescope {
		if gamescopeBin, err := exec.LookPath("gamescope"); err == nil {
			gw := resolved.GamescopeWidth
			gh := resolved.GamescopeHeight
			gfps := resolved.GamescopeFPS
			if gw <= 0 || gh <= 0 {
				disp := util.DetectPrimaryDisplay()
				gw = disp.Width
				gh = disp.Height
				if gfps <= 0 {
					gfps = disp.RefreshRate
				}
			}
			var gsArgs []string
			gsArgs = append(gsArgs, gamescopeBin, "-w", fmt.Sprintf("%d", gw), "-h", fmt.Sprintf("%d", gh), "-W", fmt.Sprintf("%d", gw), "-H", fmt.Sprintf("%d", gh))
			if gfps > 0 {
				gsArgs = append(gsArgs, "-r", fmt.Sprintf("%d", gfps))
			}
			gsArgs = append(gsArgs, "-f", "--")
			finalArgs = append(gsArgs, finalArgs...)
		}
	}

	return &LaunchSpec{
		GameDir:       absGameDir,
		WorkingDir:    workingDir,
		TargetExe:     targetExe,
		Env:           gameEnv,
		Args:          finalArgs,
		OperatingMode: domain.NormalizeOperatingMode(resolved.OperatingMode),
		Platform:      resolved.Platform,
		GbeDir:        cfg.GbeDir,
	}, nil
}

// BuildCommand constructs an *exec.Cmd configured for the specified operating mode and runner.
func BuildCommand(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, error) {
	spec, err := BuildLaunchSpec(ctx, cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("build launch spec: %w", err)
	}

	// Live environment: overlay spec.Env over os.Environ()
	envMap := make(map[string]string)
	for _, envStr := range os.Environ() {
		parts := strings.SplitN(envStr, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range spec.Env {
		if k == "LD_PRELOAD" && envMap["LD_PRELOAD"] != "" && !strings.Contains(envMap["LD_PRELOAD"], v) {
			envMap["LD_PRELOAD"] = v + ":" + envMap["LD_PRELOAD"]
		} else {
			envMap[k] = v
		}
	}

	var envList []string
	for k, v := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, spec.Args[0], spec.Args[1:]...)
	cmd.Dir = spec.WorkingDir
	cmd.Env = envList
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	slog.Info("Constructed launch command",
		"targetExe", spec.TargetExe,
		"workingDir", spec.WorkingDir,
		"args", spec.Args,
	)

	return cmd, nil
}

// GenerateLaunchScriptContent serializes a LaunchSpec into a standalone portable POSIX bash script.
func GenerateLaunchScriptContent(spec *LaunchSpec) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# Generated by gamux - Standalone Portable Game Launcher\n")
	sb.WriteString("set -e\n\n")

	sb.WriteString("# Resolve game directory relative to this script (.gamux/launch.sh)\n")
	sb.WriteString(`GAME_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"` + "\n")

	// Resolve working directory relative to GAME_DIR if applicable
	if spec.WorkingDir != "" {
		if spec.WorkingDir == spec.GameDir {
			sb.WriteString(`cd "$GAME_DIR"` + "\n\n")
		} else if strings.HasPrefix(spec.WorkingDir, spec.GameDir+"/") {
			rel := strings.TrimPrefix(spec.WorkingDir, spec.GameDir+"/")
			sb.WriteString(fmt.Sprintf(`cd "$GAME_DIR/%s"`+"\n\n", rel))
		} else {
			sb.WriteString(fmt.Sprintf(`cd %s`+"\n\n", formatBashArg(spec.WorkingDir, spec.GameDir)))
		}
	} else {
		sb.WriteString(`cd "$GAME_DIR"` + "\n\n")
	}

	// Explicit environment variables (sorted alphabetically for deterministic output)
	keys := make([]string, 0, len(spec.Env))
	for k := range spec.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := spec.Env[k]
		sb.WriteString(fmt.Sprintf("export %s=%s\n", k, formatBashEnvVal(v, spec.GameDir)))
	}
	sb.WriteString("\n")

	// Linux native loader mode Steam environment setup & teardown
	isLinuxLoader := spec.Platform == "linux" && spec.OperatingMode == ModeLoader
	if isLinuxLoader {
		gbeBase := spec.GbeDir
		if gbeBase == "" {
			gbeBase = config.DefaultConfig().GbeDir
		}
		if !filepath.IsAbs(gbeBase) {
			gbeBase = "$HOME/" + strings.TrimPrefix(gbeBase, "~/")
		}

		sb.WriteString("# Goldberg Linux Steam Environment Setup\n")
		sb.WriteString(`STEAM_BASE="$HOME/.steam"` + "\n")
		sb.WriteString(`mkdir -p "$STEAM_BASE/sdk64" "$STEAM_BASE/sdk32"` + "\n\n")
		sb.WriteString(`cleanup_steam() {` + "\n")
		sb.WriteString(`    rm -f "$STEAM_BASE/steam.pid"` + "\n")
		sb.WriteString(`    rm -f "$STEAM_BASE/sdk64/steam_settings" "$STEAM_BASE/sdk32/steam_settings"` + "\n")
		sb.WriteString(`    rm -f "$STEAM_BASE/sdk64/steamclient.so" "$STEAM_BASE/sdk32/steamclient.so"` + "\n")
		sb.WriteString(`    if [ -f "$STEAM_BASE/steam.pid.orig" ]; then mv -f "$STEAM_BASE/steam.pid.orig" "$STEAM_BASE/steam.pid"; fi` + "\n")
		sb.WriteString(`    if [ -f "$STEAM_BASE/sdk64/steamclient.so.orig" ]; then mv -f "$STEAM_BASE/sdk64/steamclient.so.orig" "$STEAM_BASE/sdk64/steamclient.so"; fi` + "\n")
		sb.WriteString(`    if [ -f "$STEAM_BASE/sdk32/steamclient.so.orig" ]; then mv -f "$STEAM_BASE/sdk32/steamclient.so.orig" "$STEAM_BASE/sdk32/steamclient.so"; fi` + "\n")
		sb.WriteString(`}` + "\n")
		sb.WriteString(`trap cleanup_steam EXIT INT TERM` + "\n\n")
		sb.WriteString(`if [ -f "$STEAM_BASE/steam.pid" ] && [ ! -f "$STEAM_BASE/steam.pid.orig" ]; then mv -f "$STEAM_BASE/steam.pid" "$STEAM_BASE/steam.pid.orig"; fi` + "\n")
		sb.WriteString(`if [ -f "$STEAM_BASE/sdk64/steamclient.so" ] && [ ! -f "$STEAM_BASE/sdk64/steamclient.so.orig" ]; then mv -f "$STEAM_BASE/sdk64/steamclient.so" "$STEAM_BASE/sdk64/steamclient.so.orig"; fi` + "\n")
		sb.WriteString(`if [ -f "$STEAM_BASE/sdk32/steamclient.so" ] && [ ! -f "$STEAM_BASE/sdk32/steamclient.so.orig" ]; then mv -f "$STEAM_BASE/sdk32/steamclient.so" "$STEAM_BASE/sdk32/steamclient.so.orig"; fi` + "\n\n")
		sb.WriteString(`echo $$ > "$STEAM_BASE/steam.pid"` + "\n")
		sb.WriteString(fmt.Sprintf(`GBE_BASE="%s"`+"\n", gbeBase))
		sb.WriteString(`ln -sf "$GBE_BASE/linux_release/experimental/x64/steamclient.so" "$STEAM_BASE/sdk64/steamclient.so"` + "\n")
		sb.WriteString(`if [ -f "$GBE_BASE/linux_release/experimental/x86/steamclient.so" ]; then` + "\n")
		sb.WriteString(`    ln -sf "$GBE_BASE/linux_release/experimental/x86/steamclient.so" "$STEAM_BASE/sdk32/steamclient.so"` + "\n")
		sb.WriteString(`elif [ -f "$GBE_BASE/linux_release/experimental/x32/steamclient.so" ]; then` + "\n")
		sb.WriteString(`    ln -sf "$GBE_BASE/linux_release/experimental/x32/steamclient.so" "$STEAM_BASE/sdk32/steamclient.so"` + "\n")
		sb.WriteString(`fi` + "\n")
		sb.WriteString(`if [ -d "$GAME_DIR/steam_settings" ]; then` + "\n")
		sb.WriteString(`    ln -sf "$GAME_DIR/steam_settings" "$STEAM_BASE/sdk64/steam_settings"` + "\n")
		sb.WriteString(`    ln -sf "$GAME_DIR/steam_settings" "$STEAM_BASE/sdk32/steam_settings"` + "\n")
		sb.WriteString(`fi` + "\n\n")
	}

	// Exec command with pass-through arguments ($@)
	var cmdParts []string
	for _, arg := range spec.Args {
		cmdParts = append(cmdParts, formatBashArg(arg, spec.GameDir))
	}
	cmdParts = append(cmdParts, `"$@"`)
	if isLinuxLoader {
		sb.WriteString(strings.Join(cmdParts, " ") + "\n")
		sb.WriteString("EXIT_CODE=$?\n")
		sb.WriteString("cleanup_steam\n")
		sb.WriteString("exit $EXIT_CODE\n")
	} else {
		sb.WriteString("exec " + strings.Join(cmdParts, " ") + "\n")
	}

	return sb.String()
}

// formatBashEnvVal formats an environment variable value for shell output.
func formatBashEnvVal(val, gameDir string) string {
	if val == "" {
		return `""`
	}
	if gameDir != "" {
		if val == gameDir {
			return `"$GAME_DIR"`
		}
		if strings.HasPrefix(val, gameDir+"/") {
			rel := strings.TrimPrefix(val, gameDir+"/")
			return fmt.Sprintf(`"$GAME_DIR/%s"`, rel)
		}
	}
	escaped := strings.ReplaceAll(val, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, `$`, `\$`)
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	return fmt.Sprintf(`"%s"`, escaped)
}

// formatBashArg formats a command-line argument for shell output.
func formatBashArg(arg, gameDir string) string {
	if arg == "" {
		return `""`
	}
	if gameDir != "" {
		if arg == gameDir {
			return `"$GAME_DIR"`
		}
		if strings.HasPrefix(arg, gameDir+"/") {
			rel := strings.TrimPrefix(arg, gameDir+"/")
			return fmt.Sprintf(`"$GAME_DIR/%s"`, rel)
		}
	}
	if strings.ContainsAny(arg, " \t\n\"'\\$`<>&|;*?~#()[]{}") {
		escaped := strings.ReplaceAll(arg, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, `$`, `\$`)
		escaped = strings.ReplaceAll(escaped, "`", "\\`")
		return fmt.Sprintf(`"%s"`, escaped)
	}
	return arg
}

// GenerateLaunchScript writes .gamux/launch.sh in the game's directory.
func GenerateLaunchScript(ctx context.Context, cfg *config.Config, opts LaunchOptions) error {
	spec, err := BuildLaunchSpec(ctx, cfg, opts)
	if err != nil {
		return fmt.Errorf("build launch spec: %w", err)
	}

	content := GenerateLaunchScriptContent(spec)
	gamuxDir := filepath.Join(spec.GameDir, ".gamux")
	if err := os.MkdirAll(gamuxDir, 0755); err != nil {
		return fmt.Errorf("create .gamux directory: %w", err)
	}
	scriptPath := filepath.Join(gamuxDir, "launch.sh")
	return util.WriteFileAtomic(scriptPath, []byte(content), 0755)
}
