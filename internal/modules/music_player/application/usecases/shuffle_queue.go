package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// ShuffleQueueInput holds the input for the ShuffleQueue use case.
type ShuffleQueueInput[C comparable] struct {
	ConnectionInfo C
}

// ShuffleQueueOutput holds the output for the ShuffleQueue use case.
type ShuffleQueueOutput struct{}

// ShuffleQueue randomizes the order of entries in the queue.
type ShuffleQueueUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewShuffleQueueUsecase creates a new ShuffleQueue use case.
func NewShuffleQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *ShuffleQueueUsecase[C] {
	return &ShuffleQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute shuffles the queue.
func (uc *ShuffleQueueUsecase[C]) Execute(
	ctx context.Context,
	input ShuffleQueueInput[C],
) (*ShuffleQueueOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
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
