package application

import (
	"context"
	"log/slog"
	"reflect"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PlaybackEventHandler handles events related to playback control.
// It subscribes to CurrentTrackChanged and TrackEnded events to manage playback flow.
type PlaybackEventHandler struct {
	playerStates    domain.PlayerStateRepository
	player          ports.TrackPlayer
	publisher       ports.EventPublisher
	subscriber      ports.EventSubscriber
	autoPlayService *domain.AutoPlayService
	botUserID       snowflake.ID
}

// NewPlaybackEventHandler creates a new PlaybackEventHandler.
func NewPlaybackEventHandler(
	playerStates domain.PlayerStateRepository,
	player ports.TrackPlayer,
	publisher ports.EventPublisher,
	subscriber ports.EventSubscriber,
	autoPlayService *domain.AutoPlayService,
	botUserID snowflake.ID,
) *PlaybackEventHandler {
	return &PlaybackEventHandler{
		playerStates:    playerStates,
		player:          player,
		subscriber:      subscriber,
		publisher:       publisher,
		autoPlayService: autoPlayService,
		botUserID:       botUserID,
	}
}

// Start registers event handlers with the subscriber.
func (h *PlaybackEventHandler) Start() error {
	err := h.subscriber.Subscribe(
		reflect.TypeFor[domain.CurrentTrackChangedEvent](),
		func(ctx context.Context, e domain.Event) {
			h.handleCurrentTrackChanged(ctx, e.(domain.CurrentTrackChangedEvent))
		},
	)
	if err != nil {
		return err
	}

	err = h.subscriber.Subscribe(
		reflect.TypeFor[domain.TrackEndedEvent](),
		func(ctx context.Context, e domain.Event) {
			h.handleTrackEnded(ctx, e.(domain.TrackEndedEvent))
		},
	)
	if err != nil {
		return err
	}

	slog.Debug("playback event handlers properly registered")

	return nil
}

func (h *PlaybackEventHandler) handleCurrentTrackChanged(
	ctx context.Context,
	event domain.CurrentTrackChangedEvent,
) {
	// Check current state to avoid race with concurrent enqueues.
	state, err := h.playerStates.Get(ctx, event.GuildID)
	if err != nil {
		slog.Warn(
			"player state not found, skipping",
			"event", event,
		)
		return
	}

	current := state.Current()

	// Queue ended — try auto-play if enabled
	if current == nil && state.IsAutoPlayEnabled() {
		track := h.autoPlayService.GetRecommendation(ctx, &state)
		if track != nil {
			entry := domain.NewQueueEntry(track.ID, h.botUserID, time.Now(), true)
			state.Enqueue(entry)

			slog.Info(
				"auto-play queued track",
				"guild", state.GetGuildID(),
				"track", track.ID,
			)

			if err := h.playerStates.Save(ctx, state); err != nil {
				slog.Error(
					"failed to save player state after auto-play",
					"event", event,
					"error", err,
				)
				return
			}

			current = state.Current()
		}
	}

	if current == nil {
		slog.Debug("no current track in the queue, stopping playback")
		if err := h.player.Stop(ctx, event.GuildID); err != nil {
			slog.Warn(
				"failed to stop the audio playback",
				"event", event,
			)
		}
		return
	}

	slog.Debug(
		"starting track",
		"event", event,
	)

	err = h.player.Play(ctx, state.GetGuildID(), current.TrackID)
	if err != nil {
		slog.Error(
			"failed to start track after queue index change",
			"event", event,
			"error", err,
		)
	}
}

func (h *PlaybackEventHandler) handleTrackEnded(ctx context.Context, event domain.TrackEndedEvent) {
	// Only advance queue for certain end reasons
	if !event.ShouldAdvanceQueue {
		return
	}

	// Get state to check loop mode and notification channel
	state, err := h.playerStates.Get(ctx, event.GuildID)
	if err != nil {
		slog.Warn(
			"track ended but player state not found",
			"event", event,
		)
		return
	}

	slog.Debug(
		"track ended, advancing queue",
		"event", event,
		"loop_mode", state.GetLoopMode().String(),
	)

	state.HandleTrackEnded(event.TrackFailed)

	// Save player state
	if err := h.playerStates.Save(ctx, state); err != nil {
		slog.Error(
			"failed to save player state",
			"event", event,
			"error", err,
		)
	}

	currentChangedEvent := domain.NewCurrentTrackChangedEvent(
		event.GuildID,
	)
	err = h.publisher.Publish(currentChangedEvent)
	if err != nil {
		slog.Warn(
			"failed to publish CurrentTrackChangedEvent",
			"event", currentChangedEvent,
			"error", err,
		)
	}
}

// NotificationEventHandler handles events related to Discord notifications.
// It subscribes to CurrentTrackChangedEvent to send/delete messages.
type NotificationEventHandler struct {
	playerStates     domain.PlayerStateRepository
	subscriber       ports.EventSubscriber
	notifier         ports.NotificationSender
	userInfoProvider ports.UserInfoProvider
}

// NewNotificationEventHandler creates a new NotificationEventHandler.
func NewNotificationEventHandler(
	playerStates domain.PlayerStateRepository,
	subscriber ports.EventSubscriber,
	notifier ports.NotificationSender,
	userInfoProvider ports.UserInfoProvider,
) *NotificationEventHandler {
	return &NotificationEventHandler{
		playerStates:     playerStates,
		subscriber:       subscriber,
		notifier:         notifier,
		userInfoProvider: userInfoProvider,
	}
}

// Start registers event handlers with the subscriber.
func (h *NotificationEventHandler) Start() error {
	err := h.subscriber.Subscribe(
		reflect.TypeFor[domain.CurrentTrackChangedEvent](),
		func(ctx context.Context, e domain.Event) {
			h.handleCurrentTrackChanged(ctx, e.(domain.CurrentTrackChangedEvent))
		},
	)
	if err != nil {
		return err
	}

	slog.Debug("notification event handlers properly registered")

	return nil
}

func (h *NotificationEventHandler) handleCurrentTrackChanged(
	ctx context.Context,
	event domain.CurrentTrackChangedEvent,
) {
	// Check if the track is still current before sending notification.
	// This prevents sending "Now Playing" for tracks that failed to load,
	// which would leave orphaned messages since handleCurrentTrackChanged already ran.
	state, err := h.playerStates.Get(ctx, event.GuildID)
	if err != nil {
		slog.Debug(
			"skipping now playing notification, state not found",
			"guild", event.GuildID,
		)
		return
	}

	oldNowPlayingMessage := state.GetNowPlayingMessage()
	if oldNowPlayingMessage != nil {
		err := h.notifier.DeleteMessage(
			oldNowPlayingMessage.ChannelID,
			oldNowPlayingMessage.MessageID,
		)
		if err != nil {
			slog.Warn(
				"failed to delete previous now playing message",
				"event", event,
				"now_playing", oldNowPlayingMessage,
				"error", err,
			)
		}
		state.SetNowPlayingMessage(nil)
	}

	current := state.Current()
	if current != nil {
		slog.Debug(
			"sending now playing notification",
			"event", event,
		)

		messageID, err := h.notifier.SendNowPlaying(
			event.GuildID,
			state.GetNotificationChannelID(),
			current.TrackID,
			current.RequesterID,
			current.EnqueuedAt,
		)
		if err == nil {
			// Store the message info for later deletion
			newNowPlayingMessage := domain.NewNowPlayingMessage(
				state.GetNotificationChannelID(),
				messageID,
			)
			state.SetNowPlayingMessage(&newNowPlayingMessage)
		} else {
			slog.Error(
				"failed to send now playing notification",
				"event", event,
				"error", err,
			)
		}
	}

	if err := h.playerStates.Save(ctx, state); err != nil {
		slog.Error(
			"failed to save player state",
			"event", event,
			"error", err,
		)
	}
}
