package ports

import (
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// NowPlayingPublisher defines the operations needed to show or clear the
// currently playing track.
type NowPlayingPublisher interface {
	// Show displays the now-playing information for the given track and requester.
	Show(
		playerStateID domain.PlayerStateID,
		track domain.Track,
		requester domain.User,
		enqueuedAt time.Time,
	) error

	// Clear removes the now-playing display for the given player state.
	Clear(playerStateID domain.PlayerStateID) error
}

// NowPlayingDestinationSetter defines the interface for configuring where
// now-playing information will be published.
type NowPlayingDestinationSetter[T comparable] interface {
	// SetDestination associates a player state with a display destination
	// where now-playing information will be shown.
	SetDestination(playerStateID domain.PlayerStateID, destination T)
}
