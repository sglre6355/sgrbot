package domain

// Queue is a queue for managing tracks using an index-based model.
// Instead of removing tracks when they finish, we maintain a currentIndex
// that advances through the track list, enabling loop functionality.
type Queue struct {
	entries      []QueueEntry
	currentIndex int
}

// NewQueue creates a new empty Queue.
func NewQueue() Queue {
	return Queue{
		entries:      make([]QueueEntry, 0),
		currentIndex: 0,
	}
}

// IsEmpty returns true if the queue has no entries.
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

// Len returns the total number of entries in the queue.
func (q *Queue) Len() int {
	return len(q.entries)
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

// Current returns the entry at currentIndex, or nil if the queue is empty.
func (q *Queue) Current() *QueueEntry {
	if q.IsEmpty() {
		return nil
	}
	return &q.entries[q.currentIndex]
}

// Upcoming returns entries after the current index (for queue display).
// Returns empty slice if no entries or no current entry.
func (q *Queue) Upcoming() []QueueEntry {
	if q.IsEmpty() {
		return []QueueEntry{}
	}

	upcoming := q.entries[q.currentIndex+1:]
	result := make([]QueueEntry, len(upcoming))
	copy(result, upcoming)
	return result
}

// List returns a copy of all entries in the queue.
func (q *Queue) List() []QueueEntry {
	result := make([]QueueEntry, q.Len())
	copy(result, q.entries)
	return result
}

// Append adds entries to the end of the queue.
func (q *Queue) Append(entries ...QueueEntry) {
	q.entries = append(q.entries, entries...)
}

// Prepend adds entries to the front of the queue.
func (q *Queue) Prepend(entries ...QueueEntry) {
	q.entries = append(entries, q.entries...)
}

// GetAt returns the entry at the given index without removing it.
// Returns nil if the index is out of bounds.
func (q *Queue) GetAt(index int) *QueueEntry {
	if !q.isValidIndex(index) {
		return nil
	}
	return &q.entries[index]
}

// RemoveAt removes and returns the entry at the given index.
// Returns nil if the index is out of bounds.
// Adjusts currentIndex if removing an entry before the current position.
func (q *Queue) RemoveAt(index int) *QueueEntry {
	if !q.isValidIndex(index) {
		return nil
	}

	entry := q.entries[index]
	q.entries = append(q.entries[:index], q.entries[index+1:]...)

	// Adjust currentIndex if we removed an entry before the current position.
	// If we removed the current entry, keep the index pointing at the next entry,
	// unless we removed the last entry (then move to the new last index).
	if q.IsEmpty() {
		q.currentIndex = 0
	} else if index < q.currentIndex {
		q.currentIndex--
	} else if index == q.currentIndex && q.currentIndex >= q.Len() {
		q.currentIndex = q.Len() - 1
	}

	return &entry
}

// Seek sets the currentIndex to the specified index.
// Returns the entry at that index, or nil if index is out of bounds.
// Does not change currentIndex if index is invalid.
func (q *Queue) Seek(index int) *QueueEntry {
	if !q.isValidIndex(index) {
		return nil
	}

	q.currentIndex = index
	return &q.entries[index]
}

// Advance moves to the next track based on loop mode.
// Returns the new current entry, or nil if queue ended.
//   - LoopModeNone: advance index, return nil if past end
//   - LoopModeTrack: don't advance, return same entry
//   - LoopModeQueue: advance, wrap to 0 if past end
func (q *Queue) Advance(mode LoopMode) *QueueEntry {
	if q.IsEmpty() {
		return nil
	}

	switch mode {
	case LoopModeTrack:
		// Don't modify currentIndex, return same entry

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

	return &q.entries[q.currentIndex]
}

// Clear removes all entries from the queue and resets the index.
func (q *Queue) Clear() {
	q.entries = make([]QueueEntry, 0)
	q.currentIndex = 0
}
