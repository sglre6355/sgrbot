package domain

import (
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

const testGuildID = snowflake.ID(1)

func newTestPlayerState() *PlayerState {
	return NewPlayerState(testGuildID, NewQueue())
}

func testEntry(id TrackID) QueueEntry {
	return QueueEntry{TrackID: id}
}

func TestNewPlayerState(t *testing.T) {
	guildID := snowflake.ID(123456789)
	queue := NewQueue()

	state := NewPlayerState(guildID, queue)

	if state.GetGuildID() != guildID {
		t.Errorf("expected GuildID %d, got %d", guildID, state.GetGuildID())
	}
	if state.IsPaused() {
		t.Error("expected not to be paused")
	}
	if state.Len() != 0 {
		t.Error("expected Queue to be empty")
	}
	if state.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
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
	nowPlayingMessage := NewNowPlayingMessage(channelID, messageID)
	state.SetNowPlayingMessage(&nowPlayingMessage)

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
	state.SetNowPlayingMessage(nil)
	if state.GetNowPlayingMessage() != nil {
		t.Error("expected nil NowPlayingMessage after clear")
	}
}

func TestPlayerState_NowPlayingMessage_ReturnsCopy(t *testing.T) {
	state := newTestPlayerState()
	channelID := snowflake.ID(123)
	messageID := snowflake.ID(456)
	nowPlayingMessage := NewNowPlayingMessage(channelID, messageID)
	state.SetNowPlayingMessage(&nowPlayingMessage)

	// Get and modify the returned message
	msg1 := state.GetNowPlayingMessage()
	msg1.ChannelID = snowflake.ID(999)

	// Original should be unchanged
	msg2 := state.GetNowPlayingMessage()
	if msg2.ChannelID != channelID {
		t.Error("GetNowPlayingMessage should return a copy")
	}
}

func TestPlayerState_Current(t *testing.T) {
	state := newTestPlayerState()
	trackID := TrackID("track-1")

	// Current on empty queue returns nil
	if got := state.Current(); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	state.Append(testEntry(trackID))
	state.SetPlaybackActive(true)

	// After Append, Current returns the first track
	got := state.Current()
	if got == nil {
		t.Fatal("expected track after Append")
	}
	if got.TrackID != trackID {
		t.Errorf("expected %s, got %s", trackID, got.TrackID)
	}
}

func TestPlayerState_Advance_LoopModeNone(t *testing.T) {
	state := newTestPlayerState()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// Advance on empty queue returns nil
	if got := state.Advance(LoopModeNone); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	state.Append(testEntry(trackID1), testEntry(trackID2))

	// Advance should return next track
	got := state.Advance(LoopModeNone)
	if got == nil || got.TrackID != trackID2 {
		t.Errorf("expected track2, got %v", got)
	}
	if state.CurrentIndex() != 1 {
		t.Errorf("expected currentIndex 1, got %d", state.CurrentIndex())
	}

	// Advance past end should return nil
	got = state.Advance(LoopModeNone)
	if got != nil {
		t.Errorf("expected nil past end, got %v", got)
	}

	// Tracks should still be in queue
	if state.Len() != 2 {
		t.Errorf("expected 2 tracks still in queue, got %d", state.Len())
	}
}

func TestPlayerState_Advance_LoopModeTrack(t *testing.T) {
	state := newTestPlayerState()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	state.Append(testEntry(trackID1), testEntry(trackID2))

	// Advance with LoopModeTrack should return same track
	got := state.Advance(LoopModeTrack)
	if got == nil || got.TrackID != trackID1 {
		t.Errorf("expected track1, got %v", got)
	}
	if state.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
	}

	// Multiple advances should keep returning same track
	for i := range 5 {
		got = state.Advance(LoopModeTrack)
		if got == nil || got.TrackID != trackID1 {
			t.Errorf("iteration %d: expected track1, got %v", i, got)
		}
	}
}

func TestPlayerState_Advance_LoopModeQueue(t *testing.T) {
	state := newTestPlayerState()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	state.Append(testEntry(trackID1), testEntry(trackID2), testEntry(trackID3))

	// Advance through all tracks
	if got := state.Advance(LoopModeQueue); got == nil || got.TrackID != trackID2 {
		t.Errorf("expected track2, got %v", got)
	}
	if got := state.Advance(LoopModeQueue); got == nil || got.TrackID != trackID3 {
		t.Errorf("expected track3, got %v", got)
	}

	// Should wrap to beginning
	got := state.Advance(LoopModeQueue)
	if got == nil || got.TrackID != trackID1 {
		t.Errorf("expected track1 (wrap), got %v", got)
	}
	if state.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after wrap, got %d", state.CurrentIndex())
	}
}

