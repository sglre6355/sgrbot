package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// Embed colors.
const (
	colorRed = 0xE74C3C
)

// Ensure Notifier implements required ports.
var (
	_ ports.NotificationSender = (*Notifier)(nil)
)

// Notifier sends notifications to Discord channels.
type Notifier struct {
	session          *discordgo.Session
	trackProvider    ports.TrackProvider
	userInfoProvider ports.UserInfoProvider
	httpClient       *http.Client
}

// NewNotifier creates a new Notifier.
func NewNotifier(
	session *discordgo.Session,
	trackProvider ports.TrackProvider,
	userInfoProvider ports.UserInfoProvider,
) *Notifier {
	return &Notifier{
		session:          session,
		trackProvider:    trackProvider,
		userInfoProvider: userInfoProvider,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SendNowPlaying sends a "Now Playing" embed to the channel and returns the message ID.
func (n *Notifier) SendNowPlaying(
	guildID snowflake.ID,
	channelID snowflake.ID,
	trackID domain.TrackID,
	requesterID snowflake.ID,
	enqueuedAt time.Time,
) (snowflake.ID, error) {
	track, err := n.trackProvider.LoadTrack(context.Background(), trackID)
	if err != nil {
		return 0, fmt.Errorf("failed to load track %q: %w", trackID, err)
	}

	var requesterName, requesterAvatarURL string
	userInfo, err := n.userInfoProvider.GetUserInfo(guildID, requesterID)
	if err != nil {
		slog.Warn(
			"failed to fetch requester info for now playing",
			"guild", guildID,
			"requester", requesterID,
			"error", err,
		)
		requesterName = "Unknown"
	} else {
		requesterName = userInfo.DisplayName
		requesterAvatarURL = userInfo.AvatarURL
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Now Playing",
			IconURL: sourceIconURL(track.Source),
		},
		Title:     track.Title,
		URL:       track.URI,
		Color:     sourceColor(track.Source),
		Timestamp: enqueuedAt.UTC().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Artist",
				Value:  track.Artist,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Requested by %s", requesterName),
			IconURL: requesterAvatarURL,
		},
	}

	// Only show duration for non-stream tracks
	if !track.IsStream {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Duration",
			Value:  formatDuration(track.Duration),
			Inline: true,
		})
	}

	if thumbnailURL := n.getBestThumbnail(
		track.Source,
		string(track.ID),
		track.ArtworkURL,
	); thumbnailURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: thumbnailURL,
		}
	}

	msg, err := n.session.ChannelMessageSendEmbed(channelID.String(), embed)
	if err != nil {
		return 0, err
	}
	messageID, err := snowflake.Parse(msg.ID)
	if err != nil {
		return 0, err
	}
	return messageID, nil
}

// DeleteMessage deletes a message from the channel.
func (n *Notifier) DeleteMessage(channelID snowflake.ID, messageID snowflake.ID) error {
	return n.session.ChannelMessageDelete(channelID.String(), messageID.String())
}

// SendError sends an error message embed to the channel.
func (n *Notifier) SendError(channelID snowflake.ID, message string) error {
	embed := &discordgo.MessageEmbed{
		Description: message,
		Color:       colorRed,
	}

	_, err := n.session.ChannelMessageSendEmbed(channelID.String(), embed)
	return err
}

// formatDuration formats a time.Duration as "m:ss" or "h:mm:ss".
func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// sourceColor returns the brand color for the given track source.
func sourceColor(s domain.TrackSource) int {
	switch s {
	case domain.TrackSourceYouTube:
		return 0xff0000
	case domain.TrackSourceSpotify:
		return 0x1ed760
	case domain.TrackSourceSoundCloud:
		return 0xff5500
	case domain.TrackSourceTwitch:
		return 0x9147ff
	default:
		return 0x000000
	}
}

// sourceIconURL returns the brand icon URL for the given track source.
func sourceIconURL(s domain.TrackSource) string {
	switch s {
	case domain.TrackSourceYouTube:
		return "https://cdn.brandfetch.io/idVfYwcuQz/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	case domain.TrackSourceSpotify:
		return "https://cdn.brandfetch.io/id20mQyGeY/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	case domain.TrackSourceSoundCloud:
		return "https://cdn.brandfetch.io/id3ytDFop3/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	case domain.TrackSourceTwitch:
		return "https://cdn.brandfetch.io/idIwZCwD2f/w/400/h/400/theme/dark/icon.jpeg?c=1dxbfHSJFAPEGdCLU4o5B"
	default:
		return "https://cdn3.iconfinder.com/data/icons/iconpark-vol-2/48/play-256.png"
	}
}

// getBestThumbnail attempts to find the best quality thumbnail for the track.
// For YouTube, it tries different quality levels (maxresdefault, sddefault, etc.).
// For Twitch, it attempts to use a higher resolution version.
// For other sources, it returns the original artwork URL.
func (n *Notifier) getBestThumbnail(
	source domain.TrackSource,
	identifier string,
	fallbackURL string,
) string {
	switch source {
	case domain.TrackSourceYouTube:
		return n.getYouTubeThumbnail(identifier, fallbackURL)
	case domain.TrackSourceTwitch:
		return n.getTwitchThumbnail(fallbackURL)
	default:
		return fallbackURL
	}
}

// getYouTubeThumbnail tries to find the highest quality YouTube thumbnail available.
func (n *Notifier) getYouTubeThumbnail(videoID string, fallbackURL string) string {
	qualities := []string{"maxresdefault", "sddefault", "hqdefault", "mqdefault"}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, quality := range qualities {
		url := fmt.Sprintf("https://img.youtube.com/vi/%s/%s.jpg", videoID, quality)
		if n.urlExists(ctx, url) {
			return url
		}
	}

	return fallbackURL
}

// getTwitchThumbnail tries to get a higher resolution Twitch thumbnail.
func (n *Notifier) getTwitchThumbnail(artworkURL string) string {
	if artworkURL == "" {
		return ""
	}

	// Try to get 1280x720 instead of 440x248
	highResURL := strings.Replace(artworkURL, "440x248", "1280x720", 1)
	if highResURL == artworkURL {
		// No replacement made, return original
		return artworkURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if n.urlExists(ctx, highResURL) {
		return highResURL
	}

	return artworkURL
}

// urlExists checks if a URL returns a successful response using a HEAD request.
func (n *Notifier) urlExists(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}
