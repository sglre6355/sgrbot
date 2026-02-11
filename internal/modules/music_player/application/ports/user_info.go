package ports

import (
	"github.com/disgoorg/snowflake/v2"
)

// UserInfo contains display information for a Discord user.
type UserInfo struct {
	DisplayName string
	AvatarURL   string
}

// UserInfoProvider defines the interface for fetching user display information.
type UserInfoProvider interface {
	// GetUserInfo returns display info for the given user in a guild.
	GetUserInfo(guildID, userID snowflake.ID) (*UserInfo, error)
}
