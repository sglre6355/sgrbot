package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sort"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Ensure TrackRecommenderAdapter implements required ports.
var _ ports.TrackRecommender = (*TrackRecommenderAdapter)(nil)

// maxMixSeeds is the maximum number of YouTube seeds to sample for mix playlists.
const maxMixSeeds = 3

// TrackRecommenderAdapter recommends tracks using YouTube Mix playlists.
type TrackRecommenderAdapter struct {
	trackProvider ports.TrackProvider
}

// NewTrackRecommenderAdapter creates a new TrackRecommenderAdapter.
func NewTrackRecommenderAdapter(trackProvider ports.TrackProvider) *TrackRecommenderAdapter {
	return &TrackRecommenderAdapter{
		trackProvider: trackProvider,
	}
}

// Recommend returns up to limit recommended tracks based on the given seed track IDs.
// It uses YouTube Mix playlists (RD{trackID}) to find related tracks, then ranks
// them by how many mixes they appear in (overlap score).
func (a *TrackRecommenderAdapter) Recommend(
	ctx context.Context,
	seeds []domain.TrackID,
	limit int,
) ([]domain.Track, error) {
	if len(seeds) == 0 || limit <= 0 {
		return nil, nil
	}

	// Load track metadata for all seeds to find YouTube-sourced tracks
	tracks, err := a.trackProvider.LoadTracks(ctx, seeds...)
	if err != nil {
		return nil, fmt.Errorf("failed to load seed tracks: %w", err)
	}

	// Build exclude set from all seed IDs
	excludeSet := make(map[domain.TrackID]struct{}, len(seeds))
	for _, id := range seeds {
		excludeSet[id] = struct{}{}
	}

	// Filter to YouTube-sourced tracks
	var ytTracks []domain.Track
	for _, t := range tracks {
		if t.Source == domain.TrackSourceYouTube {
			ytTracks = append(ytTracks, t)
		}
	}

	if len(ytTracks) == 0 {
		return nil, nil
	}

	// Randomly sample up to maxMixSeeds YouTube seeds
	sampled := ytTracks
	if len(sampled) > maxMixSeeds {
		rand.Shuffle(len(sampled), func(i, j int) {
			sampled[i], sampled[j] = sampled[j], sampled[i]
		})
		sampled = sampled[:maxMixSeeds]
	}

	// For each sampled seed, load the YouTube Mix playlist
	type scoredTrack struct {
		track domain.Track
		score int
	}
	trackScores := make(map[domain.TrackID]*scoredTrack)

	for _, seed := range sampled {
		mixURL := fmt.Sprintf(
			"https://www.youtube.com/watch?v=%s&list=RD%s",
			seed.ID.String(),
			seed.ID.String(),
		)

		trackList, err := a.trackProvider.ResolveQuery(ctx, mixURL)
		if err != nil {
			slog.Debug(
				"failed to load YouTube Mix",
				"seed", seed.ID,
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
		return nil, nil
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
