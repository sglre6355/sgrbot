package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// SeekQueueInput holds the input for the SeekQueue use case.
type SeekQueueInput[C comparable] struct {
	ConnectionInfo C
	Index          int
}

// SeekQueueOutput holds the output for the SeekQueue use case.
type SeekQueueOutput struct {
	Track dtos.TrackView
}

// SeekQueue jumps to a specific position in the queue.
type SeekQueueUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewSeekQueue creates a new SeekQueue use case.
func NewSeekQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *SeekQueueUsecase[C] {
	return &SeekQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute seeks to the given position.
func (uc *SeekQueueUsecase[C]) Execute(
	ctx context.Context,
	input SeekQueueInput[C],
) (*SeekQueueOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	entry, events := uc.player.Seek(&state, input.Index)
	if entry == nil {
		return nil, ErrInvalidIndex
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.audio.Play(ctx, state.ID(), *entry); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &SeekQueueOutput{Track: dtos.NewTrackView(entry.Track())}, nil
}
