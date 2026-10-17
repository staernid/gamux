package multiseat

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// snapshotWaylandSockets captures all existing Wayland socket filenames in XDG_RUNTIME_DIR.
func snapshotWaylandSockets() map[string]bool {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	sockets := make(map[string]bool)
	entries, err := os.ReadDir(runtimeDir)
	if err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "wayland-") && !strings.HasSuffix(e.Name(), ".lock") {
				sockets[e.Name()] = true
			}
		}
	}
	return sockets
}

// waitForNewWaylandSocket polls XDG_RUNTIME_DIR for a newly created, connectable Wayland socket.
func waitForNewWaylandSocket(existing map[string]bool, timeout time.Duration) (string, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(runtimeDir)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, "wayland-") && !strings.HasSuffix(name, ".lock") && !strings.Contains(name, "gamux") {
					if !existing[name] {
						sockPath := filepath.Join(runtimeDir, name)
						conn, err := net.DialTimeout("unix", sockPath, 100*time.Millisecond)
						if err == nil {
							_ = conn.Close()
							return sockPath, nil
						}
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "", fmt.Errorf("timeout waiting for new Wayland socket in %s", runtimeDir)
}

// snapshotEISockets captures all existing gamescope-*-ei socket filenames in XDG_RUNTIME_DIR.
func snapshotEISockets() map[string]bool {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	sockets := make(map[string]bool)
	entries, err := os.ReadDir(runtimeDir)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "gamescope-") && strings.HasSuffix(name, "-ei") {
				sockets[name] = true
			}
		}
	}
	return sockets
}

// getSocketPeerPID retrieves the server PID from a connected unix domain socket via SO_PEERCRED.
func getSocketPeerPID(conn net.Conn) (int, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, fmt.Errorf("not a unix connection")
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("get syscall conn: %w", err)
	}
	var cred *syscall.Ucred
	var credErr error
	err = raw.Control(func(fd uintptr) {
		cred, credErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil {
		return 0, err
	}
	if credErr != nil {
		return 0, credErr
	}
	if cred == nil {
		return 0, fmt.Errorf("nil credentials returned")
	}
	return int(cred.Pid), nil
}

// waitForEISocketForPID polls XDG_RUNTIME_DIR for the Gamescope EIS socket corresponding to targetPID.
// If targetPID <= 0, it matches any new socket not present in the existing snapshot.
func waitForEISocketForPID(targetPID int, existing map[string]bool, timeout time.Duration) (string, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(runtimeDir)
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, "gamescope-") && strings.HasSuffix(name, "-ei") {
					// Only exclude snapshot sockets when blindly waiting for a generic new socket (targetPID <= 0).
					// If targetPID > 0, we can deterministically match the socket's owner via SO_PEERCRED,
					// even if the socket file already existed on disk from a previous launch.
					if targetPID <= 0 && existing != nil && existing[name] {
						continue
					}
					sockPath := filepath.Join(runtimeDir, name)
					if targetPID > 0 {
						conn, err := net.DialTimeout("unix", sockPath, 150*time.Millisecond)
						if err == nil {
							pid, pidErr := getSocketPeerPID(conn)
							_ = conn.Close()
							if pidErr == nil && pid == targetPID {
								return sockPath, nil
							}
						}
					} else {
						conn, err := net.DialTimeout("unix", sockPath, 150*time.Millisecond)
						if err == nil {
							_ = conn.Close()
							return sockPath, nil
						}
					}
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if targetPID > 0 {
		return "", fmt.Errorf("timeout waiting for Gamescope EIS socket for PID %d in %s", targetPID, runtimeDir)
	}
	return "", fmt.Errorf("timeout waiting for new Gamescope EIS socket in %s", runtimeDir)
}

// waitForNewEISocket polls XDG_RUNTIME_DIR for a newly created, connectable Gamescope EIS socket.
func waitForNewEISocket(existing map[string]bool, timeout time.Duration) (string, error) {
	return waitForEISocketForPID(0, existing, timeout)
}
