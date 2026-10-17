package multiseat

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Minimal standard XKB keymap used to initialize virtual keyboards under wlroots/Cage.
const defaultXkbKeymap = `xkb_keymap {
	xkb_keycodes  { include "evdev+aliases(qwerty)" };
	xkb_types     { include "complete" };
	xkb_compat    { include "complete" };
	xkb_symbols   { include "pc+us+inet(evdev)" };
};
`

// WaylandClient provides a pure Go client to inject pointer and keyboard events
// into a Wayland compositor (such as Cage) using zwlr_virtual_pointer_v1 and zwp_virtual_keyboard_v1.
type WaylandClient struct {
	mu       sync.Mutex
	conn     *net.UnixConn
	nextID   uint32
	seatID   uint32
	ptrMgrID uint32
	kbdMgrID uint32
	ptrID    uint32
	kbdID    uint32
	closed   bool
	sockPath string
}

// NewWaylandClient initializes a new unpopulated WaylandClient instance.
func NewWaylandClient() *WaylandClient {
	return &WaylandClient{
		nextID: 1, // 1 is reserved for wl_display
	}
}

func (c *WaylandClient) allocID() uint32 {
	c.nextID++
	return c.nextID
}

// send transmits a Wayland request without file descriptor passing.
// Wire format: [objID uint32, opcode uint16, size uint16, payload...]
func (c *WaylandClient) send(objID uint32, opcode uint16, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return fmt.Errorf("wayland client is closed or not connected")
	}

	size := uint16(8 + len(payload))
	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:4], objID)
	binary.LittleEndian.PutUint16(buf[4:6], opcode)
	binary.LittleEndian.PutUint16(buf[6:8], size)
	if len(payload) > 0 {
		copy(buf[8:], payload)
	}

	_, err := c.conn.Write(buf)
	return err
}

// sendWithFD transmits a Wayland request along with an out-of-band file descriptor (SCM_RIGHTS).
func (c *WaylandClient) sendWithFD(objID uint32, opcode uint16, payload []byte, fd int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return fmt.Errorf("wayland client is closed or not connected")
	}

	size := uint16(8 + len(payload))
	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:4], objID)
	binary.LittleEndian.PutUint16(buf[4:6], opcode)
	binary.LittleEndian.PutUint16(buf[6:8], size)
	if len(payload) > 0 {
		copy(buf[8:], payload)
	}

	oob := syscall.UnixRights(fd)
	_, _, err := c.conn.WriteMsgUnix(buf, oob, nil)
	return err
}

// encodeString encodes a null-terminated string padded to a 32-bit boundary.
func encodeString(s string) []byte {
	strBytes := append([]byte(s), 0)
	l := uint32(len(strBytes))
	pad := (4 - (l % 4)) % 4
	out := make([]byte, 4+l+pad)
	binary.LittleEndian.PutUint32(out[0:4], l)
	copy(out[4:], strBytes)
	return out
}

