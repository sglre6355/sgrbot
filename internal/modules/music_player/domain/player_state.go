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
}

// NewPlayerState creates a new PlayerState for the given guild and channels.
func NewPlayerState(
	guildID snowflake.ID,
	queue Queue,
) *PlayerState {
	return &PlayerState{
		guildID: guildID,
		queue:   queue,
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

// Clear removes all entries from the queue and resets playback state.
func (p *PlayerState) Clear() {
	p.queue.Clear()
	p.currentIndex = 0
	p.isPlaybackActive = false
}

// SetPaused sets the paused state to true.
func (p *PlayerState) SetPaused(isPaused bool) {
	p.isPaused = isPaused
}

func (p *PlayerState) TogglePaused() {
	p.isPaused = !p.isPaused
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
