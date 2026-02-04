package domain

import (
	"strconv"
	"sync"
	"testing"
)

func TestNewQueue(t *testing.T) {
	q := NewQueue()

	if q == nil {
		t.Fatal("NewQueue returned nil")
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue, got length %d", q.Len())
	}
	if q.CurrentIndex() != -1 {
		t.Errorf("expected currentIndex -1, got %d", q.CurrentIndex())
	}
}

func TestQueue_Add(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// First add to idle queue should return wasIdle=true
	wasIdle := q.Add(track1)
	if !wasIdle {
		t.Error("expected wasIdle=true for first add")
	}
	if q.Len() != 1 {
		t.Errorf("expected length 1, got %d", q.Len())
	}

	// Start playback
	q.Start()

	// Second add while playing should return wasIdle=false
	wasIdle = q.Add(track2)
	if wasIdle {
		t.Error("expected wasIdle=false when playing")
	}
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}
}

func TestQueue_Add_WasIdleAfterQueueEnds(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Start()

	// Advance past the end
	q.Advance(LoopModeNone)

	// Adding now should return wasIdle=true since we're past the end
	wasIdle := q.Add(track2)
	if !wasIdle {
		t.Error("expected wasIdle=true after queue ended")
	}
}

func TestQueue_Current(t *testing.T) {
	q := NewQueue()
	track := &Track{ID: "track-1", Title: "Song 1"}

	// Current on empty queue returns nil
	if got := q.Current(); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Add(track)

	// Current before Start returns nil (currentIndex is -1)
	if got := q.Current(); got != nil {
		t.Errorf("expected nil before Start, got %v", got)
	}

	// After Start, Current returns the first track
	q.Start()
	if got := q.Current(); got != track {
		t.Errorf("expected track after Start, got %v", got)
	}
}

func TestQueue_Peek_AliasForCurrent(t *testing.T) {
	q := NewQueue()
	track := &Track{ID: "track-1", Title: "Song 1"}

	q.Add(track)
	q.Start()

	// Peek should return same as Current
	if q.Peek() != q.Current() {
		t.Error("Peek should be an alias for Current")
	}
}

func TestQueue_Start(t *testing.T) {
	q := NewQueue()
	track := &Track{ID: "track-1", Title: "Song 1"}

	// Start on empty queue returns nil
	if got := q.Start(); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}
	if q.CurrentIndex() != -1 {
		t.Errorf("expected currentIndex -1 after Start on empty, got %d", q.CurrentIndex())
	}

	q.Add(track)

	// Start should return first track and set index to 0
	got := q.Start()
	if got != track {
		t.Errorf("expected track, got %v", got)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}
}

func TestQueue_Advance_LoopModeNone(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// Advance on empty queue returns nil
	if got := q.Advance(LoopModeNone); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Add(track1)
	q.Add(track2)
	q.Start()

	// Advance should return next track
	got := q.Advance(LoopModeNone)
	if got != track2 {
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
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Add(track2)
	q.Start()

	// Advance with LoopModeTrack should return same track
	got := q.Advance(LoopModeTrack)
	if got != track1 {
		t.Errorf("expected track1, got %v", got)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}

	// Multiple advances should keep returning same track
	for i := range 5 {
		got = q.Advance(LoopModeTrack)
		if got != track1 {
			t.Errorf("iteration %d: expected track1, got %v", i, got)
		}
	}
}

func TestQueue_Advance_LoopModeQueue(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}
	track3 := &Track{ID: "track-3", Title: "Song 3"}

	q.Add(track1)
	q.Add(track2)
	q.Add(track3)
	q.Start()

	// Advance through all tracks
	if got := q.Advance(LoopModeQueue); got != track2 {
		t.Errorf("expected track2, got %v", got)
	}
	if got := q.Advance(LoopModeQueue); got != track3 {
		t.Errorf("expected track3, got %v", got)
	}

	// Should wrap to beginning
	got := q.Advance(LoopModeQueue)
	if got != track1 {
		t.Errorf("expected track1 (wrap), got %v", got)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after wrap, got %d", q.CurrentIndex())
	}
}

func TestQueue_Advance_LoopModeQueue_SingleTrack(t *testing.T) {
	q := NewQueue()
	track := &Track{ID: "track-1", Title: "Song 1"}

	q.Add(track)
	q.Start()

	// Single track with LoopModeQueue should keep returning same track
	for i := range 5 {
		got := q.Advance(LoopModeQueue)
		if got != track {
			t.Errorf("iteration %d: expected track, got %v", i, got)
		}
	}
}

