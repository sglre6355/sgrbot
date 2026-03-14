package presentation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
)

// Embed colors.
const (
	colorSuccess = 0x08c404
	colorError   = 0xE74C3C
)

// Presentation level errors.
var (
	errInvalidCommand = errors.New("invalid command")
	errUnknownCommand = errors.New("unknown command")
)

// CommandHandlers holds all the command handlers.
type CommandHandlers struct {
	addToQueue             *usecases.AddToQueueUsecase
	clearQueue             *usecases.ClearQueueUsecase
	cycleLoopMode          *usecases.CycleLoopModeUsecase
	findPlayerState        *usecases.FindPlayerStateUsecase
	joinVoiceChannel       *usecases.JoinVoiceChannelUsecase
	leaveVoiceChannel      *usecases.LeaveVoiceChannelUsecase
	listQueue              *usecases.ListQueueUsecase
	pausePlayback          *usecases.PausePlaybackUsecase
	removeFromQueue        *usecases.RemoveFromQueueUsecase
	resolveQuery           *usecases.ResolveQueryUsecase
	restartQueue           *usecases.RestartQueueUsecase
	resumePlayback         *usecases.ResumePlaybackUsecase
	seekQueue              *usecases.SeekQueueUsecase
	setAutoPlay            *usecases.SetAutoPlayUsecase
	setLoopMode            *usecases.SetLoopModeUsecase
	setNotificationChannel *usecases.SetNotificationChannelUsecase
	shuffleQueue           *usecases.ShuffleQueueUsecase
	skipTrack              *usecases.SkipTrackUsecase
}

// NewCommandHandlers creates new CommandHandlers.
func NewCommandHandlers(
	addToQueue *usecases.AddToQueueUsecase,
	clearQueue *usecases.ClearQueueUsecase,
	cycleLoopMode *usecases.CycleLoopModeUsecase,
	findPlayerState *usecases.FindPlayerStateUsecase,
	joinVoiceChannel *usecases.JoinVoiceChannelUsecase,
	leaveVoiceChannel *usecases.LeaveVoiceChannelUsecase,
	listQueue *usecases.ListQueueUsecase,
	pausePlayback *usecases.PausePlaybackUsecase,
	removeFromQueue *usecases.RemoveFromQueueUsecase,
	resolveQuery *usecases.ResolveQueryUsecase,
	restartQueue *usecases.RestartQueueUsecase,
	resumePlayback *usecases.ResumePlaybackUsecase,
	seekQueue *usecases.SeekQueueUsecase,
	setAutoPlay *usecases.SetAutoPlayUsecase,
	setLoopMode *usecases.SetLoopModeUsecase,
	setNotificationChannel *usecases.SetNotificationChannelUsecase,
	shuffleQueue *usecases.ShuffleQueueUsecase,
	skipTrack *usecases.SkipTrackUsecase,
) *CommandHandlers {
	return &CommandHandlers{
		addToQueue:             addToQueue,
		clearQueue:             clearQueue,
		cycleLoopMode:          cycleLoopMode,
		findPlayerState:        findPlayerState,
		joinVoiceChannel:       joinVoiceChannel,
		leaveVoiceChannel:      leaveVoiceChannel,
		listQueue:              listQueue,
		pausePlayback:          pausePlayback,
		removeFromQueue:        removeFromQueue,
		resolveQuery:           resolveQuery,
		restartQueue:           restartQueue,
		resumePlayback:         resumePlayback,
		seekQueue:              seekQueue,
		setAutoPlay:            setAutoPlay,
		setLoopMode:            setLoopMode,
		setNotificationChannel: setNotificationChannel,
		shuffleQueue:           shuffleQueue,
		skipTrack:              skipTrack,
	}
}

// getPlayerStateID looks up the player state for the guild.
func (h *CommandHandlers) getPlayerStateID(ctx context.Context, guildID string) (string, error) {
	output, err := h.findPlayerState.Execute(
		ctx,
		usecases.FindPlayerStateInput{
			GuildID: guildID,
		},
	)
	if err != nil {
		return "", err
	}
	return *output.PlayerStateID, nil
}

// updateNotificationChannel is a best-effort notification channel update.
func (h *CommandHandlers) updateNotificationChannel(
	playerStateID string,
	channelID string,
) {
	_, _ = h.setNotificationChannel.Execute(
		context.Background(),
		usecases.SetNotificationChannelInput{
			PlayerStateID: playerStateID,
			ChannelID:     channelID,
		},
	)
}

