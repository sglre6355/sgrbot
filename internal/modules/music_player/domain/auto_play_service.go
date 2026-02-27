package domain

import (
	"context"
	"log/slog"
	"math/rand/v2"
)

// AutoPlayService selects seeds from the queue and recommends the next track.
type AutoPlayService struct {
	tracks      TrackRepository
	recommender TrackRecommender
}

// NewAutoPlayService creates a new AutoPlayService.
func NewAutoPlayService(tracks TrackRepository, recommender TrackRecommender) *AutoPlayService {
	return &AutoPlayService{
		tracks:      tracks,
		recommender: recommender,
	}
}

// GetRecommendation selects seeds from the queue and returns a recommended track.
// Seeds are chosen as: up to 2 randomly sampled manually added YouTube tracks
// + the most recent auto-play YouTube track.
// Returns nil if no recommendation could be made.
func (s *AutoPlayService) GetRecommendation(ctx context.Context, state *PlayerState) *Track {
	allEntries := state.List()
	var manualIDs []TrackID
	var autoPlayIDs []TrackID
	for _, entry := range allEntries {
		if entry.IsAutoPlay {
			autoPlayIDs = append(autoPlayIDs, entry.TrackID)
		} else {
			manualIDs = append(manualIDs, entry.TrackID)
		}
	}

	seeds := make([]TrackID, 0, 3)
	seedSet := make(map[TrackID]struct{}, 3)

	// Sample up to 2 manual YouTube seeds
	rand.Shuffle(len(manualIDs), func(i, j int) {
		manualIDs[i], manualIDs[j] = manualIDs[j], manualIDs[i]
	})
	for _, id := range manualIDs {
		if len(seeds) >= 2 {
			break
		}
		track, err := s.tracks.FindByID(ctx, id)
		if err != nil {
			slog.Debug(
				"failed to load manual seed track",
				"track", id,
				"error", err,
			)
			continue
		}
		if track.Source != TrackSourceYouTube {
			continue
		}
		if _, exists := seedSet[id]; exists {
			continue
		}
		seedSet[id] = struct{}{}
		seeds = append(seeds, id)
	}

	// Add the most recent auto-play YouTube track as a seed
	for i := len(autoPlayIDs) - 1; i >= 0; i-- {
		id := autoPlayIDs[i]
		if _, exists := seedSet[id]; exists {
			continue
		}
		track, err := s.tracks.FindByID(ctx, id)
		if err != nil {
			slog.Debug(
				"failed to load auto-play seed track",
				"track", id,
				"error", err,
			)
			continue
		}
		if track.Source != TrackSourceYouTube {
			continue
		}
		seedSet[id] = struct{}{}
		seeds = append(seeds, id)
		break
	}

	if len(seeds) == 0 {
		slog.Debug(
			"auto-play has no YouTube seeds",
			"guild", state.GetGuildID(),
		)
		return nil
	}

	// Exclude all tracks not selected as seeds
	var exclude []TrackID
	for _, entry := range allEntries {
		if _, isSeed := seedSet[entry.TrackID]; !isSeed {
			exclude = append(exclude, entry.TrackID)
		}
	}

	tracks, err := s.recommender.Recommend(ctx, seeds, exclude, 1)
	if err != nil {
		slog.Warn(
			"auto-play recommendation failed",
			"guild", state.GetGuildID(),
			"error", err,
		)
		return nil
	}

	if len(tracks) == 0 {
		slog.Debug(
			"auto-play found no recommendations",
			"guild", state.GetGuildID(),
		)
		return nil
	}

	return &tracks[0]
}