func TestPlayerState_Advance_LoopModeQueue_SingleTrack(t *testing.T) {
	state := newTestPlayerState()
	trackID := TrackID("track-1")

	state.Append(testEntry(trackID))

	// Single track with LoopModeQueue should keep returning same track
	for i := range 5 {
		got := state.Advance(LoopModeQueue)
		if got == nil || got.TrackID != trackID {
			t.Errorf("iteration %d: expected track, got %v", i, got)
		}
	}
}

func TestPlayerState_HasNext(t *testing.T) {
	state := newTestPlayerState()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// HasNext on empty queue returns false
	if state.HasNext(LoopModeNone) {
		t.Error("expected HasNext=false for empty queue")
	}

	state.Append(testEntry(trackID1), testEntry(trackID2))

	// HasNext with LoopModeNone should be true when there are more tracks
	if !state.HasNext(LoopModeNone) {
		t.Error("expected HasNext=true with more tracks")
	}

	// Advance to last track
	state.Advance(LoopModeNone)

	// HasNext should be false at last track with LoopModeNone
	if state.HasNext(LoopModeNone) {
		t.Error("expected HasNext=false at last track with LoopModeNone")
	}

	// HasNext should be true with LoopModeTrack
	if !state.HasNext(LoopModeTrack) {
		t.Error("expected HasNext=true with LoopModeTrack")
	}

	// HasNext should be true with LoopModeQueue
	if !state.HasNext(LoopModeQueue) {
		t.Error("expected HasNext=true with LoopModeQueue")
	}
}

func TestPlayerState_Played(t *testing.T) {
	state := newTestPlayerState()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	// Played on empty queue returns empty slice
	if played := state.Played(); len(played) != 0 {
		t.Errorf("expected empty played, got %d", len(played))
	}

	state.Append(testEntry(trackID1), testEntry(trackID2), testEntry(trackID3))

	// At first track, no played tracks
	if played := state.Played(); len(played) != 0 {
		t.Errorf("expected empty played at first track, got %d", len(played))
	}

	// Advance and check played
	state.Advance(LoopModeNone)
	played := state.Played()
	if len(played) != 1 {
		t.Errorf("expected 1 played, got %d", len(played))
	}
	if played[0].TrackID != trackID1 {
		t.Error("expected track1 as played")
	}

	// Advance again
	state.Advance(LoopModeNone)
	played = state.Played()
	if len(played) != 2 {
		t.Errorf("expected 2 played, got %d", len(played))
	}
	if played[0].TrackID != trackID1 || played[1].TrackID != trackID2 {
		t.Error("unexpected played track order")
	}
}

func TestPlayerState_Upcoming(t *testing.T) {
	state := newTestPlayerState()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	state.Append(testEntry(trackID1), testEntry(trackID2), testEntry(trackID3))
	state.SetPlaybackActive(true)

	// Upcoming should return tracks after current
	upcoming := state.Upcoming()
	if len(upcoming) != 2 {
		t.Errorf("expected 2 upcoming, got %d", len(upcoming))
	}
	if upcoming[0].TrackID != trackID2 || upcoming[1].TrackID != trackID3 {
		t.Error("unexpected upcoming track order")
	}

	// Advance and check upcoming again
	state.Advance(LoopModeNone)
	upcoming = state.Upcoming()
	if len(upcoming) != 1 {
		t.Errorf("expected 1 upcoming, got %d", len(upcoming))
	}
	if upcoming[0].TrackID != trackID3 {
		t.Error("expected track3 as upcoming")
	}

	// At last track, upcoming should be empty
	state.Advance(LoopModeNone)
	upcoming = state.Upcoming()
	if len(upcoming) != 0 {
		t.Errorf("expected empty upcoming at last track, got %d", len(upcoming))
	}
}

