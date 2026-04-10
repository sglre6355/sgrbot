package lavalink

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Ensure LavalinkTrackRepository implements required interfaces.
var _ domain.TrackRepository = (*LavalinkTrackRepository)(nil)

// LavalinkTrackRepository implements domain.TrackRepository using Lavalink.
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
	id domain.TrackID,
) (domain.Track, error) {
	track, ok := r.trackCache.Get(id)
	if ok {
		return *track, nil
	}

	// Cache miss: resolve from Lavalink
	lavalinkTrack, err := resolveFromLavalink(ctx, r.link, *track)
	if err != nil {
		return domain.Track{}, fmt.Errorf("track %q not found: %w", id, err)
	}

	return r.trackCache.ConvertAndCache(lavalinkTrack), nil
}

// FindByIDs returns Tracks for the given IDs.
// It checks the local cache first, falling back to a Lavalink query for cache misses.
func (r *LavalinkTrackRepository) FindByIDs(
	ctx context.Context,
	ids ...domain.TrackID,
) ([]domain.Track, error) {
	tracks := make([]domain.Track, 0, len(ids))
	for _, id := range ids {
		track, err := r.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}

// resolveFromLavalink queries Lavalink to get a fresh track by URL.
func resolveFromLavalink(
	ctx context.Context,
	link disgolink.Client,
	track domain.Track,
) (lavalink.Track, error) {
	node := link.BestNode()
	if node == nil {
		return lavalink.Track{}, fmt.Errorf("no available Lavalink node")
	}

	result, err := node.LoadTracks(ctx, track.URL())
	if err != nil {
		return lavalink.Track{}, fmt.Errorf("failed to load track from Lavalink: %w", err)
	}

	switch data := result.Data.(type) {
	case lavalink.Track:
		return data, nil
	case lavalink.Empty:
		return lavalink.Track{}, fmt.Errorf("track %q not found on Lavalink", track.ID())
	case lavalink.Exception:
		return lavalink.Track{}, fmt.Errorf(
			"track resolution raised an exception for track %q: %w",
			track.ID(),
			data,
		)
	default:
		return lavalink.Track{}, fmt.Errorf("invalid track id: %q", track.ID())
	}
}
