package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// SetAutoPlayInput holds the input for the SetAutoPlay use case.
type SetAutoPlayInput[C comparable] struct {
	ConnectionInfo C
	Enabled        bool
}

// SetAutoPlayOutput holds the output for the SetAutoPlay use case.
type SetAutoPlayOutput struct {
	Changed bool
}

// SetAutoPlay enables or disables auto-play.
type SetAutoPlayUsecase[C comparable] struct {
	playerStates       domain.PlayerStateRepository
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewSetAutoPlay creates a new SetAutoPlay use case.
func NewSetAutoPlayUsecase[C comparable](
	playerStates domain.PlayerStateRepository,
	playerStateLocator ports.PlayerStateLocator[C],
) *SetAutoPlayUsecase[C] {
	return &SetAutoPlayUsecase[C]{
		playerStates:       playerStates,
		playerStateLocator: playerStateLocator,
	}
}

// Execute sets the auto-play state.
func (uc *SetAutoPlayUsecase[C]) Execute(
	ctx context.Context,
	input SetAutoPlayInput[C],
) (*SetAutoPlayOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	prev := state.IsAutoPlayEnabled()
	state.SetAutoPlayEnabled(input.Enabled)

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &SetAutoPlayOutput{Changed: prev != input.Enabled}, nil
}
