package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

const DefaultPageSize = 10

// QueueAddInput contains the input for the QueueAdd use case.
type QueueAddInput struct {
	GuildID               snowflake.ID
	Track                 *domain.Track
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueAddOutput contains the result of the QueueAdd use case.
type QueueAddOutput struct {
	Position int // 0-indexed position in queue (0 = now playing)
}

// QueueListInput contains the input for the QueueList use case.
type QueueListInput struct {
	GuildID               snowflake.ID
	Page                  int          // 1-indexed page number
	PageSize              int          // Items per page (optional, defaults to 10)
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueListOutput contains the result of the QueueList use case.
type QueueListOutput struct {
	CurrentTrack *domain.Track
	Tracks       []*domain.Track
	TotalTracks  int
	CurrentPage  int
	TotalPages   int
}

// QueueRemoveInput contains the input for the QueueRemove use case.
type QueueRemoveInput struct {
	GuildID               snowflake.ID
	Position              int          // 0-indexed position in queue (position 0 should be handled by Skip)
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueRemoveOutput contains the result of the QueueRemove use case.
type QueueRemoveOutput struct {
	RemovedTrack *domain.Track
}

// QueueClearInput contains the input for the QueueClear use case.
type QueueClearInput struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueClearOutput contains the result of the QueueClear use case.
type QueueClearOutput struct {
	ClearedCount int
}

// QueueService handles queue operations.
type QueueService struct {
	repo      domain.PlayerStateRepository
	publisher ports.EventPublisher
}

// NewQueueService creates a new QueueService.
func NewQueueService(
	repo domain.PlayerStateRepository,
	publisher ports.EventPublisher,
) *QueueService {
	return &QueueService{
		repo:      repo,
		publisher: publisher,
	}
}

// Add adds a track to the queue and publishes an event to trigger playback if idle.
func (q *QueueService) Add(_ context.Context, input QueueAddInput) (*QueueAddOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	wasIdle := state.IsIdle()
	state.Queue.Add(input.Track)
	// Position is 0-indexed: Queue[0] = position 0, Queue[1] = position 1, etc.
	position := state.Queue.Len() - 1

	// Publish event - PlaybackEventHandler will start playback if wasIdle
	if q.publisher != nil {
		q.publisher.PublishTrackEnqueued(ports.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   input.Track,
			WasIdle: wasIdle,
		})
	}

	return &QueueAddOutput{
		Position: position,
	}, nil
}

// List returns the current queue with pagination.
func (q *QueueService) List(input QueueListInput) (*QueueListOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	// Validate and set defaults
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}

	// Get all tracks from queue
	allTracks := state.Queue.List()

	// Separate current track (Queue[0]) from queued tracks (Queue[1:])
	var currentTrack *domain.Track
	var queuedTracks []*domain.Track
	if len(allTracks) > 0 {
		currentTrack = allTracks[0]
		queuedTracks = allTracks[1:]
	}

	// Pagination applies to queued tracks only
	totalTracks := len(queuedTracks)
	totalPages := (totalTracks + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// Clamp page to valid range
	if page > totalPages {
		page = totalPages
	}

	// Calculate slice bounds
	start := (page - 1) * pageSize
	end := start + pageSize
	end = min(end, totalTracks)

	var pageTracks []*domain.Track
	if start < totalTracks {
		pageTracks = queuedTracks[start:end]
	}

	return &QueueListOutput{
		CurrentTrack: currentTrack,
		Tracks:       pageTracks,
		TotalTracks:  totalTracks,
		CurrentPage:  page,
		TotalPages:   totalPages,
	}, nil
}

// Remove removes a track from the queue at the given position.
// Position 0 should be handled by Skip (current track requires playback control).
// Position 1+ = queued tracks (Queue[1], Queue[2], etc.).
func (q *QueueService) Remove(input QueueRemoveInput) (*QueueRemoveOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	// Queue[0] is current track, Queue[1:] are queued tracks
	// If only current track or less, no queued tracks to remove
	if state.Queue.Len() <= 1 {
		return nil, ErrQueueEmpty
	}

	// Position 0 is current track - should be handled by Skip at the handler level
	// Position 1+ maps directly to Queue index
	index := input.Position
	if index < 1 {
		return nil, ErrInvalidPosition
	}

	track := state.Queue.RemoveAt(index)
	if track == nil {
		return nil, ErrInvalidPosition
	}

	return &QueueRemoveOutput{
		RemovedTrack: track,
	}, nil
}

// Clear clears all queued tracks (keeps current track at Queue[0]).
func (q *QueueService) Clear(input QueueClearInput) (*QueueClearOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	// Queue[0] is current track, Queue[1:] are queued tracks
	// If only current track or less, no queued tracks to clear
	if state.Queue.Len() <= 1 {
		return nil, ErrQueueEmpty
	}

	count := state.Queue.ClearAfterCurrent()

	return &QueueClearOutput{
		ClearedCount: count,
	}, nil
}
