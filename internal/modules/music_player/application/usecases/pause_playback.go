package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// PausePlaybackInput holds the input for the PausePlayback use case.
type PausePlaybackInput struct {
	PlayerStateID string
}

// PausePlaybackOutput holds the output for the PausePlayback use case.
type PausePlaybackOutput struct{}

// PausePlayback pauses the current playback.
type PausePlaybackUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewPausePlayback creates a new PausePlayback use case.
func NewPausePlaybackUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *PausePlaybackUsecase {
	return &PausePlaybackUsecase{
		player:       player,
		playerStates: playerStates,
		audio:        audio,
		events:       events,
	}
}

// Execute pauses playback.
func (uc *PausePlaybackUsecase) Execute(
	ctx context.Context,
	input PausePlaybackInput,
) (*PausePlaybackOutput, error) {
	playerStateID, err := core.ParsePlayerStateID(input.PlayerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		if errors.Is(err, core.ErrPlayerStateNotFound) {
			return nil, ErrPlayerStateNotFound
		}
		return nil, errors.Join(ErrInternal, err)
	}

	events, err := uc.player.Pause(&state)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotPlaying):
			return nil, ErrNotPlaying
		case errors.Is(err, core.ErrAlreadyPaused):
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
