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
}

func TestQueue_Add(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// First add to empty queue should return wasEmpty=true
	wasEmpty := q.Add(track1)
	if !wasEmpty {
		t.Error("expected wasEmpty=true for first add")
	}
	if q.Len() != 1 {
		t.Errorf("expected length 1, got %d", q.Len())
	}

	// Second add should return wasEmpty=false
	wasEmpty = q.Add(track2)
	if wasEmpty {
		t.Error("expected wasEmpty=false for second add")
	}
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}
}

func TestQueue_Next(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	// Next on empty queue returns nil
	if got := q.Next(); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Add(track1)
	q.Add(track2)

	// Next should return first track
	got := q.Next()
	if got != track1 {
		t.Errorf("expected track1, got %v", got)
	}
	if q.Len() != 1 {
		t.Errorf("expected length 1 after Next, got %d", q.Len())
	}

	// Next should return second track
	got = q.Next()
	if got != track2 {
		t.Errorf("expected track2, got %v", got)
	}
	if q.Len() != 0 {
		t.Errorf("expected length 0 after second Next, got %d", q.Len())
	}
}

func TestQueue_Peek(t *testing.T) {
	q := NewQueue()
	track := &Track{ID: "track-1", Title: "Song 1"}

	// Peek on empty queue returns nil
	if got := q.Peek(); got != nil {
		t.Errorf("expected nil from empty queue, got %v", got)
	}

	q.Add(track)

	// Peek should return first track without removing
	got := q.Peek()
	if got != track {
		t.Errorf("expected track, got %v", got)
	}
	if q.Len() != 1 {
		t.Errorf("expected length 1 after Peek, got %d", q.Len())
	}

	// Peek again should return same track
	got = q.Peek()
	if got != track {
		t.Errorf("expected same track on second Peek, got %v", got)
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

	// Remove middle track
	removed := q.RemoveAt(1)
	if removed != track2 {
		t.Errorf("expected track2, got %v", removed)
	}
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}

	// Verify order
	list := q.List()
	if list[0] != track1 || list[1] != track3 {
		t.Error("unexpected track order after RemoveAt")
	}

	// Remove at invalid index returns nil
	if got := q.RemoveAt(-1); got != nil {
		t.Errorf("expected nil for negative index, got %v", got)
	}
	if got := q.RemoveAt(10); got != nil {
		t.Errorf("expected nil for out of bounds index, got %v", got)
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()
	track1 := &Track{ID: "track-1", Title: "Song 1"}
	track2 := &Track{ID: "track-2", Title: "Song 2"}

	q.Add(track1)
	q.Add(track2)

	count := q.Clear()
	if count != 2 {
		t.Errorf("expected cleared count 2, got %d", count)
	}
	if q.Len() != 0 {
		t.Errorf("expected empty queue after Clear, got length %d", q.Len())
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
	if q.Peek() != track1 {
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

	q.Next()
	if !q.IsEmpty() {
		t.Error("queue should be empty after removing only track")
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
			q.Peek()
		})
	}

	wg.Wait()

	if q.Len() != 100 {
		t.Errorf("expected 100 tracks after concurrent adds, got %d", q.Len())
	}
}
