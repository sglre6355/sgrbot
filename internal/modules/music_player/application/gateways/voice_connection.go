package gateways

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
)

// VoiceConnectionManager defines the interface for voice channel connection operations.
type VoiceConnectionManager interface {
	// JoinChannel connects the bot to the specified voice channel.
	JoinChannel(ctx context.Context, guildID, channelID snowflake.ID) error

	// LeaveChannel disconnects the bot from the voice channel.
	LeaveChannel(ctx context.Context, guildID snowflake.ID) error
}
