package infrastructure

import (
	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

// Ensure VoiceStateProvider implements required ports.
var (
	_ ports.VoiceStateProvider = (*VoiceStateProvider)(nil)
)

// VoiceStateProvider provides Discord voice state information.
type VoiceStateProvider struct {
	session *discordgo.Session
}

// NewVoiceStateProvider creates a new VoiceStateProvider.
func NewVoiceStateProvider(session *discordgo.Session) *VoiceStateProvider {
	return &VoiceStateProvider{
		session: session,
	}
}

// GetUserVoiceChannel returns the voice channel ID that the user is currently in.
// Returns nil if the user is not in a voice channel.
func (v *VoiceStateProvider) GetUserVoiceChannel(
	guildID, userID snowflake.ID,
) (*snowflake.ID, error) {
	// Get guild from state
	guild, err := v.session.State.Guild(guildID.String())
	if err != nil {
		return nil, err
	}

	// Find user's voice state
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID.String() && vs.ChannelID != "" {
			channelID, err := snowflake.Parse(vs.ChannelID)
			if err != nil {
				return nil, err
			}
			return &channelID, nil
		}
	}

	return nil, nil
}
