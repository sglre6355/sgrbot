package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
)

// AutoPlayService selects seeds from the queue and recommends the next track.
type AutoPlayService struct {
	botUserID   UserID
	recommender TrackRecommender
}

// Domain errors for AutoPlayService.
var (
	// ErrNoAutoPlaySeeds is returned when no suitable seeds are available.
	ErrNoAutoPlaySeeds = errors.New("no seeds available for auto-play")

	// ErrNoRecommendations is returned when the recommender returns no tracks.
	ErrNoRecommendations = errors.New("no recommendations found for auto-play")
)

// NewAutoPlayService creates a new AutoPlayService.
func NewAutoPlayService(botUserID UserID, recommender TrackRecommender) *AutoPlayService {
	return &AutoPlayService{
		botUserID:   botUserID,
		recommender: recommender,
	}
}

// GetNextRecommendation selects seeds from the queue and returns a recommended auto-play entry.
// Seeds are chosen as a combination of:
// - up to 2 randomly sampled manually added tracks
// - up to 1 most recent auto-play track
func (s *AutoPlayService) GetNextRecommendation(
	ctx context.Context,
	state *PlayerState,
) (QueueEntry, error) {
	manualPlaySeedCandidates := state.ManualPlayEntries()
	autoPlaySeedCandidates := state.AutoPlayEntries()

	// Collect up to 2 manual-play seeds and up to 1 auto-play seed accepted by the recommender
	manualPlaySeeds := make([]QueueEntry, 0, 2)
	autoPlaySeeds := make([]QueueEntry, 0, 1)

	// Sample manual play seeds
	rand.Shuffle(len(manualPlaySeedCandidates), func(i, j int) {
		manualPlaySeedCandidates[i], manualPlaySeedCandidates[j] = manualPlaySeedCandidates[j], manualPlaySeedCandidates[i]
	})
	for _, seed := range manualPlaySeedCandidates {
		if len(manualPlaySeeds) >= cap(manualPlaySeeds) {
			break
		}
		if !s.recommender.AcceptsSeed(seed) || slices.Contains(manualPlaySeeds, seed) {
			continue
		}
		manualPlaySeeds = append(manualPlaySeeds, seed)
	}

	// Sample the most recent auto-play seed
	for _, seed := range slices.Backward(autoPlaySeedCandidates) {
		if len(autoPlaySeeds) >= cap(autoPlaySeeds) {
			break
		}
		if !s.recommender.AcceptsSeed(seed) || slices.Contains(manualPlaySeeds, seed) ||
			slices.Contains(autoPlaySeeds, seed) {
			continue
		}
		autoPlaySeeds = append(autoPlaySeeds, seed)
	}

	seeds := append(manualPlaySeeds, autoPlaySeeds...)

	if len(seeds) == 0 {
		slog.Debug(
			"auto-play has no acceptable seeds",
			"player_state_id", state.ID(),
		)
		return QueueEntry{}, ErrNoAutoPlaySeeds
	}

	// Exclude all tracks not selected as seeds
	var exclusions []QueueEntry
	for _, entry := range state.List() {
		if slices.Contains(seeds, entry) {
			continue
		}
		exclusions = append(exclusions, entry)
	}

	tracks, err := s.recommender.GetRecommendation(ctx, seeds, exclusions, 1)
	if err != nil {
		slog.Warn(
			"auto-play recommendation failed",
			"player_state_id", state.ID(),
			"error", err,
		)
		return QueueEntry{}, fmt.Errorf("auto-play recommendation: %w", err)
	}

	if len(tracks) == 0 {
		slog.Debug(
			"auto-play found no recommendations",
			"player_state_id", state.ID(),
		)
		return QueueEntry{}, ErrNoRecommendations
	}

	entry := NewQueueEntry(tracks[0], s.botUserID, true)

	return entry, nil
}
