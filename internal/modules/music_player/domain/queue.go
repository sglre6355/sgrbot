package domain

import (
	"context"
	"errors"
)

// QueueID uniquely identifies a queue.
type QueueID string

// Queue is a ordered collection of track entries.
type Queue struct {
	ID      QueueID
	entries []QueueEntry
}

// Queue errors
var ErrInvalidIndex = errors.New("invalid index provided")

// NewQueue creates a new empty Queue.
func NewQueue() Queue {
	return Queue{
		entries: make([]QueueEntry, 0),
	}
}

// isValidIndex returns true if the index is within bounds.
func (q *Queue) isValidIndex(index int) bool {
	return 0 <= index && index < q.Len()
}

// Len returns the total number of entries in the queue.
func (q *Queue) Len() int {
	return len(q.entries)
}

// IsEmpty returns true if the queue has no entries.
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

// List returns a copy of all entries in the queue.
func (q *Queue) List() []QueueEntry {
	result := make([]QueueEntry, q.Len())
	copy(result, q.entries)
	return result
}

// Prepend adds entries to the front of the queue.
func (q *Queue) Prepend(entries ...QueueEntry) {
	q.entries = append(entries, q.entries...)
}

// Append adds entries to the end of the queue.
func (q *Queue) Append(entries ...QueueEntry) {
	q.entries = append(q.entries, entries...)
}

// Get returns the entry at the given index without removing it.
// Returns error if the index is out of bounds.
func (q *Queue) Get(index int) (*QueueEntry, error) {
	if !q.isValidIndex(index) {
		return nil, ErrInvalidIndex
	}
	return &q.entries[index], nil
}

// Remove removes and returns the entry at the given index.
// Returns error if the index is out of bounds.
func (q *Queue) Remove(index int) (*QueueEntry, error) {
	if !q.isValidIndex(index) {
		return nil, ErrInvalidIndex
	}

	entry := q.entries[index]
	q.entries = append(q.entries[:index], q.entries[index+1:]...)

	return &entry, nil
}

// Clear removes all entries from the queue.
func (q *Queue) Clear() {
	q.entries = make([]QueueEntry, 0)
}

// QueueRepository defines the interface for storing and retrieving queues.
type QueueRepository interface {
	// Get returns the Queue for the given queue ID, or error if not exists.
	Get(ctx context.Context, queueID QueueID) (Queue, error)

	// Save stores the Queue.
	Save(ctx context.Context, queue Queue) error

	// Delete removes the Queue for the given queue ID.
	Delete(ctx context.Context, queueID QueueID) error
}
