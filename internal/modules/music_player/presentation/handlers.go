package presentation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
)

// Embed colors.
const (
	colorSuccess = 0x08c404
	colorError   = 0xE74C3C
)

// Handlers holds all the command handlers.
type Handlers struct {
	voiceChannel *usecases.VoiceChannelService
	playback     *usecases.PlaybackService
	queue        *usecases.QueueService
	trackLoader  *usecases.TrackLoaderService
}

// NewHandlers creates new Handlers.
func NewHandlers(
	voiceChannel *usecases.VoiceChannelService,
	playback *usecases.PlaybackService,
	queue *usecases.QueueService,
	trackLoader *usecases.TrackLoaderService,
) *Handlers {
	return &Handlers{
		voiceChannel: voiceChannel,
		playback:     playback,
		queue:        queue,
		trackLoader:  trackLoader,
	}
}

// HandleJoin handles the /join command.
func (h *Handlers) HandleJoin(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	userID, err := snowflake.Parse(i.Member.User.ID)
	if err != nil {
		return respondError(r, "Invalid user")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	var voiceChannelID snowflake.ID
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "channel" {
			voiceChannelID, _ = snowflake.Parse(opt.ChannelValue(s).ID)
		}
	}

	input := usecases.JoinInput{
		GuildID:               guildID,
		UserID:                userID,
		NotificationChannelID: notificationChannelID,
		VoiceChannelID:        voiceChannelID,
	}

	output, err := h.voiceChannel.Join(context.Background(), input)
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondJoined(r, output.VoiceChannelID)
}

// HandleLeave handles the /leave command.
func (h *Handlers) HandleLeave(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	input := usecases.LeaveInput{
		GuildID: guildID,
	}

	if err := h.voiceChannel.Leave(context.Background(), input); err != nil {
		return respondError(r, err.Error())
	}

	return respondDisconnected(r)
}

