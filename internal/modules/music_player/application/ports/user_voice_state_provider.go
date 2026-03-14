package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// UserVoiceStateProvider looks up a user's current voice connection.
// C is the full connection info type, P is the partial connection info used to scope the lookup.
type UserVoiceStateProvider[C comparable, P comparable] interface {
	// GetUserVoiceConnectionInfo returns the connection info for the user's
	// current voice session within the scope of the partial connection info,
	// or nil if not connected.
	GetUserVoiceConnectionInfo(
		ctx context.Context,
		partialConnectionInfo P,
		userID core.UserID,
	) (*C, error)
}
