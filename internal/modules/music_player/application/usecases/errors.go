package usecases

import "errors"

// Domain errors for the music player module.
var (
	// ErrNotConnected is returned when an operation requires the bot to be in a voice channel.
	ErrNotConnected = errors.New("not connected to a voice channel")

	// ErrUserNotInVoice is returned when the user is not in a voice channel.
	ErrUserNotInVoice = errors.New("you must be in a voice channel")

	// ErrNotPlaying is returned when no track is currently playing.
	ErrNotPlaying = errors.New("nothing is currently playing")

	// ErrAlreadyPaused is returned when trying to pause while already paused.
	ErrAlreadyPaused = errors.New("playback is already paused")

	// ErrNotPaused is returned when trying to resume while not paused.
	ErrNotPaused = errors.New("playback is not paused")

	// ErrNoResults is returned when a search yields no results.
	ErrNoResults = errors.New("no results found")

	// ErrQueueEmpty is returned when the queue is empty.
	ErrQueueEmpty = errors.New("the queue is empty")

	// ErrInvalidPosition is returned when an invalid queue position is specified.
	ErrInvalidPosition = errors.New("invalid queue position")

	// ErrLoadFailed is returned when loading tracks fails.
	ErrLoadFailed = errors.New("failed to load track")
)
