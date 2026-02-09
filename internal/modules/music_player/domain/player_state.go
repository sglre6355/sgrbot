package domain

import (
	"context"

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
	guildID               snowflake.ID
	voiceChannelID        snowflake.ID       // Voice channel the bot is connected to
	notificationChannelID snowflake.ID       // Text channel for notifications
	nowPlayingMessage     *NowPlayingMessage // "Now Playing" message info (for deletion)
	Queue                 Queue              // Queue with index-based track management
	isPlaybackActive      bool               // true when playback is active
	isPaused              bool               // true when playback is paused
	loopMode              LoopMode           // loop mode for playback
}

// NewPlayerState creates a new PlayerState for the given guild and channels.
func NewPlayerState(guildID, voiceChannelID, notificationChannelID snowflake.ID) *PlayerState {
	return &PlayerState{
		guildID:               guildID,
		voiceChannelID:        voiceChannelID,
		notificationChannelID: notificationChannelID,
		loopMode:              LoopModeNone,
		Queue:                 NewQueue(),
	}
}

// IsPlaybackActive returns true if playback is currently active.
func (p *PlayerState) IsPlaybackActive() bool {
	return p.isPlaybackActive
}

// SetPlaybackActive sets whether playback is active.
func (p *PlayerState) SetPlaybackActive(active bool) {
	p.isPlaybackActive = active
}

// IsPaused returns true if playback is paused.
func (p *PlayerState) IsPaused() bool {
	return p.isPaused
}

// CurrentTrackID returns the currently playing track ID, or nil if playback is not active.
func (p *PlayerState) CurrentTrackID() *TrackID {
	if !p.isPlaybackActive {
		return nil
	}
	return p.Queue.Current()
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
func (p *PlayerState) SetNowPlayingMessage(channelID, messageID snowflake.ID) {
	p.nowPlayingMessage = &NowPlayingMessage{
		ChannelID: channelID,
		MessageID: messageID,
	}
}

// ClearNowPlayingMessage clears the stored "Now Playing" message info.
func (p *PlayerState) ClearNowPlayingMessage() {
	p.nowPlayingMessage = nil
}

// PlayerStateRepository defines the interface for storing and retrieving player states.
type PlayerStateRepository interface {
	// Get returns the PlayerState for the given guild, or error if not exists.
	Get(ctx context.Context, guildID snowflake.ID) (PlayerState, error)

	// Save stores the PlayerState.
	Save(ctx context.Context, state PlayerState) error

	// Delete removes the PlayerState for the given guild.
	Delete(ctx context.Context, guildID snowflake.ID) error
}
