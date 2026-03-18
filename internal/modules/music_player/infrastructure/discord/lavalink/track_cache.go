package lavalink

import (
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackCache is a thread-safe cache for domain Track objects.
type TrackCache struct {
	mu    sync.RWMutex
	cache map[domain.TrackID]*domain.Track
}

// NewTrackCache creates a new TrackCache.
func NewTrackCache() *TrackCache {
	return &TrackCache{
		cache: make(map[domain.TrackID]*domain.Track),
	}
}

// Get returns a cached track by ID, or false if not found.
func (c *TrackCache) Get(id domain.TrackID) (*domain.Track, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	track, ok := c.cache[id]
	return track, ok
}

// Set stores a track in the cache.
func (c *TrackCache) Set(id domain.TrackID, track *domain.Track) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[id] = track
}

// ConvertAndCache converts a Lavalink track to a domain Track and caches it.
func (c *TrackCache) ConvertAndCache(track lavalink.Track) domain.Track {
	info := track.Info
	trackID := domain.TrackID(info.Identifier)

	var uri, artworkURL string
	if info.URI != nil {
		uri = *info.URI
	}
	if info.ArtworkURL != nil {
		artworkURL = *info.ArtworkURL
	}
	domainTrack := domain.ConstructTrack(
		trackID,
		info.Title,
		info.Author,
		time.Duration(info.Length)*time.Millisecond,
		uri,
		artworkURL,
		domain.ParseTrackSource(info.SourceName),
		info.IsStream,
	)
	c.Set(trackID, domainTrack)

	return *domainTrack
}