// HandlePlay handles the /play command.
func (h *Handlers) HandlePlay(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	userID, err := snowflake.Parse(i.Member.User.ID)
	if err != nil {
		return respondError(r, "Invalid user")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	var query string
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "query" {
			query = opt.StringValue()
		}
	}

	// 1. Join voice channel (or update notification channel if already connected)
	_, err = h.voiceChannel.Join(ctx, usecases.JoinInput{
		GuildID:               guildID,
		UserID:                userID,
		NotificationChannelID: notificationChannelID,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	// 2. Load tracks via TrackLoaderService (may be single track or playlist)
	tracksOutput, err := h.trackLoader.ResolveQuery(ctx, usecases.ResolveQueryInput{
		Query: query,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	// 3. Add to queue based on result type
	if len(tracksOutput.Tracks) == 1 {
		// Single track - use existing Add method
		_, err = h.queue.Add(ctx, usecases.QueueAddInput{
			GuildID:               guildID,
			TrackID:               tracksOutput.Tracks[0].ID,
			RequesterID:           userID,
			NotificationChannelID: notificationChannelID,
		})
		if err != nil {
			return respondError(r, err.Error())
		}
		return respondQueueAdded(r, tracksOutput.Tracks[0])
	}

	// Playlist - use AddMultiple method
	trackIDs := make([]usecases.TrackID, len(tracksOutput.Tracks))
	for i, t := range tracksOutput.Tracks {
		trackIDs[i] = t.ID
	}
	output, err := h.queue.AddMultiple(ctx, usecases.QueueAddMultipleInput{
		GuildID:               guildID,
		TrackIDs:              trackIDs,
		RequesterID:           userID,
		NotificationChannelID: notificationChannelID,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondPlaylistAdded(r, tracksOutput.PlaylistName, output.Count)
}

// HandleStop handles the /stop command.
// Stop is a presentation concept: clear queue + skip current track.
func (h *Handlers) HandleStop(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	ctx := context.Background()

	// Clear the entire queue (ignore error if already empty)
	_, _ = h.queue.Clear(ctx, usecases.QueueClearInput{
		GuildID:               guildID,
		KeepCurrentTrack:      false, // Clear everything
		NotificationChannelID: notificationChannelID,
	})

	// Skip the current track (stops playback since queue is now empty)
	_, err = h.playback.Skip(ctx, usecases.SkipInput{
		GuildID:               guildID,
		NotificationChannelID: notificationChannelID,
	})
	if err != nil && !errors.Is(err, usecases.ErrNotPlaying) {
		return respondError(r, err.Error())
	}

	return respondStopped(r)
}

// HandlePause handles the /pause command.
func (h *Handlers) HandlePause(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	input := usecases.PauseInput{
		GuildID:               guildID,
		NotificationChannelID: notificationChannelID,
	}

	if err := h.playback.Pause(context.Background(), input); err != nil {
		return respondError(r, err.Error())
	}

	return respondPaused(r)
}

// HandleResume handles the /resume command.
func (h *Handlers) HandleResume(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	input := usecases.ResumeInput{
		GuildID:               guildID,
		NotificationChannelID: notificationChannelID,
	}

	if err := h.playback.Resume(context.Background(), input); err != nil {
		return respondError(r, err.Error())
	}

	return respondResumed(r)
}

// HandleSkip handles the /skip command.
func (h *Handlers) HandleSkip(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	input := usecases.SkipInput{
		GuildID:               guildID,
		NotificationChannelID: notificationChannelID,
	}

	output, err := h.playback.Skip(context.Background(), input)
	if err != nil {
		return respondError(r, err.Error())
	}

	// Respond with skipped confirmation - "Now Playing" is sent as a separate message by PlaybackService
	return respondSkipped(r, output.SkippedTrack)
}

// HandleQueue handles the /queue command.
func (h *Handlers) HandleQueue(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return respondError(r, "Invalid subcommand")
	}

	subCmd := options[0]
	switch subCmd.Name {
	case "list":
		return h.handleQueueList(s, i, r, subCmd.Options)
	case "remove":
		return h.handleQueueRemove(s, i, r, subCmd.Options)
	case "clear":
		return h.handleQueueClear(s, i, r)
	case "restart":
		return h.handleQueueRestart(s, i, r)
	case "seek":
		return h.handleQueueSeek(s, i, r, subCmd.Options)
	default:
		return respondError(r, "Unknown subcommand")
	}
}

func (h *Handlers) handleQueueList(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	var page int // let service default to page containing current track
	for _, opt := range options {
		if opt.Name == "page" {
			page = int(opt.IntValue())
		}
	}

	input := usecases.QueueListInput{
		GuildID:               guildID,
		Page:                  page,
		NotificationChannelID: notificationChannelID,
	}

	output, err := h.queue.List(context.Background(), input)
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondQueueList(r, output)
}

func (h *Handlers) handleQueueRemove(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	var position int
	for _, opt := range options {
		if opt.Name == "position" {
			// Convert from 1-indexed (user input) to 0-indexed (internal)
			position = int(opt.IntValue()) - 1
		}
	}

	input := usecases.QueueRemoveInput{
		GuildID:               guildID,
		Position:              position,
		NotificationChannelID: notificationChannelID,
	}

	ctx := context.Background()

	output, err := h.queue.Remove(ctx, input)
	if err != nil {
		// If trying to remove current track, skip first then remove
		if errors.Is(err, usecases.ErrIsCurrentTrack) {
			skipOutput, skipErr := h.playback.Skip(ctx, usecases.SkipInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
			})
			if skipErr != nil {
				return respondError(r, skipErr.Error())
			}
			// After skip, currentIndex has advanced, so we can now remove the track
			// at the original position (which is now in the "played" section)
			_, _ = h.queue.Remove(ctx, usecases.QueueRemoveInput{
				GuildID:               guildID,
				Position:              position,
				NotificationChannelID: notificationChannelID,
			})
			return respondQueueRemoved(r, skipOutput.SkippedTrack)
		}
		return respondError(r, err.Error())
	}

	return respondQueueRemoved(r, output.RemovedTrack)
}

func (h *Handlers) handleQueueClear(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	input := usecases.QueueClearInput{
		GuildID:               guildID,
		KeepCurrentTrack:      true, // Clear played + upcoming, keep only current
		NotificationChannelID: notificationChannelID,
	}

	_, err = h.queue.Clear(context.Background(), input)
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondQueueCleared(r)
}

func (h *Handlers) handleQueueRestart(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	_, err = h.queue.Restart(context.Background(), usecases.QueueRestartInput{
		GuildID:               guildID,
		NotificationChannelID: notificationChannelID,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondQueueRestarted(r)
}

func (h *Handlers) handleQueueSeek(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	var position int
	for _, opt := range options {
		if opt.Name == "position" {
			// Convert from 1-indexed (user input) to 0-indexed (internal)
			position = int(opt.IntValue()) - 1
		}
	}

	output, err := h.queue.Seek(context.Background(), usecases.QueueSeekInput{
		GuildID:               guildID,
		Position:              position,
		NotificationChannelID: notificationChannelID,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	// Convert back to 1-indexed for display
	return respondQueueSeeked(r, position+1, output.Track)
}

// HandleLoop handles the /loop command.
func (h *Handlers) HandleLoop(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	guildID, err := snowflake.Parse(i.GuildID)
	if err != nil {
		return respondError(r, "Invalid guild")
	}

	notificationChannelID, err := snowflake.Parse(i.ChannelID)
	if err != nil {
		return respondError(r, "Invalid channel")
	}

	ctx := context.Background()

	// Check if mode option was provided
	var modeStr string
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "mode" {
			modeStr = opt.StringValue()
		}
	}

	var newMode string
	if modeStr != "" {
		// Set specific mode
		err := h.playback.SetLoopMode(ctx, usecases.SetLoopModeInput{
			GuildID:               guildID,
			Mode:                  modeStr,
			NotificationChannelID: notificationChannelID,
		})
		if err != nil {
			return respondError(r, err.Error())
		}
		newMode = modeStr
	} else {
		// Cycle through modes
		output, err := h.playback.CycleLoopMode(ctx, usecases.CycleLoopModeInput{
			GuildID:               guildID,
			NotificationChannelID: notificationChannelID,
		})
		if err != nil {
			return respondError(r, err.Error())
		}
		newMode = output.NewMode
	}

	return respondLoopModeChanged(r, newMode)
}

// Response helpers.

func respondError(r bot.Responder, message string) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Error",
					Description: message,
					Color:       colorError,
				},
			},
		},
	})
}

func respondJoined(r bot.Responder, voiceChannelID snowflake.ID) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: fmt.Sprintf("Connected to <#%d>.", voiceChannelID),
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondDisconnected(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Disconnected.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondStopped(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Stopped playback.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondPaused(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Paused playback.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondResumed(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Resumed playback.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondSkipped(r bot.Responder, track *usecases.Track) error {
	var description string
	if track.URI != "" {
		description = fmt.Sprintf("Skipped [%s](%s).", track.Title, track.URI)
	} else {
		description = fmt.Sprintf("Skipped **%s**.", track.Title)
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: description,
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueRemoved(r bot.Responder, track *usecases.Track) error {
	var description string
	if track.URI != "" {
		description = fmt.Sprintf("Removed [%s](%s).", track.Title, track.URI)
	} else {
		description = fmt.Sprintf("Removed **%s**.", track.Title)
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: description,
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueCleared(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Cleared the queue.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueRestarted(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Restarted the queue from the beginning.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueSeeked(r bot.Responder, position int, track *usecases.Track) error {
	var description string
	if track.URI != "" {
		description = fmt.Sprintf(
			"Jumped to position %d: [%s](%s).",
			position,
			track.Title,
			track.URI,
		)
	} else {
		description = fmt.Sprintf("Jumped to position %d: **%s**.", position, track.Title)
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: description,
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondLoopModeChanged(r bot.Responder, mode string) error {
	var description string
	switch mode {
	case "track":
		description = "Now looping the current track."
	case "queue":
		description = "Now looping the queue."
	default:
		description = "Loop disabled."
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: description,
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueAdded(r bot.Responder, track *usecases.Track) error {
	var description string
	if track.URI != "" {
		description = fmt.Sprintf("Added [%s](%s) to the queue.", track.Title, track.URI)
	} else {
		description = fmt.Sprintf("Added **%s** to the queue.", track.Title)
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: description,
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondPlaylistAdded(r bot.Responder, playlistName string, trackCount int) error {
	description := fmt.Sprintf(
		"Added **%d tracks** from playlist **%s** to the queue.",
		trackCount,
		playlistName,
	)

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: description,
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueList(r bot.Responder, output *usecases.QueueListOutput) error {
	// Build title with loop mode indicator
	title := "Queue"
	switch output.LoopMode {
	case "track":
		title = "Queue \U0001F502" // 🔂
	case "queue":
		title = "Queue \U0001F501" // 🔁
	}

	embed := &discordgo.MessageEmbed{
		Title: title,
	}

	// Handle empty queue
	if len(output.Tracks) == 0 && output.TotalTracks == 0 {
		embed.Description = "Queue is empty."
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d", output.CurrentPage, output.TotalPages),
		}
		return r.Respond(&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
	}

	// Build description with sections
	var sb strings.Builder
	currentIndex := output.CurrentIndex
	needPlayedHeader := true
	needUpNextHeader := true

	for i, track := range output.Tracks {
		absIndex := output.PageStart + i
		displayIndex := absIndex + 1 // 1-indexed for display

		// Determine section and write header if needed
		if currentIndex >= 0 && absIndex < currentIndex {
			// Played section
			if needPlayedHeader {
				sb.WriteString("### Played\n")
				needPlayedHeader = false
			}
		} else if currentIndex >= 0 && absIndex == currentIndex {
			// Now Playing section
			sb.WriteString("### Now Playing\n")
		} else {
			// Up Next section (absIndex > currentIndex or currentIndex == -1)
			if needUpNextHeader {
				sb.WriteString("### Up Next\n")
				needUpNextHeader = false
			}
		}

		// Write track line (escape period to prevent Discord markdown list formatting)
		if track.URI != "" {
			fmt.Fprintf(
				&sb,
				"%d\\. [%s](%s) - %s\n",
				displayIndex,
				track.Title,
				track.URI,
				track.Artist,
			)
		} else {
			fmt.Fprintf(&sb, "%d\\. **%s** - %s\n", displayIndex, track.Title, track.Artist)
		}
	}

	embed.Description = sb.String()
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Page %d/%d", output.CurrentPage, output.TotalPages),
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
