package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// PlayerStateID uniquely identifies a player state.
type PlayerStateID string

// String returns the PlayerStateID as a string.
func (id PlayerStateID) String() string {
	return string(id)
}

// Domain errors for PlayerStateID invariants.
var (
	// ErrInvalidPlayerStateID is returned when an invalid string is passed to ParsePlayerStateID.
	ErrInvalidPlayerStateID = errors.New("invalid player state id")
)

// NewPlayerStateID generates a new unique PlayerStateID.
func NewPlayerStateID() PlayerStateID {
	id := uuid.Must(uuid.NewV7())
	return PlayerStateID(id.String())
}

// ParsePlayerStateID converts a player state id string to a PlayerStateID.
func ParsePlayerStateID(id string) (PlayerStateID, error) {
	if u, err := uuid.Parse(id); err != nil || u.Version() != 7 {
		return "", ErrInvalidPlayerStateID
	}
	return PlayerStateID(id), nil
}

// LoopMode represents the loop mode for queue playback.
type LoopMode int

const (
	LoopModeNone  LoopMode = iota // Default: no looping
	LoopModeTrack                 // Repeat current track indefinitely
	LoopModeQueue                 // Repeat entire queue when reaching end
)

// String returns a human-readable representation of the loop mode.
func (m LoopMode) String() string {
	switch m {
	case LoopModeTrack:
		return "track"
	case LoopModeQueue:
		return "queue"
	default:
		return "none"
	}
}

// ParseLoopMode converts a string to domain.LoopMode.
func ParseLoopMode(s string) LoopMode {
	switch s {
	case "track":
		return LoopModeTrack
	case "queue":
		return LoopModeQueue
	default:
		return LoopModeNone
	}
}

// PlayerState represents the state of a music player.
type PlayerState struct {
	id                PlayerStateID // Unique identifier for this player state
	queue             Queue         // Queue associated with this player state
	currentIndex      int           // Index of the currently playing track in the queue
	isPlaybackActive  bool          // true when playback is active
	isPaused          bool          // true when playback is paused
	isAutoPlayEnabled bool          // true when auto-play is enabled
	loopMode          LoopMode      // loop mode for playback
}

// Domain errors for PlayerState.
var (
	// ErrNotPlaying is returned when an operation requires active playback.
	ErrNotPlaying = errors.New("nothing is currently playing")

	// ErrAlreadyPaused is returned when trying to pause while already paused.
	ErrAlreadyPaused = errors.New("playback is already paused")

	// ErrNotPaused is returned when trying to resume while not paused.
	ErrNotPaused = errors.New("playback is not paused")
)

// NewPlayerState creates a new PlayerState for the given guild and channels.
func NewPlayerState() *PlayerState {
	return &PlayerState{
		id:                NewPlayerStateID(),
		queue:             NewQueue(),
		isAutoPlayEnabled: true,
	}
}

// ConstructPlayerState recreates a PlayerState from persisted data.
func ConstructPlayerState(
	id PlayerStateID,
	queue Queue,
	currentIndex int,
	isPlaybackActive bool,
	isPaused bool,
	isAutoPlayEnabled bool,
	loopMode LoopMode,
) PlayerState {
	return PlayerState{
		id,
		queue,
		currentIndex,
		isPlaybackActive,
		isPaused,
		isAutoPlayEnabled,
		loopMode,
	}
}

// ID returns the player state's unique identifier.
func (p *PlayerState) ID() PlayerStateID {
	return p.id
}

// CurrentIndex returns the current track index.
func (p *PlayerState) CurrentIndex() int {
	return p.currentIndex
}

// IsPlaybackActive returns true if the player is currently playing or paused with a track.
func (p *PlayerState) IsPlaybackActive() bool {
	return p.isPlaybackActive
}

// IsPaused returns true if playback is paused.
func (p *PlayerState) IsPaused() bool {
	return p.isPaused
}

// IsAutoPlayEnabled returns true if auto-play is enabled.
func (p *PlayerState) IsAutoPlayEnabled() bool {
	return p.isAutoPlayEnabled
}

// LoopMode returns the current loop mode.
func (p *PlayerState) LoopMode() LoopMode {
	return p.loopMode
}

// IsEmpty returns true if the queue has no entries.
func (p *PlayerState) IsEmpty() bool {
	return p.queue.IsEmpty()
}

// IsAtLast returns true if the current track is the last in the queue.
func (p *PlayerState) IsAtLast() bool {
	return p.currentIndex == p.queue.Len()-1
}

// HasNext returns true if there's a next track available (considering auto play and loop mode).
func (p *PlayerState) HasNext(mode LoopMode) bool {
	if p.IsAutoPlayEnabled() {
		return true
	}

	if p.queue.IsEmpty() {
		return false
	}

	switch mode {
	case LoopModeTrack, LoopModeQueue:
		// Always has next when looping (as long as queue isn't empty and not idle)
		return true
	default: // LoopModeNone
		return !p.IsAtLast()
	}
}

