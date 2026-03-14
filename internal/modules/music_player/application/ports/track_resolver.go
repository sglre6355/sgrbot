package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// TrackResolver defines the interface for resolving user queries to tracks.
type TrackResolver interface {
	// ResolveQuery searches for tracks using the given query.
	ResolveQuery(ctx context.Context, query string) (core.TrackList, error)
}
