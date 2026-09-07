package gui

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// SingleInstanceGuard manages exclusive process ownership and IPC activation.
type SingleInstanceGuard struct {
	sockPath string
	lockFile *os.File
	listener net.Listener
	mu       sync.Mutex
	closed   bool
}

// ExtractOpenTarget returns the first URL scheme or identifier argument (e.g. gamux://, steam://, http(s)://, or numeric AppID).
func ExtractOpenTarget(args []string) string {
	for _, arg := range args {
		s := strings.TrimSpace(arg)
		if s == "" || strings.HasPrefix(s, "-") {
			continue
		}
		if strings.HasPrefix(s, "gamux://") || strings.HasPrefix(s, "steam://") ||
			strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			return s
		}
		// Check if purely numeric AppID (at least 2 digits)
		if len(s) >= 2 && len(s) <= 10 {
			isNumeric := true
			for _, r := range s {
				if r < '0' || r > '9' {
					isNumeric = false
					break
				}
			}
			if isNumeric {
				return s
			}
		}
	}
	return ""
}

// AcquireOrActivate attempts to acquire exclusive ownership using the default user socket.
// If another instance is running, it activates the existing window, passes any target URL/AppID, and exits.
func AcquireOrActivate(args []string, appCtxGetter func() context.Context, onOpenTarget func(target string)) (*SingleInstanceGuard, bool) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	sockPath := filepath.Join(runtimeDir, fmt.Sprintf("gamux-gui-%d.sock", os.Getuid()))
	return AcquireOrActivateSocket(sockPath, args, appCtxGetter, onOpenTarget)
}

// AcquireOrActivateSocket attempts to acquire exclusive ownership using a specific socket path with flock-based race safety.
func AcquireOrActivateSocket(sockPath string, args []string, appCtxGetter func() context.Context, onOpenTarget func(target string)) (*SingleInstanceGuard, bool) {
	target := ExtractOpenTarget(args)
	openCmd := "FOCUS\n"
	if target != "" {
		openCmd = fmt.Sprintf("OPEN %s\n", target)
	}

	lockPath := sockPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err == nil {
		if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			// Another process holds the lock file; dial socket to send command
			conn, dErr := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
			if dErr == nil {
				_, _ = conn.Write([]byte(openCmd))
				_ = conn.Close()
				slog.Info("Another gamux-gui instance is already running; activated existing window and forwarded command", "command", strings.TrimSpace(openCmd))
			}
			_ = lockFile.Close()
			return nil, false
		}
	}

	// We hold the advisory file lock. Try to connect to existing socket in case of lingering connection
	conn, err := net.DialTimeout("unix", sockPath, 200*time.Millisecond)
	if err == nil {
		_, _ = conn.Write([]byte(openCmd))
		_ = conn.Close()
		if lockFile != nil {
			_ = lockFile.Close()
		}
		slog.Info("Another gamux-gui instance is already running; activated existing window and forwarded command", "command", strings.TrimSpace(openCmd))
		return nil, false
	}

	// Remove stale socket safely while holding flock
	_ = os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		slog.Warn("Failed to create single-instance IPC socket, proceeding without guard", "error", err)
		if lockFile != nil {
			_ = lockFile.Close()
		}
		return nil, true
	}

	guard := &SingleInstanceGuard{
		sockPath: sockPath,
		lockFile: lockFile,
		listener: listener,
	}

	go guard.listenLoop(appCtxGetter, onOpenTarget)

	return guard, true
}

func (g *SingleInstanceGuard) listenLoop(appCtxGetter func() context.Context, onOpenTarget func(target string)) {
	for {
		conn, err := g.listener.Accept()
		if err != nil {
			g.mu.Lock()
			isClosed := g.closed
			g.mu.Unlock()
			if isClosed {
				return
			}
			slog.Debug("Single instance IPC accept error", "error", err)
			return
		}

		go func(c net.Conn) {
			defer c.Close()
			scanner := bufio.NewScanner(c)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "FOCUS" {
					if appCtxGetter != nil {
						ctx := appCtxGetter()
						if ctx != nil {
							wailsRuntime.WindowShow(ctx)
							wailsRuntime.WindowUnminimise(ctx)
						}
					}
				} else if strings.HasPrefix(line, "OPEN ") {
					target := strings.TrimSpace(strings.TrimPrefix(line, "OPEN "))
					if appCtxGetter != nil {
						ctx := appCtxGetter()
						if ctx != nil {
							wailsRuntime.WindowShow(ctx)
							wailsRuntime.WindowUnminimise(ctx)
						}
					}
					if onOpenTarget != nil && target != "" {
						onOpenTarget(target)
					}
				}
			}
		}(conn)
	}
}

// Release shuts down the IPC listener and removes the socket file.
func (g *SingleInstanceGuard) Release() {
	if g == nil {
		return
	}
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	g.closed = true
	g.mu.Unlock()

	if g.listener != nil {
		_ = g.listener.Close()
	}
	_ = os.Remove(g.sockPath)
	if g.lockFile != nil {
		_ = syscall.Flock(int(g.lockFile.Fd()), syscall.LOCK_UN)
		_ = g.lockFile.Close()
		_ = os.Remove(g.sockPath + ".lock")
	}
}