// Len returns the number of entries in the queue.
func (p *PlayerState) Len() int {
	return p.queue.Len()
}

// Get returns the entry at the given index without removing it.
func (p *PlayerState) Get(index int) (*QueueEntry, error) {
	return p.queue.Get(index)
}

// List returns a copy of all entries in the queue.
func (p *PlayerState) List() []QueueEntry {
	return p.queue.List()
}

// Played returns entries before the current index.
// When playback is inactive, it returns all entries in the queue.
func (p *PlayerState) Played() []QueueEntry {
	if p.queue.IsEmpty() {
		return []QueueEntry{}
	}

	if !p.IsPlaybackActive() {
		return p.queue.List()
	}

	played := p.queue.entries[:p.currentIndex]
	result := make([]QueueEntry, len(played))
	copy(result, played)
	return result
}

// Current returns the entry at currentIndex, or nil if the queue is empty.
func (p *PlayerState) Current() *QueueEntry {
	if !p.IsPlaybackActive() || p.queue.IsEmpty() {
		return nil
	}
	return &p.queue.entries[p.currentIndex]
}

// Upcoming returns entries after the current index.
// Returns empty slice if no entries or no current entry.
func (p *PlayerState) Upcoming() []QueueEntry {
	if !p.IsPlaybackActive() || p.queue.IsEmpty() {
		return []QueueEntry{}
	}

	upcoming := p.queue.entries[p.currentIndex+1:]
	result := make([]QueueEntry, len(upcoming))
	copy(result, upcoming)
	return result
}

// ManualPlayEntries returns a copy of all entries that were manually requested.
func (p *PlayerState) ManualPlayEntries() []QueueEntry {
	return p.queue.ManualPlayEntries()
}

// AutoPlayEntries returns a copy of all entries that were added by auto-play.
func (p *PlayerState) AutoPlayEntries() []QueueEntry {
	return p.queue.AutoPlayEntries()
}

// Prepend adds entries to the front of the queue.
// If playback is active, adjusts currentIndex to keep pointing at the same track.
func (p *PlayerState) Prepend(entries ...QueueEntry) {
	p.queue.Prepend(entries...)

	if p.isPlaybackActive {
		p.currentIndex += len(entries)
	}
}

// Append adds entries to the end of the queue.
// If playback is idle, it seeks to the first new entry and activates playback.
// Returns the start index of the newly added entries and whether the playback became active.
func (p *PlayerState) Append(entries ...QueueEntry) (startIndex int, becameActive bool) {
	startIndex = p.queue.Len()
	p.queue.Append(entries...)

	if !p.isPlaybackActive {
		p.Seek(startIndex)
		becameActive = true
	}

	return startIndex, becameActive
}

// Insert inserts entries at the given index in the queue.
// If playback is active and the insertion point is at or before the current index,
// adjusts currentIndex to keep pointing at the same track.
func (p *PlayerState) Insert(index int, entries ...QueueEntry) error {
	if err := p.queue.Insert(index, entries...); err != nil {
		return err
	}

	if p.isPlaybackActive && index <= p.currentIndex {
		p.currentIndex += len(entries)
	}

	return nil
}

// Seek sets the currentIndex to the specified index.
// If the player is idle, it activates playback.
// Returns the entry at that index, or nil if index is out of bounds.
// Does not change currentIndex if index is invalid.
func (p *PlayerState) Seek(index int) *QueueEntry {
	if !p.queue.isValidIndex(index) {
		return nil
	}

	p.isPlaybackActive = true
	p.isPaused = false
	p.currentIndex = index

	return &p.queue.entries[index]
}

// Advance moves to the next track based on loop mode.
// Returns the new current entry, or nil if queue ended.
// - LoopModeNone: advance index, return nil if past end
// - LoopModeTrack: don't advance, return same entry
// - LoopModeQueue: advance, wrap to 0 if past end
func (p *PlayerState) Advance(mode LoopMode) *QueueEntry {
	if p.queue.IsEmpty() {
		return nil
	}

	p.isPaused = false

	switch mode {
	case LoopModeTrack:
		// Don't modify currentIndex, return same entry

	case LoopModeQueue:
		// Advance with wrap-around
		if p.IsAtLast() {
			p.currentIndex = 0
		} else {
			p.currentIndex++
		}

	default: // LoopModeNone
		// Advance if current track is not the last in the queue
		if p.IsAtLast() {
			p.isPlaybackActive = false
			return nil
		}
		p.currentIndex++
	}

	return &p.queue.entries[p.currentIndex]
}

// Shuffle randomizes the order of entries in the queue.
// When playback is active, the currently playing track is moved to index 0
// so playback continues uninterrupted.
func (p *PlayerState) Shuffle() {
	current := p.Current() // nil if idle
	var savedEntry QueueEntry
	if current != nil {
		savedEntry = *current // copy value before shuffle mutates the slice
	}
	p.queue.Shuffle()
	if current != nil {
		for i, e := range p.queue.entries {
			if e == savedEntry {
				p.queue.entries[0], p.queue.entries[i] = p.queue.entries[i], p.queue.entries[0]
				break
			}
		}
		p.currentIndex = 0
	}
}