func TestQueue_HasNext(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// HasNext on empty queue returns false
	if q.HasNext(LoopModeNone) {
		t.Error("expected HasNext=false for empty queue")
	}

	q.Add(track1)
	q.Add(track2)
	q.Start()

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
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}
	track3 := &Track{ID: "track-3", Title: "Song 3"}

	// Upcoming on empty queue returns empty slice
	upcoming := q.Upcoming()
	if len(upcoming) != 0 {
		t.Errorf("expected empty slice, got %d items", len(upcoming))
	}

	q.Add(track1)
	q.Add(track2)
	q.Add(track3)
	q.Start()

	// Upcoming should return tracks after current
	upcoming = q.Upcoming()
	if len(upcoming) != 2 {
		t.Errorf("expected 2 upcoming, got %d", len(upcoming))
	}
	if upcoming[0] != track2 || upcoming[1] != track3 {
		t.Error("unexpected upcoming track order")
	}

	// Advance and check upcoming again
	q.Advance(LoopModeNone)
	upcoming = q.Upcoming()
	if len(upcoming) != 1 {
		t.Errorf("expected 1 upcoming, got %d", len(upcoming))
	}
	if upcoming[0] != track3 {
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
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}
	track3 := &Track{ID: "track-3", Title: "Song 3"}

	q.Add(track1)
	q.Add(track2)
	q.Add(track3)
	q.Start()

	// Set current to track2 (index 1)
	q.Advance(LoopModeNone)

	// Remove track before current (should decrement currentIndex)
	removed := q.RemoveAt(0)
	if removed != track1 {
		t.Errorf("expected track1, got %v", removed)
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after removing before, got %d", q.CurrentIndex())
	}
	if q.Current() != track2 {
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
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Add(track2)
	q.Start()

	// Remove current track
	removed := q.RemoveAt(0)
	if removed != track1 {
		t.Errorf("expected track1, got %v", removed)
	}

	// Current should now be track2
	if q.Current() != track2 {
		t.Error("current track should be track2 after removing track1")
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}
}

func TestQueue_RemoveAt_AdjustsIndexWhenRemovingLastTrack(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Add(track2)
	q.Start()
	q.Advance(LoopModeNone) // Now at track2 (index 1)

	// Remove current track (which is the last one)
	removed := q.RemoveAt(1)
	if removed != track2 {
		t.Errorf("expected track2, got %v", removed)
	}

	// Index should be adjusted to point to last available track
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0 after removing last, got %d", q.CurrentIndex())
	}
	if q.Current() != track1 {
		t.Error("current track should be track1")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Add(track2)
	q.Start()

	count := q.Clear()
	if count != 2 {
		t.Errorf("expected cleared count 2, got %d", count)
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue after Clear, got length %d", q.Len())
	}
	if q.CurrentIndex() != -1 {
		t.Errorf("expected currentIndex -1 after Clear, got %d", q.CurrentIndex())
	}

	// Clear empty queue returns 0
	count = q.Clear()
	if count != 0 {
		t.Errorf("expected cleared count 0 for empty queue, got %d", count)
	}
}

func TestQueue_List(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// Empty list
	list := q.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	q.Add(track1)
	q.Add(track2)

	list = q.List()
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
	if list[0] != track1 || list[1] != track2 {
		t.Error("unexpected track order in List")
	}

	// Verify List returns a copy (modifying it doesn't affect queue)
	list[0] = nil
	q.Start()
	if q.Current() != track1 {
		t.Error("modifying List result affected queue")
	}
}

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue()

	if !q.IsEmpty() {
		t.Error("new queue should be empty")
	}

	q.Add(&Track{ID: "track-1"})
	if q.IsEmpty() {
		t.Error("queue with track should not be empty")
	}

	q.Clear()
	if !q.IsEmpty() {
		t.Error("queue should be empty after Clear")
	}
}

func TestQueue_IsIdle(t *testing.T) {
	q := NewQueue()

	// Empty queue is idle
	if !q.IsIdle() {
		t.Error("empty queue should be idle")
	}

	q.Add(&Track{ID: "track-1"})

	// Queue with tracks but not started is idle
	if !q.IsIdle() {
		t.Error("queue before Start should be idle")
	}

	q.Start()

	// After Start, not idle
	if q.IsIdle() {
		t.Error("queue after Start should not be idle")
	}

	// Advance past end
	q.Advance(LoopModeNone)

	// Past end is idle
	if !q.IsIdle() {
		t.Error("queue past end should be idle")
	}
}

