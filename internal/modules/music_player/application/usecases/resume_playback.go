package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// ResumePlaybackInput holds the input for the ResumePlayback use case.
type ResumePlaybackInput struct {
	PlayerStateID string
}

// ResumePlaybackOutput holds the output for the ResumePlayback use case.
type ResumePlaybackOutput struct{}

// ResumePlayback resumes paused playback.
type ResumePlaybackUsecase struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
}

// NewResumePlayback creates a new ResumePlayback use case.
func NewResumePlaybackUsecase(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
) *ResumePlaybackUsecase {
	return &ResumePlaybackUsecase{
		player:       player,
		playerStates: playerStates,
		audio:        audio,
		events:       events,
	}
}

// Execute resumes playback.
func (uc *ResumePlaybackUsecase) Execute(
	ctx context.Context,
	input ResumePlaybackInput,
) (*ResumePlaybackOutput, error) {
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

	events, err := uc.player.Resume(&state)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrNotPlaying):
			return nil, ErrNotPlaying
		case errors.Is(err, core.ErrNotPaused):
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
