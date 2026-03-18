package usecases

import "errors"

// Application-level errors returned by use cases.
var (
	ErrNotConnected    = errors.New("not connected to a voice channel")
	ErrUserNotInVoice  = errors.New("you must be in a voice channel")
	ErrNoResults       = errors.New("no results found")
	ErrQueueEmpty      = errors.New("the queue is empty")
	ErrIsCurrentTrack  = errors.New("cannot remove the currently playing track")
	ErrInvalidIndex    = errors.New("invalid position")
	ErrNotPlaying      = errors.New("nothing is currently playing")
	ErrAlreadyPaused   = errors.New("playback is already paused")
	ErrNotPaused       = errors.New("playback is not paused")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrInternal        = errors.New("an internal error occurred")
)
