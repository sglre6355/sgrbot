package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// JoinVoiceChannelInput holds the input for the JoinVoiceChannel use case.
// When ConnectionInfo is provided, it connects directly to that target.
// Otherwise, it determines the user's current voice channel within the scope
// of PartialConnectionInfo via UserID.
type JoinVoiceChannelInput[C comparable, P comparable] struct {
	UserID                string
	ConnectionInfo        *C
	PartialConnectionInfo P
}

// JoinVoiceChannelOutput holds the output for the JoinVoiceChannel use case.
type JoinVoiceChannelOutput[C comparable] struct {
	PlayerStateID  string
	ConnectionInfo C
}

// JoinVoiceChannelUsecase connects to a voice channel and initializes the player state.
type JoinVoiceChannelUsecase[C comparable, P comparable] struct {
	playerStates       domain.PlayerStateRepository
	userVoiceState     ports.UserVoiceStateProvider[C, P]
	playerStateLocator ports.PlayerStateLocator[P]
	voiceConnection    ports.VoiceConnectionGateway[C]
}

// NewJoinVoiceChannelUsecase creates a new JoinVoiceChannel use case.
func NewJoinVoiceChannelUsecase[C comparable, P comparable](
	playerStates domain.PlayerStateRepository,
	userVoiceState ports.UserVoiceStateProvider[C, P],
	playerStateLocator ports.PlayerStateLocator[P],
	voiceConnection ports.VoiceConnectionGateway[C],
) *JoinVoiceChannelUsecase[C, P] {
	return &JoinVoiceChannelUsecase[C, P]{
		playerStates:       playerStates,
		userVoiceState:     userVoiceState,
		playerStateLocator: playerStateLocator,
		voiceConnection:    voiceConnection,
	}
}

// Execute joins the voice channel, or returns the existing player state if already connected.
func (uc *JoinVoiceChannelUsecase[C, P]) Execute(
	ctx context.Context,
	input JoinVoiceChannelInput[C, P],
) (*JoinVoiceChannelOutput[C], error) {
	var connectionInfo C

	// Resolve the user's current voice channel when no explicit connection target is provided.
	if input.ConnectionInfo == nil {
		userID, err := domain.ParseUserID(input.UserID)
		if err != nil {
			return nil, errors.Join(ErrInvalidArgument, err)
		}

		info, err := uc.userVoiceState.GetUserVoiceConnectionInfo(
			ctx,
			input.PartialConnectionInfo,
			userID,
		)
		if err != nil {
			return nil, errors.Join(ErrInternal, err)
		}

		if info == nil {
			return nil, ErrUserNotInVoice
		}

		connectionInfo = *info
	} else {
		connectionInfo = *input.ConnectionInfo
	}

	if existingID := uc.playerStateLocator.FindPlayerStateID(
		ctx,
		input.PartialConnectionInfo,
	); existingID != nil {
		if err := uc.voiceConnection.Join(ctx, *existingID, connectionInfo); err != nil {
			return nil, errors.Join(ErrInternal, err)
		}

		return &JoinVoiceChannelOutput[C]{
			PlayerStateID:  existingID.String(),
			ConnectionInfo: connectionInfo,
		}, nil
	}

	state := domain.NewPlayerState()

	if err := uc.voiceConnection.Join(ctx, state.ID(), connectionInfo); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if err := uc.playerStates.Save(ctx, *state); err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	return &JoinVoiceChannelOutput[C]{
		PlayerStateID:  state.ID().String(),
		ConnectionInfo: connectionInfo,
	}, nil
}
