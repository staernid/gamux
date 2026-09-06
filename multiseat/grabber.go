package multiseat

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// EVIOCGRAB is the ioctl code (_IOW('E', 0x90, int)) to request exclusive access to an evdev device on Linux.
const EVIOCGRAB = 0x40044590

// Linux input event subsystem constants (from linux/input.h / linux/input-event-codes.h).
const (
	EV_SYN uint16 = 0x00
	EV_KEY uint16 = 0x01
	EV_REL uint16 = 0x02

	SYN_REPORT uint16 = 0x00

	REL_X      uint16 = 0x00
	REL_Y      uint16 = 0x01
	REL_HWHEEL uint16 = 0x06
	REL_WHEEL  uint16 = 0x08

	BTN_MISC     uint16 = 0x100
	BTN_MOUSE    uint16 = 0x110
	BTN_JOYSTICK uint16 = 0x120
)

// LinuxInputEvent represents a 24-byte evdev input_event struct on 64-bit Linux architectures.
type LinuxInputEvent struct {
	TimeSec  int64
	TimeUsec int64
	Type     uint16
	Code     uint16
	Value    int32
}

// DecodeLinuxInputEvent parses a raw 24-byte buffer into a LinuxInputEvent.
func DecodeLinuxInputEvent(buf []byte) (LinuxInputEvent, error) {
	if len(buf) < 24 {
		return LinuxInputEvent{}, fmt.Errorf("buffer too short for input_event: got %d bytes, need 24", len(buf))
	}
	return LinuxInputEvent{
		TimeSec:  int64(binary.LittleEndian.Uint64(buf[0:8])),
		TimeUsec: int64(binary.LittleEndian.Uint64(buf[8:16])),
		Type:     binary.LittleEndian.Uint16(buf[16:18]),
		Code:     binary.LittleEndian.Uint16(buf[18:20]),
		Value:    int32(binary.LittleEndian.Uint32(buf[20:24])),
	}, nil
}

// InputInjector represents an abstract display server input sink capable of receiving
// synthetic pointer, scrolling, button, and keyboard events.
type InputInjector interface {
	SendMotion(dx, dy int32) error
	SendButton(button uint32, state uint32) error
	SendAxis(axis uint32, value int32) error
	SendFrame() error
	SendKey(key uint32, state uint32) error
	Close() error
}

// InputGrabber manages exclusive kernel grabs (EVIOCGRAB) on assigned input devices
// and runs an active event reading loop routing events to an InputInjector.
type InputGrabber struct {
	mu       sync.Mutex
	fds      map[string]*os.File
	nodes    []string
	active   bool
	injector InputInjector
	cancel   context.CancelFunc
	ctx      context.Context
	wg       sync.WaitGroup
	onEvent  func(node string, ev LinuxInputEvent) // Optional listener for telemetry/tests
}

// NewInputGrabber instantiates an input grabber for specified device nodes.
// For each assigned device node, it automatically discovers and binds all companion
// sibling event nodes belonging to the same physical hardware device (e.g. composite
// keyboards and mice with multiple interfaces) to ensure complete capture and isolation.
func NewInputGrabber(devNodes ...string) *InputGrabber {
	var valid []string
	seen := make(map[string]bool)

	addNode := func(p string) {
		if !isExplicitDeviceNode(p) {
			return
		}
		realP := p
		if resolved, err := filepath.EvalSymlinks(p); err == nil && resolved != "" {
			realP = resolved
		}
		if !seen[realP] {
			valid = append(valid, realP)
			seen[realP] = true
		}
	}

	for _, n := range devNodes {
		if !isExplicitDeviceNode(n) {
			continue
		}
		addNode(n)

		// Discover sibling companion event nodes for this physical device
		for _, sib := range FindDeviceCompanionNodes(n) {
			addNode(sib)
		}
	}

	return &InputGrabber{
		fds:   make(map[string]*os.File),
		nodes: valid,
	}
}

// SetInjector attaches an input injector (such as EIClient or WaylandClient) to receive forwarded input events.
func (g *InputGrabber) SetInjector(injector InputInjector) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.injector = injector
}

// SetWaylandClient attaches a Wayland client to receive forwarded input events.
func (g *InputGrabber) SetWaylandClient(client *WaylandClient) {
	g.SetInjector(client)
}

