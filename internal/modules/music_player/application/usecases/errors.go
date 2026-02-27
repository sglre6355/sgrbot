package usecases

import (
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Application-layer errors for the music player module.
var (
	// ErrNotConnected is returned when an operation requires the bot to be in a voice channel.
	ErrNotConnected = errors.New("not connected to a voice channel")

	// ErrUserNotInVoice is returned when the user is not in a voice channel.
	ErrUserNotInVoice = errors.New("you must be in a voice channel")

	// ErrNoResults is returned when a search yields no results.
	ErrNoResults = errors.New("no results found")

	// ErrQueueEmpty is returned when the queue is empty.
	ErrQueueEmpty = errors.New("the queue is empty")

	// ErrNothingToClear is returned when there are no tracks to clear (only current track exists).
	ErrNothingToClear = errors.New("nothing to clear")

	// ErrInvalidIndex is returned when an invalid queue position is specified.
	ErrInvalidIndex = errors.New("invalid queue index")

	// ErrIsCurrentTrack is returned when trying to remove the currently playing track.
	// The handler should delegate to Skip instead.
	ErrIsCurrentTrack = errors.New("cannot remove current track, use skip instead")

	// ErrLoadFailed is returned when loading tracks fails.
	ErrLoadFailed = errors.New("failed to load track")
)

// Re-export domain errors for backward compatibility with presentation layer.
var (
	ErrNotPlaying    = domain.ErrNotPlaying
	ErrAlreadyPaused = domain.ErrAlreadyPaused
	ErrNotPaused     = domain.ErrNotPaused
)
