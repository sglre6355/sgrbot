package ports

import (
	"context"
)

// TrackResolver defines the interface for loading/searching tracks.
type TrackResolver interface {
	// LoadTracks searches for tracks using the given query.
	LoadTracks(ctx context.Context, query string) (*LoadResult, error)
}
