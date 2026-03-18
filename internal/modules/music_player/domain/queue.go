package domain

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

// QueueID uniquely identifies a queue.
type QueueID string

// Domain errors for QueueID invariants.
var (
	// ErrInvalidQueueID is returned when an invalid string is passed to ParseQueueID.
	ErrInvalidQueueID = errors.New("invalid queue id")
)

// NewQueueID generates a new unique QueueID.
func NewQueueID() QueueID {
	id := uuid.Must(uuid.NewV7())
	return QueueID(id.String())
}

// ParseQueueID converts a queue id string to a QueueID.
func ParseQueueID(id string) (QueueID, error) {
	if u, err := uuid.Parse(id); err != nil || u.Version() != 7 {
		return "", ErrInvalidQueueID
	}
	return QueueID(id), nil
}

// QueueEntry represents a track's placement in the queue, associating a track with who requested it and when.
type QueueEntry struct {
	track       Track
	requesterID UserID
	addedAt     time.Time
	isAutoPlay  bool
}

// NewQueueEntry creates a new QueueEntry with the provided metadata.
func NewQueueEntry(
	track Track,
	requesterID UserID,
	isAutoPlay bool,
) QueueEntry {
	addedAt := time.Now()
	return QueueEntry{
		track,
		requesterID,
		addedAt,
		isAutoPlay,
	}
}

// ConstructQueueEntry recreates a QueueEntry from persisted data.
func ConstructQueueEntry(
	track Track,
	requesterID UserID,
	addedAt time.Time,
	isAutoPlay bool,
) QueueEntry {
	return QueueEntry{
		track,
		requesterID,
		addedAt,
		isAutoPlay,
	}
}

// Track returns the queued track.
func (e *QueueEntry) Track() *Track {
	return &e.track
}

// RequesterID returns the ID of the user who requested this entry.
func (e *QueueEntry) RequesterID() UserID {
	return e.requesterID
}

// AddedAt returns the time this entry was added to the queue.
func (e *QueueEntry) AddedAt() time.Time {
	return e.addedAt
}

// IsAutoPlay returns true if this entry was added by auto-play.
func (e *QueueEntry) IsAutoPlay() bool {
	return e.isAutoPlay
}

// Queue is a ordered collection of track entries.
type Queue struct {
	id      QueueID
	entries []QueueEntry
}

// Domain errors for Queue.
var (
	ErrInvalidIndex = errors.New("invalid index provided")
)

// NewQueue creates a new empty Queue.
func NewQueue() Queue {
	return Queue{
		id:      NewQueueID(),
		entries: make([]QueueEntry, 0),
	}
}

// ConstructQueue recreates a Queue from persisted data.
func ConstructQueue(id QueueID, entries []QueueEntry) Queue {
	return Queue{id, entries}
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

// ManualPlayEntries returns a copy of all entries that were manually requested.
func (q *Queue) ManualPlayEntries() []QueueEntry {
	result := make([]QueueEntry, 0, q.Len())
	for _, e := range q.entries {
		if !e.isAutoPlay {
			result = append(result, e)
		}
	}
	return result
}

// AutoPlayEntries returns a copy of all entries that were added by auto-play.
func (q *Queue) AutoPlayEntries() []QueueEntry {
	result := make([]QueueEntry, 0, q.Len())
	for _, e := range q.entries {
		if e.isAutoPlay {
			result = append(result, e)
		}
	}
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

// Insert inserts entries at the given index, shifting existing entries to the right.
// Returns ErrInvalidIndex if the index is out of bounds.
func (q *Queue) Insert(index int, entries ...QueueEntry) error {
	if !q.isValidIndex(index) {
		return ErrInvalidIndex
	}

	q.entries = append(q.entries[:index], append(entries, q.entries[index:]...)...)

	return nil
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

// Shuffle randomizes the order of entries in the queue.
func (q *Queue) Shuffle() {
	rand.Shuffle(len(q.entries), func(i, j int) {
		q.entries[i], q.entries[j] = q.entries[j], q.entries[i]
	})
}

// QueueRepository defines the interface for storing and retrieving queues.
type QueueRepository interface {
	// FindByID returns the Queue for the given queue ID, or error if not exists.
	FindByID(ctx context.Context, queueID QueueID) (Queue, error)

	// Save stores the Queue.
	Save(ctx context.Context, queue Queue) error

	// Delete removes the Queue for the given queue ID.
	Delete(ctx context.Context, queueID QueueID) error
}
