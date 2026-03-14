package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// VoiceConnectionGateway defines the interface for voice channel connection operations.
type VoiceConnectionGateway[T comparable] interface {
	// Join establishes a voice connection and registers the mapping between
	// the player state ID and the connection info.
	Join(ctx context.Context, playerStateID core.PlayerStateID, connectionInfo T) error

	// Leave closes the voice connection for the given player state and
	// removes the mapping.
	Leave(ctx context.Context, playerStateID core.PlayerStateID) error

	// FindPlayerStateID returns the player state ID associated with the
	// given connection info, or nil if no mapping exists.
	FindPlayerStateID(ctx context.Context, connectionInfo T) *core.PlayerStateID
}
