package multiseat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/util"
)

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
		info := s.deps().DetectPrimaryDisplay()
		dispWidth = info.Width
		dispHeight = info.Height
		if dispFPS <= 0 {
			dispFPS = info.RefreshRate
		}
	}
	return dispWidth, dispHeight, dispFPS
}

// normalizeSeatConfig fills in the seat ID and resolves device node aliases.
func normalizeSeatConfig(seatCfg SeatConfig, seatIndex int) SeatConfig {
	if seatCfg.SeatID == "" {
		seatCfg.SeatID = fmt.Sprintf("p%d", seatIndex+1)
	}
	seatCfg.KeyboardNode = ResolveDeviceNode(seatCfg.KeyboardNode)
	seatCfg.MouseNode = ResolveDeviceNode(seatCfg.MouseNode)
	seatCfg.GamepadNode = ResolveDeviceNode(seatCfg.GamepadNode)
	return seatCfg
}

// resolveSeatOperatingMode picks the game's saved profile mode, then the global config
// mode, then loader mode.
func (s *Supervisor) resolveSeatOperatingMode(gameDir string) string {
	if profile, err := compat.LoadProfile(gameDir); err == nil && profile != nil && profile.OperatingMode != "" {
		return domain.NormalizeOperatingMode(profile.OperatingMode)
	}
	if s.cfg != nil && s.cfg.OperatingMode != "" {
		return domain.NormalizeOperatingMode(s.cfg.OperatingMode)
	}
	return domain.ModeLoader
}

// ensureSeatPrefixPath provisions the seat's Wine prefix, falling back to an isolated
// directory inside the profile when provisioning fails.
func (s *Supervisor) ensureSeatPrefixPath(ctx context.Context, sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, profileDir string, mode string) string {
	numAppID, _ := util.ParseAppID(sessionCfg.AppID)
	gameStub := &domain.Game{
		GameRecord: domain.GameRecord{
			Name:         sessionCfg.Name,
			AppID:        sessionCfg.AppID,
			NumericAppID: numAppID,
			GameDir:      sessionCfg.GameDir,
		},
		LaunchProfile: &domain.LaunchProfile{OperatingMode: mode},
	}

	pfxPath, err := s.deps().EnsureSeatPrefix(ctx, s.cfg, gameStub, seatCfg.SeatID, seatCfg.WinePrefix)
	if err != nil || pfxPath == "" {
		pfxPath = filepath.Join(profileDir, "prefix")
		_ = os.MkdirAll(pfxPath, 0755)
	}
	return pfxPath
}

// buildBaseLaunchSpec asks the runner for the unwrapped game launch specification.
func (s *Supervisor) buildBaseLaunchSpec(ctx context.Context, sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, mode string, pfxPath string) (*runner.LaunchSpec, error) {
	effectiveProfileID := seatCfg.SeatID
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
		OperatingMode: mode,
		ProfileID:     effectiveProfileID,
		WinePrefix:    pfxPath,
		CustomEnv:     make(map[string]string),
	}
	for k, v := range seatCfg.Player.CustomEnv {
		launchOpts.CustomEnv[k] = v
	}

	spec, err := s.deps().BuildLaunchSpec(ctx, s.cfg, launchOpts)
	if err != nil {
		return nil, fmt.Errorf("build launch spec: %w", err)
	}
	return spec, nil
}

