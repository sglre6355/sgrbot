package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// SetAutoPlayInput holds the input for the SetAutoPlay use case.
type SetAutoPlayInput struct {
	PlayerStateID string
	Enabled       bool
}

// SetAutoPlayOutput holds the output for the SetAutoPlay use case.
type SetAutoPlayOutput struct {
	Changed bool
}

// SetAutoPlay enables or disables auto-play.
type SetAutoPlayUsecase struct {
	playerStates core.PlayerStateRepository
}

// NewSetAutoPlay creates a new SetAutoPlay use case.
func NewSetAutoPlayUsecase(playerStates core.PlayerStateRepository) *SetAutoPlayUsecase {
	return &SetAutoPlayUsecase{playerStates: playerStates}
}

// Execute sets the auto-play state.
func (uc *SetAutoPlayUsecase) Execute(
	ctx context.Context,
	input SetAutoPlayInput,
) (*SetAutoPlayOutput, error) {
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

	prev := state.IsAutoPlayEnabled()
	state.SetAutoPlayEnabled(input.Enabled)

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &SetAutoPlayOutput{Changed: prev != input.Enabled}, nil
}
