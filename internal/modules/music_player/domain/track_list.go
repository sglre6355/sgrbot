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
		return "unknown"
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

// TrackListOption configures optional fields on a TrackList.
type TrackListOption func(*TrackList)

// WithPlaylistInfo sets playlist metadata on a TrackList.
func WithPlaylistInfo(identifier, name, url string) TrackListOption {
	return func(tl *TrackList) {
		tl.Identifier = &identifier
		tl.Name = &name
		tl.Url = &url
	}
}

// NewTrackList creates a TrackList.
func NewTrackList(trackListType TrackListType, tracks []Track, opts ...TrackListOption) TrackList {
	tl := TrackList{
		Type:   trackListType,
		Tracks: tracks,
	}

	for _, opt := range opts {
		opt(&tl)
	}

	return tl
}
