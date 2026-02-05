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

// StopFunc is the function signature for stopping playback.
type StopFunc func(ctx context.Context, guildID snowflake.ID) error

// PlaybackEventHandler handles events related to playback control.
// It listens for TrackEnqueued, TrackEnded, and QueueCleared events to manage playback flow.
type PlaybackEventHandler struct {
	playNextFunc PlayNextFunc
	stopFunc     StopFunc
	repo         domain.PlayerStateRepository
	bus          *Bus

	wg   sync.WaitGroup
	done chan struct{}
}

// NewPlaybackEventHandler creates a new PlaybackEventHandler.
func NewPlaybackEventHandler(
	playNextFunc PlayNextFunc,
	stopFunc StopFunc,
	repo domain.PlayerStateRepository,
	bus *Bus,
) *PlaybackEventHandler {
	return &PlaybackEventHandler{
		playNextFunc: playNextFunc,
		stopFunc:     stopFunc,
		repo:         repo,
		bus:          bus,
		done:         make(chan struct{}),
	}
}

// Start begins listening for events in background goroutines.
func (h *PlaybackEventHandler) Start(ctx context.Context) {
	h.wg.Add(3)

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

	// Handle QueueCleared events
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.done:
				return
			case event, ok := <-h.bus.QueueCleared():
				if !ok {
					return
				}
				h.handleQueueCleared(ctx, event)
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
	// Only start playback if the player was idle at enqueue time
	if !event.WasIdle {
		slog.Debug("track enqueued but player not idle, skipping auto-play",
			"guild", event.GuildID,
			"track", event.Track.Title,
		)
		return
	}

	// Re-check current state to avoid race with concurrent enqueues.
	// Multiple tracks enqueued while idle will all have WasIdle=true,
	// but only the first should trigger playback.
	state := h.repo.Get(event.GuildID)
	if state == nil {
		slog.Debug("track enqueued but state not found, skipping auto-play",
			"guild", event.GuildID,
		)
		return
	}

	// Check if playback is already active (another event already started playback).
	// IsIdle() now checks playbackActive flag, not queue position.
	if !state.IsIdle() {
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

	_, err := h.playNextFunc(ctx, event.GuildID)
	if err != nil {
		slog.Error("failed to start playback after track enqueued",
			"guild", event.GuildID,
			"error", err,
		)
	}
}

func (h *PlaybackEventHandler) handleQueueCleared(ctx context.Context, event QueueClearedEvent) {
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

	// Delete the "Now Playing" message
	state := h.repo.Get(event.GuildID)
	if state != nil {
		nowPlayingMsg := state.GetNowPlayingMessage()
		if nowPlayingMsg != nil {
			h.bus.PublishPlaybackFinished(PlaybackFinishedEvent{
				GuildID:               event.GuildID,
				NotificationChannelID: nowPlayingMsg.ChannelID,
				LastMessageID:         &nowPlayingMsg.MessageID,
			})
		}
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

	// Get state to check loop mode and notification channel
	state := h.repo.Get(event.GuildID)
	if state == nil {
		slog.Debug("track ended but no player state",
			"guild", event.GuildID,
		)
		return
	}

	loopMode := state.LoopMode()

	slog.Debug("track ended, advancing queue",
		"guild", event.GuildID,
		"reason", event.Reason,
		"loop_mode", loopMode.String(),
	)

	// Delete the old "Now Playing" message before playing next
	nowPlayingMsg := state.GetNowPlayingMessage()
	if nowPlayingMsg != nil {
		h.bus.PublishPlaybackFinished(PlaybackFinishedEvent{
			GuildID:               event.GuildID,
			NotificationChannelID: nowPlayingMsg.ChannelID,
			LastMessageID:         &nowPlayingMsg.MessageID,
		})
	}

	// Advance queue based on loop mode
	// For TrackEndLoadFailed, remove the failing track and advance to prevent infinite retry loops
	if event.Reason == TrackEndLoadFailed {
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
		state.SetStopped()
	}

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
	// Check if the track is still current before sending notification.
	// This prevents sending "Now Playing" for tracks that failed to load,
	// which would leave orphaned messages since handleTrackEnded already ran.
	state := h.repo.Get(event.GuildID)
	if state == nil {
		slog.Debug("skipping now playing notification, state not found",
			"guild", event.GuildID,
		)
		return
	}
	currentTrack := state.CurrentTrack()
	if currentTrack == nil || currentTrack.ID != event.Track.ID {
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
	state.SetNowPlayingMessage(event.NotificationChannelID, messageID)
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
