package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// Ensure DiscordUserVoiceStateProvider implements required interfaces.
var (
	_ ports.UserVoiceStateProvider[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo] = (*DiscordUserVoiceStateProvider)(
		nil,
	)
)

// DiscordUserVoiceStateProvider looks up a user's current voice connection using Discord state.
type DiscordUserVoiceStateProvider struct {
	session *discordgo.Session
}

// NewDiscordUserVoiceStateProvider creates a new DiscordUserVoiceStateProvider.
func NewDiscordUserVoiceStateProvider(session *discordgo.Session) *DiscordUserVoiceStateProvider {
	return &DiscordUserVoiceStateProvider{session: session}
}

// GetUserVoiceConnectionInfo returns the voice connection info for the user's current voice session
// within the given guild.
func (p *DiscordUserVoiceStateProvider) GetUserVoiceConnectionInfo(
	_ context.Context,
	partialConnectionInfo discord.PartialVoiceConnectionInfo,
	userID core.UserID,
) (*discord.VoiceConnectionInfo, error) {
	guild, err := p.session.State.Guild(partialConnectionInfo.GuildID)
	if err != nil {
		return nil, nil
	}
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID.String() {
			return &discord.VoiceConnectionInfo{
				GuildID:   guild.ID,
				ChannelID: vs.ChannelID,
			}, nil
		}
	}
	return nil, nil
}
