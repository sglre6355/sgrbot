package domain

import "sync"

// Queue is a thread-safe queue for managing tracks.
type Queue struct {
	mu     sync.RWMutex
	tracks []*Track
}

// NewQueue creates a new empty Queue.
func NewQueue() *Queue {
	return &Queue{
		tracks: make([]*Track, 0),
	}
}

// Add adds a track to the end of the queue.
// Returns true if the queue was empty before adding (to trigger auto-play).
func (q *Queue) Add(track *Track) (wasEmpty bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	wasEmpty = len(q.tracks) == 0
	q.tracks = append(q.tracks, track)
	return wasEmpty
}

// Next removes and returns the first track from the queue.
// Returns nil if the queue is empty.
func (q *Queue) Next() *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return nil
	}

	track := q.tracks[0]
	q.tracks = q.tracks[1:]
	return track
}

// Peek returns the first track without removing it.
// Returns nil if the queue is empty.
func (q *Queue) Peek() *Track {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.tracks) == 0 {
		return nil
	}
	return q.tracks[0]
}

// RemoveAt removes and returns the track at the given index (0-indexed).
// Returns nil if the index is out of bounds.
func (q *Queue) RemoveAt(index int) *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index < 0 || index >= len(q.tracks) {
		return nil
	}

	track := q.tracks[index]
	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)
	return track
}

// Clear removes all tracks from the queue.
// Returns the number of tracks that were removed.
func (q *Queue) Clear() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	count := len(q.tracks)
	q.tracks = make([]*Track, 0)
	return count
}

// List returns a copy of all tracks in the queue.
func (q *Queue) List() []*Track {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*Track, len(q.tracks))
	copy(result, q.tracks)
	return result
}

// Len returns the number of tracks in the queue.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.tracks)
}

// IsEmpty returns true if the queue has no tracks.
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

// GetAt returns the track at the given index without removing it.
// Returns nil if the index is out of bounds.
func (q *Queue) GetAt(index int) *Track {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if index < 0 || index >= len(q.tracks) {
		return nil
	}
	return q.tracks[index]
}

// Prepend adds a track to the front of the queue (index 0).
func (q *Queue) Prepend(track *Track) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tracks = append([]*Track{track}, q.tracks...)
}

// ClearAfterCurrent removes all tracks except index 0.
// Returns the number of tracks removed.
func (q *Queue) ClearAfterCurrent() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) <= 1 {
		return 0
	}

	count := len(q.tracks) - 1
	q.tracks = q.tracks[:1]
	return count
}
