package ports

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// EventPublisher defines the interface for publishing events asynchronously.
type EventPublisher interface {
	// PublishTrackEnqueued publishes an event when a track is added to the queue.
	PublishTrackEnqueued(event TrackEnqueuedEvent)

	// PublishPlaybackStarted publishes an event when a track starts playing.
	PublishPlaybackStarted(event PlaybackStartedEvent)

	// PublishPlaybackFinished publishes an event when playback finishes.
	PublishPlaybackFinished(event PlaybackFinishedEvent)

	// PublishTrackEnded publishes an event when a track ends (from audio player).
	PublishTrackEnded(event TrackEndedEvent)

	// PublishQueueCleared publishes an event when the queue is fully cleared.
	PublishQueueCleared(event QueueClearedEvent)
}

// TrackEnqueuedEvent is published when a track is added to the queue.
type TrackEnqueuedEvent struct {
	GuildID snowflake.ID
	Track   *domain.Track
	WasIdle bool // true if no track was playing when this was enqueued
}

// PlaybackStartedEvent is published when a track starts playing.
type PlaybackStartedEvent struct {
	GuildID               snowflake.ID
	Track                 *domain.Track
	NotificationChannelID snowflake.ID
}

// PlaybackFinishedEvent is published when a track finishes playing.
// This signals that the "Now Playing" message should be deleted.
type PlaybackFinishedEvent struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID
	LastMessageID         *snowflake.ID // "Now Playing" message to delete
}

// TrackEndedEvent is published when a track ends (from Lavalink).
type TrackEndedEvent struct {
	GuildID snowflake.ID
	Reason  TrackEndReason
}

// QueueClearedEvent is published when the queue is fully cleared (including current track).
// This triggers playback to stop.
type QueueClearedEvent struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID
}

// TrackEndReason represents why a track ended.
type TrackEndReason string

const (
	// TrackEndFinished means the track finished normally.
	TrackEndFinished TrackEndReason = "finished"
	// TrackEndLoadFailed means the track failed to load.
	TrackEndLoadFailed TrackEndReason = "load_failed"
	// TrackEndStopped means the track was stopped by the user.
	TrackEndStopped TrackEndReason = "stopped"
	// TrackEndReplaced means the track was replaced by another.
	TrackEndReplaced TrackEndReason = "replaced"
	// TrackEndCleanup means the track was cleaned up.
	TrackEndCleanup TrackEndReason = "cleanup"
)

// ShouldAdvanceQueue returns true if this end reason should advance the queue.
func (r TrackEndReason) ShouldAdvanceQueue() bool {
	return r == TrackEndFinished || r == TrackEndLoadFailed
}
