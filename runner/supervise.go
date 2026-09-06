package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/staernid/gamux/compat"
	"github.com/staernid/gamux/config"
)

// SuperviseOptions defines parameters for supervising an active process group.
type SuperviseOptions struct {
	GameDir     string
	IsGamescope bool
	LogFile     *os.File
	OnExit      func(err error, sessionSec int64)
}

// isGamescopeCmd determines if a command executes inside a Gamescope micro-compositor.
func isGamescopeCmd(cfg *config.Config, gamescopeOpt *bool, cmd *exec.Cmd) bool {
	if gamescopeOpt != nil && *gamescopeOpt {
		return true
	}
	if cfg != nil && cfg.Gamescope {
		return true
	}
	if cmd != nil {
		if len(cmd.Args) > 0 && filepath.Base(cmd.Args[0]) == "gamescope" {
			return true
		}
		if filepath.Base(cmd.Path) == "gamescope" {
			return true
		}
	}
	return false
}

// isHelperDaemon checks if a process name matches known Wine or wrapper background daemons.
func isHelperDaemon(name string) bool {
	clean := strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	base := strings.ToLower(filepath.Base(clean))
	switch base {
	case "wineserver",
		"services.exe",
		"explorer.exe",
		"conhost.exe",
		"svchost.exe",
		"rpcss.exe",
		"plugplay.exe",
		"winedevice.exe",
		"tabtip.exe",
		"start.exe",
		"gamemoded",
		"bwrap",
		"pasta",
		"reaper":
		return true
	}
	if strings.HasPrefix(base, "pressure-vessel") ||
		strings.HasPrefix(base, "steam-runtime") ||
		strings.HasPrefix(base, "pv-") {
		return true
	}
	return false
}

// isProcessHelperDaemon checks if comm or cmdline arguments indicate a helper daemon.
func isProcessHelperDaemon(comm string, cmdline []string) bool {
	if isHelperDaemon(comm) {
		return true
	}
	if len(cmdline) == 0 {
		return false
	}
	exe := cmdline[0]
	if isHelperDaemon(exe) {
		return true
	}

	exeBase := strings.ToLower(filepath.Base(strings.TrimSpace(exe)))
	commBase := strings.ToLower(filepath.Base(strings.TrimSpace(comm)))
	isWineLoader := strings.HasPrefix(exeBase, "wine") || strings.HasPrefix(commBase, "wine")
	if isWineLoader && len(cmdline) > 1 {
		hasGameBinary := false
		for _, arg := range cmdline[1:] {
			cleanArg := strings.TrimSpace(arg)
			if strings.HasPrefix(cleanArg, "-") || (strings.HasPrefix(cleanArg, "/") && !strings.Contains(cleanArg, "\\") && !strings.HasSuffix(cleanArg, ".exe")) {
				continue
			}
			argBase := strings.ToLower(filepath.Base(strings.ReplaceAll(cleanArg, "\\", "/")))
			if strings.HasSuffix(argBase, ".exe") || strings.HasSuffix(argBase, ".bat") || strings.HasSuffix(argBase, ".cmd") {
				if !isHelperDaemon(argBase) {
					hasGameBinary = true
					break
				}
			}
		}
		if !hasGameBinary {
			return true
		}
	}

	return false
}

// hasActiveGameProcess inspects /proc for non-zombie, non-helper processes belonging to pgid.
func hasActiveGameProcess(pgid int) bool {
	if pgid <= 1 {
		return false
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 1 {
			continue
		}

		statBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		statStr := string(statBytes)

		// /proc/<pid>/stat format: pid (comm) state ppid pgrp ...
		lastParen := strings.LastIndex(statStr, ")")
		firstParen := strings.Index(statStr, "(")
		if firstParen == -1 || lastParen == -1 || lastParen <= firstParen || lastParen >= len(statStr)-1 {
			continue
		}

		comm := statStr[firstParen+1 : lastParen]
		after := strings.TrimSpace(statStr[lastParen+1:])
		fields := strings.Fields(after)
		if len(fields) < 3 {
			continue
		}

		state := fields[0]
		if state == "Z" { // Skip zombies
			continue
		}

		procPgid, err := strconv.Atoi(fields[2])
		if err != nil || procPgid != pgid {
			continue
		}

		var cmdline []string
		if cmdBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil && len(cmdBytes) > 0 {
			rawParts := strings.Split(string(cmdBytes), "\x00")
			for _, p := range rawParts {
				if p != "" {
					cmdline = append(cmdline, p)
				}
			}
		}

		if !isProcessHelperDaemon(comm, cmdline) {
			return true
		}
	}

	return false
}

// terminateProcessGroup gracefully terminates all processes in pgid, falling back to SIGKILL.
func terminateProcessGroup(pgid int, gracePeriod time.Duration) {
	if pgid <= 1 {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	deadline := time.Now().Add(gracePeriod)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pgid, 0); err != nil {
			return // Process group exited cleanly
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

// superviseProcess supervises an active process group, periodic playtime flushing,
// process group monitoring for lingering Wine processes, clean teardown, and log output.
func superviseProcess(ctx context.Context, cmd *exec.Cmd, opts SuperviseOptions) error {
	started := time.Now()
	pgid := 0
	if cmd != nil && cmd.Process != nil {
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
		if opts.GameDir != "" {
			if _, _, err := compat.IncrementPlaytime(opts.GameDir, delta); err == nil {
				if delta > 0 {
					lastFlushedTime = now
				}
			}
		}
	}

	waitDone := make(chan error, 1)
	go func() {
		if cmd != nil {
			waitDone <- cmd.Wait()
		} else {
			waitDone <- nil
		}
	}()

	var waitErr error
	waiting := true
	for waiting {
		select {
		case <-ctx.Done():
			if pgid > 1 {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			} else if cmd != nil && cmd.Process != nil {
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

	// Clean up child processes in the process group
	if pgid > 1 && ctx.Err() == nil {
		if opts.IsGamescope {
			// Gamescope closed: compositor window is gone, terminate immediately
			terminateProcessGroup(pgid, 1500*time.Millisecond)
		} else {
			// Supervise pgid while active non-helper game processes exist
			superviseTicker := time.NewTicker(1 * time.Second)
			defer superviseTicker.Stop()
		supervise:
			for {
				if !hasActiveGameProcess(pgid) {
					break supervise
				}
				select {
				case <-ctx.Done():
					_ = syscall.Kill(-pgid, syscall.SIGTERM)
					waitErr = ctx.Err()
					break supervise
				case <-ticker.C:
					flushDelta()
				case <-superviseTicker.C:
				}
			}
			terminateProcessGroup(pgid, 1500*time.Millisecond)
		}
	}

	flushDelta()
	elapsed := time.Since(started)
	sessionSec := int64(elapsed.Seconds())

	if opts.LogFile != nil {
		exitMsg := fmt.Sprintf("\n\n=== Process Finished ===\nTime: %s\nSession Duration: %v\nExit State: %v\n", time.Now().Format(time.RFC1123), elapsed.Round(time.Second), waitErr)
		_, _ = opts.LogFile.WriteString(exitMsg)
		_ = opts.LogFile.Close()
	}

	if opts.OnExit != nil {
		opts.OnExit(waitErr, sessionSec)
	}

	return waitErr
}
