package gui

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
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

// AcquireOrActivate attempts to acquire exclusive ownership using the default user socket.
func AcquireOrActivate(appCtxGetter func() context.Context) (*SingleInstanceGuard, bool) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}
	sockPath := filepath.Join(runtimeDir, fmt.Sprintf("gamux-gui-%d.sock", os.Getuid()))
	return AcquireOrActivateSocket(sockPath, appCtxGetter)
}

// AcquireOrActivateSocket attempts to acquire exclusive ownership using a specific socket path with flock-based race safety.
func AcquireOrActivateSocket(sockPath string, appCtxGetter func() context.Context) (*SingleInstanceGuard, bool) {
	lockPath := sockPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err == nil {
		if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			// Another process holds the lock file; dial socket to send FOCUS
			conn, dErr := net.DialTimeout("unix", sockPath, 500*time.Millisecond)
			if dErr == nil {
				_, _ = conn.Write([]byte("FOCUS\n"))
				_ = conn.Close()
				slog.Info("Another gamux-gui instance is already running; activated existing window and exiting")
			}
			_ = lockFile.Close()
			return nil, false
		}
	}

	// We hold the advisory file lock. Try to connect to existing socket in case of lingering connection
	conn, err := net.DialTimeout("unix", sockPath, 200*time.Millisecond)
	if err == nil {
		_, _ = conn.Write([]byte("FOCUS\n"))
		_ = conn.Close()
		if lockFile != nil {
			_ = lockFile.Close()
		}
		slog.Info("Another gamux-gui instance is already running; activated existing window and exiting")
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

	go guard.listenLoop(appCtxGetter)

	return guard, true
}

func (g *SingleInstanceGuard) listenLoop(appCtxGetter func() context.Context) {
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
			buf := make([]byte, 64)
			n, _ := c.Read(buf)
			cmd := string(buf[:n])
			if cmd == "FOCUS\n" || cmd == "FOCUS" {
				if appCtxGetter != nil {
					ctx := appCtxGetter()
					if ctx != nil {
						wailsRuntime.WindowShow(ctx)
						wailsRuntime.WindowUnminimise(ctx)
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
