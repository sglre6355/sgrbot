package application

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"reflect"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// PlaybackEventHandler handles events related to playback control.
// It subscribes to CurrentTrackChanged and TrackEnded events to manage playback flow.
type PlaybackEventHandler struct {
	playerStates  domain.PlayerStateRepository
	player        ports.AudioPlayer
	publisher     ports.EventPublisher
	subscriber    ports.EventSubscriber
	recommender   ports.TrackRecommender
	trackProvider ports.TrackProvider
	botUserID     snowflake.ID
}

// NewPlaybackEventHandler creates a new PlaybackEventHandler.
func NewPlaybackEventHandler(
	playerStates domain.PlayerStateRepository,
	player ports.AudioPlayer,
	publisher ports.EventPublisher,
	subscriber ports.EventSubscriber,
	recommender ports.TrackRecommender,
	trackProvider ports.TrackProvider,
	botUserID snowflake.ID,
) *PlaybackEventHandler {
	return &PlaybackEventHandler{
		playerStates:  playerStates,
		player:        player,
		subscriber:    subscriber,
		publisher:     publisher,
		recommender:   recommender,
		trackProvider: trackProvider,
		botUserID:     botUserID,
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
	if current == nil && state.IsAutoPlayEnabled() && h.recommender != nil {
		ok := h.tryAutoPlay(ctx, &state)
		if ok {
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

	if event.TrackFailed {
		// Remove advances past the failed track (respecting loop mode),
		// then removes it from the queue.
		if _, err := state.Remove(state.CurrentIndex()); err != nil {
			slog.Warn(
				"failed to remove failing track",
				"event", event,
				"error", err,
			)
		}
	} else {
		next := state.Advance(state.GetLoopMode())
		if next == nil {
			state.SetPlaybackActive(false)
		}
	}

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

// tryAutoPlay attempts to recommend and enqueue a track for auto-play.
// Returns true if a track was successfully appended and the state was advanced.
func (h *PlaybackEventHandler) tryAutoPlay(
	ctx context.Context,
	state *domain.PlayerState,
) bool {
	// Collect seed and exclude track IDs using a weighted mix:
	// up to 2 randomly sampled manually added YouTube tracks + the most recent
	// auto-play YouTube track.
	// This keeps recommendations anchored to the user's selections while allowing
	// natural progression via the latest auto-play track.
	allEntries := state.List()
	var manualIDs []domain.TrackID
	var autoPlayIDs []domain.TrackID
	for _, entry := range allEntries {
		if entry.IsAutoPlay {
			autoPlayIDs = append(autoPlayIDs, entry.TrackID)
		} else {
			manualIDs = append(manualIDs, entry.TrackID)
		}
	}

	seeds := make([]domain.TrackID, 0, 3)
	seedSet := make(map[domain.TrackID]struct{}, 3)

	// Sample up to 2 manual YouTube seeds
	rand.Shuffle(len(manualIDs), func(i, j int) {
		manualIDs[i], manualIDs[j] = manualIDs[j], manualIDs[i]
	})
	for _, id := range manualIDs {
		if len(seeds) >= 2 {
			break
		}
		track, err := h.trackProvider.LoadTrack(ctx, id)
		if err != nil {
			slog.Debug(
				"failed to load manual seed track",
				"track", id,
				"error", err,
			)
			continue
		}
		if track.Source != domain.TrackSourceYouTube {
			continue
		}
		if _, exists := seedSet[id]; exists {
			continue
		}
		seedSet[id] = struct{}{}
		seeds = append(seeds, id)
	}

	// Add the most recent auto-play YouTube track as a seed
	for i := len(autoPlayIDs) - 1; i >= 0; i-- {
		id := autoPlayIDs[i]
		if _, exists := seedSet[id]; exists {
			continue
		}
		track, err := h.trackProvider.LoadTrack(ctx, id)
		if err != nil {
			slog.Debug(
				"failed to load auto-play seed track",
				"track", id,
				"error", err,
			)
			continue
		}
		if track.Source != domain.TrackSourceYouTube {
			continue
		}
		seedSet[id] = struct{}{}
		seeds = append(seeds, id)
		break
	}

	if len(seeds) == 0 {
		slog.Debug(
			"auto-play has no YouTube seeds",
			"guild", state.GetGuildID(),
		)
		return false
	}

	// Exclude all tracks not selected as seeds
	var exclude []domain.TrackID
	for _, entry := range allEntries {
		if _, isSeed := seedSet[entry.TrackID]; !isSeed {
			exclude = append(exclude, entry.TrackID)
		}
	}

	tracks, err := h.recommender.Recommend(ctx, seeds, exclude, 1)
	if err != nil {
		slog.Warn(
			"auto-play recommendation failed",
			"guild", state.GetGuildID(),
			"error", err,
		)
		return false
	}

	if len(tracks) == 0 {
		slog.Debug(
			"auto-play found no recommendations",
			"guild", state.GetGuildID(),
		)
		return false
	}

	// Append the recommended track as an auto-play entry
	entry := domain.NewQueueEntry(tracks[0].ID, h.botUserID, time.Now(), true)
	state.Append(entry)

	// Activate playback and advance to the newly appended track
	state.SetPlaybackActive(true)
	next := state.Advance(domain.LoopModeNone)
	if next == nil {
		return false
	}

	slog.Info(
		"auto-play queued track",
		"guild", state.GetGuildID(),
		"track", tracks[0].ID,
	)

	return true
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
