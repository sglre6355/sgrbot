package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// RemoveFromQueueInput holds the input for the RemoveFromQueue use case.
type RemoveFromQueueInput[C comparable] struct {
	ConnectionInfo C
	Index          int
}

// RemoveFromQueueOutput holds the output for the RemoveFromQueue use case.
type RemoveFromQueueOutput struct {
	RemovedTrack dtos.TrackView
}

// RemoveFromQueue removes a track at a given index from the queue.
type RemoveFromQueueUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewRemoveFromQueueUsecase creates a new RemoveFromQueue use case.
func NewRemoveFromQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *RemoveFromQueueUsecase[C] {
	return &RemoveFromQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute removes the track at the given index.
// Returns ErrIsCurrentTrack if the index points to the currently playing track.
func (uc *RemoveFromQueueUsecase[C]) Execute(
	ctx context.Context,
	input RemoveFromQueueInput[C],
) (*RemoveFromQueueOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if state.IsPlaybackActive() && input.Index == state.CurrentIndex() {
		return nil, ErrIsCurrentTrack
	}

	entry, events, err := uc.player.Remove(&state, input.Index)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidIndex) {
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
