package domain

import (
	"testing"
)

func entry(id TrackID) QueueEntry {
	return QueueEntry{TrackID: id}
}

func TestNewQueue(t *testing.T) {
	q := NewQueue()

	if q.Len() != 0 {
		t.Errorf("expected empty queue, got length %d", q.Len())
	}
}

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue()

	if !q.IsEmpty() {
		t.Error("new queue should be empty")
	}

	q.Append(entry("track-1"))
	if q.IsEmpty() {
		t.Error("queue with track should not be empty")
	}

	q.Clear()
	if !q.IsEmpty() {
		t.Error("queue should be empty after Clear")
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

	q.Append(entry(trackID1), entry(trackID2))

	list = q.List()
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
	if list[0].TrackID != trackID1 || list[1].TrackID != trackID2 {
		t.Error("unexpected track order in List")
	}

	// Verify List returns a copy (modifying it doesn't affect queue)
	list[0].TrackID = "modified"
	got, _ := q.Get(0)
	if got == nil || got.TrackID != trackID1 {
		t.Error("modifying List result affected queue")
	}
}

func TestQueue_Get(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// Get on empty queue returns error
	if _, err := q.Get(0); err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex from empty queue, got %v", err)
	}

	q.Append(entry(trackID1), entry(trackID2))

	got, err := q.Get(0)
	if err != nil || got == nil || got.TrackID != trackID1 {
		t.Errorf("expected track1 at index 0, got %v, err %v", got, err)
	}
	got, err = q.Get(1)
	if err != nil || got == nil || got.TrackID != trackID2 {
		t.Errorf("expected track2 at index 1, got %v, err %v", got, err)
	}
	if _, err := q.Get(-1); err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex for negative index, got %v", err)
	}
	if _, err := q.Get(2); err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex for out of bounds index, got %v", err)
	}
}

func TestQueue_Remove(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")
	trackID3 := TrackID("track-3")

	q.Append(entry(trackID1), entry(trackID2), entry(trackID3))

	// Remove first entry
	removed, err := q.Remove(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed == nil || removed.TrackID != trackID1 {
		t.Errorf("expected track1, got %v", removed)
	}
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}

	// Remaining entries should be in order
	list := q.List()
	if list[0].TrackID != trackID2 || list[1].TrackID != trackID3 {
		t.Error("unexpected order after removal")
	}

	// Remove at invalid index returns error
	if _, err := q.Remove(-1); err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex for negative index, got %v", err)
	}
	if _, err := q.Remove(10); err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex for out of bounds index, got %v", err)
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	q.Append(entry(trackID1), entry(trackID2))

	q.Clear()
	if q.Len() != 0 {
		t.Errorf("expected empty queue after Clear, got length %d", q.Len())
	}
}

func TestQueue_Prepend(t *testing.T) {
	q := NewQueue()
	q.Append(entry("track-2"), entry("track-3"))

	q.Prepend(entry("track-0"), entry("track-1"))

	list := q.List()
	if len(list) != 4 {
		t.Fatalf("expected 4 tracks, got %d", len(list))
	}
	if list[0].TrackID != "track-0" || list[1].TrackID != "track-1" ||
		list[2].TrackID != "track-2" ||
		list[3].TrackID != "track-3" {
		t.Error("tracks not in expected order after prepend")
	}
}

func TestQueue_Append(t *testing.T) {
	q := NewQueue()
	trackID1 := TrackID("track-1")
	trackID2 := TrackID("track-2")

	// Append single track
	q.Append(entry(trackID1))
	if q.Len() != 1 {
		t.Errorf("expected length 1, got %d", q.Len())
	}

	// Append another track
	q.Append(entry(trackID2))
	if q.Len() != 2 {
		t.Errorf("expected length 2, got %d", q.Len())
	}
}

func TestQueue_Append_Multiple(t *testing.T) {
	t.Run("append multiple to empty queue", func(t *testing.T) {
		q := NewQueue()
		entries := []QueueEntry{entry("track-1"), entry("track-2"), entry("track-3")}

		q.Append(entries...)
		if q.Len() != 3 {
			t.Errorf("expected length 3, got %d", q.Len())
		}
	})

	t.Run("append empty slice", func(t *testing.T) {
		q := NewQueue()
		entries := []QueueEntry{}

		q.Append(entries...)
		if q.Len() != 0 {
			t.Errorf("expected length 0, got %d", q.Len())
		}
	})

	t.Run("tracks are appended in order", func(t *testing.T) {
		q := NewQueue()
		q.Append(entry("track-0"))

		entries := []QueueEntry{entry("track-1"), entry("track-2")}
		q.Append(entries...)

		list := q.List()
		if len(list) != 3 {
			t.Fatalf("expected 3 tracks, got %d", len(list))
		}
		if list[0].TrackID != "track-0" || list[1].TrackID != "track-1" ||
			list[2].TrackID != "track-2" {
			t.Error("tracks not in expected order")
		}
	})
}
