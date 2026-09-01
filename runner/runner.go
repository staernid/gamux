package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
	"github.com/staernid/gamux/domain"
)

// Operating Mode Constants
const (
	ModeStandalone = string(domain.ModeStandalone)
	ModeLoader     = string(domain.ModeLoader)
	ModePortable   = string(domain.ModePortable)
	ModeNone       = string(domain.ModeNone)
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

// Launch starts a game asynchronously and returns the active *exec.Cmd process handle and log file.
func Launch(ctx context.Context, cfg *config.Config, opts LaunchOptions) (*exec.Cmd, *os.File, error) {
	resolved := ResolveEffectiveOptions(cfg, opts)
	cmd, err := BuildCommand(ctx, cfg, resolved)
	if err != nil {
		return nil, nil, fmt.Errorf("build launch command: %w", err)
	}

	// Ensure portable .gamux/launch.sh is kept up-to-date
	_ = GenerateLaunchScript(ctx, cfg, resolved)

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

// Run executes a game synchronously, waiting for it to complete and tracking playtime.
func Run(ctx context.Context, cfg *config.Config, opts LaunchOptions) error {
	resolved := ResolveEffectiveOptions(cfg, opts)
	cmd, err := BuildCommand(ctx, cfg, resolved)
	if err != nil {
		return fmt.Errorf("build launch command: %w", err)
	}

	// Ensure portable .gamux/launch.sh is kept up-to-date
	_ = GenerateLaunchScript(ctx, cfg, resolved)

	logPath := ResolveLogPath(resolved.GameDir, resolved.Name, resolved.AppID)
	logFile, _ := createLogFile(logPath, resolved, cmd)
	if logFile != nil {
		defer logFile.Close()
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("game execution failed: %w", err)
	}

	started := time.Now()
	pgid := 0
	if cmd.Process != nil {
		if p, err := syscall.Getpgid(cmd.Process.Pid); err == nil && p > 1 && p != os.Getpid() {
			pgid = p
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastFlushedTime := started
	flushDelta := func() {
		now := time.Now()
		delta := int64(now.Sub(lastFlushedTime).Seconds())
		if resolved.GameDir != "" {
			if _, _, err := compat.IncrementPlaytime(resolved.GameDir, delta); err == nil {
				if delta > 0 {
					lastFlushedTime = now
				}
			}
		}
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	var waitErr error
	waiting := true
	for waiting {
		select {
		case <-ctx.Done():
			if pgid > 1 {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			} else if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			waitErr = ctx.Err()
			waiting = false
		case <-ticker.C:
			flushDelta()
		case waitErr = <-waitDone:
			waiting = false
		}
	}

	if pgid > 1 && ctx.Err() == nil {
	supervise:
		for {
			if err := syscall.Kill(-pgid, 0); err != nil {
				break
			}
			select {
			case <-ctx.Done():
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
				waitErr = ctx.Err()
				break supervise
			case <-ticker.C:
				flushDelta()
			case <-time.After(1 * time.Second):
			}
		}
	}

	flushDelta()
	return waitErr
}