func TestPlayerState_Seek(t *testing.T) {
	t.Run("seek to valid middle position", func(t *testing.T) {
		state := newTestPlayerState()
		trackID0 := TrackID("track-0")
		trackID1 := TrackID("track-1")
		trackID2 := TrackID("track-2")

		state.Append(testEntry(trackID0), testEntry(trackID1), testEntry(trackID2))
		state.SetPlaybackActive(true)

		got := state.Seek(1)
		if got == nil || got.TrackID != trackID1 {
			t.Errorf("expected track1, got %v", got)
		}
		if state.CurrentIndex() != 1 {
			t.Errorf("expected currentIndex 1, got %d", state.CurrentIndex())
		}
		if got := state.Current(); got == nil || got.TrackID != trackID1 {
			t.Error("Current() should return track1 after seek")
		}
	})

	t.Run("seek to first position", func(t *testing.T) {
		state := newTestPlayerState()
		trackID0 := TrackID("track-0")
		trackID1 := TrackID("track-1")

		state.Append(testEntry(trackID0), testEntry(trackID1))
		state.Advance(LoopModeNone) // now at index 1

		got := state.Seek(0)
		if got == nil || got.TrackID != trackID0 {
			t.Errorf("expected track0, got %v", got)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
	})

	t.Run("seek to last position", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("track-0"), testEntry("track-1"), testEntry("track-2"))

		got := state.Seek(2)
		if got == nil || got.TrackID != "track-2" {
			t.Errorf("expected track-2, got %v", got)
		}
		if state.CurrentIndex() != 2 {
			t.Errorf("expected currentIndex 2, got %d", state.CurrentIndex())
		}
	})

	t.Run("seek to invalid negative position", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("track-0"))

		got := state.Seek(-1)
		if got != nil {
			t.Errorf("expected nil for negative position, got %v", got)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0 (unchanged), got %d", state.CurrentIndex())
		}
	})

	t.Run("seek to out of bounds position", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("track-0"), testEntry("track-1"))

		got := state.Seek(10)
		if got != nil {
			t.Errorf("expected nil for out of bounds position, got %v", got)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0 (unchanged), got %d", state.CurrentIndex())
		}
	})

	t.Run("seek on empty queue", func(t *testing.T) {
		state := newTestPlayerState()

		got := state.Seek(0)
		if got != nil {
			t.Errorf("expected nil for empty queue, got %v", got)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
	})

	t.Run("seek after advancing past end", func(t *testing.T) {
		state := newTestPlayerState()
		trackID0 := TrackID("track-0")

		state.Append(testEntry(trackID0), testEntry("track-1"))
		state.Advance(LoopModeNone) // index=1
		state.Advance(LoopModeNone) // past end

		got := state.Seek(0)
		if got == nil || got.TrackID != trackID0 {
			t.Errorf("expected track0, got %v", got)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
	})
}

func TestPlayerState_QueueRemove(t *testing.T) {
	t.Run("remove before currentIndex adjusts index", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"))
		state.SetPlaybackActive(true)
		state.Seek(2) // currentIndex=2, playing "c"

		removed, err := state.Remove(0) // remove "a"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if removed.TrackID != "a" {
			t.Errorf("expected removed track 'a', got %q", removed.TrackID)
		}
		if state.CurrentIndex() != 1 {
			t.Errorf("expected currentIndex 1, got %d", state.CurrentIndex())
		}
		if got := state.Current(); got == nil || got.TrackID != "c" {
			t.Errorf("expected current track 'c', got %v", got)
		}
	})

	t.Run("remove after currentIndex keeps index", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"))
		state.SetPlaybackActive(true)
		state.Seek(0) // currentIndex=0

		removed, err := state.Remove(2) // remove "c"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if removed.TrackID != "c" {
			t.Errorf("expected removed track 'c', got %q", removed.TrackID)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
	})

	t.Run("remove at currentIndex advances to next", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"))
		state.SetPlaybackActive(true)
		state.Seek(1) // currentIndex=1, playing "b"

		_, err := state.Remove(1) // remove "b"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Advance moved to "c" (was at index 2), then "b" removed shifts it to index 1
		if state.CurrentIndex() != 1 {
			t.Errorf("expected currentIndex 1, got %d", state.CurrentIndex())
		}
		if got := state.Current(); got == nil || got.TrackID != "c" {
			t.Errorf("expected current track 'c', got %v", got)
		}
		if !state.IsPlaybackActive() {
			t.Error("expected playback to remain active")
		}
	})

	t.Run("remove at currentIndex at last deactivates", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.Seek(1) // currentIndex=1 (last)

		_, err := state.Remove(1) // remove "b" at last position
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Advance returns nil (at last, LoopModeNone) → playback deactivates
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		if state.IsPlaybackActive() {
			t.Error("expected playback to be inactive")
		}
	})

	t.Run("remove at currentIndex at last with queue loop wraps", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.SetLoopMode(LoopModeQueue)
		state.Seek(1) // currentIndex=1 (last)

		_, err := state.Remove(1) // remove "b"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Advance wraps to 0 ("a"), then "b" removed → queue ["a"], index 0
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		if got := state.Current(); got == nil || got.TrackID != "a" {
			t.Errorf("expected current track 'a', got %v", got)
		}
		if !state.IsPlaybackActive() {
			t.Error("expected playback to remain active")
		}
	})

	t.Run("remove at currentIndex with track loop advances normally", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"))
		state.SetPlaybackActive(true)
		state.SetLoopMode(LoopModeTrack)
		state.Seek(1) // currentIndex=1, playing "b"

		_, err := state.Remove(1) // remove "b"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// LoopModeTrack treated as LoopModeNone: advance to "c" (index 2→1 after shift)
		if state.CurrentIndex() != 1 {
			t.Errorf("expected currentIndex 1, got %d", state.CurrentIndex())
		}
		if got := state.Current(); got == nil || got.TrackID != "c" {
			t.Errorf("expected current track 'c', got %v", got)
		}
		if !state.IsPlaybackActive() {
			t.Error("expected playback to remain active")
		}
	})

	t.Run("remove at currentIndex at last with track loop deactivates", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.SetLoopMode(LoopModeTrack)
		state.Seek(1) // currentIndex=1 (last)

		_, err := state.Remove(1) // remove "b"
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// LoopModeTrack treated as LoopModeNone: at last, no next → deactivates
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		if state.IsPlaybackActive() {
			t.Error("expected playback to be inactive")
		}
	})

	t.Run("remove last entry deactivates playback", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		_, err := state.Remove(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		if state.IsPlaybackActive() {
			t.Error("expected playback to be inactive after removing last entry")
		}
	})

	t.Run("remove invalid index returns error", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))

		_, err := state.Remove(5)
		if err == nil {
			t.Error("expected error for invalid index")
		}
	})
}

