package discord

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
	voiceChannel        *usecases.VoiceChannelService
	playback            *usecases.PlaybackService
	queue               *usecases.QueueService
	trackLoader         *usecases.TrackLoaderService
	notificationChannel *usecases.NotificationChannelService
}

// NewHandlers creates new Handlers.
func NewHandlers(
	voiceChannel *usecases.VoiceChannelService,
	playback *usecases.PlaybackService,
	queue *usecases.QueueService,
	trackLoader *usecases.TrackLoaderService,
	notificationChannel *usecases.NotificationChannelService,
) *Handlers {
	return &Handlers{
		voiceChannel:        voiceChannel,
		playback:            playback,
		queue:               queue,
		trackLoader:         trackLoader,
		notificationChannel: notificationChannel,
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

	// 3. Add to queue
	trackIDs := make([]string, len(tracksOutput.Tracks))
	for i, t := range tracksOutput.Tracks {
		trackIDs[i] = t.ID
	}
	output, err := h.queue.Add(ctx, usecases.QueueAddInput{
		GuildID:     guildID,
		TrackIDs:    trackIDs,
		RequesterID: userID,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	if tracksOutput.IsPlaylist {
		return respondPlaylistAdded(r, tracksOutput.PlaylistName, output.Count)
	}
	return respondQueueAdded(r, tracksOutput.Tracks[0])
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

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	// Clear the entire queue — the event handler stops playback via CurrentTrackChangedEvent
	_, err = h.queue.Clear(ctx, usecases.QueueClearInput{
		GuildID:          guildID,
		KeepCurrentTrack: false,
	})
	if err != nil && !errors.Is(err, usecases.ErrQueueEmpty) {
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	if err := h.playback.Pause(ctx, usecases.PauseInput{GuildID: guildID}); err != nil {
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	if err := h.playback.Resume(ctx, usecases.ResumeInput{GuildID: guildID}); err != nil {
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	if _, err := h.playback.Skip(ctx, usecases.SkipInput{GuildID: guildID}); err != nil {
		return respondError(r, err.Error())
	}

	// Respond with skipped confirmation - "Now Playing" is sent via CurrentTrackChangedEvent
	return respondSkipped(r)
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	var page int // let service default to page containing current track
	for _, opt := range options {
		if opt.Name == "page" {
			page = int(opt.IntValue())
		}
	}

	output, err := h.queue.List(ctx, usecases.QueueListInput{
		GuildID: guildID,
		Page:    page,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	// Collect all track IDs and load track info for display
	var allIDs []string
	allIDs = append(allIDs, output.PlayedTrackIDs...)
	if output.CurrentTrackID != "" {
		allIDs = append(allIDs, output.CurrentTrackID)
	}
	allIDs = append(allIDs, output.UpcomingTrackIDs...)

	tracks := make(map[string]*usecases.TrackInfo, len(allIDs))
	for _, id := range allIDs {
		info, err := h.trackLoader.LoadTrack(ctx, id)
		if err != nil {
			return respondError(r, err.Error())
		}
		tracks[id] = info
	}

	return respondQueueList(r, output, tracks)
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	var position int
	for _, opt := range options {
		if opt.Name == "position" {
			// Convert from 1-indexed (user input) to 0-indexed (internal)
			position = int(opt.IntValue()) - 1
		}
	}

	output, err := h.queue.Remove(ctx, usecases.QueueRemoveInput{
		GuildID:  guildID,
		Position: position,
	})
	if err != nil {
		// If trying to remove current track, skip first then remove
		if errors.Is(err, usecases.ErrIsCurrentTrack) {
			if _, skipErr := h.playback.Skip(ctx, usecases.SkipInput{
				GuildID: guildID,
			}); skipErr != nil {
				return respondError(r, skipErr.Error())
			}
			// After skip, currentIndex has advanced, so we can now remove the track
			// at the original position (which is now in the "played" section)
			removeOutput, removeErr := h.queue.Remove(ctx, usecases.QueueRemoveInput{
				GuildID:  guildID,
				Position: position,
			})
			if removeErr != nil {
				return respondError(r, removeErr.Error())
			}
			track, loadErr := h.trackLoader.LoadTrack(ctx, removeOutput.RemovedTrackID)
			if loadErr != nil {
				return respondError(r, loadErr.Error())
			}
			return respondQueueRemoved(r, track)
		}
		return respondError(r, err.Error())
	}

	track, err := h.trackLoader.LoadTrack(ctx, output.RemovedTrackID)
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondQueueRemoved(r, track)
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	_, err = h.queue.Clear(ctx, usecases.QueueClearInput{
		GuildID:          guildID,
		KeepCurrentTrack: true, // Clear played + upcoming, keep only current
	})
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	_, err = h.queue.Restart(ctx, usecases.QueueRestartInput{
		GuildID: guildID,
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

	ctx := context.Background()

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

	var position int
	for _, opt := range options {
		if opt.Name == "position" {
			// Convert from 1-indexed (user input) to 0-indexed (internal)
			position = int(opt.IntValue()) - 1
		}
	}

	output, err := h.queue.Seek(ctx, usecases.QueueSeekInput{
		GuildID:  guildID,
		Position: position,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	track, err := h.trackLoader.LoadTrack(ctx, output.TrackID)
	if err != nil {
		return respondError(r, err.Error())
	}

	// Convert back to 1-indexed for display
	return respondQueueSeeked(r, position+1, track)
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

	// Update notification channel (best-effort)
	_ = h.notificationChannel.Set(ctx, usecases.SetNotificationChannelInput{
		GuildID:   guildID,
		ChannelID: notificationChannelID,
	})

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
			GuildID: guildID,
			Mode:    modeStr,
		})
		if err != nil {
			return respondError(r, err.Error())
		}
		newMode = modeStr
	} else {
		// Cycle through modes
		output, err := h.playback.CycleLoopMode(ctx, usecases.CycleLoopModeInput{
			GuildID: guildID,
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

func respondSkipped(r bot.Responder) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Skipped.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func respondQueueRemoved(r bot.Responder, track *usecases.TrackInfo) error {
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

func respondQueueSeeked(r bot.Responder, position int, track *usecases.TrackInfo) error {
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

func respondQueueAdded(r bot.Responder, track *usecases.TrackInfo) error {
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

func respondQueueList(
	r bot.Responder,
	output *usecases.QueueListOutput,
	tracks map[string]*usecases.TrackInfo,
) error {
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
	if output.TotalTracks == 0 {
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
	displayIndex := output.PageStart + 1 // 1-indexed for display

	if len(output.PlayedTrackIDs) > 0 {
		sb.WriteString("### Played\n")
		for _, id := range output.PlayedTrackIDs {
			writeTrackLine(&sb, displayIndex, tracks[id])
			displayIndex++
		}
	}

	if output.CurrentTrackID != "" {
		sb.WriteString("### Now Playing\n")
		writeTrackLine(&sb, displayIndex, tracks[output.CurrentTrackID])
		displayIndex++
	}

	if len(output.UpcomingTrackIDs) > 0 {
		sb.WriteString("### Up Next\n")
		for _, id := range output.UpcomingTrackIDs {
			writeTrackLine(&sb, displayIndex, tracks[id])
			displayIndex++
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

// writeTrackLine writes a single track line to the string builder.
// Escapes period to prevent Discord markdown list formatting.
func writeTrackLine(sb *strings.Builder, displayIndex int, track *usecases.TrackInfo) {
	if track.URI != "" {
		fmt.Fprintf(
			sb,
			"%d\\. [%s](%s) - %s\n",
			displayIndex,
			track.Title,
			track.URI,
			track.Artist,
		)
	} else {
		fmt.Fprintf(sb, "%d\\. **%s** - %s\n", displayIndex, track.Title, track.Artist)
	}
}
