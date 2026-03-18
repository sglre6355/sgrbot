package domain

// Event is a marker interface for domain events raised by aggregates.
type Event interface {
	isEvent()
}

// TrackAddedEvent is raised when one or more tracks are added to the queue.
type TrackAddedEvent struct {
	PlayerStateID PlayerStateID
	Entries       []QueueEntry
}

func (TrackAddedEvent) isEvent() {}

// TrackRemovedEvent is raised when a track is removed from the queue.
type TrackRemovedEvent struct {
	PlayerStateID PlayerStateID
	Entry         QueueEntry
}

func (TrackRemovedEvent) isEvent() {}

// TrackStartedEvent is raised when a track starts playing.
type TrackStartedEvent struct {
	PlayerStateID PlayerStateID
	Entry         QueueEntry
}

func (TrackStartedEvent) isEvent() {}

// TrackEndedEvent is raised when a track finishes playing or fails to load.
type TrackEndedEvent struct {
	PlayerStateID PlayerStateID
	Entry         QueueEntry
	TrackFailed   bool
}

func (TrackEndedEvent) isEvent() {}

// PlaybackStartedEvent is raised when playback becomes active
// (first track added or playback resumed after being stopped).
type PlaybackStartedEvent struct {
	PlayerStateID PlayerStateID
}

func (PlaybackStartedEvent) isEvent() {}

// PlaybackStoppedEvent is raised when playback becomes inactive
// (queue ended, cleared, or last track removed).
type PlaybackStoppedEvent struct {
	PlayerStateID PlayerStateID
}

func (PlaybackStoppedEvent) isEvent() {}

// PlaybackPausedEvent is raised when playback is paused.
type PlaybackPausedEvent struct {
	PlayerStateID PlayerStateID
}

func (PlaybackPausedEvent) isEvent() {}

// PlaybackResumedEvent is raised when playback is resumed from a paused state.
type PlaybackResumedEvent struct {
	PlayerStateID PlayerStateID
}

func (PlaybackResumedEvent) isEvent() {}

// QueueClearedEvent is raised when the queue is cleared.
type QueueClearedEvent struct {
	PlayerStateID PlayerStateID
	Count         int
}

func (QueueClearedEvent) isEvent() {}

// QueueExhaustedEvent is raised when the queue runs out of tracks
// (e.g. after a skip or remove with no next track).
type QueueExhaustedEvent struct {
	PlayerStateID PlayerStateID
}

func (QueueExhaustedEvent) isEvent() {}

// QueueShuffledEvent is raised when the queue order is randomized.
type QueueShuffledEvent struct {
	PlayerStateID PlayerStateID
}

func (QueueShuffledEvent) isEvent() {}
