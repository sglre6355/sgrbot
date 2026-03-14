package ports

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// AudioGateway defines the interface for audio playback operations.
type AudioGateway interface {
	// Play starts playback of the given queue entry.
	Play(ctx context.Context, playerStateID core.PlayerStateID, entry core.QueueEntry) error

	// Stop stops the current playback.
	Stop(ctx context.Context, playerStateID core.PlayerStateID) error

	// Pause pauses the current playback.
	Pause(ctx context.Context, playerStateID core.PlayerStateID) error

	// Resume resumes the paused playback.
	Resume(ctx context.Context, playerStateID core.PlayerStateID) error
}
