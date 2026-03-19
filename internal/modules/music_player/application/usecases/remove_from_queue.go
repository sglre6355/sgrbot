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
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewRemoveFromQueueUsecase creates a new RemoveFromQueue use case.
func NewRemoveFromQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *RemoveFromQueueUsecase[C] {
	return &RemoveFromQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute removes the track at the given index.
// If the index points to the currently playing track, it skips playback first.
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

	var events []domain.Event
	shouldRemoveCurrent := state.IsPlaybackActive() && input.Index == state.CurrentIndex()

	// If removing the currently playing track, skip first so that
	// `TrackStartedEvent` is emitted by `Skip` rather than by `Remove`.
	// This means events are ordered [TrackStartedEvent, TrackRemovedEvent]
	// instead of [TrackRemovedEvent, TrackStartedEvent].
	if shouldRemoveCurrent {
		_, skipEvents, err := uc.player.Skip(&state)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
		events = append(events, skipEvents...)
	}

	entry, removeEvents, err := uc.player.Remove(&state, input.Index)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidIndex) {
			return nil, ErrInvalidIndex
		}
		return nil, errors.Join(ErrInternal, err)
	}
	events = append(events, removeEvents...)

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if shouldRemoveCurrent {
		// Update audio only when playback advanced or stopped.
		// NOTE: The earlier Skip call may emit QueueExhaustedEvent, which can
		// trigger auto-play via HandleQueueExhausted (same as SkipTrackUsecase).
		// When the queue has remaining tracks, audio.Play starts the next track;
		// otherwise audio.Stop is called, but auto-play may restart playback.
		if current := state.Current(); current != nil {
			if err := uc.audio.Play(ctx, state.ID(), *current); err != nil {
				return nil, errors.Join(ErrInternal, err)
			}
		} else {
			if err := uc.audio.Stop(ctx, state.ID()); err != nil {
				return nil, errors.Join(ErrInternal, err)
			}
		}
	}

	uc.events.Publish(ctx, events...)

	return &RemoveFromQueueOutput{
		RemovedTrack: dtos.NewTrackView(entry.Track()),
	}, nil
}
