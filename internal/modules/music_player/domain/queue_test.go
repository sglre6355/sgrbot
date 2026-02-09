package domain

import (
	"testing"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue()

	if q.Len() != 0 {
		t.Errorf("expected empty queue, got length %d", q.Len())
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}
}

func TestQueue_Append(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// Append single track
	q.Append(trackID1)
	if q.Len() != 1 {
		t.Errorf("expected length 1, got %d", q.Len())
	}

	// Append another track
	q.Append(trackID2)
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}
}

func TestQueue_Append_Multiple(t *testing.T) {
	t.Run("append multiple to empty queue", func(t *testing.T) {
		q := NewQueue()
		trackIDs := []TrackID{"track-1", "track-2", "track-3"}

		q.Append(trackIDs...)
		if q.Len() != 3 {
			t.Errorf("expected length 3, got %d", q.Len())
		}
	})

	t.Run("append empty slice", func(t *testing.T) {
		q := NewQueue()
		trackIDs := []TrackID{}

		q.Append(trackIDs...)
		if q.Len() != 0 {
			t.Errorf("expected length 0, got %d", q.Len())
		}
	})

	t.Run("tracks are appended in order", func(t *testing.T) {
		q := NewQueue()
		q.Append("track-0")

		trackIDs := []TrackID{"track-1", "track-2"}
		q.Append(trackIDs...)

		list := q.List()
		if len(list) != 3 {
			t.Fatalf("expected 3 tracks, got %d", len(list))
		}
		if list[0] != "track-0" || list[1] != "track-1" || list[2] != "track-2" {
			t.Error("tracks not in expected order")
		}
	})
}

func TestQueue_Prepend(t *testing.T) {
	q := NewQueue()
	q.Append("track-2", "track-3")

	q.Prepend("track-0", "track-1")

	list := q.List()
	if len(list) != 4 {
		t.Fatalf("expected 4 tracks, got %d", len(list))
	}
	if list[0] != "track-0" || list[1] != "track-1" || list[2] != "track-2" ||
		list[3] != "track-3" {
		t.Error("tracks not in expected order after prepend")
	}
}

func TestQueue_Current(t *testing.T) {
	q := NewQueue()
	trackID := TrackID("track-1")

	// Current on empty queue returns nil
	if got := q.Current(); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Append(trackID)

	// After Append, Current returns the first track (queue is active)
	got := q.Current()
	if got == nil {
		t.Fatal("expected track after Append")
	}
	if *got != trackID {
		t.Errorf("expected %s, got %s", trackID, *got)
	}
}

func TestQueue_Advance_LoopModeNone(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// Advance on empty queue returns nil
	if got := q.Advance(LoopModeNone); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Append(trackID1, trackID2)

	// Advance should return next track
	got := q.Advance(LoopModeNone)
	if got == nil || *got != trackID2 {
		t.Errorf("expected track2, got %v", got)
	}
	if q.CurrentIndex() != 1 {
		t.Errorf("expected currentIndex 1, got %d", q.CurrentIndex())
	}

	// Advance past end should return nil
	got = q.Advance(LoopModeNone)
	if got != nil {
		t.Errorf("expected nil past end, got %v", got)
	}

	// Tracks should still be in queue
	if q.Len() != 2 {
		t.Errorf("expected 2 tracks still in queue, got %d", q.Len())
	}
}

func TestQueue_Advance_LoopModeTrack(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	q.Append(trackID1, trackID2)

	// Advance with LoopModeTrack should return same track
	got := q.Advance(LoopModeTrack)
	if got == nil || *got != trackID1 {
		t.Errorf("expected track1, got %v", got)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}

	// Multiple advances should keep returning same track
	for i := range 5 {
		got = q.Advance(LoopModeTrack)
		if got == nil || *got != trackID1 {
			t.Errorf("iteration %d: expected track1, got %v", i, got)
		}
	}
}

func TestQueue_Advance_LoopModeQueue(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	q.Append(trackID1, trackID2, trackID3)

	// Advance through all tracks
	if got := q.Advance(LoopModeQueue); got == nil || *got != trackID2 {
		t.Errorf("expected track2, got %v", got)
	}
	if got := q.Advance(LoopModeQueue); got == nil || *got != trackID3 {
		t.Errorf("expected track3, got %v", got)
	}

	// Should wrap to beginning
	got := q.Advance(LoopModeQueue)
	if got == nil || *got != trackID1 {
		t.Errorf("expected track1 (wrap), got %v", got)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after wrap, got %d", q.CurrentIndex())
	}
}

func TestQueue_Advance_LoopModeQueue_SingleTrack(t *testing.T) {
	q := NewQueue()
	trackID := TrackID("track-1")

	q.Append(trackID)

	// Single track with LoopModeQueue should keep returning same track
	for i := range 5 {
		got := q.Advance(LoopModeQueue)
		if got == nil || *got != trackID {
			t.Errorf("iteration %d: expected track, got %v", i, got)
		}
	}
}

