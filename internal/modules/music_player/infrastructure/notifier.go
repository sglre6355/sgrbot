package infrastructure

import (
	"context"
	"fmt"
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

// Notifier sends notifications to Discord channels.
type Notifier struct {
	session    *discordgo.Session
	httpClient *http.Client
}

// NewNotifier creates a new Notifier.
func NewNotifier(session *discordgo.Session) *Notifier {
	return &Notifier{
		session: session,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SendNowPlaying sends a "Now Playing" embed to the channel and returns the message ID.
func (n *Notifier) SendNowPlaying(
	channelID snowflake.ID,
	info *ports.NowPlayingInfo,
) (snowflake.ID, error) {
	source := domain.ParseTrackSource(info.SourceName)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Now Playing",
			IconURL: source.IconURL(),
		},
		Title:     info.Title,
		URL:       info.URI,
		Color:     source.Color(),
		Timestamp: info.EnqueuedAt.UTC().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Artist",
				Value:  info.Artist,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Requested by %s", info.RequesterName),
			IconURL: info.RequesterAvatarURL,
		},
	}

	// Only show duration for non-stream tracks
	if !info.IsStream {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Duration",
			Value:  info.Duration,
			Inline: true,
		})
	}

	if thumbnailURL := n.getBestThumbnail(
		source,
		info.Identifier,
		info.ArtworkURL,
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

// SendQueueAdded sends a "Added to Queue" embed to the channel.
func (n *Notifier) SendQueueAdded(channelID snowflake.ID, info *ports.QueueAddedInfo) error {
	description := fmt.Sprintf("Added **%s** to the queue.", info.Title)

	embed := &discordgo.MessageEmbed{
		Description: description,
	}

	_, err := n.session.ChannelMessageSendEmbed(channelID.String(), embed)
	return err
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

// Ensure Notifier implements ports.NotificationSender.
var _ ports.NotificationSender = (*Notifier)(nil)
