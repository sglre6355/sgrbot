package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// LeaveVoiceChannelInput holds the input for the LeaveVoiceChannel use case.
type LeaveVoiceChannelInput[P comparable] struct {
	ConnectionInfo P
}

// LeaveVoiceChannelOutput holds the output for the LeaveVoiceChannel use case.
type LeaveVoiceChannelOutput struct{}

// LeaveVoiceChannel disconnects from the voice channel and cleans up.
type LeaveVoiceChannelUsecase[C comparable, P comparable] struct {
	playerStates       domain.PlayerStateRepository
	nowPlaying         ports.NowPlayingPublisher
	playerStateLocator ports.PlayerStateLocator[P]
	voiceConnection    ports.VoiceConnectionGateway[C]
}

// NewLeaveVoiceChannelUsecase creates a new LeaveVoiceChannel use case.
func NewLeaveVoiceChannelUsecase[C comparable, P comparable](
	playerStates domain.PlayerStateRepository,
	nowPlaying ports.NowPlayingPublisher,
	playerStateLocator ports.PlayerStateLocator[P],
	voiceConnection ports.VoiceConnectionGateway[C],
) *LeaveVoiceChannelUsecase[C, P] {
	return &LeaveVoiceChannelUsecase[C, P]{
		playerStates:       playerStates,
		nowPlaying:         nowPlaying,
		playerStateLocator: playerStateLocator,
		voiceConnection:    voiceConnection,
	}
}

// Execute leaves the voice channel and removes the player state.
func (uc *LeaveVoiceChannelUsecase[C, P]) Execute(
	ctx context.Context,
	input LeaveVoiceChannelInput[P],
) (*LeaveVoiceChannelOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	if err := uc.voiceConnection.Leave(ctx, playerStateID); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := errors.Join(
		uc.nowPlaying.Clear(playerStateID),
		uc.playerStates.Delete(ctx, playerStateID),
	); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &LeaveVoiceChannelOutput{}, nil
}
