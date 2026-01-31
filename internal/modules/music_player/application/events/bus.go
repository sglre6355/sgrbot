package events

import (
	"log/slog"
	"sync"
)

// DefaultEventBufferSize is the default buffer size for event channels.
const DefaultEventBufferSize = 100

// Bus provides a channel-based event bus for async event handling.
type Bus struct {
	trackEnqueued    chan TrackEnqueuedEvent
	playbackStarted  chan PlaybackStartedEvent
	playbackFinished chan PlaybackFinishedEvent
	trackEnded       chan TrackEndedEvent

	closed bool
	mu     sync.RWMutex
}

// NewBus creates a new Bus with the given buffer size.
func NewBus(bufferSize int) *Bus {
	if bufferSize <= 0 {
		bufferSize = DefaultEventBufferSize
	}

	return &Bus{
		trackEnqueued:    make(chan TrackEnqueuedEvent, bufferSize),
		playbackStarted:  make(chan PlaybackStartedEvent, bufferSize),
		playbackFinished: make(chan PlaybackFinishedEvent, bufferSize),
		trackEnded:       make(chan TrackEndedEvent, bufferSize),
	}
}

// Publish sends an event to the appropriate channel.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		slog.Warn("attempted to publish to closed event bus", "event", event.eventType())
		return
	}

	switch e := event.(type) {
	case TrackEnqueuedEvent:
		select {
		case b.trackEnqueued <- e:
			slog.Debug("published event", "type", e.eventType(), "guild", e.GuildID)
		default:
			slog.Warn("event buffer full, dropping event", "type", e.eventType())
		}

	case PlaybackStartedEvent:
		select {
		case b.playbackStarted <- e:
			slog.Debug("published event", "type", e.eventType(), "guild", e.GuildID)
		default:
			slog.Warn("event buffer full, dropping event", "type", e.eventType())
		}

	case PlaybackFinishedEvent:
		select {
		case b.playbackFinished <- e:
			slog.Debug("published event", "type", e.eventType(), "guild", e.GuildID)
		default:
			slog.Warn("event buffer full, dropping event", "type", e.eventType())
		}

	case TrackEndedEvent:
		select {
		case b.trackEnded <- e:
			slog.Debug("published event", "type", e.eventType(), "guild", e.GuildID)
		default:
			slog.Warn("event buffer full, dropping event", "type", e.eventType())
		}

	default:
		slog.Warn("unknown event type", "event", event)
	}
}

// TrackEnqueued returns the channel for TrackEnqueuedEvent.
func (b *Bus) TrackEnqueued() <-chan TrackEnqueuedEvent {
	return b.trackEnqueued
}

// PlaybackStarted returns the channel for PlaybackStartedEvent.
func (b *Bus) PlaybackStarted() <-chan PlaybackStartedEvent {
	return b.playbackStarted
}

// PlaybackFinished returns the channel for PlaybackFinishedEvent.
func (b *Bus) PlaybackFinished() <-chan PlaybackFinishedEvent {
	return b.playbackFinished
}

// TrackEnded returns the channel for TrackEndedEvent.
func (b *Bus) TrackEnded() <-chan TrackEndedEvent {
	return b.trackEnded
}

// Close closes all event channels.
// After calling Close, Publish will no longer send events.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	close(b.trackEnqueued)
	close(b.playbackStarted)
	close(b.playbackFinished)
	close(b.trackEnded)

	slog.Debug("event bus closed")
}
