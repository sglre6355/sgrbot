package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

const defaultPageSize = 10

// ListQueueInput holds the input for the ListQueue use case.
type ListQueueInput[C comparable] struct {
	ConnectionInfo C
	Page           int
	PageSize       int
}

// ListQueueOutput holds the output for the ListQueue use case.
type ListQueueOutput struct {
	PlayedTracks    []dtos.TrackView
	CurrentTrack    *dtos.TrackView
	UpcomingTracks  []dtos.TrackView
	LoopMode        string
	AutoPlayEnabled bool
	PageStart       int
	TotalTracks     int
	CurrentPage     int
	TotalPages      int
}

// ListQueue returns a paginated view of the queue.
type ListQueueUsecase[C comparable] struct {
	playerStates       domain.PlayerStateRepository
	playerStateLocator ports.PlayerStateLocator[C]
}

// NewListQueue creates a new ListQueue use case.
func NewListQueueUsecase[C comparable](
	playerStates domain.PlayerStateRepository,
	playerStateLocator ports.PlayerStateLocator[C],
) *ListQueueUsecase[C] {
	return &ListQueueUsecase[C]{
		playerStates:       playerStates,
		playerStateLocator: playerStateLocator,
	}
}

// Execute returns the paginated queue view.
func (uc *ListQueueUsecase[C]) Execute(
	ctx context.Context,
	input ListQueueInput[C],
) (*ListQueueOutput, error) {
	id := uc.playerStateLocator.FindPlayerStateID(ctx, input.ConnectionInfo)
	if id == nil {
		return nil, ErrNotConnected
	}
	playerStateID := *id

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	ps := defaultPageSize
	if input.PageSize > 0 {
		ps = input.PageSize
	}

	output := &ListQueueOutput{
		LoopMode:        state.LoopMode().String(),
		AutoPlayEnabled: state.IsAutoPlayEnabled(),
		TotalTracks:     state.Len(),
	}

	if output.TotalTracks == 0 {
		output.CurrentPage = 1
		output.TotalPages = 1
		return output, nil
	}

	totalPages := (output.TotalTracks + ps - 1) / ps
	output.TotalPages = totalPages

	page := input.Page
	if page <= 0 {
		if state.IsPlaybackActive() {
			page = (state.CurrentIndex() / ps) + 1
		} else {
			page = 1
		}
	}
	if page > totalPages {
		page = totalPages
	}
	output.CurrentPage = page

	pageStart := (page - 1) * ps
	pageEnd := min(pageStart+ps, output.TotalTracks)
	output.PageStart = pageStart

	played := state.Played()
	current := state.Current()
	upcoming := state.Upcoming()
	currentOffset := len(played)

	playedStart := min(pageStart, len(played))
	playedEnd := min(pageEnd, len(played))
	for _, entry := range played[playedStart:playedEnd] {
		output.PlayedTracks = append(output.PlayedTracks, dtos.NewTrackView(entry.Track()))
	}

	if current != nil && pageStart <= currentOffset && currentOffset < pageEnd {
		view := dtos.NewTrackView(current.Track())
		output.CurrentTrack = &view
	}

	upcomingStart := max(0, pageStart-currentOffset-1)
	upcomingEnd := min(max(0, pageEnd-currentOffset-1), len(upcoming))
	for _, entry := range upcoming[upcomingStart:upcomingEnd] {
		output.UpcomingTracks = append(output.UpcomingTracks, dtos.NewTrackView(entry.Track()))
	}

	return output, nil
}
