package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// CycleLoopModeInput holds the input for the CycleLoopMode use case.
type CycleLoopModeInput[C comparable] struct {
	ConnectionInfo C
}

// CycleLoopModeOutput holds the output for the CycleLoopMode use case.
type CycleLoopModeOutput struct {
	NewMode string
}

// CycleLoopMode cycles through loop modes: None -> Track -> Queue -> None.
type CycleLoopModeUsecase[C comparable] struct {
	playerStates       domain.PlayerStateRepository
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewCycleLoopMode creates a new CycleLoopMode use case.
func NewCycleLoopModeUsecase[C comparable](
	playerStates domain.PlayerStateRepository,
	playerStateLocator ports.PlayerStateLocator[C],
) *CycleLoopModeUsecase[C] {
	return &CycleLoopModeUsecase[C]{
		playerStates:       playerStates,
		playerStateLocator: playerStateLocator,
	}
}

// Execute cycles to the next loop mode.
func (uc *CycleLoopModeUsecase[C]) Execute(
	ctx context.Context,
	input CycleLoopModeInput[C],
) (*CycleLoopModeOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	newMode := state.CycleLoopMode()

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &CycleLoopModeOutput{NewMode: newMode.String()}, nil
}
