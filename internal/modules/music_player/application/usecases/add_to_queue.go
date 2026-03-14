package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// AddToQueueInput holds the input for the AddToQueue use case.
type AddToQueueInput struct {
	PlayerStateID string
	TrackIDs      []string
	RequesterID   string
}

// AddToQueueOutput holds the output for the AddToQueue use case.
type AddToQueueOutput struct {
	StartIndex int
	Count      int
}

// AddToQueue resolves track IDs, creates queue entries, and appends them.
type AddToQueueUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	tracks       core.TrackRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewAddToQueue creates a new AddToQueue use case.
func NewAddToQueueUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	tracks core.TrackRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *AddToQueueUsecase {
	return &AddToQueueUsecase{
		player:       player,
		playerStates: playerStates,
		tracks:       tracks,
		audio:        audio,
		events:       events,
	}
}

// Execute adds tracks to the queue. If the player was idle, playback starts.
func (uc *AddToQueueUsecase) Execute(
	ctx context.Context,
	input AddToQueueInput,
) (*AddToQueueOutput, error) {
	playerStateID, err := core.ParsePlayerStateID(input.PlayerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	requesterID, err := core.ParseUserID(input.RequesterID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	trackIDs := make([]core.TrackID, len(input.TrackIDs))
	for i, id := range input.TrackIDs {
		trackIDs[i], err = core.ParseTrackID(id)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		if errors.Is(err, core.ErrPlayerStateNotFound) {
			return nil, ErrPlayerStateNotFound
		}
		return nil, errors.Join(ErrInternal, err)
	}

	entries := make([]core.QueueEntry, 0, len(trackIDs))
	for _, id := range trackIDs {
		track, err := uc.tracks.FindByID(ctx, id)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
		entries = append(entries, core.NewQueueEntry(track, requesterID, false))
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
		if err := uc.audio.Play(ctx, state.ID(), *current); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &AddToQueueOutput{
		StartIndex: startIndex,
		Count:      len(entries),
	}, nil
}
