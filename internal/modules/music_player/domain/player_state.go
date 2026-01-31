package domain

import (
	"sync"

	"github.com/disgoorg/snowflake/v2"
)

// PlayerState represents the state of a music player for a guild.
type PlayerState struct {
	mu                    sync.RWMutex
	GuildID               snowflake.ID
	VoiceChannelID        snowflake.ID  // Voice channel the bot is connected to
	NotificationChannelID snowflake.ID  // Text channel for notifications
	paused                bool          // unexported to prevent direct access
	Queue                 *Queue        // Queue[0] is the current track
	NowPlayingMessageID   *snowflake.ID // Discord message ID for "Now Playing" message (for deletion)
}

// NewPlayerState creates a new PlayerState for the given guild and channels.
func NewPlayerState(guildID, voiceChannelID, notificationChannelID snowflake.ID) *PlayerState {
	return &PlayerState{
		GuildID:               guildID,
		VoiceChannelID:        voiceChannelID,
		NotificationChannelID: notificationChannelID,
		Queue:                 NewQueue(),
	}
}

// IsIdle returns true if no track is playing.
func (p *PlayerState) IsIdle() bool {
	return p.Queue.IsEmpty()
}

// IsPlaying returns true if a track is currently playing (not paused).
func (p *PlayerState) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.Queue.IsEmpty() && !p.paused
}

// IsPaused returns true if playback is paused.
func (p *PlayerState) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.Queue.IsEmpty() && p.paused
}

// CurrentTrack returns the currently playing track (head of queue).
func (p *PlayerState) CurrentTrack() *Track {
	return p.Queue.Peek()
}

// SetVoiceChannel updates the voice channel ID.
func (p *PlayerState) SetVoiceChannel(channelID snowflake.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.VoiceChannelID = channelID
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
	if !p.Queue.IsEmpty() {
		p.paused = true
	}
}

// SetResumed clears the paused state.
func (p *PlayerState) SetResumed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.Queue.IsEmpty() {
		p.paused = false
	}
}

// SetStopped removes the current track and clears the paused state.
func (p *PlayerState) SetStopped() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Queue.Next()
	p.paused = false
}

// SetNowPlayingMessageID stores the Discord message ID for the "Now Playing" message.
func (p *PlayerState) SetNowPlayingMessageID(messageID snowflake.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.NowPlayingMessageID = &messageID
}

// ClearNowPlayingMessageID clears the stored "Now Playing" message ID.
func (p *PlayerState) ClearNowPlayingMessageID() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.NowPlayingMessageID = nil
}

// GetNowPlayingMessageID returns a copy of the message ID pointer.
func (p *PlayerState) GetNowPlayingMessageID() *snowflake.ID {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.NowPlayingMessageID == nil {
		return nil
	}
	id := *p.NowPlayingMessageID
	return &id
}

// HasTrack returns true if there is a current track.
func (p *PlayerState) HasTrack() bool {
	return !p.Queue.IsEmpty()
}

// HasQueuedTracks returns true if there are tracks after the current one.
func (p *PlayerState) HasQueuedTracks() bool {
	return p.Queue.Len() > 1
}

// TotalTracks returns the total number of tracks (current + queued).
func (p *PlayerState) TotalTracks() int {
	return p.Queue.Len()
}
