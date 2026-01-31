package infrastructure

import (
	"sync"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestMemoryRepository_Get(t *testing.T) {
	repo := NewMemoryRepository()
	guildID := snowflake.ID(123)

	// Get should return nil if state doesn't exist
	state := repo.Get(guildID)
	if state != nil {
		t.Fatal("expected nil for non-existent state")
	}

	// Save a state
	newState := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	repo.Save(newState)

	// Get should return the saved state
	state = repo.Get(guildID)
	if state == nil {
		t.Fatal("expected state after save")
	}
	if state != newState {
		t.Error("expected same state instance")
	}

	// Different guild should return nil
	otherGuildID := snowflake.ID(456)
	otherState := repo.Get(otherGuildID)
	if otherState != nil {
		t.Error("expected nil for different guild")
	}
}

func TestMemoryRepository_Save(t *testing.T) {
	repo := NewMemoryRepository()
	guildID := snowflake.ID(123)

	// Save a state
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	repo.Save(state)

	// Get should return it
	retrieved := repo.Get(guildID)
	if retrieved != state {
		t.Error("expected same state instance after save")
	}

	// Save again should overwrite
	newState := domain.NewPlayerState(guildID, snowflake.ID(300), snowflake.ID(400))
	repo.Save(newState)

	retrieved = repo.Get(guildID)
	if retrieved != newState {
		t.Error("expected new state after overwrite")
	}
}

func TestMemoryRepository_Delete(t *testing.T) {
	repo := NewMemoryRepository()
	guildID := snowflake.ID(123)

	// Save a state
	state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
	repo.Save(state)

	// Delete it
	repo.Delete(guildID)

	// Get should return nil
	retrieved := repo.Get(guildID)
	if retrieved != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemoryRepository_Count(t *testing.T) {
	repo := NewMemoryRepository()

	if repo.Count() != 0 {
		t.Errorf("expected count 0, got %d", repo.Count())
	}

	repo.Save(domain.NewPlayerState(snowflake.ID(1), snowflake.ID(100), snowflake.ID(200)))
	if repo.Count() != 1 {
		t.Errorf("expected count 1, got %d", repo.Count())
	}

	repo.Save(domain.NewPlayerState(snowflake.ID(2), snowflake.ID(100), snowflake.ID(200)))
	if repo.Count() != 2 {
		t.Errorf("expected count 2, got %d", repo.Count())
	}

	repo.Delete(snowflake.ID(1))
	if repo.Count() != 1 {
		t.Errorf("expected count 1 after delete, got %d", repo.Count())
	}
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	repo := NewMemoryRepository()
	var wg sync.WaitGroup

	// Concurrent saves for different guilds
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			guildID := snowflake.ID(id)
			state := domain.NewPlayerState(guildID, snowflake.ID(100), snowflake.ID(200))
			repo.Save(state)
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
			state := repo.Get(guildID)
			if state == nil {
				t.Errorf("expected non-nil state for guild %d", id)
			}
		}(i)
	}

	wg.Wait()
}
