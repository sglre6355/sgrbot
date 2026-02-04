package domain

import "sync"

// Queue is a thread-safe queue for managing tracks using an index-based model.
// Instead of removing tracks when they finish, we maintain a currentIndex
// that advances through the track list, enabling loop functionality.
type Queue struct {
	mu           sync.RWMutex
	tracks       []*Track
	currentIndex int // -1 when empty or before first track, 0+ when playing
}

// NewQueue creates a new empty Queue.
func NewQueue() *Queue {
	return &Queue{
		tracks:       make([]*Track, 0),
		currentIndex: -1,
	}
}

// Add adds a track to the end of the queue.
// Returns true if the player was idle (no current track), to trigger auto-play.
func (q *Queue) Add(track *Track) (wasIdle bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	wasIdle = q.currentIndex < 0 || q.currentIndex >= len(q.tracks)
	q.tracks = append(q.tracks, track)
	return wasIdle
}

// Current returns the track at currentIndex, or nil if no current track.
func (q *Queue) Current() *Track {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.currentIndex < 0 || q.currentIndex >= len(q.tracks) {
		return nil
	}
	return q.tracks[q.currentIndex]
}

// Peek returns the current track without changing position.
// This is an alias for Current() for backward compatibility.
func (q *Queue) Peek() *Track {
	return q.Current()
}

// Start sets currentIndex to 0 if the queue has tracks.
// Returns the first track, or nil if queue is empty.
// Used when playback should begin from the start.
func (q *Queue) Start() *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return nil
	}
	q.currentIndex = 0
	return q.tracks[0]
}

// Advance moves to the next track based on loop mode.
// Returns the new current track, or nil if queue ended.
//   - LoopModeNone: advance index, return nil if past end
//   - LoopModeTrack: don't advance, return same track
//   - LoopModeQueue: advance, wrap to 0 if past end
func (q *Queue) Advance(mode LoopMode) *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 || q.currentIndex < 0 {
		return nil
	}

	switch mode {
	case LoopModeTrack:
		// Don't advance, return same track
		return q.tracks[q.currentIndex]

	case LoopModeQueue:
		// Advance with wrap-around
		q.currentIndex++
		if q.currentIndex >= len(q.tracks) {
			q.currentIndex = 0
		}
		return q.tracks[q.currentIndex]

	default: // LoopModeNone
		// Advance without wrap
		q.currentIndex++
		if q.currentIndex >= len(q.tracks) {
			return nil
		}
		return q.tracks[q.currentIndex]
	}
}

// HasNext returns true if there's a next track available (considering loop mode).
func (q *Queue) HasNext(mode LoopMode) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.tracks) == 0 || q.currentIndex < 0 {
		return false
	}

	switch mode {
	case LoopModeTrack, LoopModeQueue:
		// Always has next when looping (as long as queue isn't empty)
		return true
	default: // LoopModeNone
		return q.currentIndex+1 < len(q.tracks)
	}
}

// Upcoming returns tracks after the current index (for queue display).
// Returns empty slice if no tracks or no current track.
func (q *Queue) Upcoming() []*Track {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.currentIndex < 0 || q.currentIndex >= len(q.tracks)-1 {
		return make([]*Track, 0)
	}

	upcoming := q.tracks[q.currentIndex+1:]
	result := make([]*Track, len(upcoming))
	copy(result, upcoming)
	return result
}

// CurrentIndex returns the current track index (-1 if not started or empty).
func (q *Queue) CurrentIndex() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.currentIndex
}

// RemoveAt removes and returns the track at the given index (0-indexed).
// Returns nil if the index is out of bounds.
// Adjusts currentIndex if removing a track before the current position.
func (q *Queue) RemoveAt(index int) *Track {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index < 0 || index >= len(q.tracks) {
		return nil
	}

	track := q.tracks[index]
	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)

	// Adjust currentIndex if we removed a track before or at current position
	if index < q.currentIndex {
		q.currentIndex--
	} else if index == q.currentIndex {
		// Removed current track; index now points to the next track (or past end)
		// Keep index as-is; caller should handle this case
		if q.currentIndex >= len(q.tracks) && len(q.tracks) > 0 {
			// If we removed the last track and there are still tracks, adjust
			q.currentIndex = len(q.tracks) - 1
		}
	}

	return track
}

// Clear removes all tracks from the queue and resets the index.
// Returns the number of tracks that were removed.
func (q *Queue) Clear() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	count := len(q.tracks)
	q.tracks = make([]*Track, 0)
	q.currentIndex = -1
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

// Len returns the total number of tracks in the queue.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.tracks)
}

// IsEmpty returns true if the queue has no tracks.
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

// IsIdle returns true if there is no current track (not started or past end).
func (q *Queue) IsIdle() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.currentIndex < 0 || q.currentIndex >= len(q.tracks)
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

// Prepend adds a track to the front of the queue (index 0) and sets currentIndex to 0.
// This is used when starting playback of a new track immediately.
func (q *Queue) Prepend(track *Track) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tracks = append([]*Track{track}, q.tracks...)
	q.currentIndex = 0
}

// ClearAfterCurrent removes all tracks after the current track.
// Returns the number of tracks removed.
func (q *Queue) ClearAfterCurrent() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentIndex < 0 || q.currentIndex >= len(q.tracks)-1 {
		return 0
	}

	count := len(q.tracks) - q.currentIndex - 1
	q.tracks = q.tracks[:q.currentIndex+1]
	return count
}