func TestQueue_GetAt(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// GetAt on empty queue returns nil
	if got := q.GetAt(0); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Add(track1)
	q.Add(track2)

	if got := q.GetAt(0); got != track1 {
		t.Errorf("expected track1 at index 0, got %v", got)
	}
	if got := q.GetAt(1); got != track2 {
		t.Errorf("expected track2 at index 1, got %v", got)
	}
	if got := q.GetAt(-1); got != nil {
		t.Errorf("expected nil for negative index, got %v", got)
	}
	if got := q.GetAt(2); got != nil {
		t.Errorf("expected nil for out of bounds index, got %v", got)
	}
}

func TestQueue_ClearAfterCurrent(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}
	track3 := &Track{ID: "track-3", Title: "Song 3"}

	q.Add(track1)
	q.Add(track2)
	q.Add(track3)
	q.Start()

	count := q.ClearAfterCurrent()
	if count != 2 {
		t.Errorf("expected 2 tracks removed, got %d", count)
	}
	if q.Len() != 1 {
		t.Errorf("expected 1 track remaining, got %d", q.Len())
	}
	if q.Current() != track1 {
		t.Error("current track should still be track1")
	}

	// ClearAfterCurrent when already at last track returns 0
	count = q.ClearAfterCurrent()
	if count != 0 {
		t.Errorf("expected 0 tracks removed, got %d", count)
	}
}

func TestQueue_ConcurrentAccess(t *testing.T) {
	q := NewQueue()
	var wg sync.WaitGroup

	// Concurrent adds
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			q.Add(&Track{ID: TrackID(strconv.Itoa(id))})
		}(i)
	}

	// Concurrent reads while adding
	for range 50 {
		wg.Go(func() {
			q.Len()
			q.List()
			q.Current()
			q.CurrentIndex()
			q.IsIdle()
		})
	}

	wg.Wait()

	if q.Len() != 100 {
		t.Errorf("expected 100 tracks after concurrent adds, got %d", q.Len())
	}
}

func TestQueue_ConcurrentAdvance(t *testing.T) {
	q := NewQueue()

	// Add tracks
	for i := range 100 {
		q.Add(&Track{ID: TrackID(strconv.Itoa(i))})
	}
	q.Start()

	var wg sync.WaitGroup

	// Concurrent advances and reads
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			q.Advance(LoopModeNone)
		}()
		go func() {
			defer wg.Done()
			q.Current()
			q.HasNext(LoopModeNone)
		}()
	}

	wg.Wait()

	// Should not panic and index should be valid
	idx := q.CurrentIndex()
	if idx < 0 || idx > 100 {
		t.Errorf("unexpected currentIndex %d after concurrent operations", idx)
	}
}

func TestQueue_ResetToIdle(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Add(track2)
	q.Start()

	// Verify we're not idle
	if q.IsIdle() {
		t.Error("queue should not be idle after Start")
	}
	if q.CurrentIndex() != 0 {
		t.Errorf("expected currentIndex 0, got %d", q.CurrentIndex())
	}

	// Reset to idle
	q.ResetToIdle()

	// Verify idle state
	if !q.IsIdle() {
		t.Error("queue should be idle after ResetToIdle")
	}
	if q.CurrentIndex() != -1 {
		t.Errorf("expected currentIndex -1, got %d", q.CurrentIndex())
	}

	// Tracks should still be in queue
	if q.Len() != 2 {
		t.Errorf("expected 2 tracks, got %d", q.Len())
	}

	// Can start again from beginning
	q.Start()
	if q.Current() != track1 {
		t.Error("expected to start from first track")
	}
}

func TestQueue_ResetToIdle_FromMiddleOfQueue(t *testing.T) {
	q := NewQueue()
	for i := range 5 {
		q.Add(&Track{ID: TrackID(strconv.Itoa(i))})
	}
	q.Start()
	q.Advance(LoopModeNone) // index=1
	q.Advance(LoopModeNone) // index=2

	if q.CurrentIndex() != 2 {
		t.Errorf("expected currentIndex 2, got %d", q.CurrentIndex())
	}

	q.ResetToIdle()

	if q.CurrentIndex() != -1 {
		t.Errorf("expected currentIndex -1, got %d", q.CurrentIndex())
	}
	if !q.IsIdle() {
		t.Error("queue should be idle after ResetToIdle")
	}
}

func TestQueue_ResetToIdle_AfterQueueEnds(t *testing.T) {
	q := NewQueue()
	q.Add(&Track{ID: "track-1"})
	q.Start()
	q.Advance(LoopModeNone) // past end

	// Queue is already idle (past end)
	if !q.IsIdle() {
		t.Error("queue should be idle when past end")
	}

	// Reset to idle should still work
	q.ResetToIdle()

	if q.CurrentIndex() != -1 {
		t.Errorf("expected currentIndex -1, got %d", q.CurrentIndex())
	}
}
