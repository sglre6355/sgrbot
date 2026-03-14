package core

import (
	"context"
	"log/slog"
)

// PlayerService is a domain service that wraps PlayerState mutations,
// generates the correct domain events, and integrates AutoPlayService
// for auto-play recommendations.
type PlayerService struct {
	autoPlay *AutoPlayService
}

// NewPlayerService creates a new PlayerService.
func NewPlayerService(autoPlay *AutoPlayService) *PlayerService {
	return &PlayerService{autoPlay: autoPlay}
}

// Prepend adds entries to the front of the queue.
func (s *PlayerService) Prepend(
	state *PlayerState,
	entries ...QueueEntry,
) (events []Event) {
	added := make([]QueueEntry, len(entries))
	copy(added, entries)

	state.Prepend(entries...)

	events = append(events, TrackAddedEvent{PlayerStateID: state.ID(), Entries: added})

	return events
}

// Append adds entries to the end of the queue.
func (s *PlayerService) Append(
	state *PlayerState,
	entries ...QueueEntry,
) (startIndex int, becameActive bool, events []Event) {
	added := make([]QueueEntry, len(entries))
	copy(added, entries)

	startIndex, becameActive = state.Append(entries...)

	events = append(events, TrackAddedEvent{PlayerStateID: state.ID(), Entries: added})
	if becameActive {
		events = append(
			events,
			TrackStartedEvent{PlayerStateID: state.ID(), Entry: *state.Current()},
		)
	}

	return startIndex, becameActive, events
}

// Insert inserts entries at the given index in the queue.
func (s *PlayerService) Insert(
	state *PlayerState,
	index int,
	entries ...QueueEntry,
) ([]Event, error) {
	if err := state.Insert(index, entries...); err != nil {
		return nil, err
	}

	added := make([]QueueEntry, len(entries))
	copy(added, entries)

	events := []Event{TrackAddedEvent{PlayerStateID: state.ID(), Entries: added}}
	return events, nil
}

// Seek sets the current index to the specified position.
func (s *PlayerService) Seek(
	state *PlayerState,
	index int,
) (*QueueEntry, []Event) {
	entry := state.Seek(index)
	if entry == nil {
		return nil, nil
	}

	events := []Event{TrackStartedEvent{PlayerStateID: state.ID(), Entry: *entry}}
	return entry, events
}

// Shuffle randomizes the order of entries in the queue.
func (s *PlayerService) Shuffle(state *PlayerState) []Event {
	if state.IsEmpty() {
		state.Shuffle()
		return nil
	}

	state.Shuffle()
	return []Event{QueueShuffledEvent{PlayerStateID: state.ID()}}
}

// Remove removes the entry at the given index.
func (s *PlayerService) Remove(
	state *PlayerState,
	index int,
) (*QueueEntry, []Event, error) {
	removingCurrent := state.IsPlaybackActive() && index == state.CurrentIndex()

	entry, err := state.Remove(index)
	if err != nil {
		return nil, nil, err
	}

	events := []Event{TrackRemovedEvent{PlayerStateID: state.ID(), Entry: *entry}}

	if removingCurrent {
		if state.IsPlaybackActive() {
			events = append(events, TrackStartedEvent{
				PlayerStateID: state.ID(),
				Entry:         *state.Current(),
			})
		} else {
			events = append(events, PlaybackStoppedEvent{PlayerStateID: state.ID()})
		}
	}

	return entry, events, nil
}

// Clear removes all entries from the queue.
func (s *PlayerService) Clear(state *PlayerState) (int, []Event) {
	wasActive := state.IsPlaybackActive()
	count := state.Len()

	state.Clear()

	var events []Event
	if count > 0 {
		events = append(events, QueueClearedEvent{PlayerStateID: state.ID(), Count: count})
	}
	if wasActive {
		events = append(events, PlaybackStoppedEvent{PlayerStateID: state.ID()})
	}

	return count, events
}

// ClearExceptCurrent removes all entries except the currently playing track.
func (s *PlayerService) ClearExceptCurrent(
	state *PlayerState,
) (int, []Event, error) {
	count, err := state.ClearExceptCurrent()
	if err != nil {
		return 0, nil, err
	}

	var events []Event
	if count > 0 {
		events = append(events, QueueClearedEvent{PlayerStateID: state.ID(), Count: count})
	}

	return count, events, nil
}

// Pause transitions the player to the paused state.
func (s *PlayerService) Pause(state *PlayerState) ([]Event, error) {
	if err := state.Pause(); err != nil {
		return nil, err
	}

	return []Event{PlaybackPausedEvent{PlayerStateID: state.ID()}}, nil
}

// Resume transitions the player from the paused state.
func (s *PlayerService) Resume(state *PlayerState) ([]Event, error) {
	if err := state.Resume(); err != nil {
		return nil, err
	}

	return []Event{PlaybackResumedEvent{PlayerStateID: state.ID()}}, nil
}

// Skip advances past the current track.
func (s *PlayerService) Skip(
	state *PlayerState,
) (skipped *QueueEntry, events []Event, err error) {
	skipped, next, err := state.Skip()
	if err != nil {
		return nil, nil, err
	}

	if next != nil {
		events = append(events, TrackStartedEvent{PlayerStateID: state.ID(), Entry: *next})
	} else {
		events = append(events, QueueExhaustedEvent{PlayerStateID: state.ID()})
	}

	return skipped, events, nil
}

// TryAutoPlay attempts to append an auto-play recommendation and start it.
// Returns the new current entry and events, or nil if auto-play is disabled/failed.
func (s *PlayerService) TryAutoPlay(
	ctx context.Context,
	state *PlayerState,
) (*QueueEntry, []Event) {
	if !state.IsAutoPlayEnabled() || s.autoPlay == nil {
		return nil, nil
	}

	entry, err := s.autoPlay.GetNextRecommendation(ctx, state)
	if err != nil {
		slog.Debug("auto-play failed", "playerStateID", state.ID(), "error", err)
		return nil, nil
	}

	startIndex, _ := state.Append(entry)
	current := state.Seek(startIndex)
	if current == nil {
		return nil, nil
	}

	events := []Event{
		TrackAddedEvent{PlayerStateID: state.ID(), Entries: []QueueEntry{entry}},
		TrackStartedEvent{PlayerStateID: state.ID(), Entry: *current},
	}
	return current, events
}
