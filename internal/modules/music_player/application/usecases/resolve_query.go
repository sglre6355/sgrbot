package usecases

import (
	"context"
	"errors"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/dtos"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

// ResolveQueryInput holds the input for the ResolveQuery use case.
type ResolveQueryInput struct {
	Query string
}

// ResolveQueryOutput holds the output for the ResolveQuery use case.
type ResolveQueryOutput struct {
	dtos.TrackListView
}

// ResolveQuery resolves a user query (URL or search term) into tracks.
type ResolveQueryUsecase struct {
	resolver ports.TrackResolver
}

// NewResolveQuery creates a new ResolveQuery use case.
func NewResolveQueryUsecase(resolver ports.TrackResolver) *ResolveQueryUsecase {
	return &ResolveQueryUsecase{resolver: resolver}
}

// Execute resolves the query into tracks.
func (uc *ResolveQueryUsecase) Execute(
	ctx context.Context,
	input ResolveQueryInput,
) (*ResolveQueryOutput, error) {
	trackList, err := uc.resolver.ResolveQuery(ctx, input.Query)
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	if len(trackList.Tracks) == 0 {
		return nil, ErrNoResults
	}

	return &ResolveQueryOutput{
		TrackListView: dtos.NewTrackListView(trackList),
	}, nil
}
