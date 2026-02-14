package infrastructure

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func newTestPlayerState(
	guildID, voiceChannelID, notificationChannelID snowflake.ID,
) *domain.PlayerState {
	state := domain.NewPlayerState(guildID, domain.NewQueue())
	state.SetVoiceChannelID(voiceChannelID)
	state.SetNotificationChannelID(notificationChannelID)
	return state
}

func TestMemoryRepository_Get(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	guildID := snowflake.ID(123)

	// Get should return error if state doesn't exist
	_, err := repo.Get(ctx, guildID)
	if err == nil {
		t.Fatal("expected error for non-existent state")
	}
	if !errors.Is(err, domain.ErrPlayerStateNotFound) {
		t.Errorf("expected domain.ErrPlayerStateNotFound, got %v", err)
	}

	// Save a state
	newState := newTestPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	if err := repo.Save(ctx, *newState); err != nil {
		t.Fatalf("unexpected error saving state: %v", err)
	}

	// Get should return the saved state
	state, err := repo.Get(ctx, guildID)
	if err != nil {
		t.Fatalf("unexpected error getting state: %v", err)
	}
	if state.GetGuildID() != guildID {
		t.Error("expected same guild ID")
	}

	// Different guild should return error
	otherGuildID := snowflake.ID(456)
	_, err = repo.Get(ctx, otherGuildID)
	if err == nil {
		t.Error("expected error for different guild")
	}
}

func TestMemoryRepository_Save(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	guildID := snowflake.ID(123)

	// Save a state
	state := newTestPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	if err := repo.Save(ctx, *state); err != nil {
		t.Fatalf("unexpected error saving state: %v", err)
	}

	// Get should return it
	retrieved, err := repo.Get(ctx, guildID)
	if err != nil {
		t.Fatalf("unexpected error getting state: %v", err)
	}
	if retrieved.GetGuildID() != guildID {
		t.Error("expected same guild ID after save")
	}

	// Save again should overwrite
	newState := newTestPlayerState(guildID, snowflake.ID(300), snowflake.ID(400))
	if err := repo.Save(ctx, *newState); err != nil {
		t.Fatalf("unexpected error saving state: %v", err)
	}

	retrieved, err = repo.Get(ctx, guildID)
	if err != nil {
		t.Fatalf("unexpected error getting state: %v", err)
	}
	if retrieved.GetVoiceChannelID() != snowflake.ID(300) {
		t.Error("expected new voice channel ID after overwrite")
	}
}

func TestMemoryRepository_Delete(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	guildID := snowflake.ID(123)

	// Save a state
	state := newTestPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	if err := repo.Save(ctx, *state); err != nil {
		t.Fatalf("unexpected error saving state: %v", err)
	}

	// Delete it
	if err := repo.Delete(ctx, guildID); err != nil {
		t.Fatalf("unexpected error deleting state: %v", err)
	}

	// Get should return error
	_, err := repo.Get(ctx, guildID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestMemoryRepository_Count(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	if repo.Count() != 0 {
		t.Errorf("expected count 0, got %d", repo.Count())
	}

	_ = repo.Save(
		ctx,
		*newTestPlayerState(snowflake.ID(1), snowflake.ID(100), snowflake.ID(200)),
	)
	if repo.Count() != 1 {
		t.Errorf("expected count 1, got %d", repo.Count())
	}

	_ = repo.Save(
		ctx,
		*newTestPlayerState(snowflake.ID(2), snowflake.ID(100), snowflake.ID(200)),
	)
	if repo.Count() != 2 {
		t.Errorf("expected count 2, got %d", repo.Count())
	}

	_ = repo.Delete(ctx, snowflake.ID(1))
	if repo.Count() != 1 {
		t.Errorf("expected count 1 after delete, got %d", repo.Count())
	}
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	var wg sync.WaitGroup

	// Concurrent saves for different guilds
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			guildID := snowflake.ID(id)
			state := newTestPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
			_ = repo.Save(ctx, *state)
		}(i)
	}

	wg.Wait()

	// Should have 100 states
	if repo.Count() != 100 {
		t.Errorf("expected 100 states, got %d", repo.Count())
	}

	// Concurrent gets
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			guildID := snowflake.ID(id)
			_, err := repo.Get(ctx, guildID)
			if err != nil {
				t.Errorf("expected no error for guild %d, got %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}
