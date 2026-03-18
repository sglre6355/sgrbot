package youtube

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Ensure YouTubeTrackRecommender implements required interfaces.
var _ domain.TrackRecommender = (*YouTubeTrackRecommender)(nil)

// YouTubeTrackRecommender recommends tracks using YouTube Mix playlists.
type YouTubeTrackRecommender struct {
	trackResolver ports.TrackResolver
}

// NewYouTubeTrackRecommender creates a new YouTubeTrackRecommender.
func NewYouTubeTrackRecommender(trackResolver ports.TrackResolver) *YouTubeTrackRecommender {
	return &YouTubeTrackRecommender{
		trackResolver: trackResolver,
	}
}

// AcceptsSeed returns true if the given entry is a valid seed for recommendations.
// Only YouTube tracks are accepted as seeds for YouTube Mix playlists.
func (a *YouTubeTrackRecommender) AcceptsSeed(seed domain.QueueEntry) bool {
	return seed.Track().Source() == domain.TrackSourceYouTube
}

// GetRecommendation returns up to limit recommended tracks based on the given seeds.
// It uses YouTube Mix playlists (RD{trackID}) to find related tracks, then ranks
// them by how many mixes they appear in (overlap score).
func (a *YouTubeTrackRecommender) GetRecommendation(
	ctx context.Context,
	seeds []domain.QueueEntry,
	exclusions []domain.QueueEntry,
	limit int,
) ([]domain.Track, error) {
	if len(seeds) == 0 || limit <= 0 {
		return []domain.Track{}, nil
	}

	// Build exclude set from all seed IDs and explicit exclusions
	excludeSet := make(map[domain.TrackID]struct{}, len(seeds)+len(exclusions))
	for _, entry := range seeds {
		excludeSet[entry.Track().ID()] = struct{}{}
	}
	for _, entry := range exclusions {
		excludeSet[entry.Track().ID()] = struct{}{}
	}

	// For each seed, load the YouTube Mix playlist
	type scoredTrack struct {
		track domain.Track
		score int
	}
	trackScores := make(map[domain.TrackID]*scoredTrack)

	for _, seed := range seeds {
		seedID := seed.Track().ID()
		mixURL := fmt.Sprintf(
			"https://www.youtube.com/watch?v=%s&list=RD%s",
			seedID.String(),
			seedID.String(),
		)

		trackList, err := a.trackResolver.ResolveQuery(ctx, mixURL)
		if err != nil {
			slog.Debug(
				"failed to load YouTube Mix",
				"seed", seedID,
				"error", err,
			)
			continue
		}

		for _, t := range trackList.Tracks {
			// Skip tracks in the exclude set
			if _, excluded := excludeSet[t.ID()]; excluded {
				continue
			}

			if existing, ok := trackScores[t.ID()]; ok {
				existing.score++
			} else {
				trackScores[t.ID()] = &scoredTrack{track: t, score: 1}
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
