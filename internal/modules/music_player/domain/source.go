package domain

// TrackSource represents the origin platform of a track.
type TrackSource string

const (
	TrackSourceYouTube    TrackSource = "youtube"
	TrackSourceSpotify    TrackSource = "spotify"
	TrackSourceSoundCloud TrackSource = "soundcloud"
	TrackSourceTwitch     TrackSource = "twitch"
	TrackSourceOther      TrackSource = "other"
)

// ParseTrackSource converts a source name string to a TrackSource.
func ParseTrackSource(name string) TrackSource {
	switch name {
	case "youtube":
		return TrackSourceYouTube
	case "spotify":
		return TrackSourceSpotify
	case "soundcloud":
		return TrackSourceSoundCloud
	case "twitch":
		return TrackSourceTwitch
	default:
		return TrackSourceOther
	}
}
