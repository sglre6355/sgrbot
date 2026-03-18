package discord

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/platforms/discord"
)

// Ensure DiscordNowPlayingGateway implements required interfaces.
var (
	_ ports.NowPlayingPublisher = (*DiscordNowPlayingGateway)(
		nil,
	)
	_ ports.NowPlayingDestinationSetter[discord.NowPlayingDestination] = (*DiscordNowPlayingGateway)(
		nil,
	)
)

// nowPlayingMessage represents a now-playing embed that has been sent to Discord.
type nowPlayingMessage struct {
	destination discord.NowPlayingDestination
	id          string
}

// displayState tracks where to send and what's currently shown for a player.
type displayState struct {
	destination discord.NowPlayingDestination
	nowPlaying  *nowPlayingMessage
}

// DiscordNowPlayingGateway sends now-playing notifications to Discord channels.
type DiscordNowPlayingGateway struct {
	session    *discordgo.Session
	httpClient *http.Client

	mu            sync.RWMutex
	displayStates map[domain.PlayerStateID]*displayState
}

// NewDiscordNowPlayingGateway creates a new DiscordNowPlayingGateway.
func NewDiscordNowPlayingGateway(session *discordgo.Session) *DiscordNowPlayingGateway {
	return &DiscordNowPlayingGateway{
		session: session,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		displayStates: make(map[domain.PlayerStateID]*displayState),
	}
}

// SetDestination associates a player state with a display destination.
func (n *DiscordNowPlayingGateway) SetDestination(
	playerStateID domain.PlayerStateID,
	destination discord.NowPlayingDestination,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	state, ok := n.displayStates[playerStateID]
	if !ok {
		state = &displayState{}
		n.displayStates[playerStateID] = state
	}
	state.destination = destination
}

// Show displays the now-playing information for the given track and requester.
func (n *DiscordNowPlayingGateway) Show(
	playerStateID domain.PlayerStateID,
	track domain.Track,
	requester domain.User,
	enqueuedAt time.Time,
) error {
	n.mu.RLock()
	state, ok := n.displayStates[playerStateID]
	n.mu.RUnlock()

	if !ok {
		return nil // No destination set, silently skip
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Now Playing",
			IconURL: sourceIconURL(track.Source()),
		},
		Title:     track.Title(),
		URL:       track.URL(),
		Color:     sourceColor(track.Source()),
		Timestamp: enqueuedAt.UTC().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Artist",
				Value:  track.Author(),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Added by %s", requester.Name),
			IconURL: requester.AvatarURL,
		},
	}

	// Only show duration for non-stream tracks
	if !track.IsStream() {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Duration",
			Value:  formatDuration(track.Duration()),
			Inline: true,
		})
	}

	if thumbnailURL := n.getBestThumbnail(
		track.Source(),
		string(track.ID()),
		track.ArtworkURL(),
	); thumbnailURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: thumbnailURL,
		}
	}

	msg, err := n.session.ChannelMessageSendEmbed(state.destination.ChannelID, embed)
	if err != nil {
		return err
	}

	// Store message for later deletion
	n.mu.Lock()
	if s, ok := n.displayStates[playerStateID]; ok {
		s.nowPlaying = &nowPlayingMessage{
			destination: state.destination,
			id:          msg.ID,
		}
	}
	n.mu.Unlock()

	return nil
}

// Clear removes the now-playing display for the given player state.
func (n *DiscordNowPlayingGateway) Clear(playerStateID domain.PlayerStateID) error {
	n.mu.Lock()
	state, ok := n.displayStates[playerStateID]
	if !ok || state.nowPlaying == nil {
		n.mu.Unlock()
		return nil
	}

	nowPlaying := *state.nowPlaying
	state.nowPlaying = nil
	n.mu.Unlock()

	err := n.session.ChannelMessageDelete(
		nowPlaying.destination.ChannelID,
		nowPlaying.id,
	)
	if err != nil {
		slog.Warn(
			"failed to delete previous now playing message",
			"playerStateID", playerStateID,
			"channelID", nowPlaying.destination.ChannelID,
			"messageID", nowPlaying.id,
			"error", err,
		)
	}

	return nil
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
func (n *DiscordNowPlayingGateway) getBestThumbnail(
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
func (n *DiscordNowPlayingGateway) getYouTubeThumbnail(videoID string, fallbackURL string) string {
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
func (n *DiscordNowPlayingGateway) getTwitchThumbnail(artworkURL string) string {
	if artworkURL == "" {
		return ""
	}

	highResURL := strings.Replace(artworkURL, "440x248", "1280x720", 1)
	if highResURL == artworkURL {
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
func (n *DiscordNowPlayingGateway) urlExists(ctx context.Context, url string) bool {
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
