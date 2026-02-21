package discord

import (
	"context"
	"fmt"
	"log/slog"

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

// HandlePlay handles autocomplete for play command.
func (h *AutocompleteHandler) HandlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

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

	// Search for tracks (regular search behavior)
	output, err := h.trackLoader.ResolveQuery(ctx, usecases.ResolveQueryInput{
		Query: query,
	})
	if err != nil || len(output.Tracks) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{},
			},
		})
		return
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)

	if output.IsPlaylist {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name: truncate(
				fmt.Sprintf("📋 %s (%d tracks)", output.PlaylistName, len(output.Tracks)),
				100,
			),
			Value: query,
		})
	}
	for i, track := range output.Tracks {
		var optionName string
		if output.IsPlaylist {
			optionName = fmt.Sprintf("🎵 %d. %s - %s", i+1, track.Title, track.Artist)
		} else {
			optionName = fmt.Sprintf("🎵 %s - %s", track.Title, track.Artist)
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  truncate(optionName, 100),
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
	ctx := context.Background()

	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		slog.Warn("failed to parse guild ID in autocomplete", "error", err, "guildID", i.GuildID)
		return
	}

	output, err := h.queue.List(ctx, usecases.QueueListInput{
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
		if output, err := h.trackLoader.LoadTrack(
			ctx,
			usecases.LoadTrackInput{TrackID: id},
		); err == nil {
			title = output.Track.Title
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

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
