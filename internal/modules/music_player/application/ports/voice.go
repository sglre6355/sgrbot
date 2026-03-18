package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PlayerStateLocator resolves a player state from platform-specific connection scope.
type PlayerStateLocator[T comparable] interface {
	// FindPlayerStateID returns the player state ID associated with the
	// given connection info, or nil if no mapping exists.
	FindPlayerStateID(ctx context.Context, connectionInfo T) *domain.PlayerStateID
}

// VoiceConnectionGateway defines the interface for voice channel connection operations.
type VoiceConnectionGateway[T comparable] interface {
	// Join establishes a voice connection and registers the mapping between
	// the player state ID and the connection info.
	Join(
		ctx context.Context,
		playerStateID domain.PlayerStateID,
		connectionInfo T,
	) error

	// Leave closes the voice connection for the given player state and
	// removes the mapping.
	Leave(ctx context.Context, playerStateID domain.PlayerStateID) error
}

// UserVoiceStateProvider looks up a user's current voice connection.
type UserVoiceStateProvider[C comparable, P comparable] interface {
	// GetUserVoiceConnectionInfo returns the connection info for the user's
	// current voice session within the scope of the partial connection info,
	// or nil if not connected.
	GetUserVoiceConnectionInfo(
		ctx context.Context,
		partialConnectionInfo P,
		userID domain.UserID,
	) (*C, error)
}
