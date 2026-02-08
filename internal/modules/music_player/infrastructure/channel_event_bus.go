package infrastructure

import (
	"context"
	"log/slog"
	"sync"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// DefaultEventBufferSize is the default buffer size for event channels.
const DefaultEventBufferSize = 100

// Compile-time checks that ChannelEventBus implements ports interfaces.
var (
	_ ports.EventPublisher  = (*ChannelEventBus)(nil)
	_ ports.EventSubscriber = (*ChannelEventBus)(nil)
)

// ChannelEventBus provides a channel-based event bus for async event handling.
// It implements both EventPublisher and EventSubscriber interfaces.
type ChannelEventBus struct {
	// Channels for event delivery
	trackEnqueued    chan domain.TrackEnqueuedEvent
	playbackStarted  chan domain.PlaybackStartedEvent
	playbackFinished chan domain.PlaybackFinishedEvent
	trackEnded       chan domain.TrackEndedEvent
	queueCleared     chan domain.QueueClearedEvent

	// Handler slices for callback-based subscription
	trackEnqueuedHandlers    []func(context.Context, domain.TrackEnqueuedEvent)
	playbackStartedHandlers  []func(context.Context, domain.PlaybackStartedEvent)
	playbackFinishedHandlers []func(context.Context, domain.PlaybackFinishedEvent)
	trackEndedHandlers       []func(context.Context, domain.TrackEndedEvent)
	queueClearedHandlers     []func(context.Context, domain.QueueClearedEvent)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed bool
	mu     sync.RWMutex
}

// NewChannelEventBus creates a new ChannelEventBus with the given buffer size.
func NewChannelEventBus(bufferSize int) *ChannelEventBus {
	if bufferSize <= 0 {
		bufferSize = DefaultEventBufferSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &ChannelEventBus{
		trackEnqueued:    make(chan domain.TrackEnqueuedEvent, bufferSize),
		playbackStarted:  make(chan domain.PlaybackStartedEvent, bufferSize),
		playbackFinished: make(chan domain.PlaybackFinishedEvent, bufferSize),
		trackEnded:       make(chan domain.TrackEndedEvent, bufferSize),
		queueCleared:     make(chan domain.QueueClearedEvent, bufferSize),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start dispatcher goroutines
	bus.startDispatchers()

	return bus
}

// startDispatchers starts goroutines that dispatch events to registered handlers.
func (b *ChannelEventBus) startDispatchers() {
	b.wg.Add(5)

	go b.dispatchTrackEnqueued()
	go b.dispatchPlaybackStarted()
	go b.dispatchPlaybackFinished()
	go b.dispatchTrackEnded()
	go b.dispatchQueueCleared()
}

func (b *ChannelEventBus) dispatchTrackEnqueued() {
	defer b.wg.Done()
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-b.trackEnqueued:
			if !ok {
				return
			}
			b.mu.RLock()
			handlers := b.trackEnqueuedHandlers
			b.mu.RUnlock()
			for _, handler := range handlers {
				handler(b.ctx, event)
			}
		}
	}
}

func (b *ChannelEventBus) dispatchPlaybackStarted() {
	defer b.wg.Done()
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-b.playbackStarted:
			if !ok {
				return
			}
			b.mu.RLock()
			handlers := b.playbackStartedHandlers
			b.mu.RUnlock()
			for _, handler := range handlers {
				handler(b.ctx, event)
			}
		}
	}
}

func (b *ChannelEventBus) dispatchPlaybackFinished() {
	defer b.wg.Done()
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-b.playbackFinished:
			if !ok {
				return
			}
			b.mu.RLock()
			handlers := b.playbackFinishedHandlers
			b.mu.RUnlock()
			for _, handler := range handlers {
				handler(b.ctx, event)
			}
		}
	}
}

func (b *ChannelEventBus) dispatchTrackEnded() {
	defer b.wg.Done()
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-b.trackEnded:
			if !ok {
				return
			}
			b.mu.RLock()
			handlers := b.trackEndedHandlers
			b.mu.RUnlock()
			for _, handler := range handlers {
				handler(b.ctx, event)
			}
		}
	}
}

func (b *ChannelEventBus) dispatchQueueCleared() {
	defer b.wg.Done()
	for {
		select {
		case <-b.ctx.Done():
			return
		case event, ok := <-b.queueCleared:
			if !ok {
				return
			}
			b.mu.RLock()
			handlers := b.queueClearedHandlers
			b.mu.RUnlock()
			for _, handler := range handlers {
				handler(b.ctx, event)
			}
		}
	}
}

// --- EventPublisher interface ---

// PublishTrackEnqueued publishes a TrackEnqueuedEvent.
// Non-blocking: if the channel buffer is full, the event is dropped with a warning.
func (b *ChannelEventBus) PublishTrackEnqueued(event domain.TrackEnqueuedEvent) {
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
func (b *ChannelEventBus) PublishPlaybackStarted(event domain.PlaybackStartedEvent) {
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
func (b *ChannelEventBus) PublishPlaybackFinished(event domain.PlaybackFinishedEvent) {
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
func (b *ChannelEventBus) PublishTrackEnded(event domain.TrackEndedEvent) {
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
func (b *ChannelEventBus) PublishQueueCleared(event domain.QueueClearedEvent) {
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

// --- EventSubscriber interface ---

// OnTrackEnqueued registers a handler for TrackEnqueuedEvent.
func (b *ChannelEventBus) OnTrackEnqueued(
	handler func(context.Context, domain.TrackEnqueuedEvent),
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.trackEnqueuedHandlers = append(b.trackEnqueuedHandlers, handler)
}

// OnPlaybackStarted registers a handler for PlaybackStartedEvent.
func (b *ChannelEventBus) OnPlaybackStarted(
	handler func(context.Context, domain.PlaybackStartedEvent),
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.playbackStartedHandlers = append(b.playbackStartedHandlers, handler)
}

// OnPlaybackFinished registers a handler for PlaybackFinishedEvent.
func (b *ChannelEventBus) OnPlaybackFinished(
	handler func(context.Context, domain.PlaybackFinishedEvent),
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.playbackFinishedHandlers = append(b.playbackFinishedHandlers, handler)
}

// OnTrackEnded registers a handler for TrackEndedEvent.
func (b *ChannelEventBus) OnTrackEnded(handler func(context.Context, domain.TrackEndedEvent)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.trackEndedHandlers = append(b.trackEndedHandlers, handler)
}

// OnQueueCleared registers a handler for QueueClearedEvent.
func (b *ChannelEventBus) OnQueueCleared(handler func(context.Context, domain.QueueClearedEvent)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queueClearedHandlers = append(b.queueClearedHandlers, handler)
}

// Close closes all event channels and stops dispatchers.
// After calling Close, publishing will no longer send events.
func (b *ChannelEventBus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	b.mu.Unlock()

	// Cancel context to stop dispatchers
	b.cancel()

	// Close channels to unblock any pending reads
	close(b.trackEnqueued)
	close(b.playbackStarted)
	close(b.playbackFinished)
	close(b.trackEnded)
	close(b.queueCleared)

	// Wait for dispatchers to finish
	b.wg.Wait()

	slog.Debug("channel event bus closed")
}
