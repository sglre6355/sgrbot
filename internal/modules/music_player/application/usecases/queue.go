package usecases

import (
	"context"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

const DefaultPageSize = 10

// QueueService handles queue operations.
type QueueService struct {
	playerStates domain.PlayerStateRepository
	publisher    ports.EventPublisher
}

// NewQueueService creates a new QueueService.
func NewQueueService(
	playerStates domain.PlayerStateRepository,
	publisher ports.EventPublisher,
) *QueueService {
	return &QueueService{
		playerStates: playerStates,
		publisher:    publisher,
	}
}

// QueueAddInput contains the input for the QueueAdd use case.
type QueueAddInput struct {
	GuildID     snowflake.ID
	TrackIDs    []string
	RequesterID snowflake.ID
}

// QueueAddOutput contains the result of the QueueAdd use case.
type QueueAddOutput struct {
	StartIndex int // Index where first track was added
	Count      int // Number of tracks added
}

// Add adds tracks to the queue. If the queue was idle, triggers playback via CurrentTrackChangedEvent.
func (q *QueueService) Add(ctx context.Context, input QueueAddInput) (*QueueAddOutput, error) {
	if len(input.TrackIDs) == 0 {
		return &QueueAddOutput{
			StartIndex: 0,
			Count:      0,
		}, nil
	}

	state, err := q.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Capture whether queue was idle before appending
	wasActive := state.IsPlaybackActive()

	startIndex := state.Len()

	// Create entries and append
	entries := make([]domain.QueueEntry, 0, len(input.TrackIDs))
	for _, trackID := range input.TrackIDs {
		entries = append(
			entries,
			domain.NewQueueEntry(domain.TrackID(trackID), input.RequesterID, time.Now()),
		)
	}
	state.Append(entries...)

	if !wasActive {
		state.Seek(startIndex)
	}
	state.SetPlaybackActive(true)

	if err := q.playerStates.Save(ctx, state); err != nil {
		return nil, err
	}

	// Publish event if queue transitioned from idle to having a current track
	if !wasActive {
		err := q.publisher.Publish(domain.NewCurrentTrackChangedEvent(input.GuildID))
		if err != nil {
			return nil, err
		}
	}

	return &QueueAddOutput{
		StartIndex: startIndex,
		Count:      len(input.TrackIDs),
	}, nil
}

// QueueListInput contains the input for the QueueList use case.
type QueueListInput struct {
	GuildID  snowflake.ID
	Page     int // 1-indexed page number
	PageSize int // Items per page (optional, defaults to 10)
}

// QueueListOutput contains the result of the QueueList use case.
type QueueListOutput struct {
	PlayedTrackIDs   []string // Track IDs before the current track on this page
	CurrentTrackID   string   // Current track ID, empty if not on this page or idle
	UpcomingTrackIDs []string // Track IDs after the current track on this page
	TotalTracks      int      // Total tracks in queue
	CurrentPage      int
	TotalPages       int
	PageStart        int    // Start index of this page
	LoopMode         string // "none", "track", "queue"
}

// List returns the current queue with pagination.
func (q *QueueService) List(ctx context.Context, input QueueListInput) (*QueueListOutput, error) {
	state, err := q.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	// Validate and set defaults
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	// Get all entries and current index
	allEntries := state.List()
	currentIndex := -1
	if state.IsPlaybackActive() {
		currentIndex = state.CurrentIndex()
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
	totalTracks := len(allEntries)
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

	output := &QueueListOutput{
		TotalTracks: totalTracks,
		CurrentPage: page,
		TotalPages:  totalPages,
		PageStart:   start,
		LoopMode:    loopMode.String(),
	}

	if start >= totalTracks {
		return output, nil
	}

	// Split page entries into played/current/upcoming based on currentIndex
	pageEntries := allEntries[start:end]
	for i, entry := range pageEntries {
		absIndex := start + i
		trackID := entry.TrackID.String()

		switch {
		case currentIndex >= 0 && absIndex < currentIndex:
			output.PlayedTrackIDs = append(output.PlayedTrackIDs, trackID)
		case currentIndex >= 0 && absIndex == currentIndex:
			output.CurrentTrackID = trackID
		default:
			output.UpcomingTrackIDs = append(output.UpcomingTrackIDs, trackID)
		}
	}

	return output, nil
}

// QueueRemoveInput contains the input for the QueueRemove use case.
type QueueRemoveInput struct {
	GuildID snowflake.ID
	Index   int // Index in queue (cannot remove current track directly)
}

// QueueRemoveOutput contains the result of the QueueRemove use case.
type QueueRemoveOutput struct {
	RemovedTrackID string
}

// Remove removes a track from the queue at the given index.
// Returns ErrIsCurrentTrack if index is the current track (should use Skip instead).
func (q *QueueService) Remove(
	ctx context.Context,
	input QueueRemoveInput,
) (*QueueRemoveOutput, error) {
	state, err := q.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	if state.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	index := input.Index
	if index < 0 || index >= state.Len() {
		return nil, ErrInvalidIndex
	}

	// Cannot remove current track directly - must use Skip
	if state.IsPlaybackActive() && index == state.CurrentIndex() {
		return nil, ErrIsCurrentTrack
	}

	removedEntry, err := state.Remove(index)
	if err != nil {
		return nil, ErrInvalidIndex
	}

	if err := q.playerStates.Save(ctx, state); err != nil {
		return nil, err
	}

	return &QueueRemoveOutput{
		RemovedTrackID: removedEntry.TrackID.String(),
	}, nil
}

// QueueClearInput contains the input for the QueueClear use case.
type QueueClearInput struct {
	GuildID          snowflake.ID
	KeepCurrentTrack bool // true = clear played+upcoming (keep current), false = clear all
}

// QueueClearOutput contains the result of the QueueClear use case.
type QueueClearOutput struct {
	ClearedCount int
}

// Clear clears the queue.
// If KeepCurrentTrack is true, clears played + upcoming tracks, keeps only current track.
// If KeepCurrentTrack is false, clears all tracks.
func (q *QueueService) Clear(
	ctx context.Context,
	input QueueClearInput,
) (*QueueClearOutput, error) {
	state, err := q.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	var count int
	if input.KeepCurrentTrack {
		// Clear played + upcoming, keep only current track
		currentEntry := state.Current()
		if currentEntry == nil {
			// No current track (idle state) - clear all played tracks
			if state.Len() == 0 {
				return nil, ErrQueueEmpty
			}
			count = state.Len()
			state.Clear()
		} else {
			count = state.Len() - 1
			if count == 0 {
				return nil, ErrNothingToClear
			}
			// Use existing methods: clear all, add back current, activate
			savedEntry := *currentEntry
			state.Clear()
			state.Append(savedEntry)
			state.SetPlaybackActive(true)
		}
	} else {
		// Clear all tracks
		if state.Len() == 0 {
			return nil, ErrQueueEmpty
		}
		count = state.Len()
		state.Clear()
	}

	if err := q.playerStates.Save(ctx, state); err != nil {
		return nil, err
	}

	if !input.KeepCurrentTrack {
		err := q.publisher.Publish(domain.NewCurrentTrackChangedEvent(input.GuildID))
		if err != nil {
			return nil, err
		}
	}

	return &QueueClearOutput{
		ClearedCount: count,
	}, nil
}

// QueueRestartInput contains the input for the QueueRestart use case.
type QueueRestartInput struct {
	GuildID snowflake.ID
}

// QueueRestartOutput contains the result of the QueueRestart use case.
type QueueRestartOutput struct {
	TrackID string
}

// Restart restarts the queue from the beginning.
// Used to replay tracks after the queue has naturally ended.
func (q *QueueService) Restart(
	ctx context.Context,
	input QueueRestartInput,
) (*QueueRestartOutput, error) {
	output, err := q.Seek(ctx, QueueSeekInput{
		GuildID: input.GuildID,
		Index:   0,
	})
	if err != nil {
		return nil, err
	}
	return &QueueRestartOutput{TrackID: output.TrackID}, nil
}

// QueueSeekInput contains the input for the QueueSeek use case.
type QueueSeekInput struct {
	GuildID snowflake.ID
	Index   int // Index in the queue
}

// QueueSeekOutput contains the result of the QueueSeek use case.
type QueueSeekOutput struct {
	TrackID string
}

// Seek jumps to a specific index in the queue and triggers playback.
// Used to immediately play a track at any index (played or upcoming).
func (q *QueueService) Seek(
	ctx context.Context,
	input QueueSeekInput,
) (*QueueSeekOutput, error) {
	state, err := q.playerStates.Get(ctx, input.GuildID)
	if err != nil {
		return nil, ErrNotConnected
	}

	if state.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	// Seek to target index (atomic operation with bounds checking)
	entry := state.Seek(input.Index)
	if entry == nil {
		return nil, ErrInvalidIndex
	}

	// Mark playback as active before saving
	state.SetPlaybackActive(true)

	if err := q.playerStates.Save(ctx, state); err != nil {
		return nil, err
	}

	// Publish event to trigger playback
	if err := q.publisher.Publish(domain.NewCurrentTrackChangedEvent(input.GuildID)); err != nil {
		return nil, err
	}

	return &QueueSeekOutput{TrackID: entry.TrackID.String()}, nil
}