// wrapWithBwrap wraps cmdArgs in a Bubblewrap sandbox providing filesystem isolation
// (read-only game dir, isolated Wine prefix and GSE saves) and input masking.
func (s *Supervisor) wrapWithBwrap(sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, profileDir, pfxPath string, sink *VirtualSink, env map[string]string, cmdArgs []string) ([]string, error) {
	sinkName := ""
	if sink != nil {
		sinkName = sink.SinkName
	}
	bwArgs, err := BuildBwrapArgs(BwrapOptions{
		SeatID:        seatCfg.SeatID,
		ProfileDir:    profileDir,
		GameDir:       sessionCfg.GameDir,
		WinePrefix:    pfxPath,
		GamepadNode:   seatCfg.GamepadNode,
		KeyboardNode:  seatCfg.KeyboardNode,
		MouseNode:     seatCfg.MouseNode,
		AudioSink:     sinkName,
		CustomEnv:     env,
		Command:       cmdArgs,
		DieWithParent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("build bwrap args: %w", err)
	}
	return append([]string{lookPathOr(s.deps(), "bwrap")}, bwArgs...), nil
}

// seatGeometry bundles the window geometry resolved for a seat.
type seatGeometry struct {
	winW, winH, winX, winY int
	dispFPS                int
}

func (s *Supervisor) resolveSeatGeometry(sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, seatIndex, totalSeats int) seatGeometry {
	dispWidth, dispHeight, dispFPS := s.resolveDisplayDimensions(seatCfg)
	calcSlot := seatIndex
	if seatCfg.ScreenSlot != nil && *seatCfg.ScreenSlot >= 0 && *seatCfg.ScreenSlot < totalSeats {
		calcSlot = *seatCfg.ScreenSlot
	}
	w, h, x, y := CalculateSeatGeometry(sessionCfg.Layout, seatCfg.SeatID, calcSlot, totalSeats, dispWidth, dispHeight)
	return seatGeometry{winW: w, winH: h, winX: x, winY: y, dispFPS: dispFPS}
}

// hasDedicatedKBM reports whether the seat owns exclusive keyboard/mouse hardware.
func hasDedicatedKBM(seatCfg SeatConfig, seatIndex int) bool {
	dedicated := isExplicitDeviceNode(seatCfg.KeyboardNode) || isExplicitDeviceNode(seatCfg.MouseNode)
	if seatIndex == 0 && !seatCfg.ExclusiveGrab {
		return false
	}
	return dedicated
}

// wrapWithGamescope wraps cmdArgs in gamescope (scaling, resolution/FPS limit, split-screen
// or dual monitor placement, EIS input redirection).
func (s *Supervisor) wrapWithGamescope(sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, geo seatGeometry, dedicatedKBM bool, cmdArgs []string) ([]string, error) {
	renderW := seatCfg.TargetWidth
	renderH := seatCfg.TargetHeight
	if renderW <= 0 || renderH <= 0 {
		renderW = geo.winW
		renderH = geo.winH
	}

	forceGrab := seatCfg.ForceGrabCursor
	grabKbd := seatCfg.GrabKeyboard
	if dedicatedKBM {
		// In multi-seat sessions with exclusive hardware input grabbers, input is injected
		// directly via EIS. Gamescope's nested host window must NEVER grab the host's
		// desktop keyboard (-g) or mouse cursor (--force-grab-cursor), preventing conflict with Player 1.
		forceGrab = false
		grabKbd = false
	}

	gsArgs, err := BuildGamescopeArgs(DisplayOptions{
		Mode:            seatCfg.Mode,
		Layout:          sessionCfg.Layout,
		SeatID:          seatCfg.SeatID,
		DisplayIndex:    seatCfg.DisplayIndex,
		TargetWidth:     renderW,
		TargetHeight:    renderH,
		TargetFPS:       geo.dispFPS,
		WindowWidth:     geo.winW,
		WindowHeight:    geo.winH,
		WindowX:         geo.winX,
		WindowY:         geo.winY,
		Fullscreen:      seatCfg.Mode == SeatModeLocal && sessionCfg.Layout == LayoutDualMonitor,
		Borderless:      sessionCfg.Layout == LayoutSideBySide || sessionCfg.Layout == LayoutTopBottom,
		ForceGrabCursor: forceGrab,
		GrabKeyboard:    grabKbd,
		CursorPath:      seatCfg.CursorPath,
		Command:         cmdArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("build gamescope args: %w", err)
	}
	return append([]string{lookPathOr(s.deps(), "gamescope")}, gsArgs...), nil
}

// wrapWithCage wraps cmdArgs in the standalone cage kiosk compositor.
func (s *Supervisor) wrapWithCage(cmdArgs []string) ([]string, error) {
	cageArgs, err := BuildCageArgs("", cmdArgs)
	if err != nil {
		return nil, fmt.Errorf("build cage args: %w", err)
	}
	return append([]string{lookPathOr(s.deps(), "cage")}, cageArgs...), nil
}

// seatEnvOptions captures the inputs that determine cursor and window-position env vars.
type seatEnvOptions struct {
	useCage      bool
	useGamescope bool
	showCursor   bool
	dedicatedKBM bool
	layout       DisplayLayout
	winX, winY   int
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// seatEnvVars returns the extra environment for the seat process.
func seatEnvVars(o seatEnvOptions) []string {
	// Ensure cursor theme and sizing variables are set so Gamescope/Xwayland/Wine
	// always renders a visible mouse cursor sprite.
	env := []string{
		"XCURSOR_THEME=" + envOrDefault("XCURSOR_THEME", "default"),
		"XCURSOR_SIZE=" + envOrDefault("XCURSOR_SIZE", "24"),
		"XCURSOR_PATH=" + envOrDefault("XCURSOR_PATH", "/usr/share/icons:/usr/share/pixmaps:~/.local/share/icons:~/.icons"),
	}

	// Software cursor when requested or running under Cage/wlroots.
	if o.showCursor || o.useCage {
		env = append(env, "WLR_NO_HARDWARE_CURSORS=1")
	}

	// In nested mode, Gamescope defaults to cv_cursor_composite=1 ("composite when not nested"),
	// expecting the host window manager to draw the cursor. For seats with dedicated hardware
	// grabbers (or ShowCursor), physical input is redirected via EIS and the host never renders
	// a cursor, so force gamescope_cursor_composite=2 ("always composite manually").
	if o.useGamescope && (o.showCursor || o.dedicatedKBM) {
		env = append(env, "gamescope_cursor_composite=2")
	}

	// Split layouts or non-zero coordinates: position the SDL window at the calculated half.
	if o.useGamescope && (o.winX > 0 || o.winY > 0 || o.layout == LayoutSideBySide || o.layout == LayoutTopBottom) {
		pos := fmt.Sprintf("%d,%d", o.winX, o.winY)
		env = append(env, "SDL_VIDEO_WINDOW_POS="+pos, "SDL_WINDOW_POS="+pos)
	}
	return env
}

// openSeatLog creates the seat's log file; a nil file means logging is unavailable.
func openSeatLog(profileDir, seatID string) (*os.File, string) {
	logDir := filepath.Join(profileDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", seatID))
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	return logFile, logPath
}

func (s *Supervisor) buildSeatCommand(ctx context.Context, sessionCfg MultiseatSessionConfig, seatCfg SeatConfig, profileDir string, sink *VirtualSink, caps SystemCapabilities, seatIndex, totalSeats int) (*exec.Cmd, *os.File, string, error) {
	seatCfg = normalizeSeatConfig(seatCfg, seatIndex)
	seatID := seatCfg.SeatID

	mode := s.resolveSeatOperatingMode(sessionCfg.GameDir)
	pfxPath := s.ensureSeatPrefixPath(ctx, sessionCfg, seatCfg, profileDir, mode)

	spec, err := s.buildBaseLaunchSpec(ctx, sessionCfg, seatCfg, mode, pfxPath)
	if err != nil {
		return nil, nil, "", err
	}
	cmdArgs := spec.Args

	if caps.HasBwrap {
		if cmdArgs, err = s.wrapWithBwrap(sessionCfg, seatCfg, profileDir, pfxPath, sink, spec.Env, cmdArgs); err != nil {
			return nil, nil, "", err
		}
	}

	geo := s.resolveSeatGeometry(sessionCfg, seatCfg, seatIndex, totalSeats)
	useCage := resolveCageDecision(seatCfg, seatID, caps)
	useGamescope := caps.HasGamescope && !useCage
	dedicatedKBM := hasDedicatedKBM(seatCfg, seatIndex)

	switch {
	case useGamescope:
		cmdArgs, err = s.wrapWithGamescope(sessionCfg, seatCfg, geo, dedicatedKBM, cmdArgs)
	case useCage:
		cmdArgs, err = s.wrapWithCage(cmdArgs)
	}
	if err != nil {
		return nil, nil, "", err
	}

	finalCmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	finalCmd.Dir = sessionCfg.GameDir
	finalCmd.Env = append(os.Environ(), seatEnvVars(seatEnvOptions{
		useCage:      useCage,
		useGamescope: useGamescope,
		showCursor:   seatCfg.ShowCursor,
		dedicatedKBM: dedicatedKBM,
		layout:       sessionCfg.Layout,
		winX:         geo.winX,
		winY:         geo.winY,
	})...)

	logFile, logPath := openSeatLog(profileDir, seatID)
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
