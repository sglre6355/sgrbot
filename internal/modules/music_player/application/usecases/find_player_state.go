package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// FindPlayerStateInput holds the input for the FindPlayerState use case.
type FindPlayerStateInput struct {
	GuildID string
}

// FindPlayerStateOutput holds the output for the FindPlayerState use case.
type FindPlayerStateOutput struct {
	PlayerStateID *string
}

// FindPlayerState looks up the player state ID for a given connection.
type FindPlayerStateUsecase struct {
	voiceConnection ports.VoiceConnectionGateway[discord.VoiceConnectionInfo]
}

// NewFindPlayerStateUsecase creates a new FindPlayerState use case.
func NewFindPlayerStateUsecase(
	voiceConnection ports.VoiceConnectionGateway[discord.VoiceConnectionInfo],
) *FindPlayerStateUsecase {
	return &FindPlayerStateUsecase{voiceConnection: voiceConnection}
}

// Execute looks up the player state ID for the given connection info.
func (uc *FindPlayerStateUsecase) Execute(
	ctx context.Context,
	input FindPlayerStateInput,
) (*FindPlayerStateOutput, error) {
	connInfo, err := discord.NewVoiceConnectionInfo(input.GuildID, "")
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	id := uc.voiceConnection.FindPlayerStateID(ctx, connInfo)
	if id == nil {
		return nil, ErrNotConnected
	}

	s := id.String()
	return &FindPlayerStateOutput{
		PlayerStateID: &s,
	}, nil
}
