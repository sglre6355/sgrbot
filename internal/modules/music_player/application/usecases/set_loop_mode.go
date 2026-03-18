package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// SetLoopModeInput holds the input for the SetLoopMode use case.
type SetLoopModeInput[C comparable] struct {
	ConnectionInfo C
	Mode           string
}

// SetLoopModeOutput holds the output for the SetLoopMode use case.
type SetLoopModeOutput struct {
	Changed bool
}

// SetLoopMode sets the loop mode to a specific value.
type SetLoopModeUsecase[C comparable] struct {
	playerStates       domain.PlayerStateRepository
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewSetLoopModeUsecase creates a new SetLoopMode use case.
func NewSetLoopModeUsecase[C comparable](
	playerStates domain.PlayerStateRepository,
	playerStateLocator ports.PlayerStateLocator[C],
) *SetLoopModeUsecase[C] {
	return &SetLoopModeUsecase[C]{
		playerStates:       playerStates,
		playerStateLocator: playerStateLocator,
	}
}

// Execute sets the loop mode.
func (uc *SetLoopModeUsecase[C]) Execute(
	ctx context.Context,
	input SetLoopModeInput[C],
) (*SetLoopModeOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	mode := domain.ParseLoopMode(input.Mode)
	prev := state.LoopMode()
	state.SetLoopMode(mode)

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &SetLoopModeOutput{Changed: prev != mode}, nil
}