func TestQueue_HasNext(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// HasNext on empty queue returns false
	if q.HasNext(LoopModeNone) {
		t.Error("expected HasNext=false for empty queue")
	}

	q.Append(trackID1, trackID2)

	// HasNext with LoopModeNone should be true when there are more tracks
	if !q.HasNext(LoopModeNone) {
		t.Error("expected HasNext=true with more tracks")
	}

	// Advance to last track
	q.Advance(LoopModeNone)

	// HasNext should be false at last track with LoopModeNone
	if q.HasNext(LoopModeNone) {
		t.Error("expected HasNext=false at last track with LoopModeNone")
	}

	// HasNext should be true with LoopModeTrack
	if !q.HasNext(LoopModeTrack) {
		t.Error("expected HasNext=true with LoopModeTrack")
	}

	// HasNext should be true with LoopModeQueue
	if !q.HasNext(LoopModeQueue) {
		t.Error("expected HasNext=true with LoopModeQueue")
	}
}

func TestQueue_Upcoming(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	q.Append(trackID1, trackID2, trackID3)

	// Upcoming should return tracks after current
	upcoming := q.Upcoming()
	if len(upcoming) != 2 {
		t.Errorf("expected 2 upcoming, got %d", len(upcoming))
	}
	if upcoming[0] != trackID2 || upcoming[1] != trackID3 {
		t.Error("unexpected upcoming track order")
	}

	// Advance and check upcoming again
	q.Advance(LoopModeNone)
	upcoming = q.Upcoming()
	if len(upcoming) != 1 {
		t.Errorf("expected 1 upcoming, got %d", len(upcoming))
	}
	if upcoming[0] != trackID3 {
		t.Error("expected track3 as upcoming")
	}

	// At last track, upcoming should be empty
	q.Advance(LoopModeNone)
	upcoming = q.Upcoming()
	if len(upcoming) != 0 {
		t.Errorf("expected empty upcoming at last track, got %d", len(upcoming))
	}
}

func TestQueue_RemoveAt(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	q.Append(trackID1, trackID2, trackID3)

	// Set current to track2 (index 1)
	q.Advance(LoopModeNone)

	// Remove track before current (should decrement currentIndex)
	removed := q.RemoveAt(0)
	if removed == nil || *removed != trackID1 {
		t.Errorf("expected track1, got %v", removed)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after removing before, got %d", q.CurrentIndex())
	}
	if got := q.Current(); got == nil || *got != trackID2 {
		t.Error("current track should still be track2")
	}

	// Remove at invalid index returns nil
	if got := q.RemoveAt(-1); got != nil {
		t.Errorf("expected nil for negative index, got %v", got)
	}
	if got := q.RemoveAt(10); got != nil {
		t.Errorf("expected nil for out of bounds index, got %v", got)
	}
}

func TestQueue_RemoveAt_CurrentTrack(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	q.Append(trackID1, trackID2)

	// Remove current track
	removed := q.RemoveAt(0)
	if removed == nil || *removed != trackID1 {
		t.Errorf("expected track1, got %v", removed)
	}

	// Current should now be track2
	if got := q.Current(); got == nil || *got != trackID2 {
		t.Error("current track should be track2 after removing track1")
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}
}

func TestQueue_RemoveAt_AdjustsIndexWhenRemovingLastTrack(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	q.Append(trackID1, trackID2)
	q.Advance(LoopModeNone) // Now at track2 (index 1)

	// Remove current track (which is the last one)
	removed := q.RemoveAt(1)
	if removed == nil || *removed != trackID2 {
		t.Errorf("expected track2, got %v", removed)
	}

	// Index should be adjusted to point to last available track
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after removing last, got %d", q.CurrentIndex())
	}
	if got := q.Current(); got == nil || *got != trackID1 {
		t.Error("current track should be track1")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	q.Append(trackID1, trackID2)

	q.Clear()
	if q.Len() != 0 {
		t.Errorf("expected empty queue after Clear, got length %d", q.Len())
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after Clear, got %d", q.CurrentIndex())
	}
}

func TestQueue_List(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// Empty list
	list := q.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	q.Append(trackID1, trackID2)

	list = q.List()
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
	if list[0] != trackID1 || list[1] != trackID2 {
		t.Error("unexpected track order in List")
	}

	// Verify List returns a copy (modifying it doesn't affect queue)
	list[0] = "modified"
	if got := q.Current(); got == nil || *got != trackID1 {
		t.Error("modifying List result affected queue")
	}
}

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue()

	if !q.IsEmpty() {
		t.Error("new queue should be empty")
	}

	q.Append("track-1")
	if q.IsEmpty() {
		t.Error("queue with track should not be empty")
	}

	q.Clear()
	if !q.IsEmpty() {
		t.Error("queue should be empty after Clear")
	}
}

