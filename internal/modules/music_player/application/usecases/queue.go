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
	Position int // 0-indexed position in queue where track was added
}

// QueueAddMultipleInput contains the input for adding multiple tracks.
type QueueAddMultipleInput struct {
	GuildID               snowflake.ID
	Tracks                []*domain.Track
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueAddMultipleOutput contains the result of adding multiple tracks.
type QueueAddMultipleOutput struct {
	StartPosition int // 0-indexed position where first track was added
	Count         int // Number of tracks added
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
	Tracks       []*domain.Track // Paginated slice of all tracks
	CurrentIndex int             // 0-indexed position of current track (-1 if idle)
	TotalTracks  int             // Total tracks in queue
	CurrentPage  int
	TotalPages   int
	PageStart    int    // 0-indexed start position of this page
	LoopMode     string // "none", "track", "queue"
}

// QueueRemoveInput contains the input for the QueueRemove use case.
type QueueRemoveInput struct {
	GuildID               snowflake.ID
	Position              int          // 0-indexed position in queue (cannot remove current track directly)
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
	KeepCurrentTrack      bool         // true = clear played+upcoming (keep current), false = clear all
}

// QueueClearOutput contains the result of the QueueClear use case.
type QueueClearOutput struct {
	ClearedCount int
}

// QueueRestartInput contains the input for the QueueRestart use case.
type QueueRestartInput struct {
	GuildID               snowflake.ID
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueRestartOutput contains the result of the QueueRestart use case.
type QueueRestartOutput struct {
	Track *domain.Track
}

// QueueSeekInput contains the input for the QueueSeek use case.
type QueueSeekInput struct {
	GuildID               snowflake.ID
	Position              int          // 0-indexed position in the queue
	NotificationChannelID snowflake.ID // Optional: updates notification channel if non-zero
}

// QueueSeekOutput contains the result of the QueueSeek use case.
type QueueSeekOutput struct {
	Track *domain.Track
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
		q.publisher.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   input.Track,
			WasIdle: wasIdle,
		})
	}

	return &QueueAddOutput{
		Position: position,
	}, nil
}

// AddMultiple adds multiple tracks to the queue atomically.
// Publishes a single TrackEnqueuedEvent for the first track to trigger playback if idle.
func (q *QueueService) AddMultiple(
	_ context.Context,
	input QueueAddMultipleInput,
) (*QueueAddMultipleOutput, error) {
	if len(input.Tracks) == 0 {
		return &QueueAddMultipleOutput{
			StartPosition: 0,
			Count:         0,
		}, nil
	}

	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	startPosition := state.Queue.Len()
	wasIdle := state.Queue.AddMultiple(input.Tracks)

	// Publish single event for the first track - PlaybackEventHandler will start playback if wasIdle
	if q.publisher != nil {
		q.publisher.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   input.Tracks[0],
			WasIdle: wasIdle,
		})
	}

	return &QueueAddMultipleOutput{
		StartPosition: startPosition,
		Count:         len(input.Tracks),
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

	// Get all tracks and current index
	allTracks := state.Queue.List()
	currentIndex := state.Queue.CurrentIndex()
	loopMode := state.LoopMode()

	// Default to page containing the current track (or page 1 if idle)
	page := input.Page
	if page <= 0 {
		if currentIndex >= 0 {
			page = (currentIndex / pageSize) + 1
		} else {
			page = 1
		}
	}

	// Pagination applies to entire queue
	totalTracks := len(allTracks)
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
		pageTracks = allTracks[start:end]
	}

	return &QueueListOutput{
		Tracks:       pageTracks,
		CurrentIndex: currentIndex,
		TotalTracks:  totalTracks,
		CurrentPage:  page,
		TotalPages:   totalPages,
		PageStart:    start,
		LoopMode:     loopMode.String(),
	}, nil
}

