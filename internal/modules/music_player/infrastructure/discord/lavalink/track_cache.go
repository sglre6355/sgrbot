package lavalink

import (
	"sync"
	"time"

	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// TrackCache is a thread-safe cache for domain Track objects.
type TrackCache struct {
	mu    sync.RWMutex
	cache map[core.TrackID]*core.Track
}

// NewTrackCache creates a new TrackCache.
func NewTrackCache() *TrackCache {
	return &TrackCache{
		cache: make(map[core.TrackID]*core.Track),
	}
}

// Get returns a cached track by ID, or false if not found.
func (c *TrackCache) Get(id core.TrackID) (*core.Track, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	track, ok := c.cache[id]
	return track, ok
}

// Set stores a track in the cache.
func (c *TrackCache) Set(id core.TrackID, track *core.Track) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[id] = track
}

// ConvertAndCache converts a Lavalink track to a domain Track and caches it.
func (c *TrackCache) ConvertAndCache(track lavalink.Track) core.Track {
	info := track.Info
	trackID := core.TrackID(info.Identifier)

	var uri, artworkURL string
	if info.URI != nil {
		uri = *info.URI
	}
	if info.ArtworkURL != nil {
		artworkURL = *info.ArtworkURL
	}
	domainTrack := core.ConstructTrack(
		trackID,
		info.Title,
		info.Author,
		time.Duration(info.Length)*time.Millisecond,
		uri,
		artworkURL,
		core.ParseTrackSource(info.SourceName),
		info.IsStream,
	)
	c.Set(trackID, domainTrack)

	return *domainTrack
}
