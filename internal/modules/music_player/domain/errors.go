package domain

import "errors"

// Domain errors for player state invariants.
var (
	// ErrNotPlaying is returned when an operation requires active playback.
	ErrNotPlaying = errors.New("nothing is currently playing")

	// ErrAlreadyPaused is returned when trying to pause while already paused.
	ErrAlreadyPaused = errors.New("playback is already paused")

	// ErrNotPaused is returned when trying to resume while not paused.
	ErrNotPaused = errors.New("playback is not paused")
)
