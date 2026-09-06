package multiseat

/*
#cgo pkg-config: libei-1.0
#include <libei.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>

typedef struct {
    struct ei *ei;
    struct ei_device *dev;
    int ready;
    uint32_t sequence;
} ei_bridge_t;

static ei_bridge_t* ei_bridge_create(const char *socket_path) {
    if (!socket_path) return NULL;
    struct ei *ei = ei_new_sender(NULL);
    if (!ei) return NULL;
    if (ei_setup_backend_socket(ei, socket_path) < 0) {
        ei_unref(ei);
        return NULL;
    }
    ei_bridge_t *b = (ei_bridge_t*)calloc(1, sizeof(ei_bridge_t));
    if (!b) {
        ei_unref(ei);
        return NULL;
    }
    b->ei = ei;
    return b;
}

static int ei_bridge_pump(ei_bridge_t *b) {
    if (!b || !b->ei) return -1;
    ei_dispatch(b->ei);
    struct ei_event *ev;
    while ((ev = ei_get_event(b->ei))) {
        enum ei_event_type type = ei_event_get_type(ev);
        if (type == EI_EVENT_SEAT_ADDED) {
            struct ei_seat *seat = ei_event_get_seat(ev);
            ei_seat_bind_capabilities(seat,
                EI_DEVICE_CAP_POINTER,
                EI_DEVICE_CAP_POINTER_ABSOLUTE,
                EI_DEVICE_CAP_BUTTON,
                EI_DEVICE_CAP_SCROLL,
                EI_DEVICE_CAP_KEYBOARD,
                NULL);
        } else if (type == EI_EVENT_DEVICE_ADDED) {
            if (b->dev) {
                ei_device_unref(b->dev);
            }
            b->dev = ei_device_ref(ei_event_get_device(ev));
        } else if (type == EI_EVENT_DEVICE_RESUMED) {
            struct ei_device *dev = ei_event_get_device(ev);
            if (!b->dev && dev) {
                b->dev = ei_device_ref(dev);
            }
            if (b->dev) {
                b->sequence++;
                ei_device_start_emulating(b->dev, b->sequence);
                b->ready = 1;
            }
        } else if (type == EI_EVENT_DEVICE_PAUSED) {
            if (b->dev && b->ready) {
                ei_device_stop_emulating(b->dev);
                uint64_t now = ei_now(b->ei);
                ei_device_frame(b->dev, now);
                ei_dispatch(b->ei);
            }
            b->ready = 0;
        } else if (type == EI_EVENT_DEVICE_REMOVED) {
            struct ei_device *dev = ei_event_get_device(ev);
            if (b->dev && (b->dev == dev)) {
                if (b->ready) {
                    b->ready = 0;
                }
                ei_device_unref(b->dev);
                b->dev = NULL;
            }
        } else if (type == EI_EVENT_DISCONNECT) {
            b->ready = 0;
        }
        ei_event_unref(ev);
    }
    return b->ready;
}

static void ei_bridge_motion(ei_bridge_t *b, double dx, double dy) {
    if (!b || !b->dev || !b->ready) return;
    ei_device_pointer_motion(b->dev, dx, dy);
}

static void ei_bridge_motion_absolute(ei_bridge_t *b, double x, double y) {
    if (!b || !b->dev || !b->ready) return;
    ei_device_pointer_motion_absolute(b->dev, x, y);
}

static void ei_bridge_button(ei_bridge_t *b, uint32_t button, uint32_t is_press) {
    if (!b || !b->dev || !b->ready) return;
    ei_device_button_button(b->dev, button, is_press ? true : false);
}

static void ei_bridge_scroll(ei_bridge_t *b, double dx, double dy) {
    if (!b || !b->dev || !b->ready) return;
    ei_device_scroll_delta(b->dev, dx, dy);
}

static void ei_bridge_key(ei_bridge_t *b, uint32_t key, uint32_t is_press) {
    if (!b || !b->dev || !b->ready) return;
    ei_device_keyboard_key(b->dev, key, is_press ? true : false);
    uint64_t now = ei_now(b->ei);
    ei_device_frame(b->dev, now);
    ei_dispatch(b->ei);
}

static void ei_bridge_frame(ei_bridge_t *b) {
    if (!b || !b->dev || !b->ready) return;
    uint64_t now = ei_now(b->ei);
    ei_device_frame(b->dev, now);
    ei_dispatch(b->ei);
}

static void ei_bridge_destroy(ei_bridge_t *b) {
    if (!b) return;
    if (b->dev) {
        if (b->ready) {
            ei_device_stop_emulating(b->dev);
            uint64_t now = ei_now(b->ei);
            ei_device_frame(b->dev, now);
            ei_dispatch(b->ei);
            b->ready = 0;
        }
        ei_device_unref(b->dev);
        b->dev = NULL;
    }
    if (b->ei) {
        ei_unref(b->ei);
        b->ei = NULL;
    }
    free(b);
}
*/
import "C"

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unsafe"
)

// EIClient implements InputInjector using the Freedesktop libei protocol
// to inject pointer and keyboard events directly into Gamescope's internal EIS server.
type EIClient struct {
	mu       sync.Mutex
	bridge   *C.ei_bridge_t
	sockPath string
	closed   bool
	cancel   context.CancelFunc
	width    int
	height   int
	curX     float64
	curY     float64
}

