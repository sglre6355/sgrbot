package presentation

import (
	"context"
	"errors"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/platforms/discord"
)

// EventHandlers handles Discord gateway events for the music player.
type EventHandlers struct {
	botUserID         string
	leaveVoiceChannel *usecases.LeaveVoiceChannelUsecase[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo]
}

// NewEventHandlers creates a new EventHandlers.
func NewEventHandlers(
	botUserID string,
	leaveVoiceChannel *usecases.LeaveVoiceChannelUsecase[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo],
) *EventHandlers {
	return &EventHandlers{
		botUserID:         botUserID,
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

	disconnected := event.ChannelID == ""
	if !disconnected {
		return
	}

	connInfo := discord.PartialVoiceConnectionInfo{GuildID: event.GuildID}

	_, err := h.leaveVoiceChannel.Execute(
		context.Background(),
		usecases.LeaveVoiceChannelInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connInfo,
		},
	)
	if err != nil {
		if !errors.Is(err, usecases.ErrNotConnected) {
			slog.Error("failed to handle bot voice state change",
				"guild", event.GuildID,
				"error", err,
			)
		}
	}
}
