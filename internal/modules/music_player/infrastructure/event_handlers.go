package infrastructure

import (
	"context"
	"log/slog"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PlayNextFunc is the function signature for playing the next track.
type PlayNextFunc func(ctx context.Context, guildID snowflake.ID) (*domain.Track, error)

// StopFunc is the function signature for stopping playback.
type StopFunc func(ctx context.Context, guildID snowflake.ID) error

// PlaybackEventHandler handles events related to playback control.
// It subscribes to TrackEnqueued, TrackEnded, and QueueCleared events to manage playback flow.
type PlaybackEventHandler struct {
	playNextFunc PlayNextFunc
	stopFunc     StopFunc
	repo         domain.PlayerStateRepository
	subscriber   ports.EventSubscriber
	publisher    ports.EventPublisher
}

// NewPlaybackEventHandler creates a new PlaybackEventHandler.
func NewPlaybackEventHandler(
	playNextFunc PlayNextFunc,
	stopFunc StopFunc,
	repo domain.PlayerStateRepository,
	subscriber ports.EventSubscriber,
	publisher ports.EventPublisher,
) *PlaybackEventHandler {
	return &PlaybackEventHandler{
		playNextFunc: playNextFunc,
		stopFunc:     stopFunc,
		repo:         repo,
		subscriber:   subscriber,
		publisher:    publisher,
	}
}

// Start registers event handlers with the subscriber.
func (h *PlaybackEventHandler) Start() {
	h.subscriber.OnTrackEnqueued(h.handleTrackEnqueued)
	h.subscriber.OnTrackEnded(h.handleTrackEnded)
	h.subscriber.OnQueueCleared(h.handleQueueCleared)

	slog.Debug("playback event handler started")
}

func (h *PlaybackEventHandler) handleTrackEnqueued(
	ctx context.Context,
	event domain.TrackEnqueuedEvent,
) {
	// Check current state to avoid race with concurrent enqueues.
	// Multiple tracks enqueued while idle may all trigger this handler,
	// but only the first should trigger playback.
	state, err := h.repo.Get(ctx, event.GuildID)
	if err != nil {
		slog.Debug("track enqueued but state not found, skipping auto-play",
			"guild", event.GuildID,
		)
		return
	}

	// Check if playback is already active (another event already started playback).
	if state.IsPlaybackActive() {
		slog.Debug("track enqueued but playback already active, skipping auto-play",
			"guild", event.GuildID,
			"track", event.Track.Title,
		)
		return
	}

	slog.Debug("track enqueued and player idle, starting playback",
		"guild", event.GuildID,
		"track", event.Track.Title,
	)

	_, err = h.playNextFunc(ctx, event.GuildID)
	if err != nil {
		slog.Error("failed to start playback after track enqueued",
			"guild", event.GuildID,
			"error", err,
		)
	}
}

func (h *PlaybackEventHandler) handleQueueCleared(
	ctx context.Context,
	event domain.QueueClearedEvent,
) {
	slog.Debug("queue cleared, stopping playback",
		"guild", event.GuildID,
	)

	// Stop playback via audio player
	if err := h.stopFunc(ctx, event.GuildID); err != nil {
		slog.Error("failed to stop playback after queue cleared",
			"guild", event.GuildID,
			"error", err,
		)
	}

	state, err := h.repo.Get(ctx, event.GuildID)
	if err == nil {
		// Delete the "Now Playing" message
		nowPlayingMsg := state.GetNowPlayingMessage()
		if nowPlayingMsg != nil {
			h.publisher.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
				GuildID:               event.GuildID,
				NotificationChannelID: nowPlayingMsg.ChannelID,
				LastMessageID:         &nowPlayingMsg.MessageID,
			})
		}

		state.SetPlaybackActive(false)
	}
}

func (h *PlaybackEventHandler) handleTrackEnded(ctx context.Context, event domain.TrackEndedEvent) {
	// Only advance queue for certain end reasons
	if !event.Reason.ShouldAdvanceQueue() {
		slog.Debug("track ended but should not advance queue",
			"guild", event.GuildID,
			"reason", event.Reason,
		)
		return
	}

	// Get state to check loop mode and notification channel
	state, err := h.repo.Get(ctx, event.GuildID)
	if err != nil {
		slog.Debug("track ended but no player state",
			"guild", event.GuildID,
		)
		return
	}

	loopMode := state.GetLoopMode()

	slog.Debug("track ended, advancing queue",
		"guild", event.GuildID,
		"reason", event.Reason,
		"loop_mode", loopMode.String(),
	)

	// Delete the old "Now Playing" message before playing next
	nowPlayingMsg := state.GetNowPlayingMessage()
	if nowPlayingMsg != nil {
		h.publisher.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
			GuildID:               event.GuildID,
			NotificationChannelID: nowPlayingMsg.ChannelID,
			LastMessageID:         &nowPlayingMsg.MessageID,
		})
	}

	// Advance queue based on loop mode
	// For TrackEndLoadFailed, remove the failing track and advance to prevent infinite retry loops
	if event.Reason == domain.TrackEndLoadFailed {
		failedIndex := state.Queue.CurrentIndex()
		// Use LoopModeNone for LoopModeTrack to prevent infinite retry on same track,
		// but preserve LoopModeQueue to allow wrapping to first track
		advanceMode := loopMode
		if loopMode == domain.LoopModeTrack {
			advanceMode = domain.LoopModeNone
		}
		state.Queue.Advance(advanceMode)
		state.Queue.RemoveAt(failedIndex)
	} else {
		nextTrackID := state.Queue.Advance(state.GetLoopMode())
		if nextTrackID == nil {
			state.SetPlaybackActive(false)
		}
	}

	if state.IsPlaybackActive() {
		_, err = h.playNextFunc(ctx, event.GuildID)
		if err != nil {
			slog.Error("failed to play next track after track ended",
				"guild", event.GuildID,
				"error", err,
			)

			// Publish error notification if we have a channel
			if errNowPlayingMsg := state.GetNowPlayingMessage(); errNowPlayingMsg != nil {
				h.publisher.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
					GuildID:               event.GuildID,
					NotificationChannelID: errNowPlayingMsg.ChannelID,
					LastMessageID:         &errNowPlayingMsg.MessageID,
				})
			}
		}
	}

	// Save state after mutations
	if err := h.repo.Save(ctx, state); err != nil {
		slog.Error("failed to save state after track ended",
			"guild", event.GuildID,
			"error", err,
		)
	}
}

