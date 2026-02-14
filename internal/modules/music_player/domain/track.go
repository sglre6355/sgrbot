package domain

import (
	"time"
)

// TrackID is a unique identifier for a track in a queue.
type TrackID string

func (id TrackID) String() string {
	return string(id)
}

// ParseTrackID converts a track id string to a TrackID.
func ParseTrackID(id string) (TrackID, error) {
	return TrackID(id), nil
}

// Track represents a playable audio track.
type Track struct {
	ID         TrackID
	Title      string
	Artist     string
	Duration   time.Duration
	URI        string
	ArtworkURL string
	Source     TrackSource
	IsStream   bool
}

// NewTrack creates a new Track with the given parameters.
func NewTrack(
	id TrackID,
	title string,
	artist string,
	duration time.Duration,
	uri string,
	artworkURL string,
	source TrackSource,
	isStream bool,
) *Track {
	return &Track{
		ID:         id,
		Title:      title,
		Artist:     artist,
		Duration:   duration,
		URI:        uri,
		ArtworkURL: artworkURL,
		Source:     source,
		IsStream:   isStream,
	}
}
