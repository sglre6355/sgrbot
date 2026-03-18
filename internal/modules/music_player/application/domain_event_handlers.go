package application

import (
	"context"
	"log/slog"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// DomainEventHandlers handles domain events by coordinating application-level
// side effects such as advancing the queue and updating the now-playing display.
type DomainEventHandlers struct {
	player       *domain.PlayerService
	playerStates domain.PlayerStateRepository
	users        domain.UserRepository
	audio        ports.AudioGateway
	events       ports.EventPublisher
	nowPlaying   ports.NowPlayingPublisher
}

// NewDomainEventHandlers creates a new DomainEventHandlers.
func NewDomainEventHandlers(
	player *domain.PlayerService,
	playerStates domain.PlayerStateRepository,
	users domain.UserRepository,
	audio ports.AudioGateway,
	events ports.EventPublisher,
	nowPlaying ports.NowPlayingPublisher,
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
func (h *DomainEventHandlers) HandleTrackStarted(ctx context.Context, e domain.Event) {
	event := e.(domain.TrackStartedEvent)
	if err := h.updateNowPlaying(ctx, event.PlayerStateID); err != nil {
		slog.Error("failed to handle track start",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
	}
}

// HandleTrackEnded advances the queue or stops playback when a track finishes.
func (h *DomainEventHandlers) HandleTrackEnded(ctx context.Context, e domain.Event) {
	event := e.(domain.TrackEndedEvent)

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

	var next *domain.QueueEntry
	var events []domain.Event

	if event.TrackFailed {
		// Capture current entry before removal for the event.
		cp := *current

		// LoopModeTrack is treated as LoopModeNone because the track being
		// looped is being removed, so there is nothing to repeat.
		if _, err := state.Remove(state.CurrentIndex()); err != nil {
			return
		}

		events = append(events, domain.TrackRemovedEvent{PlayerStateID: state.ID(), Entry: cp})
		next = state.Current()
	} else {
		next = state.Advance(state.LoopMode())
	}

	if next != nil {
		events = append(events, domain.TrackStartedEvent{PlayerStateID: state.ID(), Entry: *next})
	} else {
		// Queue exhausted — try auto-play
		autoNext, autoEvents := h.player.TryAutoPlay(ctx, &state)
		events = append(events, autoEvents...)
		if autoNext != nil {
			next = autoNext
		} else {
			events = append(events, domain.PlaybackStoppedEvent{PlayerStateID: state.ID()})
		}
	}

	if err := h.playerStates.Save(ctx, state); err != nil {
		slog.Error("failed to handle track end",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
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

	h.events.Publish(ctx, events...)
}

// HandlePlaybackStopped clears the now-playing display when playback stops.
func (h *DomainEventHandlers) HandlePlaybackStopped(ctx context.Context, e domain.Event) {
	event := e.(domain.PlaybackStoppedEvent)
	if err := h.updateNowPlaying(ctx, event.PlayerStateID); err != nil {
		slog.Error("failed to handle playback stop",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
	}
}

// HandleQueueExhausted attempts auto-play when the queue runs out of tracks.
func (h *DomainEventHandlers) HandleQueueExhausted(ctx context.Context, e domain.Event) {
	event := e.(domain.QueueExhaustedEvent)

	state, err := h.playerStates.FindByID(ctx, event.PlayerStateID)
	if err != nil {
		slog.Error("failed to handle queue exhausted",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
	}

	next, events := h.player.TryAutoPlay(ctx, &state)
	if next == nil {
		events = append(events, domain.PlaybackStoppedEvent{PlayerStateID: state.ID()})
	}

	if err := h.playerStates.Save(ctx, state); err != nil {
		slog.Error("failed to handle queue exhausted",
			"playerStateID", event.PlayerStateID,
			"error", err,
		)
		return
	}

	if next != nil {
		if err := h.audio.Play(ctx, state.ID(), *next); err != nil {
			slog.Error("failed to handle queue exhausted",
				"playerStateID", event.PlayerStateID,
				"error", err,
			)
			return
		}
	}

	h.events.Publish(ctx, events...)
}

func (h *DomainEventHandlers) updateNowPlaying(
	ctx context.Context,
	playerStateID domain.PlayerStateID,
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
			requester = domain.User{
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