func TestQueue_GetAt(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// GetAt on empty queue returns nil
	if got := q.GetAt(0); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Append(trackID1, trackID2)

	if got := q.GetAt(0); got == nil || *got != trackID1 {
		t.Errorf("expected track1 at index 0, got %v", got)
	}
	if got := q.GetAt(1); got == nil || *got != trackID2 {
		t.Errorf("expected track2 at index 1, got %v", got)
	}
	if got := q.GetAt(-1); got != nil {
		t.Errorf("expected nil for negative index, got %v", got)
	}
	if got := q.GetAt(2); got != nil {
		t.Errorf("expected nil for out of bounds index, got %v", got)
	}
}

func TestQueue_Seek(t *testing.T) {
	t.Run("seek to valid middle position", func(t *testing.T) {
		q := NewQueue()
		trackID0 := TrackID("track-0")
		trackID1 := TrackID("track-1")
		trackID2 := TrackID("track-2")

		q.Append(trackID0, trackID1, trackID2)

		// Seek to middle position
		got := q.Seek(1)
		if got == nil || *got != trackID1 {
			t.Errorf("expected track1, got %v", got)
		}
		if q.CurrentIndex() != 1 {
			t.Errorf("expected currentIndex 1, got %d", q.CurrentIndex())
		}
		if got := q.Current(); got == nil || *got != trackID1 {
			t.Error("Current() should return track1 after seek")
		}
	})

	t.Run("seek to first position", func(t *testing.T) {
		q := NewQueue()
		trackID0 := TrackID("track-0")
		trackID1 := TrackID("track-1")

		q.Append(trackID0, trackID1)
		q.Advance(LoopModeNone) // now at index 1

		// Seek back to first position
		got := q.Seek(0)
		if got == nil || *got != trackID0 {
			t.Errorf("expected track0, got %v", got)
		}
		if q.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
		}
	})

	t.Run("seek to last position", func(t *testing.T) {
		q := NewQueue()
		trackID0 := TrackID("track-0")
		trackID1 := TrackID("track-1")
		trackID2 := TrackID("track-2")

		q.Append(trackID0, trackID1, trackID2)

		// Seek to last position
		got := q.Seek(2)
		if got == nil || *got != trackID2 {
			t.Errorf("expected track2, got %v", got)
		}
		if q.CurrentIndex() != 2 {
			t.Errorf("expected currentIndex 2, got %d", q.CurrentIndex())
		}
	})

	t.Run("seek to invalid negative position", func(t *testing.T) {
		q := NewQueue()
		q.Append("track-0")

		got := q.Seek(-1)
		if got != nil {
			t.Errorf("expected nil for negative position, got %v", got)
		}
		// currentIndex should remain unchanged
		if q.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0 (unchanged), got %d", q.CurrentIndex())
		}
	})

	t.Run("seek to out of bounds position", func(t *testing.T) {
		q := NewQueue()
		q.Append("track-0", "track-1")

		got := q.Seek(10)
		if got != nil {
			t.Errorf("expected nil for out of bounds position, got %v", got)
		}
		// currentIndex should remain unchanged
		if q.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0 (unchanged), got %d", q.CurrentIndex())
		}
	})

	t.Run("seek on empty queue", func(t *testing.T) {
		q := NewQueue()

		got := q.Seek(0)
		if got != nil {
			t.Errorf("expected nil for empty queue, got %v", got)
		}
		if q.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
		}
	})

	t.Run("seek after advancing past end", func(t *testing.T) {
		q := NewQueue()
		trackID0 := TrackID("track-0")
		trackID1 := TrackID("track-1")

		q.Append(trackID0, trackID1)
		q.Advance(LoopModeNone) // index=1
		q.Advance(LoopModeNone) // past end

		// Seek should bring us back to a valid position
		got := q.Seek(0)
		if got == nil || *got != trackID0 {
			t.Errorf("expected track0, got %v", got)
		}
		if q.CurrentIndex() != 0 {
			t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
		}
	})
}

func TestQueue_IsAtLast(t *testing.T) {
	q := NewQueue()

	// Empty queue is at last (vacuously true)
	if !q.IsAtLast() {
		t.Error("empty queue should be at last")
	}

	q.Append("track-0", "track-1", "track-2")

	// At first track, not at last
	if q.IsAtLast() {
		t.Error("queue at first track should not be at last")
	}

	// Advance to middle
	q.Advance(LoopModeNone)
	if q.IsAtLast() {
		t.Error("queue at middle track should not be at last")
	}

	// Advance to last
	q.Advance(LoopModeNone)
	if !q.IsAtLast() {
		t.Error("queue at last track should be at last")
	}
}
