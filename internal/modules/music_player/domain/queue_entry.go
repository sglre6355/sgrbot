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
}

// NewQueueEntry creates a new QueueEntry with the current time as EnqueuedAt.
func NewQueueEntry(trackID TrackID, requesterID snowflake.ID) QueueEntry {
	return QueueEntry{
		TrackID:     trackID,
		RequesterID: requesterID,
		EnqueuedAt:  time.Now().UTC(),
	}
}
