package discord

// VoiceConnectionInfo identifies the Discord voice target to connect to.
type VoiceConnectionInfo struct {
	GuildID   string
	ChannelID string
}

// NewVoiceConnectionInfo creates a new VoiceConnectionInfo.
func NewVoiceConnectionInfo(guildID, channelID string) (VoiceConnectionInfo, error) {
	return VoiceConnectionInfo{GuildID: guildID, ChannelID: channelID}, nil
}

// PartialVoiceConnectionInfo scopes voice lookups to a specific guild.
type PartialVoiceConnectionInfo struct {
	GuildID string
}

// NewPartialVoiceConnectionInfo creates a new PartialVoiceConnectionInfo.
func NewPartialVoiceConnectionInfo(guildID string) (PartialVoiceConnectionInfo, error) {
	return PartialVoiceConnectionInfo{GuildID: guildID}, nil
}
