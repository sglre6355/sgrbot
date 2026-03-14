package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// SeekQueueInput holds the input for the SeekQueue use case.
type SeekQueueInput struct {
	PlayerStateID string
	Index         int
}

// SeekQueueOutput holds the output for the SeekQueue use case.
type SeekQueueOutput struct {
	Track dtos.TrackView
}

// SeekQueue jumps to a specific position in the queue.
type SeekQueueUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewSeekQueue creates a new SeekQueue use case.
func NewSeekQueueUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *SeekQueueUsecase {
	return &SeekQueueUsecase{
		player:       player,
		playerStates: playerStates,
		audio:        audio,
		events:       events,
	}
}

// Execute seeks to the given position.
func (uc *SeekQueueUsecase) Execute(
	ctx context.Context,
	input SeekQueueInput,
) (*SeekQueueOutput, error) {
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

	entry, events := uc.player.Seek(&state, input.Index)
	if entry == nil {
		return nil, ErrInvalidIndex
	}

	if err := uc.audio.Play(ctx, state.ID(), *entry); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &SeekQueueOutput{Track: dtos.NewTrackView(entry.Track())}, nil
}
