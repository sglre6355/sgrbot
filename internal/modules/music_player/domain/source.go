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

// Color returns the brand color for the source.
// Source: https://brandfetch.com/
func (s TrackSource) Color() int {
	switch s {
	case TrackSourceYouTube:
		return 0xff0000
	case TrackSourceSpotify:
		return 0x1ed760
	case TrackSourceSoundCloud:
		return 0xff5500
	case TrackSourceTwitch:
		return 0x9147ff
	default:
		return 0x000000
	}
}

// IconURL returns the brand icon URL for the source.
// Source: https://brandfetch.com/
func (s TrackSource) IconURL() string {
	switch s {
	case TrackSourceYouTube:
		return "https://cdn.brandfetch.io/idVfYwcuQz/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	case TrackSourceSpotify:
		return "https://cdn.brandfetch.io/id20mQyGeY/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	case TrackSourceSoundCloud:
		return "https://cdn.brandfetch.io/id3ytDFop3/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	case TrackSourceTwitch:
		return "https://cdn.brandfetch.io/idIwZCwD2f/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	default:
		return "https://cdn3.iconfinder.com/data/icons/iconpark-vol-2/48/play-256.png"
	}
}
