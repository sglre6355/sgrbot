package gateways

import (
	"context"
	"reflect"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// EventPublisher defines the interface for publishing events asynchronously.
type EventPublisher interface {
	Publish(event domain.Event) error
}

// EventSubscriber defines the interface for subscribing to events.
// Handlers are registered with the subscriber and invoked when events occur.
type EventSubscriber interface {
	Subscribe(eventType reflect.Type, handler func(context.Context, domain.Event)) error
}
