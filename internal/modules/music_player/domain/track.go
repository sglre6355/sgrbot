package domain

import (
	"context"
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

// TrackRepository defines the interface for retrieving tracks by ID.
type TrackRepository interface {
	// FindByID returns the Track for the given ID, or error if not found.
	FindByID(ctx context.Context, id TrackID) (Track, error)

	// FindByIDs returns Tracks for the given IDs, or error if any not found.
	FindByIDs(ctx context.Context, ids ...TrackID) ([]Track, error)
}
