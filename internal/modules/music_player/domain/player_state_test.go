package domain

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

const (
	testGuildID        = snowflake.ID(1)
	testVoiceChannelID = snowflake.ID(100)
	testNotifyChannel  = snowflake.ID(200)
)

func newTestPlayerState() *PlayerState {
	return NewPlayerState(testGuildID, testVoiceChannelID, testNotifyChannel)
}

func TestNewPlayerState(t *testing.T) {
	guildID := snowflake.ID(123456789)
	voiceID := snowflake.ID(111)
	notifyID := snowflake.ID(222)

	state := NewPlayerState(guildID, voiceID, notifyID)

	if state.GetGuildID() != guildID {
		t.Errorf("expected GuildID %d, got %d", guildID, state.GetGuildID())
	}
	if state.GetVoiceChannelID() != voiceID {
		t.Errorf("expected VoiceChannelID %d, got %d", voiceID, state.GetVoiceChannelID())
	}
	if state.GetNotificationChannelID() != notifyID {
		t.Errorf(
			"expected NotificationChannelID %d, got %d",
			notifyID,
			state.GetNotificationChannelID(),
		)
	}
	if state.IsPlaybackActive() {
		t.Error("expected playback to not be active")
	}
	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
	if state.Queue.Len() != 0 {
		t.Error("expected Queue to be empty")
	}
	if state.CurrentTrackID() != nil {
		t.Error("expected CurrentTrack to be nil")
	}
}

func TestPlayerState_SetVoiceChannelID(t *testing.T) {
	state := newTestPlayerState()
	newVoiceID := snowflake.ID(999)

	state.SetVoiceChannelID(newVoiceID)

	if state.GetVoiceChannelID() != newVoiceID {
		t.Errorf("expected VoiceChannelID %d, got %d", newVoiceID, state.GetVoiceChannelID())
	}
}

func TestPlayerState_SetNotificationChannelID(t *testing.T) {
	state := newTestPlayerState()
	newNotifyID := snowflake.ID(888)

	state.SetNotificationChannelID(newNotifyID)

	if state.GetNotificationChannelID() != newNotifyID {
		t.Errorf(
			"expected NotificationChannelID %d, got %d",
			newNotifyID,
			state.GetNotificationChannelID(),
		)
	}
}

func TestPlayerState_SetPaused(t *testing.T) {
	state := newTestPlayerState()

	// Initially not paused
	if state.IsPaused() {
		t.Error("expected not to be paused initially")
	}

	// Set paused to true
	state.SetPaused(true)
	if !state.IsPaused() {
		t.Error("expected to be paused")
	}

	// Set paused to false
	state.SetPaused(false)
	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
}

func TestPlayerState_TogglePaused(t *testing.T) {
	state := newTestPlayerState()

	// Initially not paused
	if state.IsPaused() {
		t.Error("expected not to be paused initially")
	}

	// Toggle to paused
	state.TogglePaused()
	if !state.IsPaused() {
		t.Error("expected to be paused after toggle")
	}

	// Toggle back to not paused
	state.TogglePaused()
	if state.IsPaused() {
		t.Error("expected not to be paused after second toggle")
	}
}

func TestPlayerState_PlaybackActive(t *testing.T) {
	state := newTestPlayerState()

	// Initially not active
	if state.IsPlaybackActive() {
		t.Error("expected playback to not be active initially")
	}

	// Set active
	state.SetPlaybackActive(true)
	if !state.IsPlaybackActive() {
		t.Error("expected playback to be active")
	}

	// Set inactive
	state.SetPlaybackActive(false)
	if state.IsPlaybackActive() {
		t.Error("expected playback to not be active")
	}
}

func TestPlayerState_CurrentTrack(t *testing.T) {
	state := newTestPlayerState()

	// No current track initially
	if state.CurrentTrackID() != nil {
		t.Error("expected no current track initially")
	}

	// Add a track and activate playback
	trackID := TrackID("track-1")
	state.Queue.Append(trackID)
	state.SetPlaybackActive(true)

	// Now should have a current track
	current := state.CurrentTrackID()
	if current == nil {
		t.Fatal("expected current track after adding")
	}
	if *current != trackID {
		t.Errorf("expected track ID %s, got %s", trackID, *current)
	}
}

func TestPlayerState_LoopMode(t *testing.T) {
	state := newTestPlayerState()

	// Default loop mode is None
	if got := state.GetLoopMode(); got != LoopModeNone {
		t.Errorf("expected LoopModeNone, got %v", got)
	}

	// Set loop mode
	state.SetLoopMode(LoopModeTrack)
	if got := state.GetLoopMode(); got != LoopModeTrack {
		t.Errorf("expected LoopModeTrack, got %v", got)
	}

	state.SetLoopMode(LoopModeQueue)
	if got := state.GetLoopMode(); got != LoopModeQueue {
		t.Errorf("expected LoopModeQueue, got %v", got)
	}
}

func TestPlayerState_CycleLoopMode(t *testing.T) {
	state := newTestPlayerState()

	// None -> Track
	got := state.CycleLoopMode()
	if got != LoopModeTrack {
		t.Errorf("expected LoopModeTrack, got %v", got)
	}
	if state.GetLoopMode() != LoopModeTrack {
		t.Error("state loop mode should be updated")
	}

	// Track -> Queue
	got = state.CycleLoopMode()
	if got != LoopModeQueue {
		t.Errorf("expected LoopModeQueue, got %v", got)
	}

	// Queue -> None
	got = state.CycleLoopMode()
	if got != LoopModeNone {
		t.Errorf("expected LoopModeNone, got %v", got)
	}
}

func TestPlayerState_NowPlayingMessage(t *testing.T) {
	state := newTestPlayerState()

	// Initially nil
	if state.GetNowPlayingMessage() != nil {
		t.Error("expected nil NowPlayingMessage initially")
	}

	// Set message
	channelID := snowflake.ID(123)
	messageID := snowflake.ID(456)
	state.SetNowPlayingMessage(channelID, messageID)

	msg := state.GetNowPlayingMessage()
	if msg == nil {
		t.Fatal("expected NowPlayingMessage to be set")
	}
	if msg.ChannelID != channelID {
		t.Errorf("expected ChannelID %d, got %d", channelID, msg.ChannelID)
	}
	if msg.MessageID != messageID {
		t.Errorf("expected MessageID %d, got %d", messageID, msg.MessageID)
	}

	// Clear message
	state.ClearNowPlayingMessage()
	if state.GetNowPlayingMessage() != nil {
		t.Error("expected nil NowPlayingMessage after clear")
	}
}

func TestPlayerState_NowPlayingMessage_ReturnsCopy(t *testing.T) {
	state := newTestPlayerState()
	channelID := snowflake.ID(123)
	messageID := snowflake.ID(456)
	state.SetNowPlayingMessage(channelID, messageID)

	// Get and modify the returned message
	msg1 := state.GetNowPlayingMessage()
	msg1.ChannelID = snowflake.ID(999)

	// Original should be unchanged
	msg2 := state.GetNowPlayingMessage()
	if msg2.ChannelID != channelID {
		t.Error("GetNowPlayingMessage should return a copy")
	}
}
