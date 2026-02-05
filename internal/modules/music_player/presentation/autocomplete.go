package presentation

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
	autocomplete *usecases.AutocompleteService
}

// NewAutocompleteHandler creates a new AutocompleteHandler.
func NewAutocompleteHandler(
	autocomplete *usecases.AutocompleteService,
) *AutocompleteHandler {
	return &AutocompleteHandler{
		autocomplete: autocomplete,
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

	output := h.autocomplete.GetQueueTracks(usecases.GetQueueTracksInput{
		GuildID: guildID,
	})
	if output.Tracks == nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{},
			},
		})
		return
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, min(len(output.Tracks), 25))
	for idx, track := range output.Tracks {
		if idx >= 25 {
			break
		}
		// Use 1-indexed positions to match queue list display
		displayPos := idx + 1
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  fmt.Sprintf("%d. %s", displayPos, truncate(track.Title, 90)),
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
	if h.autocomplete == nil {
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

	// Search for tracks
	result, err := h.autocomplete.SearchTracks(context.Background(), usecases.SearchTracksInput{
		Query: query,
		Limit: 10,
	})
	if err != nil || result.Tracks == nil {
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
			Name:  truncate(fmt.Sprintf("%s - %s", track.Title, track.Artist), 100),
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

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