// Connect dials the Wayland compositor socket and negotiates virtual input protocols.
func (c *WaylandClient) Connect(sockPath string) error {
	c.mu.Lock()
	c.sockPath = sockPath
	c.closed = false
	c.mu.Unlock()

	raddr, err := net.ResolveUnixAddr("unix", sockPath)
	if err != nil {
		return fmt.Errorf("resolve wayland socket address %s: %w", sockPath, err)
	}

	conn, err := net.DialUnix("unix", nil, raddr)
	if err != nil {
		return fmt.Errorf("dial wayland socket %s: %w", sockPath, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.nextID = 1
	c.mu.Unlock()

	// 1. Request registry (wl_display opcode 1)
	regID := c.allocID()
	var regPayload [4]byte
	binary.LittleEndian.PutUint32(regPayload[:], regID)
	if err := c.send(1, 1, regPayload[:]); err != nil {
		_ = conn.Close()
		return fmt.Errorf("get registry: %w", err)
	}

	// 2. Request sync callback (wl_display opcode 0) to wait until all globals are announced
	syncID := c.allocID()
	var syncPayload [4]byte
	binary.LittleEndian.PutUint32(syncPayload[:], syncID)
	if err := c.send(1, 0, syncPayload[:]); err != nil {
		_ = conn.Close()
		return fmt.Errorf("display sync: %w", err)
	}

	// 3. Read registry globals
	header := make([]byte, 8)
	var seatName, ptrMgrName, kbdMgrName uint32

	for {
		if _, err := io.ReadFull(conn, header); err != nil {
			_ = conn.Close()
			return fmt.Errorf("read wayland event header: %w", err)
		}

		objID := binary.LittleEndian.Uint32(header[0:4])
		opcode := binary.LittleEndian.Uint16(header[4:6])
		size := binary.LittleEndian.Uint16(header[6:8])

		if size < 8 {
			_ = conn.Close()
			return fmt.Errorf("invalid wayland message size: %d", size)
		}

		bodySize := int(size) - 8
		body := make([]byte, bodySize)
		if bodySize > 0 {
			if _, err := io.ReadFull(conn, body); err != nil {
				_ = conn.Close()
				return fmt.Errorf("read wayland event body: %w", err)
			}
		}

		// Check for wl_display.error (objID 1, opcode 0)
		if objID == 1 && opcode == 0 && bodySize >= 12 {
			errObj := binary.LittleEndian.Uint32(body[0:4])
			errCode := binary.LittleEndian.Uint32(body[4:8])
			strLen := binary.LittleEndian.Uint32(body[8:12])
			msg := ""
			if bodySize >= int(12+strLen) && strLen > 1 {
				msg = string(body[12 : 12+strLen-1])
			}
			_ = conn.Close()
			return fmt.Errorf("wayland compositor error on obj %d (code %d): %s", errObj, errCode, msg)
		}

		// Check for wl_callback.done (sync complete)
		if objID == syncID && opcode == 0 {
			break
		}

		// Check for wl_registry.global (objID regID, opcode 0)
		if objID == regID && opcode == 0 && bodySize >= 8 {
			name := binary.LittleEndian.Uint32(body[0:4])
			strLen := binary.LittleEndian.Uint32(body[4:8])
			if bodySize >= int(8+strLen) && strLen > 1 {
				iface := string(body[8 : 8+strLen-1])
				switch iface {
				case "wl_seat":
					seatName = name
				case "zwlr_virtual_pointer_manager_v1":
					ptrMgrName = name
				case "zwp_virtual_keyboard_manager_v1":
					kbdMgrName = name
				}
			}
		}
	}

	if ptrMgrName == 0 {
		_ = conn.Close()
		return fmt.Errorf("compositor at %s does not export zwlr_virtual_pointer_manager_v1", sockPath)
	}
	if kbdMgrName == 0 {
		_ = conn.Close()
		return fmt.Errorf("compositor at %s does not export zwp_virtual_keyboard_manager_v1", sockPath)
	}

	// 4. Bind globals: wl_seat (if available), pointer manager, keyboard manager
	var seatObjID uint32
	if seatName != 0 {
		seatObjID = c.allocID()
		bindSeat := make([]byte, 4)
		binary.LittleEndian.PutUint32(bindSeat[0:4], seatName)
		bindSeat = append(bindSeat, encodeString("wl_seat")...)
		var vAndID [8]byte
		binary.LittleEndian.PutUint32(vAndID[0:4], 1)
		binary.LittleEndian.PutUint32(vAndID[4:8], seatObjID)
		bindSeat = append(bindSeat, vAndID[:]...)
		if err := c.send(regID, 0, bindSeat); err != nil {
			_ = conn.Close()
			return fmt.Errorf("bind wl_seat: %w", err)
		}
	}

	ptrMgrObjID := c.allocID()
	bindPtr := make([]byte, 4)
	binary.LittleEndian.PutUint32(bindPtr[0:4], ptrMgrName)
	bindPtr = append(bindPtr, encodeString("zwlr_virtual_pointer_manager_v1")...)
	var vAndID [8]byte
	binary.LittleEndian.PutUint32(vAndID[0:4], 1)
	binary.LittleEndian.PutUint32(vAndID[4:8], ptrMgrObjID)
	bindPtr = append(bindPtr, vAndID[:]...)
	if err := c.send(regID, 0, bindPtr); err != nil {
		_ = conn.Close()
		return fmt.Errorf("bind zwlr_virtual_pointer_manager_v1: %w", err)
	}

	kbdMgrObjID := c.allocID()
	bindKbd := make([]byte, 4)
	binary.LittleEndian.PutUint32(bindKbd[0:4], kbdMgrName)
	bindKbd = append(bindKbd, encodeString("zwp_virtual_keyboard_manager_v1")...)
	binary.LittleEndian.PutUint32(vAndID[0:4], 1)
	binary.LittleEndian.PutUint32(vAndID[4:8], kbdMgrObjID)
	bindKbd = append(bindKbd, vAndID[:]...)
	if err := c.send(regID, 0, bindKbd); err != nil {
		_ = conn.Close()
		return fmt.Errorf("bind zwp_virtual_keyboard_manager_v1: %w", err)
	}

	// 5. Create virtual pointer (zwlr_virtual_pointer_manager_v1.create_virtual_pointer opcode 0)
	ptrObjID := c.allocID()
	var createPtrPayload [8]byte
	binary.LittleEndian.PutUint32(createPtrPayload[0:4], seatObjID) // seat or 0 (compositor default)
	binary.LittleEndian.PutUint32(createPtrPayload[4:8], ptrObjID)
	if err := c.send(ptrMgrObjID, 0, createPtrPayload[:]); err != nil {
		_ = conn.Close()
		return fmt.Errorf("create virtual pointer: %w", err)
	}

	// 6. Create virtual keyboard (zwp_virtual_keyboard_manager_v1.create_virtual_keyboard opcode 0)
	kbdObjID := c.allocID()
	var createKbdPayload [8]byte
	binary.LittleEndian.PutUint32(createKbdPayload[0:4], seatObjID)
	binary.LittleEndian.PutUint32(createKbdPayload[4:8], kbdObjID)
	if err := c.send(kbdMgrObjID, 0, createKbdPayload[:]); err != nil {
		_ = conn.Close()
		return fmt.Errorf("create virtual keyboard: %w", err)
	}

	// 7. Upload standard XKB keymap via memfd and SCM_RIGHTS
	kmBytes := []byte(defaultXkbKeymap)
	kmLen := len(kmBytes)
	memfd, err := unix.MemfdCreate("gamux_keymap", unix.MFD_CLOEXEC)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("create keymap memfd: %w", err)
	}
	defer func() { _ = syscall.Close(memfd) }()

	if _, err := syscall.Write(memfd, kmBytes); err != nil {
		_ = conn.Close()
		return fmt.Errorf("write keymap to memfd: %w", err)
	}

	// zwp_virtual_keyboard_v1.keymap: opcode 0, format=1 (WL_KEYBOARD_KEYMAP_FORMAT_XKB_V1), fd, size
	var kmPayload [8]byte
	binary.LittleEndian.PutUint32(kmPayload[0:4], 1)
	binary.LittleEndian.PutUint32(kmPayload[4:8], uint32(kmLen))
	if err := c.sendWithFD(kbdObjID, 0, kmPayload[:], memfd); err != nil {
		_ = conn.Close()
		return fmt.Errorf("upload keymap to virtual keyboard: %w", err)
	}

	c.mu.Lock()
	c.seatID = seatObjID
	c.ptrMgrID = ptrMgrObjID
	c.kbdMgrID = kbdMgrObjID
	c.ptrID = ptrObjID
	c.kbdID = kbdObjID
	c.mu.Unlock()

	slog.Info("Wayland virtual input protocols initialized successfully",
		"socket", sockPath, "ptr_id", ptrObjID, "kbd_id", kbdObjID)
	return nil
}

// SendMotion emits a relative pointer motion event.
// dx and dy are integer delta displacements converted to 24.8 fixed point.
func (c *WaylandClient) SendMotion(dx, dy int32) error {
	c.mu.Lock()
	ptrID := c.ptrID
	c.mu.Unlock()
	if ptrID == 0 {
		return fmt.Errorf("virtual pointer not initialized")
	}

	// zwlr_virtual_pointer_v1.motion: opcode 0, time (uint), dx (fixed 24.8), dy (fixed 24.8)
	var payload [12]byte
	nowMs := uint32(time.Now().UnixMilli())
	binary.LittleEndian.PutUint32(payload[0:4], nowMs)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(dx<<8))
	binary.LittleEndian.PutUint32(payload[8:12], uint32(dy<<8))
	return c.send(ptrID, 0, payload[:])
}

