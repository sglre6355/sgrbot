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

// LoadTracksForAutocompleteInput contains the input for playlist-aware autocomplete.
type LoadTracksForAutocompleteInput struct {
	Query string
	Limit int // Max individual tracks to return (default 24, leaving room for playlist option)
}

// LoadTracksForAutocompleteOutput contains the result for playlist-aware autocomplete.
type LoadTracksForAutocompleteOutput struct {
	IsPlaylist   bool
	PlaylistName string
	PlaylistURL  string             // Original URL for "add all" option
	TrackCount   int                // Total tracks in playlist
	Tracks       []*ports.TrackInfo // Individual tracks (limited)
}

// LoadTracksForAutocomplete loads tracks for autocomplete, with special handling for playlists.
// For playlists, returns playlist metadata and a limited list of individual tracks.
// For non-playlists, returns the tracks normally.
func (s *AutocompleteService) LoadTracksForAutocomplete(
	ctx context.Context,
	input LoadTracksForAutocompleteInput,
) (*LoadTracksForAutocompleteOutput, error) {
	if s.trackLoader == nil {
		return &LoadTracksForAutocompleteOutput{}, nil
	}

	query := domain.NewSearchQuery(input.Query)
	result, err := s.trackLoader.LoadTracks(ctx, query.LavalinkQuery())
	if err != nil {
		return nil, err
	}

	if result.Type == ports.LoadTypeEmpty || result.Type == ports.LoadTypeError ||
		len(result.Tracks) == 0 {
		return &LoadTracksForAutocompleteOutput{}, nil
	}

	// Determine limit (default 24 to leave room for playlist option)
	limit := input.Limit
	if limit <= 0 {
		limit = 24
	}

	// Limit tracks
	tracks := result.Tracks
	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	return &LoadTracksForAutocompleteOutput{
		IsPlaylist:   result.Type == ports.LoadTypePlaylist,
		PlaylistName: result.PlaylistID,
		PlaylistURL:  input.Query, // Original query URL
		TrackCount:   len(result.Tracks),
		Tracks:       tracks,
	}, nil
}
