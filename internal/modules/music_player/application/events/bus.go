package events

import (
	"log/slog"
	"sync"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

// DefaultEventBufferSize is the default buffer size for event channels.
const DefaultEventBufferSize = 100

// Compile-time check that Bus implements ports.EventPublisher.
var _ ports.EventPublisher = (*Bus)(nil)

// Bus provides a channel-based event bus for async event handling.
type Bus struct {
	trackEnqueued    chan TrackEnqueuedEvent
	playbackStarted  chan PlaybackStartedEvent
	playbackFinished chan PlaybackFinishedEvent
	trackEnded       chan TrackEndedEvent
	queueCleared     chan QueueClearedEvent

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
		queueCleared:     make(chan QueueClearedEvent, bufferSize),
	}
}

// PublishTrackEnqueued publishes a TrackEnqueuedEvent.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *Bus) PublishTrackEnqueued(event TrackEnqueuedEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		slog.Warn("attempted to publish to closed event bus", "type", "TrackEnqueued")
		return
	}

	select {
	case b.trackEnqueued <- event:
		slog.Debug("published event", "type", "TrackEnqueued", "guild", event.GuildID)
	default:
		slog.Warn("event buffer full, dropping event", "type", "TrackEnqueued")
	}
}

// PublishPlaybackStarted publishes a PlaybackStartedEvent.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *Bus) PublishPlaybackStarted(event PlaybackStartedEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		slog.Warn("attempted to publish to closed event bus", "type", "PlaybackStarted")
		return
	}

	select {
	case b.playbackStarted <- event:
		slog.Debug("published event", "type", "PlaybackStarted", "guild", event.GuildID)
	default:
		slog.Warn("event buffer full, dropping event", "type", "PlaybackStarted")
	}
}

// PublishPlaybackFinished publishes a PlaybackFinishedEvent.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *Bus) PublishPlaybackFinished(event PlaybackFinishedEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		slog.Warn("attempted to publish to closed event bus", "type", "PlaybackFinished")
		return
	}

	select {
	case b.playbackFinished <- event:
		slog.Debug("published event", "type", "PlaybackFinished", "guild", event.GuildID)
	default:
		slog.Warn("event buffer full, dropping event", "type", "PlaybackFinished")
	}
}

// PublishTrackEnded publishes a TrackEndedEvent.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *Bus) PublishTrackEnded(event TrackEndedEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		slog.Warn("attempted to publish to closed event bus", "type", "TrackEnded")
		return
	}

	select {
	case b.trackEnded <- event:
		slog.Debug("published event", "type", "TrackEnded", "guild", event.GuildID)
	default:
		slog.Warn("event buffer full, dropping event", "type", "TrackEnded")
	}
}

// PublishQueueCleared publishes a QueueClearedEvent.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *Bus) PublishQueueCleared(event QueueClearedEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		slog.Warn("attempted to publish to closed event bus", "type", "QueueCleared")
		return
	}

	select {
	case b.queueCleared <- event:
		slog.Debug("published event", "type", "QueueCleared", "guild", event.GuildID)
	default:
		slog.Warn("event buffer full, dropping event", "type", "QueueCleared")
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

// QueueCleared returns the channel for QueueClearedEvent.
func (b *Bus) QueueCleared() <-chan QueueClearedEvent {
	return b.queueCleared
}

// Close closes all event channels.
// After calling Close, publishing will no longer send events.
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
	close(b.queueCleared)

	slog.Debug("event bus closed")
}
