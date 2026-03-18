package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// SkipTrackInput holds the input for the SkipTrack use case.
type SkipTrackInput[C comparable] struct {
	ConnectionInfo C
}

// SkipTrackOutput holds the output for the SkipTrack use case.
type SkipTrackOutput struct {
	SkippedTrack dtos.TrackView
}

// SkipTrack advances past the current track.
type SkipTrackUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewSkipTrackUsecase creates a new SkipTrack use case.
func NewSkipTrackUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *SkipTrackUsecase[C] {
	return &SkipTrackUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute skips the current track.
func (uc *SkipTrackUsecase[C]) Execute(
	ctx context.Context,
	input SkipTrackInput[C],
) (*SkipTrackOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	skipped, events, err := uc.player.Skip(&state)
	if err != nil {
		if errors.Is(err, domain.ErrNotPlaying) {
			return nil, ErrNotPlaying
		}
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if current := state.Current(); current != nil {
		if err := uc.audio.Play(ctx, state.ID(), *current); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	} else {
		if err := uc.audio.Stop(ctx, state.ID()); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}
	}

	uc.events.Publish(ctx, events...)

	return &SkipTrackOutput{
		SkippedTrack: dtos.NewTrackView(skipped.Track()),
	}, nil
}
