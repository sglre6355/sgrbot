package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// AddToQueueInput holds the input for the AddToQueue use case.
type AddToQueueInput[C comparable] struct {
	ConnectionInfo C
	TrackIDs       []string
	RequesterID    string
}

// AddToQueueOutput holds the output for the AddToQueue use case.
type AddToQueueOutput struct {
	StartIndex int
	Count      int
}

// AddToQueue resolves track IDs, creates queue entries, and appends them.
type AddToQueueUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	tracks             domain.TrackRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewAddToQueue creates a new AddToQueue use case.
func NewAddToQueueUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	tracks domain.TrackRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *AddToQueueUsecase[C] {
	return &AddToQueueUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		tracks:             tracks,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute adds tracks to the queue. If the player was idle, playback starts.
func (uc *AddToQueueUsecase[C]) Execute(
	ctx context.Context,
	input AddToQueueInput[C],
) (*AddToQueueOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	requesterID, err := domain.ParseUserID(input.RequesterID)
	if err != nil {
		return nil, errors.Join(ErrInvalidArgument, err)
	}

	trackIDs := make([]domain.TrackID, len(input.TrackIDs))
	for i, tid := range input.TrackIDs {
		trackIDs[i], err = domain.ParseTrackID(tid)
		if err != nil {
			return nil, errors.Join(ErrInvalidArgument, err)
		}
	}

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	entries := make([]domain.QueueEntry, 0, len(trackIDs))
	for _, tid := range trackIDs {
		track, err := uc.tracks.FindByID(ctx, tid)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
		entries = append(entries, domain.NewQueueEntry(track, requesterID, false))
	}

	startIndex, becameActive, events := uc.player.Append(&state, entries...)

	if becameActive {
		current := state.Current()
		if current == nil {
			return nil, errors.Join(
				ErrInternal,
				errors.New("player became active but has no current track"),
			)
		}
		if err := uc.playerStates.Save(ctx, state); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
		if err := uc.audio.Play(ctx, state.ID(), *current); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	} else {
		if err := uc.playerStates.Save(ctx, state); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	uc.events.Publish(ctx, events...)

	return &AddToQueueOutput{
		StartIndex: startIndex,
		Count:      len(entries),
	}, nil
}
