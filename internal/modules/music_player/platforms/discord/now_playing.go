package discord

// NowPlayingDestination identifies the Discord text channel used for now-playing updates.
type NowPlayingDestination struct {
	ChannelID string
}

// NewNowPlayingDestination creates a new NowPlayingDestination.
func NewNowPlayingDestination(channelID string) (NowPlayingDestination, error) {
	return NowPlayingDestination{ChannelID: channelID}, nil
}
