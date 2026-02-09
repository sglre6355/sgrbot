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
	repo          domain.PlayerStateRepository
	publisher     ports.EventPublisher
	trackProvider ports.TrackProvider
}

// NewQueueService creates a new QueueService.
func NewQueueService(
	repo domain.PlayerStateRepository,
	publisher ports.EventPublisher,
	trackProvider ports.TrackProvider,
) *QueueService {
	return &QueueService{
		repo:          repo,
		publisher:     publisher,
		trackProvider: trackProvider,
	}
}

// Add adds a track to the queue and publishes an event to trigger playback if idle.
func (q *QueueService) Add(ctx context.Context, input QueueAddInput) (*QueueAddOutput, error) {
	state, err := q.repo.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	state.Queue.Append(input.Track.ID)
	// Position is 0-indexed: Queue[0] = position 0, Queue[1] = position 1, etc.
	position := state.Queue.Len() - 1

	if err := q.repo.Save(ctx, state); err != nil {
		return nil, err
	}

	// Publish event - PlaybackEventHandler will start playback if idle
	if q.publisher != nil {
		q.publisher.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   input.Track,
		})
	}

	return &QueueAddOutput{
		Position: position,
	}, nil
}

// AddMultiple adds multiple tracks to the queue atomically.
// Publishes a single TrackEnqueuedEvent for the first track to trigger playback if idle.
func (q *QueueService) AddMultiple(
	ctx context.Context,
	input QueueAddMultipleInput,
) (*QueueAddMultipleOutput, error) {
	if len(input.Tracks) == 0 {
		return &QueueAddMultipleOutput{
			StartPosition: 0,
			Count:         0,
		}, nil
	}

	state, err := q.repo.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	startPosition := state.Queue.Len()

	// Collect IDs and append
	ids := make([]domain.TrackID, 0, len(input.Tracks))
	for _, track := range input.Tracks {
		ids = append(ids, track.ID)
	}
	state.Queue.Append(ids...)

	if err := q.repo.Save(ctx, state); err != nil {
		return nil, err
	}

	// Publish single event for the first track - PlaybackEventHandler will start playback if idle
	if q.publisher != nil {
		q.publisher.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   input.Tracks[0],
		})
	}

	return &QueueAddMultipleOutput{
		StartPosition: startPosition,
		Count:         len(input.Tracks),
	}, nil
}

// List returns the current queue with pagination.
func (q *QueueService) List(ctx context.Context, input QueueListInput) (*QueueListOutput, error) {
	state, err := q.repo.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
		_ = q.repo.Save(ctx, state)
	}

	// Validate and set defaults
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	// Get all track IDs and current index
	allTrackIDs := state.Queue.List()
	currentIndex := -1
	if state.IsPlaybackActive() {
		currentIndex = state.Queue.CurrentIndex()
	}
	loopMode := state.GetLoopMode()

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
	totalTracks := len(allTrackIDs)
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
		pageIDs := allTrackIDs[start:end]
		tracks, err := q.trackProvider.LoadTracks(pageIDs...)
		if err != nil {
			return nil, err
		}
		pageTracks = make([]*domain.Track, len(tracks))
		for i := range tracks {
			pageTracks[i] = &tracks[i]
		}
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
func (q *QueueService) Remove(
	ctx context.Context,
	input QueueRemoveInput,
) (*QueueRemoveOutput, error) {
	state, err := q.repo.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	if state.Queue.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	index := input.Position
	if index < 0 || index >= state.Queue.Len() {
		return nil, ErrInvalidPosition
	}

	// Cannot remove current track directly - must use Skip
	if state.IsPlaybackActive() && index == state.Queue.CurrentIndex() {
		return nil, ErrIsCurrentTrack
	}

	removedID := state.Queue.RemoveAt(index)
	if removedID == nil {
		return nil, ErrInvalidPosition
	}

	track, err := q.trackProvider.LoadTrack(*removedID)
	if err != nil {
		return nil, err
	}

	if err := q.repo.Save(ctx, state); err != nil {
		return nil, err
	}

	return &QueueRemoveOutput{
		RemovedTrack: &track,
	}, nil
}

// Clear clears the queue.
// If KeepCurrentTrack is true, clears played + upcoming tracks, keeps only current track.
// If KeepCurrentTrack is false, clears all tracks.
func (q *QueueService) Clear(
	ctx context.Context,
	input QueueClearInput,
) (*QueueClearOutput, error) {
	state, err := q.repo.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	var count int
	if input.KeepCurrentTrack {
		// Clear played + upcoming, keep only current track
		currentTrackID := state.CurrentTrackID()
		if currentTrackID == nil {
			// No current track (idle state) - clear all played tracks
			if state.Queue.Len() == 0 {
				return nil, ErrQueueEmpty
			}
			count = state.Queue.Len()
			state.Queue.Clear()
		} else {
			count = state.Queue.Len() - 1
			if count == 0 {
				return nil, ErrNothingToClear
			}
			// Use existing methods: clear all, add back current, activate
			savedID := *currentTrackID
			state.Queue.Clear()
			state.Queue.Append(savedID)
		}
	} else {
		// Clear all tracks
		if state.Queue.Len() == 0 {
			return nil, ErrQueueEmpty
		}
		count = state.Queue.Len()
		state.Queue.Clear()

		// Publish event to stop playback
		if q.publisher != nil {
			q.publisher.PublishQueueCleared(domain.QueueClearedEvent{
				GuildID:               input.GuildID,
				NotificationChannelID: input.NotificationChannelID,
			})
		}
	}

	if err := q.repo.Save(ctx, state); err != nil {
		return nil, err
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
	ctx context.Context,
	input QueueSeekInput,
) (*QueueSeekOutput, error) {
	state, err := q.repo.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Update notification channel if provided
	if input.NotificationChannelID != 0 {
		state.SetNotificationChannelID(input.NotificationChannelID)
	}

	if state.Queue.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	// Seek to target position (atomic operation with bounds checking)
	trackID := state.Queue.Seek(input.Position)
	if trackID == nil {
		return nil, ErrInvalidPosition
	}

	track, err := q.trackProvider.LoadTrack(*trackID)
	if err != nil {
		return nil, err
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
	state.SetPlaybackActive(false)

	if err := q.repo.Save(ctx, state); err != nil {
		return nil, err
	}

	// Publish event to trigger playback.
	// PlayNext will see currentIndex >= 0 (not idle) and play Current() directly,
	// which is the track we just seeked to.
	if q.publisher != nil {
		q.publisher.PublishTrackEnqueued(domain.TrackEnqueuedEvent{
			GuildID: input.GuildID,
			Track:   &track,
		})
	}

	return &QueueSeekOutput{Track: &track}, nil
}
