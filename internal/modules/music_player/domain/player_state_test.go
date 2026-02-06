package domain

import (
	"sync"
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
	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
	if state.Queue == nil {
		t.Error("expected Queue to be initialized")
	}
	if state.CurrentTrack() != nil {
		t.Error("expected CurrentTrack to be nil")
	}
}

func TestPlayerState_StatusMethods(t *testing.T) {
	state := newTestPlayerState()

	// Initial state is idle (playbackActive=false)
	if !state.IsIdle() {
		t.Error("new state should be idle")
	}
	if state.IsPaused() {
		t.Error("new state should not be paused")
	}

	// Set playing (playbackActive=true, paused=false)
	state.SetPlaying(&Track{ID: "track-1"})
	if state.IsIdle() {
		t.Error("playing state should not be idle")
	}
	if state.IsPaused() {
		t.Error("playing state should not be paused")
	}

	// Set paused (playbackActive=true, paused=true)
	state.SetPaused()
	if state.IsIdle() {
		t.Error("paused state should not be idle")
	}
	if !state.IsPaused() {
		t.Error("paused state should be paused")
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

func TestPlayerState_SetPlaying(t *testing.T) {
	state := newTestPlayerState()
	track := &Track{ID: "track-1", Title: "Test Song"}

	state.SetPlaying(track)

	if state.CurrentTrack() != track {
		t.Error("expected CurrentTrack to be set")
	}
	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
	if state.IsIdle() {
		t.Error("expected playback to be active")
	}
}

func TestPlayerState_SetPaused(t *testing.T) {
	state := newTestPlayerState()

	// Pausing with no track should not change state
	state.SetPaused()
	if state.IsPaused() {
		t.Error("should not pause without current track")
	}

	// Set a track and pause
	state.SetPlaying(&Track{ID: "track-1"})
	state.SetPaused()

	if !state.IsPaused() {
		t.Error("expected to be paused")
	}
}

func TestPlayerState_SetResumed(t *testing.T) {
	state := newTestPlayerState()

	// Resume with no track should not change state
	state.SetResumed()
	if state.IsPaused() {
		t.Error("should not change paused state without current track")
	}

	// Set a track, pause, then resume
	state.SetPlaying(&Track{ID: "track-1"})
	state.SetPaused()
	state.SetResumed()

	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
	if state.IsIdle() {
		t.Error("expected playback to be active")
	}
}

func TestPlayerState_SetStopped(t *testing.T) {
	state := newTestPlayerState()
	state.SetPlaying(&Track{ID: "track-1"})
	state.SetPaused()

	state.SetStopped()

	if state.CurrentTrack() != nil {
		t.Error("expected CurrentTrack to be nil")
	}
	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
	if !state.IsIdle() {
		t.Error("expected IsIdle to return true")
	}
}

func TestPlayerState_StopPlayback_ResetsLoopMode(t *testing.T) {
	state := newTestPlayerState()
	state.SetPlaying(&Track{ID: "track-1"})
	state.SetLoopMode(LoopModeQueue)

	if state.LoopMode() != LoopModeQueue {
		t.Error("expected loop mode to be queue")
	}

	state.StopPlayback()

	if state.LoopMode() != LoopModeNone {
		t.Error("expected loop mode to be reset to none after StopPlayback")
	}
	if !state.IsIdle() {
		t.Error("expected IsIdle to return true")
	}
}

func TestPlayerState_HasTrack(t *testing.T) {
	state := newTestPlayerState()

	if state.HasTrack() {
		t.Error("new state should not have track")
	}

	state.SetPlaying(&Track{ID: "track-1"})
	if !state.HasTrack() {
		t.Error("state with track should have track")
	}
}

func TestPlayerState_HasQueuedTracks(t *testing.T) {
	state := newTestPlayerState()

	if state.HasQueuedTracks() {
		t.Error("new state should not have queued tracks")
	}

	// Just a current track, no queue
	state.SetPlaying(&Track{ID: "current"})
	if state.HasQueuedTracks() {
		t.Error("state with only current track should not have queued tracks")
	}

	// Add a track to queue (after current)
	state.Queue.Add(&Track{ID: "queued"})
	if !state.HasQueuedTracks() {
		t.Error("state with queued track should have queued tracks")
	}
}

func TestPlayerState_TotalTracks(t *testing.T) {
	state := newTestPlayerState()

	if got := state.TotalTracks(); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}

	// Add current track
	state.SetPlaying(&Track{ID: "current"})
	if got := state.TotalTracks(); got != 1 {
		t.Errorf("expected 1, got %d", got)
	}

	// Add queued tracks
	state.Queue.Add(&Track{ID: "queued-1"})
	state.Queue.Add(&Track{ID: "queued-2"})
	if got := state.TotalTracks(); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}

	// Stop advances to next track but keeps all tracks (index-based model)
	state.SetStopped()
	if got := state.TotalTracks(); got != 3 {
		t.Errorf("expected 3 (tracks kept in queue), got %d", got)
	}
}