func TestPlayerState_QueueClear(t *testing.T) {
	state := newTestPlayerState()
	state.Append(testEntry("a"), testEntry("b"), testEntry("c"))
	state.SetPlaybackActive(true)
	state.Seek(2)

	state.Clear()

	if state.Len() != 0 {
		t.Errorf("expected empty queue, got %d", state.Len())
	}
	if state.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
	}
	if state.IsPlaybackActive() {
		t.Error("expected playback to be inactive")
	}
}

func TestPlayerState_QueuePrepend(t *testing.T) {
	t.Run("prepend with active playback shifts currentIndex", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.Seek(1) // currentIndex=1, playing "b"

		state.Prepend(testEntry("x"), testEntry("y"))

		// Queue: [x, y, a, b], "b" was at index 1, now at index 3
		if state.CurrentIndex() != 3 {
			t.Errorf("expected currentIndex 3, got %d", state.CurrentIndex())
		}
		if got := state.Current(); got == nil || got.TrackID != "b" {
			t.Errorf("expected current track 'b', got %v", got)
		}
	})

	t.Run("prepend with inactive playback does not shift", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		// playback not active

		state.Prepend(testEntry("x"))

		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
	})
}

func TestPlayerState_Shuffle(t *testing.T) {
	t.Run("active playback - current track moves to index 0", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"), testEntry("d"), testEntry("e"))
		state.SetPlaybackActive(true)
		state.Seek(2) // playing "c"

		state.Shuffle()

		// Current track should now be at index 0
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		current := state.Current()
		if current == nil || current.TrackID != "c" {
			t.Errorf("expected current track 'c', got %v", current)
		}

		// All entries should be preserved
		if state.Len() != 5 {
			t.Fatalf("expected 5 entries, got %d", state.Len())
		}
		seen := make(map[TrackID]bool)
		for _, e := range state.List() {
			seen[e.TrackID] = true
		}
		for _, id := range []TrackID{"a", "b", "c", "d", "e"} {
			if !seen[id] {
				t.Errorf("missing entry %q after shuffle", id)
			}
		}
	})

	t.Run("idle - all entries shuffled, currentIndex unchanged", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"), testEntry("d"), testEntry("e"))
		// playback not active

		state.Shuffle()

		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		if state.Len() != 5 {
			t.Fatalf("expected 5 entries, got %d", state.Len())
		}
		seen := make(map[TrackID]bool)
		for _, e := range state.List() {
			seen[e.TrackID] = true
		}
		for _, id := range []TrackID{"a", "b", "c", "d", "e"} {
			if !seen[id] {
				t.Errorf("missing entry %q after shuffle", id)
			}
		}
	})

	t.Run("empty queue is no-op", func(t *testing.T) {
		state := newTestPlayerState()
		state.Shuffle() // should not panic
		if state.Len() != 0 {
			t.Errorf("expected empty queue, got %d", state.Len())
		}
	})

	t.Run("single entry is no-op", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("only"))
		state.SetPlaybackActive(true)

		state.Shuffle()

		if state.Len() != 1 {
			t.Fatalf("expected 1 entry, got %d", state.Len())
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		current := state.Current()
		if current == nil || current.TrackID != "only" {
			t.Errorf("expected current track 'only', got %v", current)
		}
	})
}

