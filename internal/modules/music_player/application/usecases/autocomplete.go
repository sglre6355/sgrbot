package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// GetQueueTracksInput contains the input for the GetQueueTracks use case.
type GetQueueTracksInput struct {
	GuildID snowflake.ID
}

// GetQueueTracksOutput contains the output for the GetQueueTracks use case.
type GetQueueTracksOutput struct {
	Tracks []*domain.Track
}

// AutocompleteService handles autocomplete-related operations.
type AutocompleteService struct {
	repo        domain.PlayerStateRepository
	trackLoader ports.TrackResolver
}

// NewAutocompleteService creates a new AutocompleteService.
func NewAutocompleteService(
	repo domain.PlayerStateRepository,
	trackLoader ports.TrackResolver,
) *AutocompleteService {
	return &AutocompleteService{
		repo:        repo,
		trackLoader: trackLoader,
	}
}

// GetQueueTracks returns the current queue tracks for autocomplete suggestions.
func (s *AutocompleteService) GetQueueTracks(input GetQueueTracksInput) *GetQueueTracksOutput {
	state := s.repo.Get(input.GuildID)
	if state == nil {
		return &GetQueueTracksOutput{Tracks: nil}
	}

	return &GetQueueTracksOutput{
		Tracks: state.Queue.List(),
	}
}

// SearchTracks searches for tracks matching the query.
// This is a pass-through to TrackLoaderService.SearchTracks.
func (s *AutocompleteService) SearchTracks(
	ctx context.Context,
	input SearchTracksInput,
) (*SearchTracksOutput, error) {
	if s.trackLoader == nil {
		return &SearchTracksOutput{Tracks: nil}, nil
	}

	// Use TrackLoaderService's search functionality
	loader := NewTrackLoaderService(s.trackLoader)
	return loader.SearchTracks(ctx, input)
}
