package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// RestartQueueInput holds the input for the RestartQueue use case.
type RestartQueueInput struct {
	PlayerStateID string
}

// RestartQueueOutput holds the output for the RestartQueue use case.
type RestartQueueOutput struct{}

// RestartQueue seeks to the first track, replaying from the beginning.
type RestartQueueUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewRestartQueue creates a new RestartQueue use case.
func NewRestartQueueUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *RestartQueueUsecase {
	return &RestartQueueUsecase{
		player:       player,
		playerStates: playerStates,
		audio:        audio,
		events:       events,
	}
}

// Execute restarts the queue from the beginning.
func (uc *RestartQueueUsecase) Execute(
	ctx context.Context,
	input RestartQueueInput,
) (*RestartQueueOutput, error) {
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

	entry, events := uc.player.Seek(&state, 0)
	if entry != nil {
		if err := uc.audio.Play(ctx, state.ID(), *entry); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &RestartQueueOutput{}, nil
}
