package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackProvider defines the interface for resolving TrackIDs to full Track objects.
type TrackProvider interface {
	// LoadTrack returns the Track for the given ID, or error if not found.
	LoadTrack(ctx context.Context, id domain.TrackID) (domain.Track, error)

	// LoadTracks returns Tracks for the given IDs, or error if any not found.
	LoadTracks(ctx context.Context, ids ...domain.TrackID) ([]domain.Track, error)

	// ResolveQuery searches for tracks using the given query.
	ResolveQuery(ctx context.Context, query string) (domain.TrackList, error)
}
