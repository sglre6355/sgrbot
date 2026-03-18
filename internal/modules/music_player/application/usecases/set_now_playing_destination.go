package usecases

import (
	"context"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

// SetNowPlayingDestinationInput holds the input for the SetNowPlayingDestination use case.
type SetNowPlayingDestinationInput[C comparable, D comparable] struct {
	ConnectionInfo        C
	NowPlayingDestination D
}

// SetNowPlayingDestinationOutput holds the output for the SetNowPlayingDestination use case.
type SetNowPlayingDestinationOutput struct{}

// SetNowPlayingDestinationUsecase updates the now-playing display destination.
type SetNowPlayingDestinationUsecase[C comparable, D comparable] struct {
	nowPlaying         ports.NowPlayingDestinationSetter[D]
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewSetNowPlayingDestinationUsecase creates a new SetNowPlayingDestination use case.
func NewSetNowPlayingDestinationUsecase[C comparable, D comparable](
	nowPlaying ports.NowPlayingDestinationSetter[D],
	playerStateLocator ports.PlayerStateLocator[C],
) *SetNowPlayingDestinationUsecase[C, D] {
	return &SetNowPlayingDestinationUsecase[C, D]{
		nowPlaying:         nowPlaying,
		playerStateLocator: playerStateLocator,
	}
}

// Execute sets the now-playing display destination.
func (uc *SetNowPlayingDestinationUsecase[C, D]) Execute(
	ctx context.Context,
	input SetNowPlayingDestinationInput[C, D],
) (*SetNowPlayingDestinationOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	uc.nowPlaying.SetDestination(playerStateID, input.NowPlayingDestination)
	return &SetNowPlayingDestinationOutput{}, nil
}
