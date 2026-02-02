package infrastructure

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// MemoryRepository is an in-memory implementation of PlayerStateRepository.
type MemoryRepository struct {
	mu     sync.RWMutex
	states map[snowflake.ID]*domain.PlayerState
}

// NewMemoryRepository creates a new MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		states: make(map[snowflake.ID]*domain.PlayerState),
	}
}

// Get returns the PlayerState for the given guild, or nil if not exists.
func (r *MemoryRepository) Get(guildID snowflake.ID) *domain.PlayerState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.states[guildID]
}

// Save stores the PlayerState.
func (r *MemoryRepository) Save(state *domain.PlayerState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.states[state.GuildID] = state
}

// Delete removes the PlayerState for the given guild.
func (r *MemoryRepository) Delete(guildID snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.states, guildID)
}

// Count returns the number of player states (for testing/monitoring).
func (r *MemoryRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.states)
}

// Ensure MemoryRepository implements PlayerStateRepository.
var _ domain.PlayerStateRepository = (*MemoryRepository)(nil)