func TestPlayerState_AutoPlayEnabled(t *testing.T) {
	state := newTestPlayerState()

	// Default is true
	if !state.IsAutoPlayEnabled() {
		t.Error("expected auto-play to be enabled by default")
	}

	// Disable auto-play
	state.SetAutoPlayEnabled(false)
	if state.IsAutoPlayEnabled() {
		t.Error("expected auto-play to be disabled")
	}

	// Re-enable auto-play
	state.SetAutoPlayEnabled(true)
	if !state.IsAutoPlayEnabled() {
		t.Error("expected auto-play to be enabled")
	}
}

func TestPlayerState_IsAtLast(t *testing.T) {
	state := newTestPlayerState()

	state.Append(testEntry("track-0"), testEntry("track-1"), testEntry("track-2"))

	// At first track, not at last
	if state.IsAtLast() {
		t.Error("state at first track should not be at last")
	}

	// Advance to middle
	state.Advance(LoopModeNone)
	if state.IsAtLast() {
		t.Error("state at middle track should not be at last")
	}

	// Advance to last
	state.Advance(LoopModeNone)
	if !state.IsAtLast() {
		t.Error("state at last track should be at last")
	}
}

func TestPlayerState_Pause(t *testing.T) {
	t.Run("pause successfully", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		err := state.Pause()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !state.IsPaused() {
			t.Error("expected paused")
		}
	})

	t.Run("not playing", func(t *testing.T) {
		state := newTestPlayerState()

		err := state.Pause()
		if err != ErrNotPlaying {
			t.Errorf("expected ErrNotPlaying, got %v", err)
		}
	})

	t.Run("already paused", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)
		state.SetPaused(true)

		err := state.Pause()
		if err != ErrAlreadyPaused {
			t.Errorf("expected ErrAlreadyPaused, got %v", err)
		}
	})
}

func TestPlayerState_Resume(t *testing.T) {
	t.Run("resume successfully", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)
		state.SetPaused(true)

		err := state.Resume()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.IsPaused() {
			t.Error("expected not paused")
		}
	})

	t.Run("not playing", func(t *testing.T) {
		state := newTestPlayerState()

		err := state.Resume()
		if err != ErrNotPlaying {
			t.Errorf("expected ErrNotPlaying, got %v", err)
		}
	})

	t.Run("not paused", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		err := state.Resume()
		if err != ErrNotPaused {
			t.Errorf("expected ErrNotPaused, got %v", err)
		}
	})
}

func TestPlayerState_Skip(t *testing.T) {
	t.Run("skip to next track", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)

		skipped, next, err := state.Skip()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skipped.TrackID != "a" {
			t.Errorf("expected skipped 'a', got %q", skipped.TrackID)
		}
		if next == nil || next.TrackID != "b" {
			t.Errorf("expected next 'b', got %v", next)
		}
		if !state.IsPlaybackActive() {
			t.Error("expected playback to remain active")
		}
	})

	t.Run("skip at last track", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		skipped, next, err := state.Skip()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skipped.TrackID != "a" {
			t.Errorf("expected skipped 'a', got %q", skipped.TrackID)
		}
		if next != nil {
			t.Errorf("expected next nil, got %v", next)
		}
		if state.IsPlaybackActive() {
			t.Error("expected playback to be inactive")
		}
	})

	t.Run("skip overrides LoopModeTrack", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.SetLoopMode(LoopModeTrack)

		skipped, next, err := state.Skip()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skipped.TrackID != "a" {
			t.Errorf("expected skipped 'a', got %q", skipped.TrackID)
		}
		if next == nil || next.TrackID != "b" {
			t.Errorf("expected next 'b', got %v", next)
		}
	})

	t.Run("skip respects LoopModeQueue", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.SetLoopMode(LoopModeQueue)
		state.Seek(1) // at last

		_, next, err := state.Skip()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if next == nil || next.TrackID != "a" {
			t.Errorf("expected next 'a' (wrap), got %v", next)
		}
	})

	t.Run("not playing", func(t *testing.T) {
		state := newTestPlayerState()

		_, _, err := state.Skip()
		if err != ErrNotPlaying {
			t.Errorf("expected ErrNotPlaying, got %v", err)
		}
	})
}

