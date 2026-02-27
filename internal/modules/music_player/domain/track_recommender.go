package domain

import "context"

// TrackRecommender defines the interface for recommending tracks based on seed tracks.
type TrackRecommender interface {
	// Recommend returns up to limit recommended tracks based on the given
	// seed track IDs. Returned tracks must not have IDs in the seed list
	// or the exclude list.
	Recommend(
		ctx context.Context,
		seeds []TrackID,
		exclude []TrackID,
		limit int,
	) ([]Track, error)
}
