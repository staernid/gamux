package tasks

import (
	"github.com/staernid/gamux/events"
)

// EventSink defines the emission seam for task lifecycle and progress telemetry.
type EventSink interface {
	Emit(ev events.Event)
}

// BusEventSink adapts tasks events to the central events.Bus.
type BusEventSink struct {
	bus *events.Bus
}

// NewBusEventSink creates an EventSink connected to a Bus.
func NewBusEventSink(bus *events.Bus) *BusEventSink {
	return &BusEventSink{bus: bus}
}

// Emit broadcasts the event over the bus if available.
func (s *BusEventSink) Emit(ev events.Event) {
	if s.bus != nil {
		s.bus.Publish(ev)
	}
}

// NoopEventSink is a silent sink for tests and benchmarks.
type NoopEventSink struct{}

// NewNoopEventSink creates a NoopEventSink.
func NewNoopEventSink() *NoopEventSink {
	return &NoopEventSink{}
}

// Emit does nothing.
func (s *NoopEventSink) Emit(ev events.Event) {}

// ChannelEventSink sends events to a channel for deterministic test assertions.
type ChannelEventSink struct {
	ch chan events.Event
}

// NewChannelEventSink creates a ChannelEventSink with specified buffer size.
func NewChannelEventSink(bufSize int) *ChannelEventSink {
	return &ChannelEventSink{
		ch: make(chan events.Event, bufSize),
	}
}

// Events returns the receive-only channel.
func (s *ChannelEventSink) Events() <-chan events.Event {
	return s.ch
}

// Emit sends the event to the channel non-blockingly.
func (s *ChannelEventSink) Emit(ev events.Event) {
	select {
	case s.ch <- ev:
	default:
	}
}
