package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// JoinVoiceChannelInput holds the input for the JoinVoiceChannel use case.
// When ChannelID is provided, it connects directly to that channel.
// Otherwise, it determines the user's current voice channel via UserID.
type JoinVoiceChannelInput struct {
	GuildID   string
	UserID    string
	ChannelID *string
}

// JoinVoiceChannelOutput holds the output for the JoinVoiceChannel use case.
type JoinVoiceChannelOutput struct {
	PlayerStateID string
	ChannelID     string
}

// JoinVoiceChannelUsecase connects to a voice channel and initializes the player state.
type JoinVoiceChannelUsecase struct {
	playerStates    core.PlayerStateRepository
	userVoiceState  ports.UserVoiceStateProvider[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo]
	voiceConnection ports.VoiceConnectionGateway[discord.VoiceConnectionInfo]
}

// NewJoinVoiceChannelUsecase creates a new JoinVoiceChannel use case.
func NewJoinVoiceChannelUsecase(
	playerStates core.PlayerStateRepository,
	userVoiceState ports.UserVoiceStateProvider[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo],
	voiceConnection ports.VoiceConnectionGateway[discord.VoiceConnectionInfo],
) *JoinVoiceChannelUsecase {
	return &JoinVoiceChannelUsecase{
		playerStates:    playerStates,
		userVoiceState:  userVoiceState,
		voiceConnection: voiceConnection,
	}
}

// Execute joins the voice channel, or returns the existing player state if already connected.
func (uc *JoinVoiceChannelUsecase) Execute(
	ctx context.Context,
	input JoinVoiceChannelInput,
) (*JoinVoiceChannelOutput, error) {
	channelID := input.ChannelID

	// Resolve the user's current voice channel when no explicit channel is provided.
	if input.ChannelID == nil {
		userID, err := core.ParseUserID(input.UserID)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}

		partialConnectionInfo, err := discord.NewPartialVoiceConnectionInfo(input.GuildID)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}

		info, err := uc.userVoiceState.GetUserVoiceConnectionInfo(
			ctx,
			partialConnectionInfo,
			userID,
		)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}

		if info == nil {
			return nil, ErrUserNotInVoice
		}

		channelID = &info.ChannelID
	}

	connectionInfo, err := discord.NewVoiceConnectionInfo(input.GuildID, *channelID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if existingID := uc.voiceConnection.FindPlayerStateID(ctx, connectionInfo); existingID != nil {
		return &JoinVoiceChannelOutput{
			PlayerStateID: existingID.String(),
			ChannelID:     *channelID,
		}, nil
	}

	state := core.NewPlayerState()

	if err := uc.voiceConnection.Join(ctx, state.ID(), connectionInfo); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, *state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &JoinVoiceChannelOutput{
		PlayerStateID: state.ID().String(),
		ChannelID:     *channelID,
	}, nil
}
