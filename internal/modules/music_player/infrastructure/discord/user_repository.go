package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Ensure DiscordUserRepository implements domain.UserRepository.
var (
	_ domain.UserRepository = (*DiscordUserRepository)(nil)
)

// DiscordUserRepository implements domain.UserRepository using a Discord session.
type DiscordUserRepository struct {
	session *discordgo.Session
}

// NewDiscordUserRepository creates a new DiscordUserRepository.
func NewDiscordUserRepository(session *discordgo.Session) *DiscordUserRepository {
	return &DiscordUserRepository{session: session}
}

// FindByID fetches display info for a user.
func (r *DiscordUserRepository) FindByID(userID domain.UserID) (domain.User, error) {
	user, err := r.session.User(userID.String())
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to fetch user: %w", err)
	}

	return domain.User{
		ID:        userID,
		Name:      user.DisplayName(),
		AvatarURL: user.AvatarURL(""),
	}, nil
}
