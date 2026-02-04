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
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
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

	// 2. Load track via TrackLoaderService
	trackOutput, err := h.trackLoader.LoadTrack(ctx, usecases.LoadTrackInput{
		Query:              query,
		RequesterID:        userID,
		RequesterName:      getDisplayName(i.Member),
		RequesterAvatarURL: i.Member.User.AvatarURL(""),
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	// 3. Add to queue (auto-starts playback if CurrentTrack is nil)
	_, err = h.queue.Add(ctx, usecases.QueueAddInput{
		GuildID:               guildID,
		Track:                 trackOutput.Track,
		NotificationChannelID: notificationChannelID,
	})
	if err != nil {
		return respondError(r, err.Error())
	}

	// Always respond with "Added to Queue" - "Now Playing" is sent as a separate message
	return respondQueueAdded(r, trackOutput.Track)
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
	_, _ = h.queue.Clear(usecases.QueueClearInput{
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

	output, err := h.queue.List(input)
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
			position = int(opt.IntValue())
		}
	}

	input := usecases.QueueRemoveInput{
		GuildID:               guildID,
		Position:              position,
		NotificationChannelID: notificationChannelID,
	}

	output, err := h.queue.Remove(input)
	if err != nil {
		// If trying to remove current track, delegate to Skip
		if errors.Is(err, usecases.ErrIsCurrentTrack) {
			skipOutput, skipErr := h.playback.Skip(context.Background(), usecases.SkipInput{
				GuildID:               guildID,
				NotificationChannelID: notificationChannelID,
			})
			if skipErr != nil {
				return respondError(r, skipErr.Error())
			}
			return respondSkipped(r, skipOutput.SkippedTrack)
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

	_, err = h.queue.Clear(input)
	if err != nil {
		return respondError(r, err.Error())
	}

	return respondQueueCleared(r)
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

	var newMode domain.LoopMode
	if modeStr != "" {
		// Set specific mode
		mode := parseLoopMode(modeStr)
		err := h.playback.SetLoopMode(ctx, usecases.SetLoopModeInput{
			GuildID:               guildID,
			Mode:                  mode,
			NotificationChannelID: notificationChannelID,
		})
		if err != nil {
			return respondError(r, err.Error())
		}
		newMode = mode
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

func parseLoopMode(s string) domain.LoopMode {
	switch s {
	case "track":
		return domain.LoopModeTrack
	case "queue":
		return domain.LoopModeQueue
	default:
		return domain.LoopModeNone
	}
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

func respondLoopModeChanged(r bot.Responder, mode domain.LoopMode) error {
	var description string
	switch mode {
	case domain.LoopModeTrack:
		description = "Now looping the current track."
	case domain.LoopModeQueue:
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

func respondQueueList(r bot.Responder, output *usecases.QueueListOutput) error {
	// Build title with loop mode indicator
	title := "Queue"
	switch output.LoopMode {
	case domain.LoopModeTrack:
		title = "Queue \U0001F502" // 🔂
	case domain.LoopModeQueue:
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

// getDisplayName returns the effective display name for a guild member.
// Priority: guild nickname > global display name > username.
func getDisplayName(member *discordgo.Member) string {
	if member.Nick != "" {
		return member.Nick
	}
	if member.User.GlobalName != "" {
		return member.User.GlobalName
	}
	return member.User.Username
}
