package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// ResumePlaybackInput holds the input for the ResumePlayback use case.
type ResumePlaybackInput[C comparable] struct {
	ConnectionInfo C
}

// ResumePlaybackOutput holds the output for the ResumePlayback use case.
type ResumePlaybackOutput struct{}

// ResumePlayback resumes paused playback.
type ResumePlaybackUsecase[C comparable] struct {
	player             *domain.PlayerService
	playerStates       domain.PlayerStateRepository
	audio              ports.AudioGateway
	events             ports.EventPublisher
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewResumePlayback creates a new ResumePlayback use case.
func NewResumePlaybackUsecase[C comparable](
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	playerStateLocator ports.PlayerStateLocator[C],
) *ResumePlaybackUsecase[C] {
	return &ResumePlaybackUsecase[C]{
		player:             player,
		playerStates:       playerStates,
		audio:              audio,
		events:             events,
		playerStateLocator: playerStateLocator,
	}
}

// Execute resumes playback.
func (uc *ResumePlaybackUsecase[C]) Execute(
	ctx context.Context,
	input ResumePlaybackInput[C],
) (*ResumePlaybackOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	events, err := uc.player.Resume(&state)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotPlaying):
			return nil, ErrNotPlaying
		case errors.Is(err, domain.ErrNotPaused):
			return nil, ErrNotPaused
		default:
			return nil, errors.Join(ErrInternal, err)
		}
	}

	if err := uc.audio.Resume(ctx, state.ID()); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.events.Publish(ctx, events...)

	return &ResumePlaybackOutput{}, nil
}
