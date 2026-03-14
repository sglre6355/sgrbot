package in_memory

import (
	"context"
	"sync"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// Ensure InMemoryPlayerStateRepository implements required interfaces.
var (
	_ core.PlayerStateRepository = (*InMemoryPlayerStateRepository)(nil)
)

// InMemoryPlayerStateRepository is an in-memory implementation of PlayerStateRepository.
type InMemoryPlayerStateRepository struct {
	mu     sync.RWMutex
	states map[core.PlayerStateID]core.PlayerState
}

// NewInMemoryPlayerStateRepository creates a new InMemoryPlayerStateRepository.
func NewInMemoryPlayerStateRepository() *InMemoryPlayerStateRepository {
	return &InMemoryPlayerStateRepository{
		states: make(map[core.PlayerStateID]core.PlayerState),
	}
}

// FindByID returns the PlayerState for the given ID, or error if not exists.
func (r *InMemoryPlayerStateRepository) FindByID(
	_ context.Context,
	id core.PlayerStateID,
) (core.PlayerState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, ok := r.states[id]
	if !ok {
		return core.PlayerState{}, core.ErrPlayerStateNotFound
	}
	return state, nil
}

// Save stores the PlayerState.
func (r *InMemoryPlayerStateRepository) Save(_ context.Context, state core.PlayerState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.states[state.ID()] = state
	return nil
}

// Delete removes the PlayerState for the given ID.
func (r *InMemoryPlayerStateRepository) Delete(_ context.Context, id core.PlayerStateID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.states, id)
	return nil
}

// Count returns the number of player states (for testing/monitoring).
func (r *InMemoryPlayerStateRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.states)
}
