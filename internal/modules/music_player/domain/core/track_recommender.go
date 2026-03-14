package core

import "context"

// TrackRecommender defines the interface for recommending tracks based on seed tracks.
type TrackRecommender interface {
	// Recommend returns up to limit recommended tracks based on the given seed track IDs.
	// Returned tracks must not have IDs in the seed list or the exclude list.
	GetRecommendation(
		ctx context.Context,
		seeds []QueueEntry,
		exclusions []QueueEntry,
		limit int,
	) ([]Track, error)

	// AcceptsSeed returns true if the given entry is a valid seed for recommendations.
	AcceptsSeed(seed QueueEntry) bool
}
