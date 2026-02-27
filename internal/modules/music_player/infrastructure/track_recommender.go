package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Ensure TrackRecommenderAdapter implements required interfaces.
var _ domain.TrackRecommender = (*TrackRecommenderAdapter)(nil)

// TrackRecommenderAdapter recommends tracks using YouTube Mix playlists.
type TrackRecommenderAdapter struct {
	trackResolver ports.TrackResolver
}

// NewTrackRecommenderAdapter creates a new TrackRecommenderAdapter.
func NewTrackRecommenderAdapter(trackResolver ports.TrackResolver) *TrackRecommenderAdapter {
	return &TrackRecommenderAdapter{
		trackResolver: trackResolver,
	}
}

// Recommend returns up to limit recommended tracks based on the given seed track IDs.
// Tracks with IDs in the exclude list are also filtered out. Seeds should be
// YouTube track IDs so they can be used for YouTube Mix playlists.
// It uses YouTube Mix playlists (RD{trackID}) to find related tracks, then ranks
// them by how many mixes they appear in (overlap score).
func (a *TrackRecommenderAdapter) Recommend(
	ctx context.Context,
	seeds []domain.TrackID,
	exclude []domain.TrackID,
	limit int,
) ([]domain.Track, error) {
	if len(seeds) == 0 || limit <= 0 {
		return []domain.Track{}, nil
	}

	// Build exclude set from all seed IDs and explicit excludes
	excludeSet := make(map[domain.TrackID]struct{}, len(seeds)+len(exclude))
	for _, id := range seeds {
		excludeSet[id] = struct{}{}
	}
	for _, id := range exclude {
		excludeSet[id] = struct{}{}
	}

	// For each seed, load the YouTube Mix playlist
	type scoredTrack struct {
		track domain.Track
		score int
	}
	trackScores := make(map[domain.TrackID]*scoredTrack)

	for _, seed := range seeds {
		mixURL := fmt.Sprintf(
			"https://www.youtube.com/watch?v=%s&list=RD%s",
			seed.String(),
			seed.String(),
		)

		trackList, err := a.trackResolver.ResolveQuery(ctx, mixURL)
		if err != nil {
			slog.Debug(
				"failed to load YouTube Mix",
				"seed", seed,
				"error", err,
			)
			continue
		}

		for _, t := range trackList.Tracks {
			// Skip tracks in the exclude set
			if _, excluded := excludeSet[t.ID]; excluded {
				continue
			}

			if existing, ok := trackScores[t.ID]; ok {
				existing.score++
			} else {
				trackScores[t.ID] = &scoredTrack{track: t, score: 1}
			}
		}
	}

	if len(trackScores) == 0 {
		return []domain.Track{}, nil
	}

	// Sort by overlap score descending
	ranked := make([]*scoredTrack, 0, len(trackScores))
	for _, st := range trackScores {
		ranked = append(ranked, st)
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	// Return up to limit tracks
	result := make([]domain.Track, 0, limit)
	for i := range ranked {
		if len(result) >= limit {
			break
		}
		result = append(result, ranked[i].track)
	}

	return result, nil
}
