package core

import (
	"context"
	"errors"
	"time"
)

// TrackID is a unique identifier for a track in a queue.
type TrackID string

// Domain errors for TrackID invariants.
var (
	// ErrEmptyTrackID is returned when an empty string is passed to ParseTrackID.
	ErrEmptyTrackID = errors.New("track id must not be empty")
)

// ParseTrackID converts a track id string to a TrackID.
func ParseTrackID(id string) (TrackID, error) {
	if len(id) == 0 {
		return "", ErrEmptyTrackID
	}

	return TrackID(id), nil
}

func (id TrackID) String() string {
	return string(id)
}

// TrackSource represents the origin platform of a track.
type TrackSource string

const (
	TrackSourceYouTube    TrackSource = "youtube"
	TrackSourceSpotify    TrackSource = "spotify"
	TrackSourceSoundCloud TrackSource = "soundcloud"
	TrackSourceTwitch     TrackSource = "twitch"
	TrackSourceOther      TrackSource = "other"
)

// ParseTrackSource converts a source name string to a TrackSource.
func ParseTrackSource(name string) TrackSource {
	switch name {
	case "youtube":
		return TrackSourceYouTube
	case "spotify":
		return TrackSourceSpotify
	case "soundcloud":
		return TrackSourceSoundCloud
	case "twitch":
		return TrackSourceTwitch
	default:
		return TrackSourceOther
	}
}

// String returns the TrackSource as a string.
func (s TrackSource) String() string {
	return string(s)
}

// Track represents a playable audio track.
type Track struct {
	id         TrackID
	title      string
	author     string
	duration   time.Duration
	url        string
	artworkURL string
	source     TrackSource
	isStream   bool
}

// ConstructTrack creates a new Track with the given parameters.
func ConstructTrack(
	id TrackID,
	title string,
	author string,
	duration time.Duration,
	url string,
	artworkURL string,
	source TrackSource,
	isStream bool,
) *Track {
	return &Track{
		id,
		title,
		author,
		duration,
		url,
		artworkURL,
		source,
		isStream,
	}
}

// ID returns the track's unique identifier.
func (t *Track) ID() TrackID {
	return t.id
}

// Title returns the track's title.
func (t *Track) Title() string {
	return t.title
}

// Author returns the track's author.
func (t *Track) Author() string {
	return t.author
}

// Duration returns the track's duration.
func (t *Track) Duration() time.Duration {
	return t.duration
}

// URL returns the track's URL.
func (t *Track) URL() string {
	return t.url
}

// ArtworkURL returns the URL of the track's artwork.
func (t *Track) ArtworkURL() string {
	return t.artworkURL
}

// Source returns the track's origin platform.
func (t *Track) Source() TrackSource {
	return t.source
}

// IsStream returns true if the track is a live stream.
func (t *Track) IsStream() bool {
	return t.isStream
}

// TrackRepository defines the interface for retrieving tracks by ID.
type TrackRepository interface {
	// FindByID returns the Track for the given ID, or error if not found.
	FindByID(ctx context.Context, id TrackID) (Track, error)

	// FindByIDs returns Tracks for the given IDs, or error if any not found.
	FindByIDs(ctx context.Context, ids ...TrackID) ([]Track, error)
}
