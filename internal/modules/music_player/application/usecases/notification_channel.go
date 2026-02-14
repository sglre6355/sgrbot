package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// NotificationChannelService handles updating the notification channel for a guild's player.
type NotificationChannelService struct {
	repo domain.PlayerStateRepository
}

// NewNotificationChannelService creates a new NotificationChannelService.
func NewNotificationChannelService(repo domain.PlayerStateRepository) *NotificationChannelService {
	return &NotificationChannelService{repo: repo}
}

// SetNotificationChannelInput contains the input for the Set use case.
type SetNotificationChannelInput struct {
	GuildID   snowflake.ID
	ChannelID snowflake.ID
}

// Set updates the notification channel for the guild's player state.
func (n *NotificationChannelService) Set(
	ctx context.Context,
	input SetNotificationChannelInput,
) error {
	state, err := n.repo.Get(ctx, input.GuildID)
	if err != nil {
		return ErrNotConnected
	}

	state.SetNotificationChannelID(input.ChannelID)

	return n.repo.Save(ctx, state)
}