// SendButton emits a mouse button press or release event.
// button is the Linux evdev button code (e.g. 0x110 / 272 for BTN_LEFT).
// state is 0 for released, 1 for pressed.
func (c *WaylandClient) SendButton(button uint32, state uint32) error {
	c.mu.Lock()
	ptrID := c.ptrID
	c.mu.Unlock()
	if ptrID == 0 {
		return fmt.Errorf("virtual pointer not initialized")
	}

	// zwlr_virtual_pointer_v1.button: opcode 2, time (uint), button (uint), state (uint)
	var payload [12]byte
	nowMs := uint32(time.Now().UnixMilli())
	binary.LittleEndian.PutUint32(payload[0:4], nowMs)
	binary.LittleEndian.PutUint32(payload[4:8], button)
	binary.LittleEndian.PutUint32(payload[8:12], state)
	return c.send(ptrID, 2, payload[:])
}

// SendAxis emits a pointer scroll axis movement event.
// axis is 0 for vertical scroll, 1 for horizontal scroll.
// value is the displacement in 24.8 fixed point.
func (c *WaylandClient) SendAxis(axis uint32, value int32) error {
	c.mu.Lock()
	ptrID := c.ptrID
	c.mu.Unlock()
	if ptrID == 0 {
		return fmt.Errorf("virtual pointer not initialized")
	}

	// zwlr_virtual_pointer_v1.axis: opcode 3, time (uint), axis (uint), value (fixed 24.8)
	var payload [12]byte
	nowMs := uint32(time.Now().UnixMilli())
	binary.LittleEndian.PutUint32(payload[0:4], nowMs)
	binary.LittleEndian.PutUint32(payload[4:8], axis)
	binary.LittleEndian.PutUint32(payload[8:12], uint32(value))
	return c.send(ptrID, 3, payload[:])
}

