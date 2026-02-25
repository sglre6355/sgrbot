package domain

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
)

// QueueEntry represents a track's placement in the queue,
// associating a track with who requested it and when.
type QueueEntry struct {
	TrackID     TrackID
	RequesterID snowflake.ID
	EnqueuedAt  time.Time
	IsAutoPlay  bool
}

// NewQueueEntry creates a new QueueEntry with the provided metadata.
func NewQueueEntry(
	trackID TrackID,
	requesterID snowflake.ID,
	enqueuedAt time.Time,
	isAutoPlay bool,
) QueueEntry {
	return QueueEntry{
		TrackID:     trackID,
		RequesterID: requesterID,
		EnqueuedAt:  enqueuedAt,
		IsAutoPlay:  isAutoPlay,
	}
}
