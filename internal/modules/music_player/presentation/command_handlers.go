package presentation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sglre6355/sgrbot/internal/bot"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/usecases"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/platforms/discord"
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
	addToQueue               *usecases.AddToQueueUsecase[discord.PartialVoiceConnectionInfo]
	clearQueue               *usecases.ClearQueueUsecase[discord.PartialVoiceConnectionInfo]
	joinVoiceChannel         *usecases.JoinVoiceChannelUsecase[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo]
	leaveVoiceChannel        *usecases.LeaveVoiceChannelUsecase[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo]
	listQueue                *usecases.ListQueueUsecase[discord.PartialVoiceConnectionInfo]
	pausePlayback            *usecases.PausePlaybackUsecase[discord.PartialVoiceConnectionInfo]
	removeFromQueue          *usecases.RemoveFromQueueUsecase[discord.PartialVoiceConnectionInfo]
	resolveQuery             *usecases.ResolveQueryUsecase
	restartQueue             *usecases.RestartQueueUsecase[discord.PartialVoiceConnectionInfo]
	resumePlayback           *usecases.ResumePlaybackUsecase[discord.PartialVoiceConnectionInfo]
	seekQueue                *usecases.SeekQueueUsecase[discord.PartialVoiceConnectionInfo]
	setAutoPlay              *usecases.SetAutoPlayUsecase[discord.PartialVoiceConnectionInfo]
	setLoopMode              *usecases.SetLoopModeUsecase[discord.PartialVoiceConnectionInfo]
	setNowPlayingDestination *usecases.SetNowPlayingDestinationUsecase[discord.PartialVoiceConnectionInfo, discord.NowPlayingDestination]
	shuffleQueue             *usecases.ShuffleQueueUsecase[discord.PartialVoiceConnectionInfo]
	skipTrack                *usecases.SkipTrackUsecase[discord.PartialVoiceConnectionInfo]
}

// NewCommandHandlers creates new CommandHandlers.
func NewCommandHandlers(
	addToQueue *usecases.AddToQueueUsecase[discord.PartialVoiceConnectionInfo],
	clearQueue *usecases.ClearQueueUsecase[discord.PartialVoiceConnectionInfo],
	joinVoiceChannel *usecases.JoinVoiceChannelUsecase[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo],
	leaveVoiceChannel *usecases.LeaveVoiceChannelUsecase[discord.VoiceConnectionInfo, discord.PartialVoiceConnectionInfo],
	listQueue *usecases.ListQueueUsecase[discord.PartialVoiceConnectionInfo],
	pausePlayback *usecases.PausePlaybackUsecase[discord.PartialVoiceConnectionInfo],
	removeFromQueue *usecases.RemoveFromQueueUsecase[discord.PartialVoiceConnectionInfo],
	resolveQuery *usecases.ResolveQueryUsecase,
	restartQueue *usecases.RestartQueueUsecase[discord.PartialVoiceConnectionInfo],
	resumePlayback *usecases.ResumePlaybackUsecase[discord.PartialVoiceConnectionInfo],
	seekQueue *usecases.SeekQueueUsecase[discord.PartialVoiceConnectionInfo],
	setAutoPlay *usecases.SetAutoPlayUsecase[discord.PartialVoiceConnectionInfo],
	setLoopMode *usecases.SetLoopModeUsecase[discord.PartialVoiceConnectionInfo],
	setNowPlayingDestination *usecases.SetNowPlayingDestinationUsecase[discord.PartialVoiceConnectionInfo, discord.NowPlayingDestination],
	shuffleQueue *usecases.ShuffleQueueUsecase[discord.PartialVoiceConnectionInfo],
	skipTrack *usecases.SkipTrackUsecase[discord.PartialVoiceConnectionInfo],
) *CommandHandlers {
	return &CommandHandlers{
		addToQueue:               addToQueue,
		clearQueue:               clearQueue,
		joinVoiceChannel:         joinVoiceChannel,
		leaveVoiceChannel:        leaveVoiceChannel,
		listQueue:                listQueue,
		pausePlayback:            pausePlayback,
		removeFromQueue:          removeFromQueue,
		resolveQuery:             resolveQuery,
		restartQueue:             restartQueue,
		resumePlayback:           resumePlayback,
		seekQueue:                seekQueue,
		setAutoPlay:              setAutoPlay,
		setLoopMode:              setLoopMode,
		setNowPlayingDestination: setNowPlayingDestination,
		shuffleQueue:             shuffleQueue,
		skipTrack:                skipTrack,
	}
}

