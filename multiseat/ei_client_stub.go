//go:build !cgo

package multiseat

import (
	"errors"
	"sync"
)

// EIClient is a stub when compiled without CGO/libei support.
type EIClient struct {
	mu     sync.Mutex
	width  int
	height int
	curX   float64
	curY   float64
}

// NewEIClient instantiates a stub EIClient instance with default 1080p center coordinates.
func NewEIClient() *EIClient {
	return &EIClient{
		width:  1920,
		height: 1080,
		curX:   960,
		curY:   540,
	}
}

// SetDimensions sets stub virtual display bounds and centers the cursor.
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

// GetCursorPos returns current stub coordinates.
func (c *EIClient) GetCursorPos() (float64, float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.curX, c.curY
}

// Connect returns an error indicating CGO/libei is required.
func (c *EIClient) Connect(sockPath string) error {
	return errors.New("multiseat EI client requires CGO with libei-1.0")
}

// SendMotion returns an error indicating CGO/libei is required.
func (c *EIClient) SendMotion(dx, dy int32) error {
	return errors.New("multiseat EI client requires CGO with libei-1.0")
}

// SendMotionAbsolute returns an error indicating CGO/libei is required.
func (c *EIClient) SendMotionAbsolute(x, y float64) error {
	return errors.New("multiseat EI client requires CGO with libei-1.0")
}

// SendButton returns an error indicating CGO/libei is required.
func (c *EIClient) SendButton(button uint32, state uint32) error {
	return errors.New("multiseat EI client requires CGO with libei-1.0")
}

// SendAxis returns an error indicating CGO/libei is required.
func (c *EIClient) SendAxis(axis uint32, value int32) error {
	return errors.New("multiseat EI client requires CGO with libei-1.0")
}

// SendKey returns an error indicating CGO/libei is required.
func (c *EIClient) SendKey(key uint32, state uint32) error {
	return errors.New("multiseat EI client requires CGO with libei-1.0")
}

// SendFrame returns an error since stub client is never connected.
func (c *EIClient) SendFrame() error {
	return errors.New("ei bridge not connected")
}

// Close is a no-op stub.
func (c *EIClient) Close() error {
	return nil
}
