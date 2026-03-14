package ports

import (
	"context"
	"reflect"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// EventPublisher defines the interface for publishing events asynchronously.
type EventPublisher interface {
	Publish(ctx context.Context, events ...core.Event)
}

// EventSubscriber defines the interface for subscribing to events.
// Handlers are registered with the subscriber and invoked when events occur.
type EventSubscriber interface {
	Subscribe(eventType reflect.Type, handler func(context.Context, core.Event)) error
}
