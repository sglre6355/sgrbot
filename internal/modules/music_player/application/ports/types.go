package ports

import (
	"time"

	"github.com/disgoorg/snowflake/v2"
)

// LoadResult represents the result of loading tracks.
type LoadResult struct {
	Type       LoadType
	Tracks     []*TrackInfo
	PlaylistID string
}

// LoadType represents the type of load result.
type LoadType string

const (
	LoadTypeTrack    LoadType = "track"
	LoadTypePlaylist LoadType = "playlist"
	LoadTypeSearch   LoadType = "search"
	LoadTypeEmpty    LoadType = "empty"
	LoadTypeError    LoadType = "error"
)

// TrackInfo contains information about a loaded track.
type TrackInfo struct {
	Identifier string // Unique identifier from Lavalink
	Encoded    string
	Title      string
	Artist     string
	Duration   time.Duration
	URI        string
	ArtworkURL string
	SourceName string // e.g., "youtube", "spotify", "soundcloud"
	IsStream   bool
}

// NowPlayingInfo contains information for the "Now Playing" notification.
type NowPlayingInfo struct {
	Identifier         string // Unique identifier (e.g., YouTube video ID)
	Title              string
	Artist             string
	Duration           string
	URI                string
	ArtworkURL         string
	SourceName         string // e.g., "youtube", "spotify", "soundcloud"
	IsStream           bool
	RequesterID        snowflake.ID
	RequesterName      string
	RequesterAvatarURL string
	EnqueuedAt         time.Time
}
