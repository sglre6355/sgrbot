package application

import (
	"context"
	"log/slog"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/discord"
)

// DomainEventHandlers handles domain events by coordinating application-level
// side effects such as advancing the queue and updating the now-playing display.
type DomainEventHandlers struct {
	player       *core.PlayerService
	playerStates core.PlayerStateRepository
	users        core.UserRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
	nowPlaying   ports.NowPlayingGateway[discord.NowPlayingDestination]
}

// NewDomainEventHandlers creates a new DomainEventHandlers.
func NewDomainEventHandlers(
	player *core.PlayerService,
	playerStates core.PlayerStateRepository,
	users core.UserRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	nowPlaying ports.NowPlayingGateway[discord.NowPlayingDestination],
) *DomainEventHandlers {
	return &DomainEventHandlers{
		player:       player,
		playerStates: playerStates,
		users:        users,
		audio:        audio,
		events:       events,
		nowPlaying:   nowPlaying,
	}
}

// HandleTrackStarted updates the now-playing display when a new track begins.
func (h *DomainEventHandlers) HandleTrackStarted(ctx context.Context, e core.Event) {
	event := e.(core.TrackStartedEvent)
	if err := h.updateNowPlaying(ctx, event.PlayerStateID); err != nil {
		slog.Error("failed to handle track start",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
	}
}

// HandleTrackEnded advances the queue or stops playback when a track finishes.
func (h *DomainEventHandlers) HandleTrackEnded(ctx context.Context, e core.Event) {
	event := e.(core.TrackEndedEvent)

	state, err := h.playerStates.FindByID(ctx, event.PlayerStateID)
	if err != nil {
		slog.Error("failed to handle track end",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
	}

	// Guard against stale events: the track that ended must still be current.
	current := state.Current()
	if current == nil || *current != event.Entry {
		return
	}

	var next *core.QueueEntry
	var events []core.Event

	if event.TrackFailed {
		// Capture current entry before removal for the event.
		cp := *current

		// LoopModeTrack is treated as LoopModeNone because the track being
		// looped is being removed, so there is nothing to repeat.
		if _, err := state.Remove(state.CurrentIndex()); err != nil {
			return
		}

		events = append(events, core.TrackRemovedEvent{PlayerStateID: state.ID(), Entry: cp})
		next = state.Current()
	} else {
		next = state.Advance(state.LoopMode())
	}

	if next != nil {
		events = append(events, core.TrackStartedEvent{PlayerStateID: state.ID(), Entry: *next})
	} else {
		// Queue exhausted — try auto-play
		autoNext, autoEvents := h.player.TryAutoPlay(ctx, &state)
		events = append(events, autoEvents...)
		if autoNext != nil {
			next = autoNext
		} else {
			events = append(events, core.PlaybackStoppedEvent{PlayerStateID: state.ID()})
		}
	}

	if next != nil {
		if err := h.audio.Play(ctx, state.ID(), *next); err != nil {
			slog.Error("failed to handle track end",
				"playerStateID", event.PlayerStateID,
				"error", err,
			)
			return
		}
	}

	if err := h.playerStates.Save(ctx, state); err != nil {
		slog.Error("failed to handle track end",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
	}

	h.events.Publish(ctx, events...)
}

// HandlePlaybackStopped clears the now-playing display when playback stops.
func (h *DomainEventHandlers) HandlePlaybackStopped(ctx context.Context, e core.Event) {
	event := e.(core.PlaybackStoppedEvent)
	if err := h.updateNowPlaying(ctx, event.PlayerStateID); err != nil {
		slog.Error("failed to handle playback stop",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
	}
}

// HandleQueueExhausted attempts auto-play when the queue runs out of tracks.
func (h *DomainEventHandlers) HandleQueueExhausted(ctx context.Context, e core.Event) {
	event := e.(core.QueueExhaustedEvent)

	state, err := h.playerStates.FindByID(ctx, event.PlayerStateID)
	if err != nil {
		slog.Error("failed to handle queue exhausted",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
	}

	next, events := h.player.TryAutoPlay(ctx, &state)
	if next != nil {
		if err := h.audio.Play(ctx, state.ID(), *next); err != nil {
			slog.Error("failed to handle queue exhausted",
				"playerStateID", event.PlayerStateID,
				"error", err,
			)
			return
		}
	} else {
		events = append(events, core.PlaybackStoppedEvent{PlayerStateID: state.ID()})
	}

	if err := h.playerStates.Save(ctx, state); err != nil {
		slog.Error("failed to handle queue exhausted",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
	}

	h.events.Publish(ctx, events...)
}

func (h *DomainEventHandlers) updateNowPlaying(
	ctx context.Context,
	playerStateID core.PlayerStateID,
) error {
	state, err := h.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return err
	}

	if err := h.nowPlaying.Clear(playerStateID); err != nil {
		return err
	}

	current := state.Current()

	if current != nil {
		requester, err := h.users.FindByID(current.RequesterID())
		if err != nil {
			requester = core.User{
				ID:   current.RequesterID(),
				Name: "Unknown",
			}
		}

		if err := h.nowPlaying.Show(
			playerStateID,
			*current.Track(),
			requester,
			current.AddedAt(),
		); err != nil {
			return err
		}
	}

	return nil
}
