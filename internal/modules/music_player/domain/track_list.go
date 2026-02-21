package domain

// TrackListType represents the type of track list.
type TrackListType int

const (
	TrackListTypeTrack TrackListType = iota
	TrackListTypePlaylist
	TrackListTypeSearch
)

func (t TrackListType) String() string {
	switch t {
	case TrackListTypeTrack:
		return "track"
	case TrackListTypePlaylist:
		return "playlist"
	case TrackListTypeSearch:
		return "search"
	default:
		return ""
	}
}

// TrackList represents a collection of tracks.
type TrackList struct {
	Type       TrackListType
	Identifier *string
	Name       *string
	Url        *string
	Tracks     []Track
}
