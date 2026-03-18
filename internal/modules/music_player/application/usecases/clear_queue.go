package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// ClearQueueInput holds the input for the ClearQueue use case.
type ClearQueueInput[C comparable] struct {
	ConnectionInfo   C
	KeepCurrentTrack bool
}

// ClearQueueOutput holds the output for the ClearQueue use case.
type ClearQueueOutput struct {
	ClearedCount int
}

// ClearQueue removes entries from the queue.
type ClearQueueUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewClearQueue creates a new ClearQueue use case.
func NewClearQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *ClearQueueUsecase[C] {
	return &ClearQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute clears the queue. If KeepCurrentTrack is true, only played and
// upcoming entries are removed. Otherwise, all entries are removed and
// playback stops.
func (uc *ClearQueueUsecase[C]) Execute(
	ctx context.Context,
	input ClearQueueInput[C],
) (*ClearQueueOutput, error) {
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

	var clearedCount int
	var events []domain.Event
	if state.IsPlaybackActive() && input.KeepCurrentTrack {
		clearedCount, events, err = uc.player.ClearExceptCurrent(&state)
		if err != nil {
			if errors.Is(err, domain.ErrNotPlaying) {
				return nil, ErrNotPlaying
			}
			return nil, errors.Join(ErrInternal, err)
		}
	} else {
		clearedCount, events = uc.player.Clear(&state)
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if !input.KeepCurrentTrack {
		if err := uc.audio.Stop(ctx, state.ID()); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	uc.events.Publish(ctx, events...)

	return &ClearQueueOutput{ClearedCount: clearedCount}, nil
}
