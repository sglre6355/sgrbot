package ports

import (
	"github.com/disgoorg/snowflake/v2"
)

// VoiceStateProvider defines the interface for getting Discord voice state information.
type VoiceStateProvider interface {
	// GetUserVoiceChannel returns the voice channel ID the user is currently in.
	// Returns nil if the user is not in a voice channel.
	GetUserVoiceChannel(guildID, userID snowflake.ID) (*snowflake.ID, error)
}
