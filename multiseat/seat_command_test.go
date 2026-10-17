package multiseat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
	"github.com/staernid/gamux/runner"
	"github.com/staernid/gamux/util"
)

// fakeSeatDeps is a seatDeps test double with configurable failure modes that records calls.
type fakeSeatDeps struct {
	binaries   map[string]string // name -> resolved path; missing names fail LookPath
	prefix     string
	prefixErr  error
	specErr    error
	specArgs   []string
	specEnv    map[string]string
	display    util.DisplayInfo
	gotOpts    runner.LaunchOptions
	gotSeatID  string
	displayHit int
}

func (f *fakeSeatDeps) LookPath(name string) (string, error) {
	if p, ok := f.binaries[name]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}

func (f *fakeSeatDeps) EnsureSeatPrefix(_ context.Context, _ *config.Config, _ *domain.Game, seatID, _ string) (string, error) {
	f.gotSeatID = seatID
	return f.prefix, f.prefixErr
}

func (f *fakeSeatDeps) BuildLaunchSpec(_ context.Context, _ *config.Config, opts runner.LaunchOptions) (*runner.LaunchSpec, error) {
	f.gotOpts = opts
	if f.specErr != nil {
		return nil, f.specErr
	}
	args := f.specArgs
	if args == nil {
		args = []string{"/fake/game", "-flag"}
	}
	return &runner.LaunchSpec{Args: slices.Clone(args), Env: f.specEnv}, nil
}

func (f *fakeSeatDeps) DetectPrimaryDisplay() util.DisplayInfo {
	f.displayHit++
	return f.display
}

func newFakeSupervisor(t *testing.T, fake *fakeSeatDeps) (*Supervisor, MultiseatSessionConfig, string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	sup := &Supervisor{cfg: config.DefaultConfig(), seatDeps: fake}
	sessionCfg := MultiseatSessionConfig{
		GameDir: filepath.Join(tmp, "Game"),
		Name:    "Game",
		AppID:   "70",
		Layout:  LayoutDualMonitor,
	}
	return sup, sessionCfg, filepath.Join(tmp, "profile")
}

func buildWithFake(t *testing.T, fake *fakeSeatDeps, caps SystemCapabilities, seat SeatConfig) (cmdArgs, env []string, err error) {
	t.Helper()
	sup, sessionCfg, profileDir := newFakeSupervisor(t, fake)
	if fake.prefix == "" && fake.prefixErr == nil {
		fake.prefix = filepath.Join(profileDir, "fake-prefix")
	}
	cmd, logFile, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, seat, profileDir, nil, caps, 1, 2)
	if err != nil {
		return nil, nil, err
	}
	if logFile != nil {
		t.Cleanup(func() { logFile.Close() })
	}
	return cmd.Args, cmd.Env, nil
}

func TestBuildSeatCommand_NoWrappersRunsGameDirectly(t *testing.T) {
	args, _, err := buildWithFake(t, &fakeSeatDeps{}, SystemCapabilities{}, SeatConfig{Mode: SeatModeLocal})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"/fake/game", "-flag"}; !slices.Equal(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func TestBuildSeatCommand_LaunchSpecErrorIsWrapped(t *testing.T) {
	fake := &fakeSeatDeps{specErr: errors.New("boom")}
	_, _, err := buildWithFake(t, fake, SystemCapabilities{HasGamescope: true}, SeatConfig{})
	if err == nil || !strings.Contains(err.Error(), "build launch spec") || !errors.Is(err, fake.specErr) {
		t.Fatalf("err = %v, want wrapped launch spec error", err)
	}
}

func TestBuildSeatCommand_PrefixFailureFallsBackToProfileDir(t *testing.T) {
	fake := &fakeSeatDeps{prefixErr: errors.New("no space")}
	sup, sessionCfg, profileDir := newFakeSupervisor(t, fake)

	cmd, logFile, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, SeatConfig{}, profileDir, nil, SystemCapabilities{}, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	want := filepath.Join(profileDir, "prefix")
	if fake.gotOpts.WinePrefix != want {
		t.Errorf("WinePrefix = %q, want fallback %q", fake.gotOpts.WinePrefix, want)
	}
	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Errorf("fallback prefix dir not created: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected a command despite prefix failure")
	}
}

