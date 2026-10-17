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

	// 3. Wrap in compositors:
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
