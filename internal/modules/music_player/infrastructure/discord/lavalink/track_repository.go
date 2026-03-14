package lavalink

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// Ensure LavalinkTrackRepository implements required interfaces.
var _ core.TrackRepository = (*LavalinkTrackRepository)(nil)

// LavalinkTrackRepository implements core.TrackRepository using Lavalink.
type LavalinkTrackRepository struct {
	link       disgolink.Client
	trackCache *TrackCache
}

// NewLavalinkTrackRepository creates a new LavalinkTrackRepository.
func NewLavalinkTrackRepository(
	link disgolink.Client,
	trackCache *TrackCache,
) *LavalinkTrackRepository {
	return &LavalinkTrackRepository{
		link:       link,
		trackCache: trackCache,
	}
}

// FindByID returns the Track for the given ID.
// It checks the local cache first, falling back to a Lavalink query on cache miss.
func (r *LavalinkTrackRepository) FindByID(
	ctx context.Context,
	id core.TrackID,
) (core.Track, error) {
	track, ok := r.trackCache.Get(id)
	if ok {
		return *track, nil
	}

	// Cache miss: resolve from Lavalink
	lavalinkTrack, err := resolveFromLavalink(ctx, r.link, id)
	if err != nil {
		return core.Track{}, fmt.Errorf("track %q not found: %w", id, err)
	}

	return r.trackCache.ConvertAndCache(lavalinkTrack), nil
}

// FindByIDs returns Tracks for the given IDs.
// It checks the local cache first, falling back to a Lavalink query for cache misses.
func (r *LavalinkTrackRepository) FindByIDs(
	ctx context.Context,
	ids ...core.TrackID,
) ([]core.Track, error) {
	tracks := make([]core.Track, 0, len(ids))
	for _, id := range ids {
		track, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}

// resolveFromLavalink queries Lavalink to get a fresh track by identifier.
func resolveFromLavalink(
	ctx context.Context,
	link disgolink.Client,
	trackID core.TrackID,
) (lavalink.Track, error) {
	node := link.BestNode()
	if node == nil {
		return lavalink.Track{}, fmt.Errorf("no available Lavalink node")
	}

	result, err := node.LoadTracks(ctx, trackID.String())
	if err != nil {
		return lavalink.Track{}, fmt.Errorf("failed to load track from Lavalink: %w", err)
	}

	switch data := result.Data.(type) {
	case lavalink.Track:
		return data, nil
	case lavalink.Empty:
		return lavalink.Track{}, fmt.Errorf("track %q not found on Lavalink", trackID)
	case lavalink.Exception:
		return lavalink.Track{}, fmt.Errorf(
			"track resolution raised an exception for track %q: %w",
			trackID,
			data,
		)
	default:
		return lavalink.Track{}, fmt.Errorf("invalid track id: %q", trackID)
	}
}
