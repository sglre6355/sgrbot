package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// DefaultEventBufferSize is the default buffer size for event channels.
const DefaultEventBufferSize = 100

// Ensure ChannelEventBus implements required ports.
var (
	_ ports.EventPublisher  = (*ChannelEventBus)(nil)
	_ ports.EventSubscriber = (*ChannelEventBus)(nil)
)

// ChannelEventBus provides a channel-based event bus for async event handling.
// It implements both EventPublisher and EventSubscriber interfaces.
type ChannelEventBus struct {
	events   chan domain.Event
	handlers map[reflect.Type][]func(context.Context, domain.Event)

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
		events:   make(chan domain.Event, bufferSize),
		handlers: make(map[reflect.Type][]func(context.Context, domain.Event)),
		ctx:      ctx,
		cancel:   cancel,
	}

	bus.wg.Go(func() {
		for event := range bus.events {
			eventType := reflect.TypeOf(event)

			bus.mu.RLock()
			handlers := make([]func(context.Context, domain.Event), len(bus.handlers[eventType]))
			copy(handlers, bus.handlers[eventType])
			bus.mu.RUnlock()

			for _, handler := range handlers {
				handler(bus.ctx, event)
			}
		}
	})

	return bus
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

	// Close channel to unblock any pending reads
	close(b.events)

	// Wait for dispatchers to finish
	b.wg.Wait()

	slog.Debug("channel event bus closed")
}

func (b *ChannelEventBus) Publish(event domain.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("attempted to publish to closed event bus")
	}

	select {
	case b.events <- event:
		slog.Debug("published event", "type", fmt.Sprintf("%T", event))
	default:
		return fmt.Errorf("event buffer full, dropping %T event", event)
	}

	return nil
}

func (b *ChannelEventBus) Subscribe(
	eventType reflect.Type,
	handler func(context.Context, domain.Event),
) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	eventInterface := reflect.TypeFor[domain.Event]()
	if !eventType.Implements(eventInterface) {
		return errors.New("invalid event type provided")
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return nil
}
