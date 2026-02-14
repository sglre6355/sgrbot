package discord

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
)

// AutocompleteHandler handles autocomplete requests.
type AutocompleteHandler struct {
	queue       *usecases.QueueService
	trackLoader *usecases.TrackLoaderService
}

// NewAutocompleteHandler creates a new AutocompleteHandler.
func NewAutocompleteHandler(
	queue *usecases.QueueService,
	trackLoader *usecases.TrackLoaderService,
) *AutocompleteHandler {
	return &AutocompleteHandler{
		queue:       queue,
		trackLoader: trackLoader,
	}
}

// HandleQueueRemove handles autocomplete for queue remove command.
func (h *AutocompleteHandler) HandleQueueRemove(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	h.handleQueuePositionAutocomplete(s, i)
}

// HandleQueueSeek handles autocomplete for queue seek command.
func (h *AutocompleteHandler) HandleQueueSeek(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	h.handleQueuePositionAutocomplete(s, i)
}

// handleQueuePositionAutocomplete is a shared helper for queue position autocomplete.
func (h *AutocompleteHandler) handleQueuePositionAutocomplete(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		slog.Warn("failed to parse guild ID in autocomplete", "error", err, "guildID", i.GuildID)
		return
	}

	output, err := h.queue.List(context.Background(), usecases.QueueListInput{
		GuildID:  guildID,
		Page:     1,
		PageSize: 25,
	})
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{},
			},
		})
		return
	}

	// Combine all track IDs into a flat list for autocomplete choices
	var allIDs []string
	allIDs = append(allIDs, output.PlayedTrackIDs...)
	if output.CurrentTrackID != "" {
		allIDs = append(allIDs, output.CurrentTrackID)
	}
	allIDs = append(allIDs, output.UpcomingTrackIDs...)

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(allIDs))
	for idx, id := range allIDs {
		// Use 1-indexed positions to match queue list display
		displayPos := output.PageStart + idx + 1
		title := id // fallback to ID if track info unavailable
		if info, err := h.trackLoader.LoadTrack(context.Background(), id); err == nil {
			title = info.Title
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%d. %s", displayPos, truncate(title, 90)),
			Value: displayPos,
		})
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// HandlePlay handles autocomplete for play command.
func (h *AutocompleteHandler) HandlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if h.trackLoader == nil {
		return
	}

	// Get the current query value
	var query string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "query" && opt.Focused {
			query = opt.StringValue()
			break
		}
	}

	// Don't search for very short queries
	if len(query) < 2 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{},
			},
		})
		return
	}

	// Check if query is a URL (potential playlist)
	if _, err := url.ParseRequestURI(query); err == nil {
		result, err := h.trackLoader.PreviewQuery(
			context.Background(),
			usecases.PreviewQueryInput{
				Query: query,
				Limit: 24, // Leave room for playlist option
			},
		)
		if err == nil && result.IsPlaylist && len(result.Tracks) > 0 {
			choices := buildPlaylistChoices(result, query)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: choices,
				},
			})
			return
		}
		// If not a playlist or error, fall through to regular search
	}

	// Search for tracks (regular search behavior)
	result, err := h.trackLoader.PreviewQuery(context.Background(), usecases.PreviewQueryInput{
		Query: query,
		Limit: 10,
	})
	if err != nil || len(result.Tracks) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{},
			},
		})
		return
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(result.Tracks))
	for _, track := range result.Tracks {
		// Use the URI as the value so it can be played directly
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  truncate(fmt.Sprintf("🎵 %s - %s", track.Title, track.Artist), 100),
			Value: track.URI,
		})
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// buildPlaylistChoices builds autocomplete choices for a playlist result.
func buildPlaylistChoices(
	result *usecases.PreviewQueryOutput,
	playlistURL string,
) []*discordgo.ApplicationCommandOptionChoice {
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(result.Tracks)+1)

	// First choice: entire playlist
	choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
		Name: truncate(
			fmt.Sprintf("📋 %s (%d tracks)", result.PlaylistName, result.TotalTracks),
			100,
		),
		Value: playlistURL,
	})

	// Remaining choices: individual tracks
	for i, track := range result.Tracks {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  truncate(fmt.Sprintf("🎵 %d. %s - %s", i+1, track.Title, track.Artist), 100),
			Value: track.URI,
		})
	}

	return choices
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
