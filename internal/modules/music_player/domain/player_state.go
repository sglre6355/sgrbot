package domain

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
)

// NowPlayingMessage stores the channel and message ID for a "Now Playing" message.
// Both values are needed for deletion since the message may be in a different channel
// than the current notification channel if the user switched channels while playing.
type NowPlayingMessage struct {
	ChannelID snowflake.ID
	MessageID snowflake.ID
}

// PlayerState represents the state of a music player for a guild.
type PlayerState struct {
	mu                    sync.RWMutex
	GuildID               snowflake.ID
	VoiceChannelID        snowflake.ID       // Voice channel the bot is connected to
	NotificationChannelID snowflake.ID       // Text channel for notifications
	paused                bool               // unexported to prevent direct access
	loopMode              LoopMode           // loop mode for playback
	Queue                 *Queue             // Queue with index-based track management
	nowPlayingMessage     *NowPlayingMessage // "Now Playing" message info (for deletion)
}

// NewPlayerState creates a new PlayerState for the given guild and channels.
func NewPlayerState(guildID, voiceChannelID, notificationChannelID snowflake.ID) *PlayerState {
	return &PlayerState{
		GuildID:               guildID,
		VoiceChannelID:        voiceChannelID,
		NotificationChannelID: notificationChannelID,
		loopMode:              LoopModeNone,
		Queue:                 NewQueue(),
	}
}

// IsIdle returns true if no track is currently active (queue not started or past end).
func (p *PlayerState) IsIdle() bool {
	return p.Queue.IsIdle()
}

// IsPlaying returns true if a track is currently playing (not paused).
func (p *PlayerState) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.Queue.IsIdle() && !p.paused
}

// IsPaused returns true if playback is paused.
func (p *PlayerState) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.Queue.IsIdle() && p.paused
}

// CurrentTrack returns the currently playing track.
func (p *PlayerState) CurrentTrack() *Track {
	return p.Queue.Current()
}

// SetVoiceChannel updates the voice channel ID.
func (p *PlayerState) SetVoiceChannel(channelID snowflake.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.VoiceChannelID = channelID
}

// GetVoiceChannelID returns the current voice channel ID.
func (p *PlayerState) GetVoiceChannelID() snowflake.ID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.VoiceChannelID
}

// SetNotificationChannel updates the notification channel ID.
func (p *PlayerState) SetNotificationChannel(channelID snowflake.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.NotificationChannelID = channelID
}

// SetPlaying sets the current track (prepends to queue) and clears the paused state.
func (p *PlayerState) SetPlaying(track *Track) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Queue.Prepend(track)
	p.paused = false
}

// SetPaused sets the paused state to true.
func (p *PlayerState) SetPaused() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.Queue.IsIdle() {
		p.paused = true
	}
}

// SetResumed clears the paused state.
func (p *PlayerState) SetResumed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.Queue.IsIdle() {
		p.paused = false
	}
}

// SetStopped advances the queue based on the current loop mode and clears the paused state.
func (p *PlayerState) SetStopped() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Queue.Advance(p.loopMode)
	p.paused = false
}

// LoopMode returns the current loop mode.
func (p *PlayerState) LoopMode() LoopMode {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loopMode
}

// SetLoopMode sets the loop mode.
func (p *PlayerState) SetLoopMode(mode LoopMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loopMode = mode
}

// CycleLoopMode cycles through loop modes: None -> Track -> Queue -> None.
// Returns the new loop mode.
func (p *PlayerState) CycleLoopMode() LoopMode {
	p.mu.Lock()
	defer p.mu.Unlock()

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

// SetNowPlayingMessage stores the "Now Playing" message info for later deletion.
func (p *PlayerState) SetNowPlayingMessage(channelID, messageID snowflake.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nowPlayingMessage = &NowPlayingMessage{
		ChannelID: channelID,
		MessageID: messageID,
	}
}

// ClearNowPlayingMessage clears the stored "Now Playing" message info.
func (p *PlayerState) ClearNowPlayingMessage() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nowPlayingMessage = nil
}

// GetNowPlayingMessage returns a copy of the "Now Playing" message info.
func (p *PlayerState) GetNowPlayingMessage() *NowPlayingMessage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.nowPlayingMessage == nil {
		return nil
	}
	return &NowPlayingMessage{
		ChannelID: p.nowPlayingMessage.ChannelID,
		MessageID: p.nowPlayingMessage.MessageID,
	}
}

// HasTrack returns true if there is a current track.
func (p *PlayerState) HasTrack() bool {
	return !p.Queue.IsIdle()
}

// HasQueuedTracks returns true if there are tracks after the current one.
func (p *PlayerState) HasQueuedTracks() bool {
	return len(p.Queue.Upcoming()) > 0
}

// TotalTracks returns the total number of tracks in the queue.
func (p *PlayerState) TotalTracks() int {
	return p.Queue.Len()
}
