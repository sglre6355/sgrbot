package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// LeaveVoiceChannelInput holds the input for the LeaveVoiceChannel use case.
type LeaveVoiceChannelInput struct {
	PlayerStateID string
}

// LeaveVoiceChannelOutput holds the output for the LeaveVoiceChannel use case.
type LeaveVoiceChannelOutput struct{}

// LeaveVoiceChannel disconnects from the voice channel and cleans up.
type LeaveVoiceChannelUsecase struct {
	playerStates    core.PlayerStateRepository
	nowPlaying      ports.NowPlayingGateway[discord.NowPlayingDestination]
	voiceConnection ports.VoiceConnectionGateway[discord.VoiceConnectionInfo]
}

// NewLeaveVoiceChannelUsecase creates a new LeaveVoiceChannel use case.
func NewLeaveVoiceChannelUsecase(
	playerStates core.PlayerStateRepository,
	nowPlaying ports.NowPlayingGateway[discord.NowPlayingDestination],
	voiceConnection ports.VoiceConnectionGateway[discord.VoiceConnectionInfo],
) *LeaveVoiceChannelUsecase {
	return &LeaveVoiceChannelUsecase{
		playerStates:    playerStates,
		nowPlaying:      nowPlaying,
		voiceConnection: voiceConnection,
	}
}

// Execute leaves the voice channel and removes the player state.
func (uc *LeaveVoiceChannelUsecase) Execute(
	ctx context.Context,
	input LeaveVoiceChannelInput,
) (*LeaveVoiceChannelOutput, error) {
	playerStateID, err := core.ParsePlayerStateID(input.PlayerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.voiceConnection.Leave(ctx, playerStateID); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.nowPlaying.Clear(playerStateID); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Delete(ctx, playerStateID); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &LeaveVoiceChannelOutput{}, nil
}
