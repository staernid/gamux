package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/tools"
	"github.com/staernid/gamux/util"
)

// Runner represents a polymorphic execution engine adapter.
type Runner interface {
	Name() string
	Available(ctx context.Context, cfg *config.Config) bool
	BuildArgs(ctx context.Context, cfg *config.Config, targetExe string, exeArgs []string, opts LaunchOptions) (binary string, args []string, err error)
}

// NativeRunner executes native Linux ELF binaries and scripts directly.
type NativeRunner struct{}

func (r *NativeRunner) Name() string {
	return RunnerLinux
}

func (r *NativeRunner) Available(ctx context.Context, cfg *config.Config) bool {
	return true
}

func (r *NativeRunner) BuildArgs(ctx context.Context, cfg *config.Config, targetExe string, exeArgs []string, opts LaunchOptions) (string, []string, error) {
	return targetExe, exeArgs, nil
}

// UMURunner executes Windows binaries through Open-Wine-Components umu-run.
type UMURunner struct{}

func (r *UMURunner) Name() string {
	return RunnerUMU
}

func (r *UMURunner) Available(ctx context.Context, cfg *config.Config) bool {
	bin, err := FindUMUBinary()
	return err == nil && bin != ""
}

func (r *UMURunner) BuildArgs(ctx context.Context, cfg *config.Config, targetExe string, exeArgs []string, opts LaunchOptions) (string, []string, error) {
	umuBin, err := FindUMUBinary()
	if err != nil {
		slog.Info("umu-run not found locally, auto-bootstrapping from GitHub release")
		if booted, bErr := tools.NewUMUTool(cfg).EnsureAvailable(ctx); bErr == nil && util.FileExists(booted) {
			umuBin = booted
		} else {
			return "", nil, fmt.Errorf("runner 'umu' selected: %w (auto-bootstrap failed: %v)", err, bErr)
		}
	}
	return umuBin, append([]string{targetExe}, exeArgs...), nil
}

// WineRunner executes Windows binaries through system Wine.
type WineRunner struct{}

func (r *WineRunner) Name() string {
	return RunnerWine
}

func (r *WineRunner) Available(ctx context.Context, cfg *config.Config) bool {
	_, err := exec.LookPath("wine")
	return err == nil
}

func (r *WineRunner) BuildArgs(ctx context.Context, cfg *config.Config, targetExe string, exeArgs []string, opts LaunchOptions) (string, []string, error) {
	wineBin, err := exec.LookPath("wine")
	if err != nil {
		return "", nil, fmt.Errorf("wine binary not found in PATH: %w", err)
	}
	return wineBin, append([]string{targetExe}, exeArgs...), nil
}

// ProtonRunner executes Windows binaries directly through a Valve/GE Proton script.
type ProtonRunner struct{}

func (r *ProtonRunner) Name() string {
	return RunnerProton
}

func (r *ProtonRunner) Available(ctx context.Context, cfg *config.Config) bool {
	if _, err := exec.LookPath("proton"); err == nil {
		return true
	}
	if cfg != nil && cfg.ProtonPath != "" {
		return util.FileExists(filepath.Join(cfg.ProtonPath, "proton"))
	}
	return false
}

func (r *ProtonRunner) BuildArgs(ctx context.Context, cfg *config.Config, targetExe string, exeArgs []string, opts LaunchOptions) (string, []string, error) {
	protonBin, err := exec.LookPath("proton")
	if err != nil {
		if opts.ProtonPath != "" {
			customProton := filepath.Join(opts.ProtonPath, "proton")
			if util.FileExists(customProton) {
				protonBin = customProton
			}
		} else if cfg != nil && cfg.ProtonPath != "" {
			customProton := filepath.Join(cfg.ProtonPath, "proton")
			if util.FileExists(customProton) {
				protonBin = customProton
			}
		}
	}
	if protonBin == "" {
		return "", nil, fmt.Errorf("proton binary not found in PATH or configured proton path")
	}
	return protonBin, append([]string{"run", targetExe}, exeArgs...), nil
}

// ResolveRunner returns the appropriate Runner adapter for the given runner name and platform.
func ResolveRunner(runnerName, platform string) Runner {
	cleanName := strings.ToLower(strings.TrimSpace(runnerName))
	cleanPlat := strings.ToLower(strings.TrimSpace(platform))

	if cleanPlat == "linux" && (cleanName == "" || cleanName == RunnerAuto || cleanName == RunnerLinux) {
		return &NativeRunner{}
	}

	switch cleanName {
	case RunnerLinux:
		return &NativeRunner{}
	case RunnerWine:
		return &WineRunner{}
	case RunnerProton:
		return &ProtonRunner{}
	case RunnerUMU, RunnerAuto, "":
		return &UMURunner{}
	default:
		if cleanPlat == "linux" {
			return &NativeRunner{}
		}
		return &UMURunner{}
	}
}
