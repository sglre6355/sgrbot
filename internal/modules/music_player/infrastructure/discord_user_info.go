package infrastructure

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

// Ensure DiscordUserInfoProvider implements ports.UserInfoProvider.
var (
	_ ports.UserInfoProvider = (*DiscordUserInfoProvider)(nil)
)

// DiscordUserInfoProvider implements ports.UserInfoProvider using a Discord session.
type DiscordUserInfoProvider struct {
	session *discordgo.Session
}

// NewDiscordUserInfoProvider creates a new DiscordUserInfoProvider.
func NewDiscordUserInfoProvider(session *discordgo.Session) *DiscordUserInfoProvider {
	return &DiscordUserInfoProvider{session: session}
}

// GetUserInfo fetches display info for a user in a guild.
func (p *DiscordUserInfoProvider) GetUserInfo(
	guildID, userID snowflake.ID,
) (*ports.UserInfo, error) {
	member, err := p.session.GuildMember(guildID.String(), userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guild member: %w", err)
	}

	displayName := getDisplayName(member)
	avatarURL := member.User.AvatarURL("")

	return &ports.UserInfo{
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}, nil
}

// getDisplayName returns the effective display name for a guild member.
// Priority: guild nickname > global display name > username.
func getDisplayName(member *discordgo.Member) string {
	if member.Nick != "" {
		return member.Nick
	}
	if member.User.GlobalName != "" {
		return member.User.GlobalName
	}
	return member.User.Username
}