// SendFrame commits an accumulated batch of pointer events (motion, buttons, scroll).
func (c *WaylandClient) SendFrame() error {
	c.mu.Lock()
	ptrID := c.ptrID
	c.mu.Unlock()
	if ptrID == 0 {
		return fmt.Errorf("virtual pointer not initialized")
	}

	// zwlr_virtual_pointer_v1.frame: opcode 4, no payload
	return c.send(ptrID, 4, nil)
}

// SendKey emits a keyboard key press or release event.
// key is the Linux evdev keycode (e.g. KEY_A = 30).
// state is 0 for released, 1 for pressed.
func (c *WaylandClient) SendKey(key uint32, state uint32) error {
	c.mu.Lock()
	kbdID := c.kbdID
	c.mu.Unlock()
	if kbdID == 0 {
		return fmt.Errorf("virtual keyboard not initialized")
	}

	// zwp_virtual_keyboard_v1.key: opcode 1, time (uint), key (uint), state (uint)
	var payload [12]byte
	nowMs := uint32(time.Now().UnixMilli())
	binary.LittleEndian.PutUint32(payload[0:4], nowMs)
	binary.LittleEndian.PutUint32(payload[4:8], key)
	binary.LittleEndian.PutUint32(payload[8:12], state)
	return c.send(kbdID, 1, payload[:])
}

// Close gracefully releases the Wayland client connection and virtual objects.
func (c *WaylandClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true

	conn := c.conn
	ptrID := c.ptrID
	kbdID := c.kbdID
	ptrMgrID := c.ptrMgrID
	kbdMgrID := c.kbdMgrID
	c.conn = nil
	c.ptrID = 0
	c.kbdID = 0
	c.mu.Unlock()

	if conn == nil {
		return nil
	}

	// Send destructors if possible before closing socket
	if ptrID != 0 {
		// zwlr_virtual_pointer_v1.destroy: opcode 8
		_ = c.send(ptrID, 8, nil)
	}
	if kbdID != 0 {
		// zwp_virtual_keyboard_v1.destroy: opcode 3
		_ = c.send(kbdID, 3, nil)
	}
	if ptrMgrID != 0 {
		// zwlr_virtual_pointer_manager_v1.destroy: opcode 1
		_ = c.send(ptrMgrID, 1, nil)
	}
	_ = kbdMgrID // zwp_virtual_keyboard_manager_v1 has no destroy request in v1

	err := conn.Close()
	slog.Info("Closed Wayland virtual input connection", "socket", c.sockPath)
	return err
}
