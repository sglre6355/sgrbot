package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain/core"
)

const defaultPageSize = 10

// ListQueueInput holds the input for the ListQueue use case.
type ListQueueInput struct {
	PlayerStateID string
	Page          int
	PageSize      int
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
type ListQueueUsecase struct {
	playerStates core.PlayerStateRepository
}

// NewListQueue creates a new ListQueue use case.
func NewListQueueUsecase(playerStates core.PlayerStateRepository) *ListQueueUsecase {
	return &ListQueueUsecase{playerStates: playerStates}
}

// Execute returns the paginated queue view.
func (uc *ListQueueUsecase) Execute(
	ctx context.Context,
	input ListQueueInput,
) (*ListQueueOutput, error) {
	playerStateID, err := core.ParsePlayerStateID(input.PlayerStateID)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	state, err := uc.playerStates.FindByID(ctx, playerStateID)
	if err != nil {
		if errors.Is(err, core.ErrPlayerStateNotFound) {
			return nil, ErrNotConnected
		}
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

	entries := state.List()
	for i := pageStart; i < pageEnd; i++ {
		view := dtos.NewTrackView(entries[i].Track())

		if !state.IsPlaybackActive() {
			output.PlayedTracks = append(output.PlayedTracks, view)
			continue
		}

		if i < state.CurrentIndex() {
			output.PlayedTracks = append(output.PlayedTracks, view)
		} else if i == state.CurrentIndex() {
			output.CurrentTrack = &view
		} else {
			output.UpcomingTracks = append(output.UpcomingTracks, view)
		}
	}

	return output, nil
}
