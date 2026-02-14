package ports

import "github.com/sglre6355/sgrbot/internal/modules/music_player/domain"

// EventPublisher defines the interface for publishing events asynchronously.
type EventPublisher interface {
	Publish(event domain.Event) error
}