func TestBuildSeatCommand_EmptyPrefixPathAlsoFallsBack(t *testing.T) {
	fake := &fakeSeatDeps{prefix: ""}
	sup, sessionCfg, profileDir := newFakeSupervisor(t, fake)
	_, logFile, _, err := sup.buildSeatCommand(context.Background(), sessionCfg, SeatConfig{}, profileDir, nil, SystemCapabilities{}, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	if want := filepath.Join(profileDir, "prefix"); fake.gotOpts.WinePrefix != want {
		t.Errorf("WinePrefix = %q, want %q", fake.gotOpts.WinePrefix, want)
	}
}

func TestBuildSeatCommand_SeatIDDefaultsFromIndex(t *testing.T) {
	fake := &fakeSeatDeps{}
	if _, _, err := buildWithFake(t, fake, SystemCapabilities{}, SeatConfig{}); err != nil {
		t.Fatal(err)
	}
	if fake.gotSeatID != "p2" {
		t.Errorf("seat id = %q, want p2 for index 1", fake.gotSeatID)
	}
}

func TestBuildSeatCommand_GamescopeBinaryResolution(t *testing.T) {
	caps := SystemCapabilities{HasGamescope: true}
	seat := SeatConfig{Mode: SeatModeLocal, TargetWidth: 1280, TargetHeight: 720}

	args, _, err := buildWithFake(t, &fakeSeatDeps{binaries: map[string]string{"gamescope": "/opt/bin/gamescope"}}, caps, seat)
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "/opt/bin/gamescope" {
		t.Errorf("args[0] = %q, want resolved gamescope path", args[0])
	}
	if !slices.Contains(args, "/fake/game") {
		t.Errorf("game command missing from %v", args)
	}

	args, _, err = buildWithFake(t, &fakeSeatDeps{}, caps, seat)
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "gamescope" {
		t.Errorf("args[0] = %q, want bare name when LookPath fails", args[0])
	}
}

func TestBuildSeatCommand_CageFallbackWhenNoGamescope(t *testing.T) {
	fake := &fakeSeatDeps{binaries: map[string]string{"cage": "/usr/bin/cage"}}
	args, env, err := buildWithFake(t, fake, SystemCapabilities{HasCage: true}, SeatConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "/usr/bin/cage" || !slices.Contains(args, "/fake/game") {
		t.Errorf("args = %v, want cage-wrapped game", args)
	}
	if !slices.Contains(env, "WLR_NO_HARDWARE_CURSORS=1") {
		t.Error("cage seats must force software cursors")
	}
}

func TestBuildSeatCommand_BwrapWrapsGameAndRunsInsideCompositor(t *testing.T) {
	fake := &fakeSeatDeps{binaries: map[string]string{"bwrap": "/usr/bin/bwrap", "cage": "/usr/bin/cage"}}
	args, _, err := buildWithFake(t, fake, SystemCapabilities{HasBwrap: true, HasCage: true}, SeatConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "/usr/bin/cage" {
		t.Fatalf("args[0] = %q, want compositor outermost", args[0])
	}
	if !slices.Contains(args, "/usr/bin/bwrap") {
		t.Errorf("bwrap should sit inside the compositor: %v", args)
	}
}

func TestResolveDisplayDimensions_FallsBackToDetectedDisplay(t *testing.T) {
	fake := &fakeSeatDeps{display: util.DisplayInfo{Width: 2560, Height: 1440, RefreshRate: 144}}
	sup := &Supervisor{cfg: &config.Config{}, seatDeps: fake}

	w, h, fps := sup.resolveDisplayDimensions(SeatConfig{})
	if w != 2560 || h != 1440 || fps != 144 || fake.displayHit != 1 {
		t.Errorf("got %dx%d@%d (hits=%d), want detected 2560x1440@144", w, h, fps, fake.displayHit)
	}

	// An explicit seat FPS survives detection.
	_, _, fps = sup.resolveDisplayDimensions(SeatConfig{TargetFPS: 30})
	if fps != 30 {
		t.Errorf("fps = %d, want explicit 30", fps)
	}
}

func TestResolveDisplayDimensions_ConfigAvoidsDetection(t *testing.T) {
	fake := &fakeSeatDeps{}
	sup := &Supervisor{cfg: &config.Config{GamescopeWidth: 1920, GamescopeHeight: 1080, GamescopeFPS: 75}, seatDeps: fake}

	w, h, fps := sup.resolveDisplayDimensions(SeatConfig{})
	if w != 1920 || h != 1080 || fps != 75 || fake.displayHit != 0 {
		t.Errorf("got %dx%d@%d (hits=%d), want config values without probing", w, h, fps, fake.displayHit)
	}
}

func TestSeatEnvVars(t *testing.T) {
	t.Setenv("XCURSOR_THEME", "")
	t.Setenv("XCURSOR_SIZE", "")
	t.Setenv("XCURSOR_PATH", "")

	env := seatEnvVars(seatEnvOptions{})
	for _, want := range []string{"XCURSOR_THEME=default", "XCURSOR_SIZE=24"} {
		if !slices.Contains(env, want) {
			t.Errorf("missing %q in %v", want, env)
		}
	}
	for _, unwanted := range []string{"WLR_NO_HARDWARE_CURSORS=1", "gamescope_cursor_composite=2", "SDL_WINDOW_POS=0,0"} {
		if slices.Contains(env, unwanted) {
			t.Errorf("unexpected %q in %v", unwanted, env)
		}
	}

	env = seatEnvVars(seatEnvOptions{useGamescope: true, dedicatedKBM: true, layout: LayoutSideBySide, winX: 960})
	for _, want := range []string{"gamescope_cursor_composite=2", "SDL_VIDEO_WINDOW_POS=960,0", "SDL_WINDOW_POS=960,0"} {
		if !slices.Contains(env, want) {
			t.Errorf("missing %q in %v", want, env)
		}
	}

	t.Setenv("XCURSOR_THEME", "Adwaita")
	if env := seatEnvVars(seatEnvOptions{showCursor: true}); !slices.Contains(env, "XCURSOR_THEME=Adwaita") || !slices.Contains(env, "WLR_NO_HARDWARE_CURSORS=1") {
		t.Errorf("env = %v", env)
	}
}

func TestHasDedicatedKBM(t *testing.T) {
	explicit := SeatConfig{KeyboardNode: "/dev/input/event3"}
	if !hasDedicatedKBM(explicit, 1) {
		t.Error("secondary seat with explicit keyboard should be dedicated")
	}
	if hasDedicatedKBM(explicit, 0) {
		t.Error("primary seat without ExclusiveGrab must not be dedicated")
	}
	explicit.ExclusiveGrab = true
	if !hasDedicatedKBM(explicit, 0) {
		t.Error("primary seat with ExclusiveGrab should be dedicated")
	}
	if hasDedicatedKBM(SeatConfig{}, 1) {
		t.Error("seat without explicit nodes is not dedicated")
	}
}
