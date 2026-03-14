package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// SetLoopModeInput holds the input for the SetLoopMode use case.
type SetLoopModeInput struct {
	PlayerStateID string
	Mode          string
}

// SetLoopModeOutput holds the output for the SetLoopMode use case.
type SetLoopModeOutput struct {
	Changed bool
}

// SetLoopMode sets the loop mode to a specific value.
type SetLoopModeUsecase struct {
	playerStates core.PlayerStateRepository
}

// NewSetLoopMode creates a new SetLoopMode use case.
func NewSetLoopModeUsecase(playerStates core.PlayerStateRepository) *SetLoopModeUsecase {
	return &SetLoopModeUsecase{playerStates: playerStates}
}

// Execute sets the loop mode.
func (uc *SetLoopModeUsecase) Execute(
	ctx context.Context,
	input SetLoopModeInput,
) (*SetLoopModeOutput, error) {
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

	mode := core.ParseLoopMode(input.Mode)
	prev := state.LoopMode()
	state.SetLoopMode(mode)

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &SetLoopModeOutput{Changed: prev != mode}, nil
}