// NewEIClient instantiates an unpopulated EIClient instance with default 1080p geometry.
func NewEIClient() *EIClient {
	return &EIClient{
		width:  1920,
		height: 1080,
		curX:   960,
		curY:   540,
	}
}

// SetDimensions configures the target coordinate boundary for absolute motion clamping
// and initializes the virtual cursor position to the center of the viewport.
func (c *EIClient) SetDimensions(w, h int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if w <= 0 {
		w = 1920
	}
	if h <= 0 {
		h = 1080
	}
	c.width = w
	c.height = h
	c.curX = float64(w) / 2.0
	c.curY = float64(h) / 2.0
}

// GetCursorPos returns the current accumulated (x, y) coordinates of the cursor.
func (c *EIClient) GetCursorPos() (float64, float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.curX, c.curY
}

// Connect dials the Gamescope EIS socket and performs the device handshake.
func (c *EIClient) Connect(sockPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sockPath = sockPath
	c.closed = false

	cSock := C.CString(sockPath)
	defer C.free(unsafe.Pointer(cSock))

	bridge := C.ei_bridge_create(cSock)
	if bridge == nil {
		return fmt.Errorf("failed to create libei sender for socket %s", sockPath)
	}

	// Poll until Gamescope accepts the capabilities and resumes the virtual device
	deadline := time.Now().Add(4 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		rc := C.ei_bridge_pump(bridge)
		if rc == 1 {
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !ready {
		C.ei_bridge_destroy(bridge)
		return fmt.Errorf("timed out waiting for Gamescope EIS device ready on %s", sockPath)
	}

	c.bridge = bridge

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	// Background goroutine to pump incoming EIS messages and stay healthy
	go func() {
		ticker := time.NewTicker(40 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				b := c.bridge
				if b != nil {
					C.ei_bridge_pump(b)
				}
				c.mu.Unlock()
			}
		}
	}()

	slog.Info("Gamescope EIS virtual input connected successfully", "socket", sockPath)
	return nil
}

// SendMotion generates relative pointer motion events.
// Gamescope's internal EIS handler routes this directly to wlserver_mousemotion,
// updating cursor coordinates for 2D desktop navigation while seamlessly supporting
// relative pointer constraints for 3D camera look.
func (c *EIClient) SendMotion(dx, dy int32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.bridge == nil {
		return fmt.Errorf("ei client is closed")
	}

	c.curX += float64(dx)
	c.curY += float64(dy)
	if c.curX < 0 {
		c.curX = 0
	}
	if c.width > 0 && c.curX > float64(c.width) {
		c.curX = float64(c.width)
	}
	if c.curY < 0 {
		c.curY = 0
	}
	if c.height > 0 && c.curY > float64(c.height) {
		c.curY = float64(c.height)
	}

	C.ei_bridge_motion(c.bridge, C.double(dx), C.double(dy))
	return nil
}

// SendMotionAbsolute generates absolute pointer motion events, warping the Gamescope cursor.
func (c *EIClient) SendMotionAbsolute(x, y float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.bridge == nil {
		return fmt.Errorf("ei client is closed")
	}

	c.curX = x
	c.curY = y
	C.ei_bridge_motion_absolute(c.bridge, C.double(x), C.double(y))
	return nil
}

// SendButton generates a mouse button press or release event.
func (c *EIClient) SendButton(button uint32, state uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.bridge == nil {
		return fmt.Errorf("ei client is closed")
	}
	C.ei_bridge_button(c.bridge, C.uint32_t(button), C.uint32_t(state))
	return nil
}

// SendAxis generates a mouse wheel scroll event.
// axis 0 = vertical (dy), axis 1 = horizontal (dx).
// value is in 24.8 fixed point (value / 256.0 = logical pixels).
func (c *EIClient) SendAxis(axis uint32, value int32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.bridge == nil {
		return fmt.Errorf("ei client is closed")
	}

	delta := float64(value) / 256.0
	var dx, dy float64
	if axis == 0 {
		dy = delta
	} else {
		dx = delta
	}

	C.ei_bridge_scroll(c.bridge, C.double(dx), C.double(dy))
	return nil
}

// SendKey generates a keyboard key press or release event.
func (c *EIClient) SendKey(key uint32, state uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.bridge == nil {
		return fmt.Errorf("ei client is closed")
	}
	C.ei_bridge_key(c.bridge, C.uint32_t(key), C.uint32_t(state))
	return nil
}

// SendFrame commits an accumulated batch of pointer events.
func (c *EIClient) SendFrame() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.bridge == nil {
		return fmt.Errorf("ei client is closed")
	}
	C.ei_bridge_frame(c.bridge)
	return nil
}

// Close disconnects and releases the libei sender context.
func (c *EIClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	cancel := c.cancel
	bridge := c.bridge
	c.bridge = nil
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if bridge != nil {
		C.ei_bridge_destroy(bridge)
	}

	slog.Info("Closed Gamescope EIS virtual input connection", "socket", c.sockPath)
	return nil
}
