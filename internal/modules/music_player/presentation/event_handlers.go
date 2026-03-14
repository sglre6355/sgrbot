package presentation

import (
	"context"
	"errors"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
)

// EventHandlers handles Discord gateway events for the music player.
type EventHandlers struct {
	botUserID         string
	findPlayerState   *usecases.FindPlayerStateUsecase
	leaveVoiceChannel *usecases.LeaveVoiceChannelUsecase
}

// NewEventHandlers creates a new EventHandlers.
func NewEventHandlers(
	botUserID string,
	findPlayerState *usecases.FindPlayerStateUsecase,
	leaveVoiceChannel *usecases.LeaveVoiceChannelUsecase,
) *EventHandlers {
	return &EventHandlers{
		botUserID:         botUserID,
		findPlayerState:   findPlayerState,
		leaveVoiceChannel: leaveVoiceChannel,
	}
}

// HandleVoiceStateUpdate handles VoiceStateUpdate events for the bot.
func (h *EventHandlers) HandleVoiceStateUpdate(
	_ *discordgo.Session,
	event *discordgo.VoiceStateUpdate,
) {
	// Only handle updates for the bot itself
	if event.UserID != h.botUserID {
		return
	}

	ctx := context.Background()
	findPlayerStateOutput, err := h.findPlayerState.Execute(
		ctx,
		usecases.FindPlayerStateInput{
			GuildID: event.GuildID,
		},
	)
	if err != nil {
		if !errors.Is(err, usecases.ErrNotConnected) {
			slog.Error("failed to find player state",
				"guild", event.GuildID,
				"error", err,
			)
		}
		return
	}
	playerStateID := *findPlayerStateOutput.PlayerStateID

	disconnected := event.ChannelID == ""
	if !disconnected {
		return
	}

	_, err = h.leaveVoiceChannel.Execute(
		context.Background(),
		usecases.LeaveVoiceChannelInput{
			PlayerStateID: playerStateID,
		},
	)
	if err != nil {
		slog.Error("failed to handle bot voice state change",
			"guild", event.GuildID,
			"error", err,
		)
	}
}
