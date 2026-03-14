package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// ShuffleQueueInput holds the input for the ShuffleQueue use case.
type ShuffleQueueInput struct {
	PlayerStateID string
}

// ShuffleQueueOutput holds the output for the ShuffleQueue use case.
type ShuffleQueueOutput struct{}

// ShuffleQueue randomizes the order of entries in the queue.
type ShuffleQueueUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	events       ports.EventPublisher
}

// NewShuffleQueue creates a new ShuffleQueue use case.
func NewShuffleQueueUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	events ports.EventPublisher,
) *ShuffleQueueUsecase {
	return &ShuffleQueueUsecase{
		player:       player,
		playerStates: playerStates,
		events:       events,
	}
}

// Execute shuffles the queue.
func (uc *ShuffleQueueUsecase) Execute(
	ctx context.Context,
	input ShuffleQueueInput,
) (*ShuffleQueueOutput, error) {
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

	if state.IsEmpty() {
		return nil, ErrQueueEmpty
	}

	events := uc.player.Shuffle(&state)

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &ShuffleQueueOutput{}, nil
}
