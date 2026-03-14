package presentation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
)

const queueAutocompletePageSize = 25

// AutocompleteHandler handles autocomplete requests.
type AutocompleteHandler struct {
	findPlayerState *usecases.FindPlayerStateUsecase
	listQueue       *usecases.ListQueueUsecase
	resolveQuery    *usecases.ResolveQueryUsecase
}

// NewAutocompleteHandler creates a new AutocompleteHandler.
func NewAutocompleteHandler(
	findPlayerState *usecases.FindPlayerStateUsecase,
	listQueue *usecases.ListQueueUsecase,
	resolveQuery *usecases.ResolveQueryUsecase,
) *AutocompleteHandler {
	return &AutocompleteHandler{
		findPlayerState: findPlayerState,
		listQueue:       listQueue,
		resolveQuery:    resolveQuery,
	}
}

// HandleInteractionCreate routes autocomplete interactions to the appropriate handler.
func (h *AutocompleteHandler) HandleInteractionCreate(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	if i.Type != discordgo.InteractionApplicationCommandAutocomplete {
		return
	}

	data := i.ApplicationCommandData()

	switch data.Name {
	case "play":
		h.handlePlay(s, i)
	case "queue":
		if len(data.Options) > 0 {
			switch data.Options[0].Name {
			case "remove":
				h.handleQueueRemove(s, i)
			case "seek":
				h.handleQueueSeek(s, i)
			}
		}
	}
}

// handlePlay handles autocomplete for play command.
func (h *AutocompleteHandler) handlePlay(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	output, err := h.resolveQuery.Execute(ctx, usecases.ResolveQueryInput{
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

	if output.Type == "playlist" && output.Name != nil && output.URL != nil {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name: truncate(
				fmt.Sprintf("📋 %s (%d tracks)", *output.Name, len(output.Tracks)),
				100,
			),
			Value: *output.URL,
		})
	}
	for idx, track := range output.Tracks {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  truncate(fmt.Sprintf("🎵 %d. %s - %s", idx+1, track.Title, track.Author), 100),
			Value: track.URL,
		})
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// handleQueueRemove handles autocomplete for queue remove command.
func (h *AutocompleteHandler) handleQueueRemove(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	h.handleQueuePositionAutocomplete(s, i)
}

// handleQueueSeek handles autocomplete for queue seek command.
func (h *AutocompleteHandler) handleQueueSeek(
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

	findPlayerStateOutput, err := h.findPlayerState.Execute(
		ctx,
		usecases.FindPlayerStateInput{
			GuildID: i.GuildID,
		},
	)
	if err != nil {
		if errors.Is(err, usecases.ErrNotConnected) {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: []*discordgo.ApplicationCommandOptionChoice{},
				},
			})
		} else {
			slog.Error("failed to find player state",
				"guild", i.GuildID,
				"error", err,
			)
		}
		return
	}

	listQueueOutput, err := h.listQueue.Execute(ctx, usecases.ListQueueInput{
		PlayerStateID: *findPlayerStateOutput.PlayerStateID,
		Page:          1,
		PageSize:      queueAutocompletePageSize,
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

	// Combine all tracks into a flat list for autocomplete choices
	var allTracks []dtos.TrackView
	allTracks = append(allTracks, listQueueOutput.PlayedTracks...)
	if listQueueOutput.CurrentTrack != nil {
		allTracks = append(allTracks, *listQueueOutput.CurrentTrack)
	}
	allTracks = append(allTracks, listQueueOutput.UpcomingTracks...)

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(allTracks))
	for i, track := range allTracks {
		displayPos := listQueueOutput.PageStart + i + 1
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

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