func TestPlayerState_Enqueue(t *testing.T) {
	t.Run("enqueue to idle player activates", func(t *testing.T) {
		state := newTestPlayerState()

		startIndex, becameActive := state.Enqueue(testEntry("a"), testEntry("b"))
		if startIndex != 0 {
			t.Errorf("expected startIndex 0, got %d", startIndex)
		}
		if !becameActive {
			t.Error("expected becameActive=true")
		}
		if !state.IsPlaybackActive() {
			t.Error("expected playback active")
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		current := state.Current()
		if current == nil || current.TrackID != "a" {
			t.Errorf("expected current 'a', got %v", current)
		}
	})

	t.Run("enqueue to active player appends", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		startIndex, becameActive := state.Enqueue(testEntry("b"), testEntry("c"))
		if startIndex != 1 {
			t.Errorf("expected startIndex 1, got %d", startIndex)
		}
		if becameActive {
			t.Error("expected becameActive=false")
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
		if state.Len() != 3 {
			t.Errorf("expected 3 entries, got %d", state.Len())
		}
	})
}

func TestPlayerState_HandleTrackEnded(t *testing.T) {
	t.Run("normal end advances", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)

		next := state.HandleTrackEnded(false)
		if next == nil || next.TrackID != "b" {
			t.Errorf("expected next 'b', got %v", next)
		}
		if !state.IsPlaybackActive() {
			t.Error("expected playback active")
		}
	})

	t.Run("normal end at last deactivates", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		next := state.HandleTrackEnded(false)
		if next != nil {
			t.Errorf("expected nil, got %v", next)
		}
		if state.IsPlaybackActive() {
			t.Error("expected playback inactive")
		}
	})

	t.Run("normal end with LoopModeTrack repeats", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)
		state.SetLoopMode(LoopModeTrack)

		next := state.HandleTrackEnded(false)
		if next == nil || next.TrackID != "a" {
			t.Errorf("expected next 'a', got %v", next)
		}
	})

	t.Run("failed removes track and advances", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"))
		state.SetPlaybackActive(true)

		next := state.HandleTrackEnded(true)
		if next == nil || next.TrackID != "b" {
			t.Errorf("expected next 'b', got %v", next)
		}
		if state.Len() != 1 {
			t.Errorf("expected 1 entry, got %d", state.Len())
		}
	})

	t.Run("failed removes last track deactivates", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		next := state.HandleTrackEnded(true)
		if next != nil {
			t.Errorf("expected nil, got %v", next)
		}
		if state.IsPlaybackActive() {
			t.Error("expected playback inactive")
		}
	})
}

func TestPlayerState_ClearExceptCurrent(t *testing.T) {
	t.Run("clears except current", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"), testEntry("b"), testEntry("c"))
		state.SetPlaybackActive(true)
		state.Seek(1) // playing "b"

		count, err := state.ClearExceptCurrent()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
		if state.Len() != 1 {
			t.Errorf("expected 1 entry, got %d", state.Len())
		}
		current := state.Current()
		if current == nil || current.TrackID != "b" {
			t.Errorf("expected current 'b', got %v", current)
		}
		if state.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", state.CurrentIndex())
		}
	})

	t.Run("not playing returns error", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))

		_, err := state.ClearExceptCurrent()
		if err != ErrNotPlaying {
			t.Errorf("expected ErrNotPlaying, got %v", err)
		}
	})

	t.Run("only current track returns zero count", func(t *testing.T) {
		state := newTestPlayerState()
		state.Append(testEntry("a"))
		state.SetPlaybackActive(true)

		count, err := state.ClearExceptCurrent()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
	})
}
