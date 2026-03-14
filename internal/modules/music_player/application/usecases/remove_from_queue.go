package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// RemoveFromQueueInput holds the input for the RemoveFromQueue use case.
type RemoveFromQueueInput struct {
	PlayerStateID string
	Index         int
}

// RemoveFromQueueOutput holds the output for the RemoveFromQueue use case.
type RemoveFromQueueOutput struct {
	RemovedTrack dtos.TrackView
}

// RemoveFromQueue removes a track at a given index from the queue.
type RemoveFromQueueUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	events       ports.EventPublisher
}

// NewRemoveFromQueue creates a new RemoveFromQueue use case.
func NewRemoveFromQueueUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	events ports.EventPublisher,
) *RemoveFromQueueUsecase {
	return &RemoveFromQueueUsecase{
		player:       player,
		playerStates: playerStates,
		events:       events,
	}
}

// Execute removes the track at the given index.
// Returns ErrIsCurrentTrack if the index points to the currently playing track.
func (uc *RemoveFromQueueUsecase) Execute(
	ctx context.Context,
	input RemoveFromQueueInput,
) (*RemoveFromQueueOutput, error) {
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

	if state.IsPlaybackActive() && input.Index == state.CurrentIndex() {
		return nil, ErrIsCurrentTrack
	}

	entry, events, err := uc.player.Remove(&state, input.Index)
	if err != nil {
		if errors.Is(err, core.ErrInvalidIndex) {
			return nil, ErrInvalidIndex
		}
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &RemoveFromQueueOutput{
		RemovedTrack: dtos.NewTrackView(entry.Track()),
	}, nil
}