// SetEventListener registers a callback invoked on each received evdev event.
func (g *InputGrabber) SetEventListener(fn func(node string, ev LinuxInputEvent)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onEvent = fn
}

// Grab takes exclusive ownership of configured devices via EVIOCGRAB and starts active event loops.
// When grabbed, the host OS (Player 1) receives zero events from these devices.
func (g *InputGrabber) Grab() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.active {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	g.ctx = ctx
	g.cancel = cancel

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

		// Start background event loop for this grabbed node
		g.wg.Add(1)
		go g.readLoop(ctx, node, f)
	}

	g.active = len(g.fds) > 0
	return nil
}

// readLoop actively reads 24-byte input_event packets from the grabbed device and forwards them.
func (g *InputGrabber) readLoop(ctx context.Context, node string, f *os.File) {
	defer g.wg.Done()

	buf := make([]byte, 24)
	var pendingDX, pendingDY int32
	var hasPointerEvent bool

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, err := io.ReadFull(f, buf)
		if err != nil {
			// Expected on close/release
			return
		}

		ev, err := DecodeLinuxInputEvent(buf)
		if err != nil {
			continue
		}

		g.mu.Lock()
		inj := g.injector
		listener := g.onEvent
		g.mu.Unlock()

		if listener != nil {
			listener(node, ev)
		}

		if inj == nil {
			continue
		}

		switch ev.Type {
		case EV_REL:
			switch ev.Code {
			case REL_X:
				pendingDX += ev.Value
			case REL_Y:
				pendingDY += ev.Value
			case REL_WHEEL:
				// Vertical scroll wheel: In Wayland/EIS, scrolling down is positive.
				// Invert Linux REL_WHEEL so wheel-down scrolls down (positive).
				// 15 pixels (in 24.8 fixed point: 15 * 256 = 3840) per click.
				val := -ev.Value * 15 * 256
				_ = inj.SendAxis(0, val)
				hasPointerEvent = true
			case REL_HWHEEL:
				// Horizontal scroll wheel
				val := ev.Value * 15 * 256
				_ = inj.SendAxis(1, val)
				hasPointerEvent = true
			}

		case EV_KEY:
			if ev.Code >= BTN_MOUSE && ev.Code < BTN_JOYSTICK {
				// Mouse button
				var state uint32
				if ev.Value != 0 {
					state = 1
				}
				_ = inj.SendButton(uint32(ev.Code), state)
				hasPointerEvent = true
			} else if ev.Code < BTN_MISC || (ev.Code >= 0x160 && ev.Code < 0x2ff) {
				// Keyboard key (state: 0 = release, 1 = press)
				if ev.Value == 2 {
					// Ignore evdev kernel autorepeat events; Wayland/Xwayland manages client-side repeat
					continue
				}
				var state uint32
				if ev.Value != 0 {
					state = 1
				}
				_ = inj.SendKey(uint32(ev.Code), state)
			}

		case EV_SYN:
			if ev.Code == SYN_REPORT {
				if pendingDX != 0 || pendingDY != 0 {
					_ = inj.SendMotion(pendingDX, pendingDY)
					pendingDX, pendingDY = 0, 0
					hasPointerEvent = true
				}
				if hasPointerEvent {
					_ = inj.SendFrame()
					hasPointerEvent = false
				}
			}
		}
	}
}

// Release releases all grabbed input devices back to the host system and closes the input injector.
func (g *InputGrabber) Release() {
	if g == nil {
		return
	}
	g.mu.Lock()
	if !g.active && len(g.fds) == 0 && g.injector == nil {
		g.mu.Unlock()
		return
	}

	if g.cancel != nil {
		g.cancel()
	}

	for node, f := range g.fds {
		if f != nil {
			// EVIOCGRAB (arg = 0 to release)
			_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(EVIOCGRAB), 0)
			_ = f.Close()
			slog.Info("Released input device grab", "node", node)
		}
	}

	inj := g.injector
	g.injector = nil
	g.fds = make(map[string]*os.File)
	g.active = false
	g.mu.Unlock()

	g.wg.Wait()

	if inj != nil {
		_ = inj.Close()
	}
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
