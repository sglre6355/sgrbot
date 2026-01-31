package domain

import (
	"github.com/disgoorg/snowflake/v2"
)

// PlayerStateRepository defines the interface for storing and retrieving player states.
type PlayerStateRepository interface {
	// Get returns the PlayerState for the given guild, or nil if not exists.
	Get(guildID snowflake.ID) *PlayerState

	// Save stores the PlayerState.
	Save(state *PlayerState)

	// Delete removes the PlayerState for the given guild.
	Delete(guildID snowflake.ID)
}
