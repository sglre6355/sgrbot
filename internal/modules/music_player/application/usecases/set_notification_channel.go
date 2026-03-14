package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// SetNotificationChannelInput holds the input for the SetNotificationChannel use case.
type SetNotificationChannelInput struct {
	PlayerStateID string
	ChannelID     string
}

// SetNotificationChannelOutput holds the output for the SetNotificationChannel use case.
type SetNotificationChannelOutput struct{}

// SetNotificationChannel updates the now-playing display destination.
type SetNotificationChannelUsecase struct {
	nowPlaying ports.NowPlayingGateway[discord.NowPlayingDestination]
}

// NewSetNotificationChannelUsecase creates a new SetNotificationChannel use case.
func NewSetNotificationChannelUsecase(
	nowPlaying ports.NowPlayingGateway[discord.NowPlayingDestination],
) *SetNotificationChannelUsecase {
	return &SetNotificationChannelUsecase{nowPlaying: nowPlaying}
}

// Execute sets the notification channel destination.
func (uc *SetNotificationChannelUsecase) Execute(
	_ context.Context,
	input SetNotificationChannelInput,
) (*SetNotificationChannelOutput, error) {
	playerStateID, err := core.ParsePlayerStateID(input.PlayerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	destination, err := discord.NewNowPlayingDestination(input.ChannelID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	uc.nowPlaying.SetDestination(playerStateID, destination)
	return &SetNotificationChannelOutput{}, nil
}
