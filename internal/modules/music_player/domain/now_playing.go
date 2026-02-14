package domain

import "github.com/disgoorg/snowflake/v2"

// NowPlayingMessage stores the channel and message ID for a "Now Playing" message.
// Both values are needed for deletion since the message may be in a different channel
// than the current notification channel if the user switched channels while playing.
type NowPlayingMessage struct {
	ChannelID snowflake.ID
	MessageID snowflake.ID
}

func NewNowPlayingMessage(channelID snowflake.ID, messageID snowflake.ID) NowPlayingMessage {
	return NowPlayingMessage{
		ChannelID: channelID,
		MessageID: messageID,
	}
}
