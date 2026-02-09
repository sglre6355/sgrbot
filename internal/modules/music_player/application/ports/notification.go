package ports

import (
	"github.com/disgoorg/snowflake/v2"
)

// NotificationSender defines the interface for sending notifications to Discord channels.
type NotificationSender interface {
	// SendNowPlaying sends a "Now Playing" embed to the channel and returns the message ID.
	SendNowPlaying(channelID snowflake.ID, info *NowPlayingInfo) (messageID snowflake.ID, err error)

	// DeleteMessage deletes a message from the channel.
	DeleteMessage(channelID snowflake.ID, messageID snowflake.ID) error

	// SendError sends an error message embed to the channel.
	SendError(channelID snowflake.ID, message string) error
}
