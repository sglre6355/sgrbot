package ports

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackPlayer defines the interface for track playback operations.
type TrackPlayer interface {
	// Play starts playback of the given track.
	Play(ctx context.Context, guildID snowflake.ID, trackID domain.TrackID) error

	// Stop stops the current playback.
	Stop(ctx context.Context, guildID snowflake.ID) error

	// Pause pauses the current playback.
	Pause(ctx context.Context, guildID snowflake.ID) error

	// Resume resumes the paused playback.
	Resume(ctx context.Context, guildID snowflake.ID) error
}
