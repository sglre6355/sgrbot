package infrastructure

import (
	"context"
	"errors"
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// ErrPlayerStateNotFound is returned when a player state is not found.
var ErrPlayerStateNotFound = errors.New("player state not found")

// MemoryRepository is an in-memory implementation of PlayerStateRepository.
type MemoryRepository struct {
	mu     sync.RWMutex
	states map[snowflake.ID]domain.PlayerState
}

// NewMemoryRepository creates a new MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		states: make(map[snowflake.ID]domain.PlayerState),
	}
}

// Get returns the PlayerState for the given guild, or error if not exists.
func (r *MemoryRepository) Get(
	_ context.Context,
	guildID snowflake.ID,
) (domain.PlayerState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, ok := r.states[guildID]
	if !ok {
		return domain.PlayerState{}, ErrPlayerStateNotFound
	}
	return state, nil
}

// Save stores the PlayerState.
func (r *MemoryRepository) Save(_ context.Context, state domain.PlayerState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.states[state.GetGuildID()] = state
	return nil
}

// Delete removes the PlayerState for the given guild.
func (r *MemoryRepository) Delete(_ context.Context, guildID snowflake.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.states, guildID)
	return nil
}

// Count returns the number of player states (for testing/monitoring).
func (r *MemoryRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.states)
}

// Ensure MemoryRepository implements PlayerStateRepository.
var _ domain.PlayerStateRepository = (*MemoryRepository)(nil)
