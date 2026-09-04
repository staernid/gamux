package multiseat

import (
	"log/slog"
	"os"
	"sync"
	"syscall"
)

// EVIOCGRAB is the ioctl code (_IOW('E', 0x90, int)) to request exclusive access to an evdev device on Linux.
const EVIOCGRAB = 0x40044590

// InputGrabber manages exclusive kernel grab (EVIOCGRAB) on assigned input devices.
type InputGrabber struct {
	mu     sync.Mutex
	fds    map[string]*os.File
	nodes  []string
	active bool
}

// NewInputGrabber instantiates an input grabber for specified device nodes.
func NewInputGrabber(devNodes ...string) *InputGrabber {
	var valid []string
	seen := make(map[string]bool)
	for _, n := range devNodes {
		if n != "" && n != "none" && !seen[n] {
			valid = append(valid, n)
			seen[n] = true
		}
	}
	return &InputGrabber{
		fds:   make(map[string]*os.File),
		nodes: valid,
	}
}

// Grab takes exclusive ownership of the configured devices via EVIOCGRAB.
// When grabbed, the host OS and other processes receive zero events from these devices.
func (g *InputGrabber) Grab() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.active {
		return nil
	}

	for _, node := range g.nodes {
		f, err := os.OpenFile(node, os.O_RDWR, 0)
		if err != nil {
			// Fallback to read-only
			f, err = os.OpenFile(node, os.O_RDONLY, 0)
			if err != nil {
				slog.Warn("Failed to open input device for exclusive grab", "node", node, "error", err)
				continue
			}
		}

		// Perform EVIOCGRAB (arg = 1 to grab)
		_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(EVIOCGRAB), 1)
		if errno != 0 {
			slog.Warn("ioctl EVIOCGRAB failed on device", "node", node, "errno", errno)
			_ = f.Close()
			continue
		}

		g.fds[node] = f
		slog.Info("Exclusively grabbed input device for seat", "node", node)
	}

	g.active = len(g.fds) > 0
	return nil
}

// Release releases all grabbed input devices back to the host system.
func (g *InputGrabber) Release() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	for node, f := range g.fds {
		if f != nil {
			// EVIOCGRAB (arg = 0 to release)
			_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(EVIOCGRAB), 0)
			_ = f.Close()
			slog.Info("Released input device grab", "node", node)
		}
	}

	g.fds = make(map[string]*os.File)
	g.active = false
}

// IsActive returns whether any devices are currently grabbed.
func (g *InputGrabber) IsActive() bool {
	if g == nil {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active
}