// updateNowPlayingDestination is a best-effort now-playing destination update.
func (h *CommandHandlers) updateNowPlayingDestination(
	connectionInfo discord.PartialVoiceConnectionInfo,
	channelID string,
) {
	_, err := h.setNowPlayingDestination.Execute(
		context.Background(),
		usecases.SetNowPlayingDestinationInput[discord.PartialVoiceConnectionInfo, discord.NowPlayingDestination]{
			ConnectionInfo:        connectionInfo,
			NowPlayingDestination: discord.NowPlayingDestination{ChannelID: channelID},
		},
	)
	if err != nil && !errors.Is(err, usecases.ErrNotConnected) {
		slog.Warn(
			"failed to update now-playing destination",
			"guild", connectionInfo.GuildID,
			"channel", channelID,
			"error", err,
		)
	}
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

	var joinConnectionInfo *discord.VoiceConnectionInfo
	if channelID != nil {
		joinConnectionInfo = &discord.VoiceConnectionInfo{
			GuildID:   i.GuildID,
			ChannelID: *channelID,
		}
	}
	partialInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	joinOutput, err := h.joinVoiceChannel.Execute(
		ctx,
		usecases.JoinVoiceChannelInput[
			discord.VoiceConnectionInfo,
			discord.PartialVoiceConnectionInfo,
		]{
			UserID:                i.Member.User.ID,
			ConnectionInfo:        joinConnectionInfo,
			PartialConnectionInfo: partialInfo,
		},
	)
	if err != nil {
		return respondError(r, err)
	}

	// Set notification channel
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}
	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	return r.Respond(&discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: fmt.Sprintf(
						"Connected to <#%s>.",
						joinOutput.ConnectionInfo.ChannelID,
					),
					Color: colorSuccess,
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	_, err := h.leaveVoiceChannel.Execute(
		ctx,
		usecases.LeaveVoiceChannelInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
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

	partialInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	if _, err := h.joinVoiceChannel.Execute(
		ctx,
		usecases.JoinVoiceChannelInput[
			discord.VoiceConnectionInfo,
			discord.PartialVoiceConnectionInfo,
		]{
			UserID:                userID,
			PartialConnectionInfo: partialInfo,
		},
	); err != nil {
		return respondError(r, err)
	}

	// Update notification channel
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}
	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	// Resolve query
	resolveOutput, err := h.resolveQuery.Execute(ctx, usecases.ResolveQueryInput{
		Query: query,
		Limit: 1,
	})
	if err != nil {
		return respondError(r, err)
	}

	// Add to queue
	trackIDs := make([]string, len(resolveOutput.Tracks))
	for idx, t := range resolveOutput.Tracks {
		trackIDs[idx] = t.ID
	}
	addOutput, err := h.addToQueue.Execute(
		ctx,
		usecases.AddToQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
			TrackIDs:       trackIDs,
			RequesterID:    userID,
		},
	)
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	_, err := h.clearQueue.Execute(
		ctx,
		usecases.ClearQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo:   connectionInfo,
			KeepCurrentTrack: false,
		},
	)
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	if _, err := h.pausePlayback.Execute(
		ctx,
		usecases.PausePlaybackInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
		},
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	if _, err := h.resumePlayback.Execute(
		ctx,
		usecases.ResumePlaybackInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
		},
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	if _, err := h.skipTrack.Execute(
		ctx,
		usecases.SkipTrackInput[discord.PartialVoiceConnectionInfo]{ConnectionInfo: connectionInfo},
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	var page int
	for _, opt := range options {
		if opt.Name == "page" {
			page = int(opt.IntValue())
		}
	}

	listOutput, err := h.listQueue.Execute(
		ctx,
		usecases.ListQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
			Page:           page,
		},
	)
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	var index int
	for _, opt := range options {
		if opt.Name == "position" {
			index = int(opt.IntValue()) - 1 // 1-indexed → 0-indexed
		}
	}

	removeOutput, err := h.removeFromQueue.Execute(
		ctx,
		usecases.RemoveFromQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
			Index:          index,
		},
	)
	if err != nil {
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	_, err := h.clearQueue.Execute(
		ctx,
		usecases.ClearQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo:   connectionInfo,
			KeepCurrentTrack: true,
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	_, err := h.restartQueue.Execute(
		ctx,
		usecases.RestartQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	_, err := h.shuffleQueue.Execute(
		ctx,
		usecases.ShuffleQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	var index int
	for _, opt := range options {
		if opt.Name == "position" {
			index = int(opt.IntValue()) - 1 // 1-indexed → 0-indexed
		}
	}

	seekOutput, err := h.seekQueue.Execute(
		ctx,
		usecases.SeekQueueInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
			Index:          index,
		},
	)
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	var enabled bool
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "enabled" {
			enabled = opt.BoolValue()
		}
	}

	_, err := h.setAutoPlay.Execute(
		ctx,
		usecases.SetAutoPlayInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
			Enabled:        enabled,
		},
	)
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
	connectionInfo := discord.PartialVoiceConnectionInfo{GuildID: i.GuildID}

	h.updateNowPlayingDestination(connectionInfo, i.ChannelID)

	var newMode string
	options := i.ApplicationCommandData().Options
	for _, opt := range options {
		if opt.Name == "mode" {
			newMode = opt.StringValue()
		}
	}

	_, err := h.setLoopMode.Execute(
		ctx,
		usecases.SetLoopModeInput[discord.PartialVoiceConnectionInfo]{
			ConnectionInfo: connectionInfo,
			Mode:           newMode,
		},
	)
	if err != nil {
		return respondError(r, err)
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
	usecases.ErrNotConnected:   "Not connected to a voice channel.",
	usecases.ErrUserNotInVoice: "You must be in a voice channel, or use /join to specify one.",
	usecases.ErrNoResults:      "No results found for your query. Try a different search term.",
	usecases.ErrQueueEmpty:     "The queue is empty.",
	usecases.ErrInvalidIndex:   "The provided position doesn't exist in the queue.",
	usecases.ErrNotPlaying:     "Nothing is playing right now.",
	usecases.ErrAlreadyPaused:  "Playback is already paused.",
	usecases.ErrNotPaused:      "Playback isn't paused.",
	errInvalidCommand:          "Invalid command.",
	errUnknownCommand:          "Unknown command.",
}

// userFacingMessage returns a user-friendly message for the given error and
// reports whether the error matched a known user-facing case.
func userFacingMessage(err error) (string, bool) {
	for known, message := range userFacingMessages {
		if errors.Is(err, known) {
			return message, true
		}
	}

	return "Something went wrong. Please try again later.", false
}

func respondError(r bot.Responder, err error) error {
	message, known := userFacingMessage(err)
	if !known {
		slog.Warn("unknown music player command error", "error", err)
	}

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
