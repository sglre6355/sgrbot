package domain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

type mockTrackRepo struct {
	tracks map[TrackID]*Track
}

func newMockTrackRepo() *mockTrackRepo {
	return &mockTrackRepo{tracks: make(map[TrackID]*Track)}
}

func (m *mockTrackRepo) FindByID(_ context.Context, id TrackID) (Track, error) {
	t, ok := m.tracks[id]
	if !ok {
		return Track{}, fmt.Errorf("track %q not found", id)
	}
	return *t, nil
}

func (m *mockTrackRepo) FindByIDs(_ context.Context, ids ...TrackID) ([]Track, error) {
	result := make([]Track, 0, len(ids))
	for _, id := range ids {
		t, ok := m.tracks[id]
		if !ok {
			return nil, fmt.Errorf("track %q not found", id)
		}
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockTrackRepo) Store(track *Track) {
	m.tracks[track.ID] = track
}

type mockRecommender struct {
	result []Track
	err    error
}

func (m *mockRecommender) Recommend(
	_ context.Context,
	_ []TrackID,
	_ []TrackID,
	_ int,
) ([]Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func ytTrack(id string) *Track {
	return &Track{
		ID:     TrackID(id),
		Title:  "Track " + id,
		Source: TrackSourceYouTube,
	}
}

func TestAutoPlayService_GetRecommendation(t *testing.T) {
	guildID := snowflake.ID(1)

	t.Run("no seeds returns nil", func(t *testing.T) {
		repo := newMockTrackRepo()
		rec := &mockRecommender{}
		svc := NewAutoPlayService(repo, rec)

		state := NewPlayerState(guildID, NewQueue())
		// No tracks in queue at all
		result := svc.GetRecommendation(context.Background(), state)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("manual YouTube seeds produce recommendation", func(t *testing.T) {
		repo := newMockTrackRepo()
		repo.Store(ytTrack("yt1"))
		repo.Store(ytTrack("yt2"))

		recommended := Track{ID: "rec1", Title: "Recommended", Source: TrackSourceYouTube}
		rec := &mockRecommender{result: []Track{recommended}}

		svc := NewAutoPlayService(repo, rec)

		state := NewPlayerState(guildID, NewQueue())
		state.Append(QueueEntry{TrackID: "yt1"})
		state.Append(QueueEntry{TrackID: "yt2"})
		state.SetPlaybackActive(true)

		result := svc.GetRecommendation(context.Background(), state)
		if result == nil {
			t.Fatal("expected recommendation, got nil")
		}
		if result.ID != "rec1" {
			t.Errorf("expected rec1, got %s", result.ID)
		}
	})

	t.Run("non-YouTube tracks are excluded from seeds", func(t *testing.T) {
		repo := newMockTrackRepo()
		repo.Store(&Track{ID: "sp1", Title: "Spotify Track", Source: TrackSourceSpotify})

		rec := &mockRecommender{}
		svc := NewAutoPlayService(repo, rec)

		state := NewPlayerState(guildID, NewQueue())
		state.Append(QueueEntry{TrackID: "sp1"})
		state.SetPlaybackActive(true)

		result := svc.GetRecommendation(context.Background(), state)
		if result != nil {
			t.Errorf("expected nil (no YouTube seeds), got %v", result)
		}
	})

	t.Run("mixed manual and auto-play tracks", func(t *testing.T) {
		repo := newMockTrackRepo()
		repo.Store(ytTrack("manual1"))
		repo.Store(ytTrack("auto1"))

		recommended := Track{ID: "rec1", Title: "Recommended", Source: TrackSourceYouTube}
		rec := &mockRecommender{result: []Track{recommended}}

		svc := NewAutoPlayService(repo, rec)

		state := NewPlayerState(guildID, NewQueue())
		state.Append(QueueEntry{TrackID: "manual1"})
		state.Append(QueueEntry{
			TrackID:    "auto1",
			IsAutoPlay: true,
			EnqueuedAt: time.Now(),
		})
		state.SetPlaybackActive(true)

		result := svc.GetRecommendation(context.Background(), state)
		if result == nil {
			t.Fatal("expected recommendation, got nil")
		}
		if result.ID != "rec1" {
			t.Errorf("expected rec1, got %s", result.ID)
		}
	})

	t.Run("recommender error returns nil", func(t *testing.T) {
		repo := newMockTrackRepo()
		repo.Store(ytTrack("yt1"))

		rec := &mockRecommender{err: fmt.Errorf("recommendation failed")}
		svc := NewAutoPlayService(repo, rec)

		state := NewPlayerState(guildID, NewQueue())
		state.Append(QueueEntry{TrackID: "yt1"})
		state.SetPlaybackActive(true)

		result := svc.GetRecommendation(context.Background(), state)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("recommender returns empty results", func(t *testing.T) {
		repo := newMockTrackRepo()
		repo.Store(ytTrack("yt1"))

		rec := &mockRecommender{result: []Track{}}
		svc := NewAutoPlayService(repo, rec)

		state := NewPlayerState(guildID, NewQueue())
		state.Append(QueueEntry{TrackID: "yt1"})
		state.SetPlaybackActive(true)

		result := svc.GetRecommendation(context.Background(), state)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}
