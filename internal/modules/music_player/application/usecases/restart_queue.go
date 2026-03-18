package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// RestartQueueInput holds the input for the RestartQueue use case.
type RestartQueueInput[C comparable] struct {
	ConnectionInfo C
}

// RestartQueueOutput holds the output for the RestartQueue use case.
type RestartQueueOutput struct{}

// RestartQueue seeks to the first track, replaying from the beginning.
type RestartQueueUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewRestartQueue creates a new RestartQueue use case.
func NewRestartQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *RestartQueueUsecase[C] {
	return &RestartQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute restarts the queue from the beginning.
func (uc *RestartQueueUsecase[C]) Execute(
	ctx context.Context,
	input RestartQueueInput[C],
) (*RestartQueueOutput, error) {
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

	entry, events := uc.player.Seek(&state, 0)
	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if entry != nil {
		if err := uc.audio.Play(ctx, state.ID(), *entry); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	uc.events.Publish(ctx, events...)

	return &RestartQueueOutput{}, nil
}