func TestPlayerState_LoopMode(t *testing.T) {
	state := newTestPlayerState()

	// Default loop mode is None
	if got := state.LoopMode(); got != LoopModeNone {
		t.Errorf("expected LoopModeNone, got %v", got)
	}

	// Set loop mode
	state.SetLoopMode(LoopModeTrack)
	if got := state.LoopMode(); got != LoopModeTrack {
		t.Errorf("expected LoopModeTrack, got %v", got)
	}

	state.SetLoopMode(LoopModeQueue)
	if got := state.LoopMode(); got != LoopModeQueue {
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
	if state.LoopMode() != LoopModeTrack {
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

func TestPlayerState_SetStopped_WithLoopModes(t *testing.T) {
	t.Run("LoopModeNone advances to next track", func(t *testing.T) {
		state := newTestPlayerState()
		track1 := &Track{ID: "track-1"}
		track2 := &Track{ID: "track-2"}

		state.Queue.Add(track1)
		state.Queue.Add(track2)
		state.Queue.Start()

		state.SetStopped()

		if state.CurrentTrack() != track2 {
			t.Error("expected current track to be track2")
		}
	})

	t.Run("LoopModeTrack replays same track", func(t *testing.T) {
		state := newTestPlayerState()
		track1 := &Track{ID: "track-1"}
		track2 := &Track{ID: "track-2"}

		state.Queue.Add(track1)
		state.Queue.Add(track2)
		state.Queue.Start()
		state.SetLoopMode(LoopModeTrack)

		state.SetStopped()

		if state.CurrentTrack() != track1 {
			t.Error("expected current track to still be track1")
		}
	})

	t.Run("LoopModeQueue wraps to start", func(t *testing.T) {
		state := newTestPlayerState()
		track1 := &Track{ID: "track-1"}
		track2 := &Track{ID: "track-2"}

		state.Queue.Add(track1)
		state.Queue.Add(track2)
		state.Queue.Start()
		state.SetLoopMode(LoopModeQueue)

		// Advance to track2
		state.SetStopped()
		if state.CurrentTrack() != track2 {
			t.Error("expected current track to be track2")
		}

		// Should wrap to track1
		state.SetStopped()
		if state.CurrentTrack() != track1 {
			t.Error("expected current track to wrap to track1")
		}
	})
}

func TestPlayerState_ConcurrentAccess(t *testing.T) {
	state := newTestPlayerState()
	state.SetPlaying(&Track{ID: "track-1"})

	var wg sync.WaitGroup
	const numGoroutines = 100

	// Run concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			_ = state.IsIdle()
		}()

		go func() {
			defer wg.Done()
			_ = state.IsPaused()
		}()

		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				state.SetPaused()
			} else {
				state.SetResumed()
			}
		}(i)
	}

	wg.Wait()
}
