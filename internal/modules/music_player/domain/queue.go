package domain

// Queue is a queue for managing tracks using an index-based model.
// Instead of removing tracks when they finish, we maintain a currentIndex
// that advances through the track list, enabling loop functionality.
type Queue struct {
	trackIDs     []TrackID
	currentIndex int
}

// NewQueue creates a new empty Queue.
func NewQueue() Queue {
	return Queue{
		trackIDs:     make([]TrackID, 0),
		currentIndex: 0,
	}
}

// IsEmpty returns true if the queue has no track IDs.
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

// IsAtLast returns true if the current track is the last in the queue.
func (q *Queue) IsAtLast() bool {
	return q.Len() <= q.currentIndex+1
}

func (q *Queue) isValidIndex(index int) bool {
	return 0 <= index && index < q.Len()
}

// Len returns the total number of track IDs in the queue.
func (q *Queue) Len() int {
	return len(q.trackIDs)
}

// CurrentIndex returns the current track index.
func (q *Queue) CurrentIndex() int {
	return q.currentIndex
}

// HasNext returns true if there's a next track available (considering loop mode).
func (q *Queue) HasNext(mode LoopMode) bool {
	if q.IsEmpty() {
		return false
	}

	switch mode {
	case LoopModeTrack, LoopModeQueue:
		// Always has next when looping (as long as queue isn't empty and not idle)
		return true
	default: // LoopModeNone
		return !q.IsAtLast()
	}
}

// Current returns the track ID at currentIndex, or nil if the queue is empty.
func (q *Queue) Current() *TrackID {
	if q.IsEmpty() {
		return nil
	}
	return &q.trackIDs[q.currentIndex]
}

// Upcoming returns track IDs after the current index (for queue display).
// Returns empty slice if no track IDs or no current track.
func (q *Queue) Upcoming() []TrackID {
	upcoming := q.trackIDs[q.currentIndex+1:]
	result := make([]TrackID, len(upcoming))
	copy(result, upcoming)
	return result
}

// List returns a copy of all track IDs in the queue.
func (q *Queue) List() []TrackID {
	result := make([]TrackID, q.Len())
	copy(result, q.trackIDs)
	return result
}

// Append adds track ID(s) to the end of the queue.
func (q *Queue) Append(trackIDs ...TrackID) {
	q.trackIDs = append(q.trackIDs, trackIDs...)
}

// Prepend adds track ID(s) to the front of the queue.
func (q *Queue) Prepend(trackIDs ...TrackID) {
	q.trackIDs = append(trackIDs, q.trackIDs...)
}

// GetAt returns the track ID at the given index without removing it.
// Returns nil if the index is out of bounds.
func (q *Queue) GetAt(index int) *TrackID {
	if !q.isValidIndex(index) {
		return nil
	}
	return &q.trackIDs[index]
}

// RemoveAt removes and returns the track at the given index.
// Returns nil if the index is out of bounds.
// Adjusts currentIndex if removing a track before the current position.
func (q *Queue) RemoveAt(index int) *TrackID {
	if !q.isValidIndex(index) {
		return nil
	}

	trackID := q.trackIDs[index]
	q.trackIDs = append(q.trackIDs[:index], q.trackIDs[index+1:]...)

	// Adjust currentIndex if we removed a track before the current position.
	// If we removed the current track, keep the index pointing at the next track,
	// unless we removed the last track (then move to the new last index).
	if q.IsEmpty() {
		q.currentIndex = 0
	} else if index < q.currentIndex {
		q.currentIndex--
	} else if index == q.currentIndex && q.currentIndex >= q.Len() {
		q.currentIndex = q.Len() - 1
	}

	return &trackID
}

// Seek sets the currentIndex to the specified index.
// Returns the track at that index, or nil if index is out of bounds.
// Does not change currentIndex if index is invalid.
func (q *Queue) Seek(index int) *TrackID {
	if !q.isValidIndex(index) {
		return nil
	}

	q.currentIndex = index
	return &q.trackIDs[index]
}

// Advance moves to the next track based on loop mode.
// Returns the new current track, or nil if queue ended.
//   - LoopModeNone: advance index, return nil if past end
//   - LoopModeTrack: don't advance, return same track
//   - LoopModeQueue: advance, wrap to 0 if past end
func (q *Queue) Advance(mode LoopMode) *TrackID {
	if q.IsEmpty() {
		return nil
	}

	switch mode {
	case LoopModeTrack:
		// Don't modify currentIndex, return same track

	case LoopModeQueue:
		// Advance with wrap-around
		if q.IsAtLast() {
			q.currentIndex = 0
		} else {
			q.currentIndex++
		}

	default: // LoopModeNone
		// Advance if current track is not the last in the queue
		if q.IsAtLast() {
			return nil
		}
		q.currentIndex++
	}

	return &q.trackIDs[q.currentIndex]
}

// Clear removes all track IDs from the queue and resets the index.
func (q *Queue) Clear() {
	q.trackIDs = make([]TrackID, 0)
	q.currentIndex = 0
}
