package ports

import (
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// EventPublisher defines the interface for publishing events asynchronously.
type EventPublisher interface {
	// PublishTrackEnqueued publishes an event when a track is added to the queue.
	PublishTrackEnqueued(event domain.TrackEnqueuedEvent)

	// PublishPlaybackStarted publishes an event when a track starts playing.
	PublishPlaybackStarted(event domain.PlaybackStartedEvent)

	// PublishPlaybackFinished publishes an event when playback finishes.
	PublishPlaybackFinished(event domain.PlaybackFinishedEvent)

	// PublishTrackEnded publishes an event when a track ends (from audio player).
	PublishTrackEnded(event domain.TrackEndedEvent)

	// PublishQueueCleared publishes an event when the queue is fully cleared.
	PublishQueueCleared(event domain.QueueClearedEvent)
}
