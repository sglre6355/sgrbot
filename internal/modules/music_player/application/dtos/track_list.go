package dtos

import (
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackListView is a read-only view of a resolved track list using only primitive types.
type TrackListView struct {
	Type   string
	Name   *string
	URL    *string
	Tracks []TrackView
}

// NewTrackListView converts a domain TrackList to a TrackListView.
func NewTrackListView(tl domain.TrackList) TrackListView {
	tracks := make([]TrackView, len(tl.Tracks))
	for i := range tl.Tracks {
		tracks[i] = NewTrackView(&tl.Tracks[i])
	}
	return TrackListView{
		Type:   tl.Type.String(),
		Name:   tl.Name,
		URL:    tl.Url,
		Tracks: tracks,
	}
}
