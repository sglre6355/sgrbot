package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// EventSubscriber defines the interface for subscribing to events.
// Handlers are registered with the subscriber and invoked when events occur.
// This abstraction allows swapping the underlying transport (channels, message queues, etc.).
type EventSubscriber interface {
	// OnTrackEnqueued registers a handler for TrackEnqueuedEvent.
	OnTrackEnqueued(handler func(context.Context, domain.TrackEnqueuedEvent))

	// OnPlaybackStarted registers a handler for PlaybackStartedEvent.
	OnPlaybackStarted(handler func(context.Context, domain.PlaybackStartedEvent))

	// OnPlaybackFinished registers a handler for PlaybackFinishedEvent.
	OnPlaybackFinished(handler func(context.Context, domain.PlaybackFinishedEvent))

	// OnTrackEnded registers a handler for TrackEndedEvent.
	OnTrackEnded(handler func(context.Context, domain.TrackEndedEvent))

	// OnQueueCleared registers a handler for QueueClearedEvent.
	OnQueueCleared(handler func(context.Context, domain.QueueClearedEvent))
}