// NotificationEventHandler handles events related to Discord notifications.
// It subscribes to PlaybackStarted and PlaybackFinished events to send/delete messages.
type NotificationEventHandler struct {
	notifier     ports.NotificationSender
	repo         domain.PlayerStateRepository
	subscriber   ports.EventSubscriber
	userInfoProv ports.UserInfoProvider
}

// NewNotificationEventHandler creates a new NotificationEventHandler.
func NewNotificationEventHandler(
	notifier ports.NotificationSender,
	repo domain.PlayerStateRepository,
	subscriber ports.EventSubscriber,
	userInfoProv ports.UserInfoProvider,
) *NotificationEventHandler {
	return &NotificationEventHandler{
		notifier:     notifier,
		repo:         repo,
		subscriber:   subscriber,
		userInfoProv: userInfoProv,
	}
}

// Start registers event handlers with the subscriber.
func (h *NotificationEventHandler) Start() {
	h.subscriber.OnPlaybackStarted(h.handlePlaybackStarted)
	h.subscriber.OnPlaybackFinished(h.handlePlaybackFinished)

	slog.Debug("notification event handler started")
}

func (h *NotificationEventHandler) handlePlaybackStarted(
	ctx context.Context,
	event domain.PlaybackStartedEvent,
) {
	// Check if the track is still current before sending notification.
	// This prevents sending "Now Playing" for tracks that failed to load,
	// which would leave orphaned messages since handleTrackEnded already ran.
	state, err := h.repo.Get(ctx, event.GuildID)
	if err != nil {
		slog.Debug("skipping now playing notification, state not found",
			"guild", event.GuildID,
		)
		return
	}
	currentEntry := state.CurrentEntry()
	if currentEntry == nil || currentEntry.TrackID != event.Track.ID {
		slog.Debug("skipping now playing notification, track no longer current",
			"guild", event.GuildID,
			"track", event.Track.Title,
		)
		return
	}

	slog.Debug("sending now playing notification",
		"guild", event.GuildID,
		"track", event.Track.Title,
	)

	// Fetch requester display info via port
	var requesterName, requesterAvatarURL string
	if h.userInfoProv != nil {
		userInfo, err := h.userInfoProv.GetUserInfo(event.GuildID, event.RequesterID)
		if err != nil {
			slog.Warn("failed to fetch requester info for now playing",
				"guild", event.GuildID,
				"requester", event.RequesterID,
				"error", err,
			)
			requesterName = "Unknown"
		} else {
			requesterName = userInfo.DisplayName
			requesterAvatarURL = userInfo.AvatarURL
		}
	}

	messageID, err := h.notifier.SendNowPlaying(event.NotificationChannelID, &ports.NowPlayingInfo{
		Identifier:         string(event.Track.ID),
		Title:              event.Track.Title,
		Artist:             event.Track.Artist,
		Duration:           event.Track.FormattedDuration(),
		URI:                event.Track.URI,
		ArtworkURL:         event.Track.ArtworkURL,
		SourceName:         event.Track.SourceName,
		IsStream:           event.Track.IsStream,
		RequesterID:        event.RequesterID,
		RequesterName:      requesterName,
		RequesterAvatarURL: requesterAvatarURL,
		EnqueuedAt:         event.EnqueuedAt,
	})
	if err != nil {
		slog.Error("failed to send now playing notification",
			"guild", event.GuildID,
			"error", err,
		)
		return
	}

	// Store the message info for later deletion
	state.SetNowPlayingMessage(event.NotificationChannelID, messageID)
	if err := h.repo.Save(ctx, state); err != nil {
		slog.Error("failed to save state after setting now playing message",
			"guild", event.GuildID,
			"error", err,
		)
	}
}

func (h *NotificationEventHandler) handlePlaybackFinished(
	ctx context.Context,
	event domain.PlaybackFinishedEvent,
) {
	// Delete the "Now Playing" message if it exists
	if event.LastMessageID == nil {
		return
	}

	slog.Debug("deleting now playing message",
		"guild", event.GuildID,
		"message_id", *event.LastMessageID,
	)

	if err := h.notifier.DeleteMessage(
		event.NotificationChannelID,
		*event.LastMessageID,
	); err != nil {
		slog.Warn("failed to delete now playing message",
			"guild", event.GuildID,
			"error", err,
		)
	}

	// Only clear the message info if it matches the one we just deleted.
	// This prevents a race condition where a new track's message info
	// could be cleared if events are processed out of order.
	state, err := h.repo.Get(ctx, event.GuildID)
	if err == nil {
		currentMsg := state.GetNowPlayingMessage()
		if currentMsg != nil && currentMsg.MessageID == *event.LastMessageID {
			state.ClearNowPlayingMessage()
			_ = h.repo.Save(ctx, state)
		}
	}
}
