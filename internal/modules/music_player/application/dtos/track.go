package dtos

import (
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackView is a read-only view of a track using only standard library types.
type TrackView struct {
	ID         string
	Title      string
	Author     string
	Duration   time.Duration
	URL        string
	ArtworkURL string
	Source     string
	IsStream   bool
}

// NewTrackView converts a domain Track to a TrackView.
func NewTrackView(t *domain.Track) TrackView {
	return TrackView{
		ID:         t.ID().String(),
		Title:      t.Title(),
		Author:     t.Author(),
		Duration:   t.Duration(),
		URL:        t.URL(),
		ArtworkURL: t.ArtworkURL(),
		Source:     t.Source().String(),
		IsStream:   t.IsStream(),
	}
}
