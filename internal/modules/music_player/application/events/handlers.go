package events

import (
	"context"
	"log/slog"
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PlayNextFunc is the function signature for playing the next track.
type PlayNextFunc func(ctx context.Context, guildID snowflake.ID) (*domain.Track, error)

// PlaybackEventHandler handles events related to playback control.
// It listens for TrackEnqueued and TrackEnded events to manage playback flow.
type PlaybackEventHandler struct {
	playNextFunc PlayNextFunc
	repo         domain.PlayerStateRepository
	bus          *Bus

	wg   sync.WaitGroup
	done chan struct{}
}

// NewPlaybackEventHandler creates a new PlaybackEventHandler.
func NewPlaybackEventHandler(
	playNextFunc PlayNextFunc,
	repo domain.PlayerStateRepository,
	bus *Bus,
) *PlaybackEventHandler {
	return &PlaybackEventHandler{
		playNextFunc: playNextFunc,
		repo:         repo,
		bus:          bus,
		done:         make(chan struct{}),
	}
}

// Start begins listening for events in background goroutines.
func (h *PlaybackEventHandler) Start(ctx context.Context) {
	h.wg.Add(2)

	// Handle TrackEnqueued events
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.done:
				return
			case event, ok := <-h.bus.TrackEnqueued():
				if !ok {
					return
				}
				h.handleTrackEnqueued(ctx, event)
			}
		}
	}()

	// Handle TrackEnded events
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.done:
				return
			case event, ok := <-h.bus.TrackEnded():
				if !ok {
					return
				}
				h.handleTrackEnded(ctx, event)
			}
		}
	}()

	slog.Debug("playback event handler started")
}

// Stop stops the event handler and waits for goroutines to finish.
func (h *PlaybackEventHandler) Stop() {
	close(h.done)
	h.wg.Wait()
	slog.Debug("playback event handler stopped")
}

func (h *PlaybackEventHandler) handleTrackEnqueued(ctx context.Context, event TrackEnqueuedEvent) {
	// Only start playback if the player was idle
	if !event.WasIdle {
		slog.Debug("track enqueued but player not idle, skipping auto-play",
			"guild", event.GuildID,
			"track", event.Track.Title,
		)
		return
	}

	slog.Debug("track enqueued and player idle, starting playback",
		"guild", event.GuildID,
		"track", event.Track.Title,
	)

	_, err := h.playNextFunc(ctx, event.GuildID)
	if err != nil {
		slog.Error("failed to start playback after track enqueued",
			"guild", event.GuildID,
			"error", err,
		)
	}
}

func (h *PlaybackEventHandler) handleTrackEnded(ctx context.Context, event TrackEndedEvent) {
	// Only advance queue for certain end reasons
	if !event.Reason.ShouldAdvanceQueue() {
		slog.Debug("track ended but should not advance queue",
			"guild", event.GuildID,
			"reason", event.Reason,
		)
		return
	}

	slog.Debug("track ended, advancing queue",
		"guild", event.GuildID,
		"reason", event.Reason,
	)

	// Get state to check notification channel for error reporting
	state := h.repo.Get(event.GuildID)
	if state == nil {
		slog.Debug("track ended but no player state",
			"guild", event.GuildID,
		)
		return
	}

	// Delete the old "Now Playing" message before playing next
	nowPlayingMsg := state.GetNowPlayingMessage()
	if nowPlayingMsg != nil {
		h.bus.PublishPlaybackFinished(PlaybackFinishedEvent{
			GuildID:               event.GuildID,
			NotificationChannelID: nowPlayingMsg.ChannelID,
			LastMessageID:         &nowPlayingMsg.MessageID,
		})
	}

	// Remove finished track from queue before playing next
	state.SetStopped()

	_, err := h.playNextFunc(ctx, event.GuildID)
	if err != nil {
		slog.Error("failed to play next track after track ended",
			"guild", event.GuildID,
			"error", err,
		)

		// Publish error notification if we have a channel
		if errNowPlayingMsg := state.GetNowPlayingMessage(); errNowPlayingMsg != nil {
			h.bus.PublishPlaybackFinished(PlaybackFinishedEvent{
				GuildID:               event.GuildID,
				NotificationChannelID: errNowPlayingMsg.ChannelID,
				LastMessageID:         &errNowPlayingMsg.MessageID,
			})
		}
	}
}

// NotificationEventHandler handles events related to Discord notifications.
// It listens for PlaybackStarted and PlaybackFinished events to send/delete messages.
type NotificationEventHandler struct {
	notifier ports.NotificationSender
	repo     domain.PlayerStateRepository
	bus      *Bus

	wg   sync.WaitGroup
	done chan struct{}
}

// NewNotificationEventHandler creates a new NotificationEventHandler.
func NewNotificationEventHandler(
	notifier ports.NotificationSender,
	repo domain.PlayerStateRepository,
	bus *Bus,
) *NotificationEventHandler {
	return &NotificationEventHandler{
		notifier: notifier,
		repo:     repo,
		bus:      bus,
		done:     make(chan struct{}),
	}
}

// Start begins listening for events in background goroutines.
func (h *NotificationEventHandler) Start(ctx context.Context) {
	h.wg.Add(2)

	// Handle PlaybackStarted events
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.done:
				return
			case event, ok := <-h.bus.PlaybackStarted():
				if !ok {
					return
				}
				h.handlePlaybackStarted(event)
			}
		}
	}()

	// Handle PlaybackFinished events
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.done:
				return
			case event, ok := <-h.bus.PlaybackFinished():
				if !ok {
					return
				}
				h.handlePlaybackFinished(event)
			}
		}
	}()

	slog.Debug("notification event handler started")
}

// Stop stops the event handler and waits for goroutines to finish.
func (h *NotificationEventHandler) Stop() {
	close(h.done)
	h.wg.Wait()
	slog.Debug("notification event handler stopped")
}

func (h *NotificationEventHandler) handlePlaybackStarted(event PlaybackStartedEvent) {
	slog.Debug("sending now playing notification",
		"guild", event.GuildID,
		"track", event.Track.Title,
	)

	messageID, err := h.notifier.SendNowPlaying(event.NotificationChannelID, &ports.NowPlayingInfo{
		Identifier:         string(event.Track.ID),
		Title:              event.Track.Title,
		Artist:             event.Track.Artist,
		Duration:           event.Track.FormattedDuration(),
		URI:                event.Track.URI,
		ArtworkURL:         event.Track.ArtworkURL,
		SourceName:         event.Track.SourceName,
		IsStream:           event.Track.IsStream,
		RequesterID:        event.Track.RequesterID,
		RequesterName:      event.Track.RequesterName,
		RequesterAvatarURL: event.Track.RequesterAvatarURL,
		EnqueuedAt:         event.Track.EnqueuedAt,
	})
	if err != nil {
		slog.Error("failed to send now playing notification",
			"guild", event.GuildID,
			"error", err,
		)
		return
	}

	// Store the message info for later deletion
	state := h.repo.Get(event.GuildID)
	if state != nil {
		state.SetNowPlayingMessage(event.NotificationChannelID, messageID)
	}
}

func (h *NotificationEventHandler) handlePlaybackFinished(event PlaybackFinishedEvent) {
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
	state := h.repo.Get(event.GuildID)
	if state != nil {
		currentMsg := state.GetNowPlayingMessage()
		if currentMsg != nil && currentMsg.MessageID == *event.LastMessageID {
			state.ClearNowPlayingMessage()
		}
	}
}
