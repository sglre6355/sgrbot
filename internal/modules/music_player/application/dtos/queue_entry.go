package dtos

import (
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// QueueEntryView is a read-only view of a queue entry using only standard library types.
type QueueEntryView struct {
	Track       TrackView
	RequesterID string
	AddedAt     time.Time
	IsAutoPlay  bool
}

// NewQueueEntryView converts a domain QueueEntry to a QueueEntryView.
func NewQueueEntryView(e *domain.QueueEntry) QueueEntryView {
	return QueueEntryView{
		Track:       NewTrackView(e.Track()),
		RequesterID: e.RequesterID().String(),
		AddedAt:     e.AddedAt(),
		IsAutoPlay:  e.IsAutoPlay(),
	}
}
