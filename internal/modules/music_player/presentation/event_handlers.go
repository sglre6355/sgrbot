package presentation

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
)

// EventHandlers handles Discord gateway events for the music player.
type EventHandlers struct {
	botID        snowflake.ID
	voiceChannel *usecases.VoiceChannelService
}

// NewEventHandlers creates a new EventHandlers.
func NewEventHandlers(
	botID snowflake.ID,
	voiceChannel *usecases.VoiceChannelService,
) *EventHandlers {
	return &EventHandlers{
		botID:        botID,
		voiceChannel: voiceChannel,
	}
}

// HandleVoiceStateUpdate handles VoiceStateUpdate events for the bot.
func (h *EventHandlers) HandleVoiceStateUpdate(
	_ *discordgo.Session,
	event *discordgo.VoiceStateUpdate,
) {
	// Only handle updates for the bot itself
	if event.UserID != h.botID.String() {
		return
	}

	guildID, err := snowflake.Parse(event.GuildID)
	if err != nil {
		slog.Error("failed to parse guild ID in voice state update", "error", err)
		return
	}

	// Parse the channel ID - nil means disconnected
	var newChannelID *snowflake.ID
	if event.ChannelID != "" {
		id, err := snowflake.Parse(event.ChannelID)
		if err != nil {
			slog.Error("failed to parse channel ID in voice state update", "error", err)
			return
		}
		newChannelID = &id
	}

	h.voiceChannel.HandleBotVoiceStateChange(usecases.BotVoiceStateChangeInput{
		GuildID:      guildID,
		NewChannelID: newChannelID,
	})
}