// HandleJoin handles the /join command.
func (h *CommandHandlers) HandleJoin(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	var channelID *string
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "channel" {
			id := opt.ChannelValue(s).ID
			channelID = &id
		}
	}

	joinOutput, err := h.joinVoiceChannel.Execute(ctx, usecases.JoinVoiceChannelInput{
		GuildID:   i.GuildID,
		UserID:    i.Member.User.ID,
		ChannelID: channelID,
	})
	if err != nil {
		return respondError(r, err)
	}

	// Set notification channel
	h.updateNotificationChannel(joinOutput.PlayerStateID, i.ChannelID)

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: fmt.Sprintf("Connected to <#%s>.", joinOutput.ChannelID),
					Color:       colorSuccess,
				},
			},
		},
	})
}

// HandleLeave handles the /leave command.
func (h *CommandHandlers) HandleLeave(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()
	guildID := i.GuildID

	playerStateID, err := h.getPlayerStateID(ctx, guildID)
	if err != nil {
		return respondError(r, err)
	}

	_, err = h.leaveVoiceChannel.Execute(
		ctx,
		usecases.LeaveVoiceChannelInput{
			PlayerStateID: playerStateID,
		},
	)
	if err != nil {
		return respondError(r, err)
	}

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

// HandlePlay handles the /play command.
func (h *CommandHandlers) HandlePlay(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()
	userID := i.Member.User.ID

	var query string
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "query" {
			query = opt.StringValue()
		}
	}

	joinOutput, err := h.joinVoiceChannel.Execute(ctx, usecases.JoinVoiceChannelInput{
		GuildID: i.GuildID,
		UserID:  userID,
	})
	if err != nil {
		return respondError(r, err)
	}

	// Update notification channel
	h.updateNotificationChannel(joinOutput.PlayerStateID, i.ChannelID)

	// Resolve query
	resolveOutput, err := h.resolveQuery.Execute(ctx, usecases.ResolveQueryInput{
		Query: query,
	})
	if err != nil {
		return respondError(r, err)
	}

	// Add to queue
	trackIDs := make([]string, len(resolveOutput.Tracks))
	for idx, t := range resolveOutput.Tracks {
		trackIDs[idx] = t.ID
	}
	addOutput, err := h.addToQueue.Execute(ctx, usecases.AddToQueueInput{
		PlayerStateID: joinOutput.PlayerStateID,
		TrackIDs:      trackIDs,
		RequesterID:   userID,
	})
	if err != nil {
		return respondError(r, err)
	}

	var description string
	if resolveOutput.Type == "playlist" && resolveOutput.Name != nil {
		description = fmt.Sprintf(
			"Added **%d tracks** from playlist **%s** to the queue.",
			addOutput.Count,
			*resolveOutput.Name,
		)
	} else {
		track := resolveOutput.Tracks[0]
		if track.URL != "" {
			description = fmt.Sprintf("Added [%s](%s) to the queue.", track.Title, track.URL)
		} else {
			description = fmt.Sprintf("Added **%s** to the queue.", track.Title)
		}
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

// HandleStop handles the /stop command.
func (h *CommandHandlers) HandleStop(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	_, err = h.clearQueue.Execute(ctx, usecases.ClearQueueInput{
		PlayerStateID:    playerStateID,
		KeepCurrentTrack: false,
	})
	if err != nil && !errors.Is(err, usecases.ErrQueueEmpty) {
		return respondError(r, err)
	}

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

// HandlePause handles the /pause command.
func (h *CommandHandlers) HandlePause(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	if _, err := h.pausePlayback.Execute(
		ctx,
		usecases.PausePlaybackInput{PlayerStateID: playerStateID},
	); err != nil {
		return respondError(r, err)
	}

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

// HandleResume handles the /resume command.
func (h *CommandHandlers) HandleResume(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	if _, err := h.resumePlayback.Execute(
		ctx,
		usecases.ResumePlaybackInput{PlayerStateID: playerStateID},
	); err != nil {
		return respondError(r, err)
	}

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

// HandleSkip handles the /skip command.
func (h *CommandHandlers) HandleSkip(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	if _, err := h.skipTrack.Execute(
		ctx,
		usecases.SkipTrackInput{PlayerStateID: playerStateID},
	); err != nil {
		return respondError(r, err)
	}

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

// HandleQueue handles the /queue command.
func (h *CommandHandlers) HandleQueue(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return respondError(r, errInvalidCommand)
	}

	Cmd := options[0]
	switch Cmd.Name {
	case "list":
		return h.handleQueueList(s, i, r, Cmd.Options)
	case "remove":
		return h.handleQueueRemove(s, i, r, Cmd.Options)
	case "clear":
		return h.handleQueueClear(s, i, r)
	case "restart":
		return h.handleQueueRestart(s, i, r)
	case "shuffle":
		return h.handleQueueShuffle(s, i, r)
	case "seek":
		return h.handleQueueSeek(s, i, r, Cmd.Options)
	default:
		return respondError(r, errUnknownCommand)
	}
}

func (h *CommandHandlers) handleQueueList(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	var page int
	for _, opt := range options {
		if opt.Name == "page" {
			page = int(opt.IntValue())
		}
	}

	listOutput, err := h.listQueue.Execute(ctx, usecases.ListQueueInput{
		PlayerStateID: playerStateID,
		Page:          page,
	})
	if err != nil {
		return respondError(r, err)
	}

	// Build embed with mode fields
	embed := &discordgo.MessageEmbed{
		Title: "Queue",
	}

	var loopModeValue string
	switch listOutput.LoopMode {
	case "track":
		loopModeValue = "\U0001F502 Track" // 🔂
	case "queue":
		loopModeValue = "\U0001F501 Queue" // 🔁
	default:
		loopModeValue = "Off"
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Loop",
		Value:  loopModeValue,
		Inline: true,
	})

	var autoPlayValue string
	if listOutput.AutoPlayEnabled {
		autoPlayValue = "\u2705 Enabled" // ✅
	} else {
		autoPlayValue = "Off"
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Auto-Play",
		Value:  autoPlayValue,
		Inline: true,
	})

	// Handle empty queue
	if listOutput.TotalTracks == 0 {
		embed.Description = "Queue is empty."
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d", listOutput.CurrentPage, listOutput.TotalPages),
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
	displayIndex := listOutput.PageStart + 1 // 1-indexed for display

	if len(listOutput.PlayedTracks) > 0 {
		sb.WriteString("### Played\n")
		for _, track := range listOutput.PlayedTracks {
			writeTrackLine(&sb, displayIndex, track)
			displayIndex++
		}
	}

	if listOutput.CurrentTrack != nil {
		sb.WriteString("### Now Playing\n")
		writeTrackLine(&sb, displayIndex, *listOutput.CurrentTrack)
		displayIndex++
	}

	if len(listOutput.UpcomingTracks) > 0 {
		sb.WriteString("### Up Next\n")
		for _, track := range listOutput.UpcomingTracks {
			writeTrackLine(&sb, displayIndex, track)
			displayIndex++
		}
	}

	embed.Description = sb.String()
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Page %d/%d", listOutput.CurrentPage, listOutput.TotalPages),
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (h *CommandHandlers) handleQueueRemove(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	var index int
	for _, opt := range options {
		if opt.Name == "position" {
			index = int(opt.IntValue()) - 1 // 1-indexed → 0-indexed
		}
	}

	removeOutput, err := h.removeFromQueue.Execute(ctx, usecases.RemoveFromQueueInput{
		PlayerStateID: playerStateID,
		Index:         index,
	})
	if err != nil {
		if errors.Is(err, usecases.ErrIsCurrentTrack) {
			// Skip first, then remove
			if _, skipErr := h.skipTrack.Execute(ctx, usecases.SkipTrackInput{
				PlayerStateID: playerStateID,
			}); skipErr != nil {
				return respondError(r, skipErr)
			}
			removeOutput, removeErr := h.removeFromQueue.Execute(ctx, usecases.RemoveFromQueueInput{
				PlayerStateID: playerStateID,
				Index:         index,
			})
			if removeErr != nil {
				return respondError(r, removeErr)
			}
			return respondQueueRemoved(r, removeOutput.RemovedTrack)
		}
		return respondError(r, err)
	}

	return respondQueueRemoved(r, removeOutput.RemovedTrack)
}

func (h *CommandHandlers) handleQueueClear(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	_, err = h.clearQueue.Execute(ctx, usecases.ClearQueueInput{
		PlayerStateID:    playerStateID,
		KeepCurrentTrack: true,
	})
	if err != nil {
		return respondError(r, err)
	}

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

func (h *CommandHandlers) handleQueueRestart(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	_, err = h.restartQueue.Execute(ctx, usecases.RestartQueueInput{
		PlayerStateID: playerStateID,
	})
	if err != nil {
		return respondError(r, err)
	}

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

func (h *CommandHandlers) handleQueueShuffle(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	_, err = h.shuffleQueue.Execute(ctx, usecases.ShuffleQueueInput{
		PlayerStateID: playerStateID,
	})
	if err != nil {
		return respondError(r, err)
	}

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "Shuffled the queue.",
					Color:       colorSuccess,
				},
			},
		},
	})
}

func (h *CommandHandlers) handleQueueSeek(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
	options []*discordgo.ApplicationCommandInteractionDataOption,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	var index int
	for _, opt := range options {
		if opt.Name == "position" {
			index = int(opt.IntValue()) - 1 // 1-indexed → 0-indexed
		}
	}

	seekOutput, err := h.seekQueue.Execute(ctx, usecases.SeekQueueInput{
		PlayerStateID: playerStateID,
		Index:         index,
	})
	if err != nil {
		return respondError(r, err)
	}

	var description string
	if seekOutput.Track.URL != "" {
		description = fmt.Sprintf(
			"Jumped to position %d: [%s](%s).",
			index+1,
			seekOutput.Track.Title,
			seekOutput.Track.URL,
		)
	} else {
		description = fmt.Sprintf(
			"Jumped to position %d: **%s**.",
			index+1,
			seekOutput.Track.Title,
		)
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

// HandleAutoPlay handles the /autoplay command.
func (h *CommandHandlers) HandleAutoPlay(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	var enabled bool
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "enabled" {
			enabled = opt.BoolValue()
		}
	}

	_, err = h.setAutoPlay.Execute(ctx, usecases.SetAutoPlayInput{
		PlayerStateID: playerStateID,
		Enabled:       enabled,
	})
	if err != nil {
		return respondError(r, err)
	}

	var description string
	if enabled {
		description = "Auto-play enabled."
	} else {
		description = "Auto-play disabled."
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

// HandleLoop handles the /loop command.
func (h *CommandHandlers) HandleLoop(
	_ *discordgo.Session,
	i *discordgo.InteractionCreate,
	r bot.Responder,
) error {
	ctx := context.Background()

	playerStateID, err := h.getPlayerStateID(ctx, i.GuildID)
	if err != nil {
		return respondError(r, err)
	}

	h.updateNotificationChannel(playerStateID, i.ChannelID)

	var modeStr string
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "mode" {
			modeStr = opt.StringValue()
		}
	}

	var newMode string
	if modeStr != "" {
		_, err := h.setLoopMode.Execute(ctx, usecases.SetLoopModeInput{
			PlayerStateID: playerStateID,
			Mode:          modeStr,
		})
		if err != nil {
			return respondError(r, err)
		}
		newMode = modeStr
	} else {
		output, err := h.cycleLoopMode.Execute(ctx, usecases.CycleLoopModeInput{
			PlayerStateID: playerStateID,
		})
		if err != nil {
			return respondError(r, err)
		}
		newMode = output.NewMode
	}

	var description string
	switch newMode {
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

// Response helpers.

// userFacingMessages maps known usecase errors to user-friendly messages.
var userFacingMessages = map[error]string{
	usecases.ErrNotConnected:        "I'm not connected to a voice channel. Use `/join` first.",
	usecases.ErrUserNotInVoice:      "You need to be in a voice channel to use this command.",
	usecases.ErrNoResults:           "No results found for your query. Try a different search term.",
	usecases.ErrQueueEmpty:          "The queue is empty.",
	usecases.ErrIsCurrentTrack:      "That track is currently playing and can't be removed directly. Use `/skip` instead.",
	usecases.ErrInvalidIndex:        "That position doesn't exist in the queue.",
	usecases.ErrNotPlaying:          "Nothing is playing right now.",
	usecases.ErrAlreadyPaused:       "Playback is already paused.",
	usecases.ErrNotPaused:           "Playback isn't paused.",
	usecases.ErrPlayerStateNotFound: "No active player found for this server. Use `/join` to get started.",
	errInvalidCommand:               "Invalid command.",
	errUnknownCommand:               "Unknown command.",
}

// userFacingMessage returns a user-friendly message for the given error.
func userFacingMessage(err error) string {
	for known, msg := range userFacingMessages {
		if errors.Is(err, known) {
			return msg
		}
	}
	return "Something went wrong. Please try again later."
}

func respondError(r bot.Responder, err error) error {
	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Error",
					Description: userFacingMessage(err),
					Color:       colorError,
				},
			},
		},
	})
}

func respondQueueRemoved(r bot.Responder, track dtos.TrackView) error {
	var description string
	if track.URL != "" {
		description = fmt.Sprintf("Removed [%s](%s).", track.Title, track.URL)
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

// writeTrackLine writes a single track line to the string builder.
func writeTrackLine(sb *strings.Builder, displayIndex int, track dtos.TrackView) {
	if track.URL != "" {
		fmt.Fprintf(
			sb,
			"%d\\. [%s](%s) - %s\n",
			displayIndex,
			track.Title,
			track.URL,
			track.Author,
		)
	} else {
		fmt.Fprintf(sb, "%d\\. **%s** - %s\n", displayIndex, track.Title, track.Author)
	}
}
