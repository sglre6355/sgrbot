package events

import (
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

// Re-export event types from ports for use by event handlers.
type (
	TrackEnqueuedEvent    = ports.TrackEnqueuedEvent
	PlaybackStartedEvent  = ports.PlaybackStartedEvent
	PlaybackFinishedEvent = ports.PlaybackFinishedEvent
	TrackEndedEvent       = ports.TrackEndedEvent
	TrackEndReason        = ports.TrackEndReason
	QueueClearedEvent     = ports.QueueClearedEvent
)

// Re-export TrackEndReason constants.
const (
	TrackEndFinished   = ports.TrackEndFinished
	TrackEndLoadFailed = ports.TrackEndLoadFailed
	TrackEndStopped    = ports.TrackEndStopped
	TrackEndReplaced   = ports.TrackEndReplaced
	TrackEndCleanup    = ports.TrackEndCleanup
)