// Remove removes and returns the entry at the given index.
// If removing the current track, advances to the next track first (respecting loop mode).
// Adjusts currentIndex and playback state to maintain consistency.
func (p *PlayerState) Remove(index int) (*QueueEntry, error) {
	if !p.queue.isValidIndex(index) {
		return nil, ErrInvalidIndex
	}

	// If removing the current track, advance first so we know what to play next.
	// LoopModeTrack is treated as LoopModeNone here because the track being
	// looped is being removed, so there is nothing to repeat.
	removingCurrent := p.IsPlaybackActive() && index == p.currentIndex
	if removingCurrent {
		loopmode := p.loopMode
		if loopmode == LoopModeTrack {
			loopmode = LoopModeNone
		}
		next := p.Advance(loopmode)
		if next == nil {
			p.isPlaybackActive = false
		}
	}

	entry, err := p.queue.Remove(index)
	if err != nil {
		return nil, err
	}

	if p.queue.IsEmpty() {
		p.currentIndex = 0
		p.isPlaybackActive = false
	} else if index < p.currentIndex {
		p.currentIndex--
	} else if p.currentIndex >= p.queue.Len() {
		p.currentIndex = p.queue.Len() - 1
	}

	return entry, nil
}

// Clear removes all entries from the queue and resets playback state.
func (p *PlayerState) Clear() {
	p.queue.Clear()
	p.currentIndex = 0
	p.isPlaybackActive = false
}

// ClearExceptCurrent removes all entries except the currently playing track.
// Returns the number of entries removed and an error if not playing.
func (p *PlayerState) ClearExceptCurrent() (int, error) {
	current := p.Current()
	if current == nil {
		return 0, ErrNotPlaying
	}

	count := p.queue.Len() - 1
	savedEntry := *current
	p.queue.Clear()
	p.queue.Append(savedEntry)
	p.currentIndex = 0
	p.isPlaybackActive = true

	return count, nil
}

// Pause transitions the player to the paused state.
// Returns ErrNotPlaying if playback is not active, ErrAlreadyPaused if already paused.
func (p *PlayerState) Pause() error {
	if !p.isPlaybackActive {
		return ErrNotPlaying
	}
	if p.isPaused {
		return ErrAlreadyPaused
	}
	p.isPaused = true
	return nil
}

// Resume transitions the player from the paused state to the playing state.
// Returns ErrNotPlaying if playback is not active, ErrNotPaused if not paused.
func (p *PlayerState) Resume() error {
	if !p.isPlaybackActive {
		return ErrNotPlaying
	}
	if !p.isPaused {
		return ErrNotPaused
	}
	p.isPaused = false
	return nil
}

// Skip advances past the current track.
// If LoopModeTrack is active, it is overridden to LoopModeNone (explicit skip breaks track loop).
// Returns the skipped entry, the next entry (nil if queue ended), and an error if not playing.
func (p *PlayerState) Skip() (skipped *QueueEntry, next *QueueEntry, err error) {
	current := p.Current()
	if current == nil {
		return nil, nil, ErrNotPlaying
	}

	skipped = current

	loopmode := p.loopMode
	if loopmode == LoopModeTrack {
		loopmode = LoopModeNone
	}
	next = p.Advance(loopmode)

	if next == nil {
		p.isPlaybackActive = false
	}

	return skipped, next, nil
}

// SetLoopMode sets the loop mode.
func (p *PlayerState) SetLoopMode(mode LoopMode) {
	p.loopMode = mode
}

// CycleLoopMode cycles through loop modes: None -> Track -> Queue -> None.
// Returns the new loop mode.
func (p *PlayerState) CycleLoopMode() LoopMode {
	switch p.loopMode {
	case LoopModeNone:
		p.loopMode = LoopModeTrack
	case LoopModeTrack:
		p.loopMode = LoopModeQueue
	case LoopModeQueue:
		p.loopMode = LoopModeNone
	}
	return p.loopMode
}

// SetAutoPlayEnabled sets the auto-play enabled state.
func (p *PlayerState) SetAutoPlayEnabled(enabled bool) {
	p.isAutoPlayEnabled = enabled
}

// Domain errors for PlayerStateRepository.
var (
	ErrPlayerStateNotFound = errors.New("player state not found")
)

// PlayerStateRepository defines the interface for storing and retrieving player states.
type PlayerStateRepository interface {
	// FindByID returns the PlayerState for the given player state ID, or error if not exists.
	FindByID(ctx context.Context, id PlayerStateID) (PlayerState, error)

	// Save stores the PlayerState.
	Save(ctx context.Context, state PlayerState) error

	// Delete removes the PlayerState for the given player state ID.
	Delete(ctx context.Context, id PlayerStateID) error
}
