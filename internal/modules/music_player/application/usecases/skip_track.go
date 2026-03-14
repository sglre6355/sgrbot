package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// SkipTrackInput holds the input for the SkipTrack use case.
type SkipTrackInput struct {
	PlayerStateID string
}

// SkipTrackOutput holds the output for the SkipTrack use case.
type SkipTrackOutput struct {
	SkippedTrack dtos.TrackView
}

// SkipTrack advances past the current track.
type SkipTrackUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewSkipTrack creates a new SkipTrack use case.
func NewSkipTrackUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *SkipTrackUsecase {
	return &SkipTrackUsecase{
		player:       player,
		playerStates: playerStates,
		audio:        audio,
		events:       events,
	}
}

// Execute skips the current track.
func (uc *SkipTrackUsecase) Execute(
	ctx context.Context,
	input SkipTrackInput,
) (*SkipTrackOutput, error) {
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

	skipped, events, err := uc.player.Skip(&state)
	if err != nil {
		if errors.Is(err, core.ErrNotPlaying) {
			return nil, ErrNotPlaying
		}
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

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &SkipTrackOutput{
		SkippedTrack: dtos.NewTrackView(skipped.Track()),
	}, nil
}
