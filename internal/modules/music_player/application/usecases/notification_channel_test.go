package usecases

import (
	"context"
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func TestNotificationChannelService_Set(t *testing.T) {
	guildID := snowflake.ID(1)
	voiceChannelID := snowflake.ID(4)
	oldChannelID := snowflake.ID(3)
	newChannelID := snowflake.ID(99)

	t.Run("successfully updates notification channel", func(t *testing.T) {
		repo := newMockRepository()
		repo.createConnectedState(guildID, voiceChannelID, oldChannelID)

		service := NewNotificationChannelService(repo)
		err := service.Set(context.Background(), SetNotificationChannelInput{
			GuildID:   guildID,
			ChannelID: newChannelID,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		state, _ := repo.Get(context.Background(), guildID)
		if state.GetNotificationChannelID() != newChannelID {
			t.Errorf(
				"expected notification channel ID %d, got %d",
				newChannelID,
				state.GetNotificationChannelID(),
			)
		}
	})

	t.Run("returns ErrNotConnected when no state exists", func(t *testing.T) {
		repo := newMockRepository()

		service := NewNotificationChannelService(repo)
		err := service.Set(context.Background(), SetNotificationChannelInput{
			GuildID:   guildID,
			ChannelID: newChannelID,
		})
		if err != ErrNotConnected {
			t.Errorf("expected ErrNotConnected, got %v", err)
		}
	})
}
