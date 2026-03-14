package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// CycleLoopModeInput holds the input for the CycleLoopMode use case.
type CycleLoopModeInput struct {
	PlayerStateID string
}

// CycleLoopModeOutput holds the output for the CycleLoopMode use case.
type CycleLoopModeOutput struct {
	NewMode string
}

// CycleLoopMode cycles through loop modes: None -> Track -> Queue -> None.
type CycleLoopModeUsecase struct {
	playerStates core.PlayerStateRepository
}

// NewCycleLoopMode creates a new CycleLoopMode use case.
func NewCycleLoopModeUsecase(playerStates core.PlayerStateRepository) *CycleLoopModeUsecase {
	return &CycleLoopModeUsecase{playerStates: playerStates}
}

// Execute cycles to the next loop mode.
func (uc *CycleLoopModeUsecase) Execute(
	ctx context.Context,
	input CycleLoopModeInput,
) (*CycleLoopModeOutput, error) {
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

	newMode := state.CycleLoopMode()

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &CycleLoopModeOutput{NewMode: newMode.String()}, nil
}
