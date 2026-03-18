package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PausePlaybackInput holds the input for the PausePlayback use case.
type PausePlaybackInput[C comparable] struct {
	ConnectionInfo C
}

// PausePlaybackOutput holds the output for the PausePlayback use case.
type PausePlaybackOutput struct{}

// PausePlayback pauses the current playback.
type PausePlaybackUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewPausePlayback creates a new PausePlayback use case.
func NewPausePlaybackUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *PausePlaybackUsecase[C] {
	return &PausePlaybackUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute pauses playback.
func (uc *PausePlaybackUsecase[C]) Execute(
	ctx context.Context,
	input PausePlaybackInput[C],
) (*PausePlaybackOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	events, err := uc.player.Pause(&state)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotPlaying):
			return nil, ErrNotPlaying
		case errors.Is(err, domain.ErrAlreadyPaused):
			return nil, ErrAlreadyPaused
		default:
			return nil, errors.Join(ErrInternal, err)
		}
	}

	if err := uc.audio.Pause(ctx, state.ID()); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &PausePlaybackOutput{}, nil
}
