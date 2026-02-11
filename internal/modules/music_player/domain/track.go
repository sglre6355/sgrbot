package domain

import (
	"strconv"
	"time"
)

// TrackID is a unique identifier for a track in a queue.
type TrackID string

// Track represents a playable audio track.
type Track struct {
	ID         TrackID
	Title      string
	Artist     string
	Duration   time.Duration
	URI        string
	ArtworkURL string
	SourceName string // e.g., "youtube", "spotify", "soundcloud"
	IsStream   bool
}

// Source returns the parsed TrackSource for this track.
func (t *Track) Source() TrackSource {
	return ParseTrackSource(t.SourceName)
}

// NewTrack creates a new Track with the given parameters.
func NewTrack(
	id TrackID,
	title string,
	artist string,
	duration time.Duration,
	uri string,
	artworkURL string,
	sourceName string,
	isStream bool,
) *Track {
	return &Track{
		ID:         id,
		Title:      title,
		Artist:     artist,
		Duration:   duration,
		URI:        uri,
		ArtworkURL: artworkURL,
		SourceName: sourceName,
		IsStream:   isStream,
	}
}

// IsValid returns true if the track has the minimum required fields.
func (t *Track) IsValid() bool {
	return t.Title != ""
}

// FormattedDuration returns the duration as a human-readable string (mm:ss or hh:mm:ss).
func (t *Track) FormattedDuration() string {
	if t.IsStream {
		return "LIVE"
	}

	totalSeconds := int(t.Duration.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return formatTime(hours, minutes, seconds)
	}
	return formatTimeShort(minutes, seconds)
}

func formatTime(hours, minutes, seconds int) string {
	return pad(hours) + ":" + pad(minutes) + ":" + pad(seconds)
}

func formatTimeShort(minutes, seconds int) string {
	return pad(minutes) + ":" + pad(seconds)
}

func pad(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
