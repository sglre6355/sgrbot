package ports

import (
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// NowPlayingGateway defines the interface for displaying the currently playing track.
type NowPlayingGateway[T comparable] interface {
	// SetDestination associates a player state with a display destination
	// where now-playing information will be shown.
	SetDestination(playerStateID core.PlayerStateID, destination T)

	// Show displays the now-playing information for the given track and requester.
	Show(
		playerStateID core.PlayerStateID,
		track core.Track,
		requester core.User,
		enqueuedAt time.Time,
	) error

	// Clear removes the now-playing display for the given player state.
	Clear(playerStateID core.PlayerStateID) error
}
