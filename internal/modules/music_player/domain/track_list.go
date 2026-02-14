package domain

// TrackListType represents the type of track list.
type TrackListType int

const (
	TrackListTypeTrack TrackListType = iota
	TrackListTypePlaylist
	TrackListTypeSearch
)

// TrackList represents a collection of tracks.
type TrackList struct {
	Type       TrackListType
	Identifier *string
	Name       *string
	Url        *string
	Tracks     []Track
}
