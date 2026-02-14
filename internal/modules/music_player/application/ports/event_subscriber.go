package ports

import (
	"context"
	"reflect"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// EventSubscriber defines the interface for subscribing to events.
// Handlers are registered with the subscriber and invoked when events occur.
type EventSubscriber interface {
	Subscribe(eventType reflect.Type, handler func(context.Context, domain.Event)) error
}