// Remove removes a track from the queue at the given position (0-indexed).
// Returns ErrIsCurrentTrack if position is the current track (should use Skip instead).
func (q *QueueService) Remove(input QueueRemoveInput) (*QueueRemoveOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	if state.Queue.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	index := input.Position
	if index < 0 || index >= state.Queue.Len() {
		return nil, ErrInvalidPosition
	}

	// Cannot remove current track directly - must use Skip
	if index == state.Queue.CurrentIndex() {
		return nil, ErrIsCurrentTrack
	}

	track := state.Queue.RemoveAt(index)
	if track == nil {
		return nil, ErrInvalidPosition
	}

	return &QueueRemoveOutput{
		RemovedTrack: track,
	}, nil
}

// Clear clears the queue.
// If KeepCurrentTrack is true, clears played + upcoming tracks, keeps only current track.
// If KeepCurrentTrack is false, clears all tracks.
func (q *QueueService) Clear(input QueueClearInput) (*QueueClearOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	var count int
	if input.KeepCurrentTrack {
		// Clear played + upcoming, keep only current track
		currentTrack := state.Queue.Current()
		if currentTrack == nil {
			// No current track (idle state) - clear all played tracks
			if state.Queue.Len() == 0 {
				return nil, ErrQueueEmpty
			}
			count = state.Queue.Clear()
		} else {
			count = state.Queue.Len() - 1
			if count == 0 {
				return nil, ErrNothingToClear
			}
			// Use existing methods: clear all, add back current, start
			state.Queue.Clear()
			state.Queue.Add(currentTrack)
			state.Queue.Start()
		}
	} else {
		// Clear all tracks
		if state.Queue.Len() == 0 {
			return nil, ErrQueueEmpty
		}
		count = state.Queue.Clear()

		// Publish event to stop playback
		if q.publisher != nil {
			q.publisher.PublishQueueCleared(domain.QueueClearedEvent{
				GuildID:               input.GuildID,
				NotificationChannelID: input.NotificationChannelID,
			})
		}
	}

	return &QueueClearOutput{
		ClearedCount: count,
	}, nil
}

// Restart restarts the queue from the beginning.
// Used to replay tracks after the queue has naturally ended.
func (q *QueueService) Restart(
	ctx context.Context,
	input QueueRestartInput,
) (*QueueRestartOutput, error) {
	output, err := q.Seek(ctx, QueueSeekInput{
		GuildID:               input.GuildID,
		Position:              0,
		NotificationChannelID: input.NotificationChannelID,
	})
	if err != nil {
		return nil, err
	}
	return &QueueRestartOutput{Track: output.Track}, nil
}

// Seek jumps to a specific position in the queue and triggers playback.
// Used to immediately play a track at any position (played or upcoming).
func (q *QueueService) Seek(
	_ context.Context,
	input QueueSeekInput,
) (*QueueSeekOutput, error) {
	state := q.repo.Get(input.GuildID)
	if state == nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannel(input.NotificationChannelID)
	}

	if state.Queue.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	// Seek to target position (atomic operation with bounds checking)
	track := state.Queue.Seek(input.Position)
	if track == nil {
		return nil, ErrInvalidPosition
	}

	// Delete the old "Now Playing" message before starting the new track
	if q.publisher != nil {
		nowPlayingMsg := state.GetNowPlayingMessage()
		if nowPlayingMsg != nil {
			q.publisher.PublishPlaybackFinished(domain.PlaybackFinishedEvent{
				GuildID:               input.GuildID,
				NotificationChannelID: nowPlayingMsg.ChannelID,
				LastMessageID:         &nowPlayingMsg.MessageID,
			})
		}
	}

	// Mark playback as inactive so the event handler will trigger new playback
	state.StopPlayback()

	// Publish event to trigger playback.
	// PlayNext will see currentIndex >= 0 (not idle) and play Current() directly,
	// which is the track we just seeked to.
	if q.publisher != nil {
		q.publisher.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   track,
			WasIdle: true,
		})
	}

	return &QueueSeekOutput{Track: track}, nil
}
