package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// ClearQueueInput holds the input for the ClearQueue use case.
type ClearQueueInput struct {
	PlayerStateID    string
	KeepCurrentTrack bool
}

// ClearQueueOutput holds the output for the ClearQueue use case.
type ClearQueueOutput struct {
	ClearedCount int
}

// ClearQueue removes entries from the queue.
type ClearQueueUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewClearQueue creates a new ClearQueue use case.
func NewClearQueueUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *ClearQueueUsecase {
	return &ClearQueueUsecase{
		player:       player,
		playerStates: playerStates,
		audio:        audio,
		events:       events,
	}
}

// Execute clears the queue. If KeepCurrentTrack is true, only played and
// upcoming entries are removed. Otherwise, all entries are removed and
// playback stops.
func (uc *ClearQueueUsecase) Execute(
	ctx context.Context,
	input ClearQueueInput,
) (*ClearQueueOutput, error) {
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

	var clearedCount int
	var events []core.Event
	if state.IsPlaybackActive() && input.KeepCurrentTrack {
		clearedCount, events, err = uc.player.ClearExceptCurrent(&state)
		if err != nil {
			if errors.Is(err, core.ErrNotPlaying) {
				return nil, ErrNotPlaying
			}
			return nil, errors.Join(ErrInternal, err)
		}
	} else {
		clearedCount, events = uc.player.Clear(&state)
		if err := uc.audio.Stop(ctx, state.ID()); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &ClearQueueOutput{ClearedCount: clearedCount}, nil
}
