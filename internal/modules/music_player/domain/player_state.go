package domain

import (
	"context"
	"errors"

	"github.com/disgoorg/snowflake/v2"
)

// PlayerState represents the state of a music player for a guild.
type PlayerState struct {
	guildID               snowflake.ID       // Guild this player state belongs to
	voiceChannelID        snowflake.ID       // Voice channel the bot is connected to
	notificationChannelID snowflake.ID       // Text channel for notifications
	nowPlayingMessage     *NowPlayingMessage // "Now Playing" message info (for deletion)
	queue                 Queue              // Queue associated with this player state
	currentIndex          int                // Index of the currently playing track in the queue
	isPlaybackActive      bool               // true when playback is active
	isPaused              bool               // true when playback is paused
	loopMode              LoopMode           // loop mode for playback
	autoPlayEnabled       bool               // true when auto-play is enabled
}

// NewPlayerState creates a new PlayerState for the given guild and channels.
func NewPlayerState(
	guildID snowflake.ID,
	queue Queue,
) *PlayerState {
	return &PlayerState{
		guildID:         guildID,
		queue:           queue,
		autoPlayEnabled: true,
	}
}

// IsPaused returns true if playback is paused.
func (p *PlayerState) IsPaused() bool {
	return p.isPaused
}

// GetGuildID returns the guild ID.
func (p *PlayerState) GetGuildID() snowflake.ID {
	// No read mutex: guildID must not be modified after initialization
	return p.guildID
}

// No SetGuildID method: guildID must not be modified after initialization

// GetVoiceChannelID returns the current voice channel ID.
func (p *PlayerState) GetVoiceChannelID() snowflake.ID {
	return p.voiceChannelID
}

// SetVoiceChannelID updates the voice channel ID.
func (p *PlayerState) SetVoiceChannelID(channelID snowflake.ID) {
	p.voiceChannelID = channelID
}

// GetNotificationChannelID returns the current voice channel ID.
func (p *PlayerState) GetNotificationChannelID() snowflake.ID {
	return p.notificationChannelID
}

// SetNotificationChannelID updates the notification channel ID.
func (p *PlayerState) SetNotificationChannelID(channelID snowflake.ID) {
	p.notificationChannelID = channelID
}

// GetNowPlayingMessage returns a copy of the "Now Playing" message info.
func (p *PlayerState) GetNowPlayingMessage() *NowPlayingMessage {
	if p.nowPlayingMessage == nil {
		return nil
	}
	return &NowPlayingMessage{
		ChannelID: p.nowPlayingMessage.ChannelID,
		MessageID: p.nowPlayingMessage.MessageID,
	}
}

// SetNowPlayingMessage stores the "Now Playing" message info for later deletion.
func (p *PlayerState) SetNowPlayingMessage(nowPlayingMessage *NowPlayingMessage) {
	p.nowPlayingMessage = nowPlayingMessage
}

// CurrentIndex returns the current track index.
func (p *PlayerState) CurrentIndex() int {
	return p.currentIndex
}

func (p *PlayerState) IsPlaybackActive() bool {
	return p.isPlaybackActive
}

func (p *PlayerState) SetPlaybackActive(isPlaybackActive bool) {
	p.isPlaybackActive = isPlaybackActive
}

// IsAtLast returns true if the current track is the last in the queue.
func (p *PlayerState) IsAtLast() bool {
	return p.currentIndex == p.queue.Len()-1
}

