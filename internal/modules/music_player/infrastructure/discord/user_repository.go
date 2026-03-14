package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

// Ensure DiscordUserRepository implements core.UserRepository.
var (
	_ core.UserRepository = (*DiscordUserRepository)(nil)
)

// DiscordUserRepository implements core.UserRepository using a Discord session.
type DiscordUserRepository struct {
	session *discordgo.Session
}

// NewDiscordUserRepository creates a new DiscordUserRepository.
func NewDiscordUserRepository(session *discordgo.Session) *DiscordUserRepository {
	return &DiscordUserRepository{session: session}
}

// FindByID fetches display info for a user.
func (r *DiscordUserRepository) FindByID(userID core.UserID) (core.User, error) {
	user, err := r.session.User(userID.String())
	if err != nil {
		return core.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}

	return core.User{
		ID:        userID,
		Name:      user.DisplayName(),
		AvatarURL: user.AvatarURL(""),
	}, nil
}