// HasNext returns true if there's a next track available (considering loop mode).
func (p *PlayerState) HasNext(mode LoopMode) bool {
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

// Played returns entries before the current index.
// Returns empty slice if no entries or at the first track.
func (p *PlayerState) Played() []QueueEntry {
	if p.queue.IsEmpty() {
		return []QueueEntry{}
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

// Seek sets the currentIndex to the specified index.
// Returns the entry at that index, or nil if index is out of bounds.
// Does not change currentIndex if index is invalid.
func (p *PlayerState) Seek(index int) *QueueEntry {
	if !p.queue.isValidIndex(index) {
		return nil
	}

	p.currentIndex = index
	return &p.queue.entries[index]
}

// Advance moves to the next track based on loop mode.
// Returns the new current entry, or nil if queue ended.
//   - LoopModeNone: advance index, return nil if past end
//   - LoopModeTrack: don't advance, return same entry
//   - LoopModeQueue: advance, wrap to 0 if past end
func (p *PlayerState) Advance(mode LoopMode) *QueueEntry {
	if p.queue.IsEmpty() {
		return nil
	}

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
			return nil
		}
		p.currentIndex++
	}

	return &p.queue.entries[p.currentIndex]
}

// Len returns the number of entries in the queue.
func (p *PlayerState) Len() int {
	return p.queue.Len()
}

// IsEmpty returns true if the queue has no entries.
func (p *PlayerState) IsEmpty() bool {
	return p.queue.IsEmpty()
}

// List returns a copy of all entries in the queue.
func (p *PlayerState) List() []QueueEntry {
	return p.queue.List()
}

// Get returns the entry at the given index without removing it.
func (p *PlayerState) Get(index int) (*QueueEntry, error) {
	return p.queue.Get(index)
}

// Append adds entries to the end of the queue.
func (p *PlayerState) Append(entries ...QueueEntry) {
	p.queue.Append(entries...)
}

// Prepend adds entries to the front of the queue.
// If playback is active, adjusts currentIndex to keep pointing at the same track.
func (p *PlayerState) Prepend(entries ...QueueEntry) {
	p.queue.Prepend(entries...)
	if p.isPlaybackActive {
		p.currentIndex += len(entries)
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
	if p.IsPlaybackActive() && index == p.currentIndex {
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

// Clear removes all entries from the queue and resets playback state.
func (p *PlayerState) Clear() {
	p.queue.Clear()
	p.currentIndex = 0
	p.isPlaybackActive = false
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

// Enqueue appends entries to the queue. If the player is idle, it seeks to the first
// new entry and activates playback.
// Returns the start index of the newly added entries and whether the player became active.
func (p *PlayerState) Enqueue(entries ...QueueEntry) (startIndex int, becameActive bool) {
	startIndex = p.queue.Len()
	p.queue.Append(entries...)

	if !p.isPlaybackActive {
		p.Seek(startIndex)
		p.isPlaybackActive = true
		becameActive = true
	}

	return startIndex, becameActive
}

// HandleTrackEnded processes the end of a track.
// If failed is true, the current track is removed from the queue.
// Otherwise, the queue advances according to the current loop mode.
// Returns the next entry to play, or nil if the queue has ended.
func (p *PlayerState) HandleTrackEnded(failed bool) *QueueEntry {
	if failed {
		if _, err := p.Remove(p.currentIndex); err != nil {
			return nil
		}
		return p.Current()
	}

	next := p.Advance(p.loopMode)
	if next == nil {
		p.isPlaybackActive = false
	}
	return next
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

// SetPaused sets the paused state.
func (p *PlayerState) SetPaused(isPaused bool) {
	p.isPaused = isPaused
}

// GetLoopMode returns the current loop mode.
func (p *PlayerState) GetLoopMode() LoopMode {
	return p.loopMode
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

// IsAutoPlayEnabled returns true if auto-play is enabled.
func (p *PlayerState) IsAutoPlayEnabled() bool {
	return p.autoPlayEnabled
}

// SetAutoPlayEnabled sets the auto-play enabled state.
func (p *PlayerState) SetAutoPlayEnabled(enabled bool) {
	p.autoPlayEnabled = enabled
}

// Errors for PlayerStateRepository
var ErrPlayerStateNotFound = errors.New("player state not found")

// PlayerStateRepository defines the interface for storing and retrieving player states.
type PlayerStateRepository interface {
	// Get returns the PlayerState for the given guild, or error if not exists.
	Get(ctx context.Context, guildID snowflake.ID) (PlayerState, error)

	// Save stores the PlayerState.
	Save(ctx context.Context, state PlayerState) error

	// Delete removes the PlayerState for the given guild.
	Delete(ctx context.Context, guildID snowflake.ID) error
}
