package usecases

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// LoadTrackInput contains the input for the LoadTrack use case.
type LoadTrackInput struct {
	Query              string
	RequesterID        snowflake.ID
	RequesterName      string
	RequesterAvatarURL string
}

// LoadTrackOutput contains the result of the LoadTrack use case.
type LoadTrackOutput struct {
	Track *domain.Track
}

// LoadTracksInput contains the input for the LoadTracks use case.
type LoadTracksInput struct {
	Query              string
	RequesterID        snowflake.ID
	RequesterName      string
	RequesterAvatarURL string
}

// LoadTracksOutput contains the result of the LoadTracks use case.
type LoadTracksOutput struct {
	Tracks       []*domain.Track
	IsPlaylist   bool
	PlaylistName string
}

// TrackLoaderService handles track loading operations.
type TrackLoaderService struct {
	trackResolver ports.TrackResolver
}

// NewTrackLoaderService creates a new TrackLoaderService.
func NewTrackLoaderService(trackResolver ports.TrackResolver) *TrackLoaderService {
	return &TrackLoaderService{
		trackResolver: trackResolver,
	}
}

// LoadTrack loads a track from the given query.
func (s *TrackLoaderService) LoadTrack(
	ctx context.Context,
	input LoadTrackInput,
) (*LoadTrackOutput, error) {
	// Parse and search for the track
	query := domain.NewSearchQuery(input.Query)
	result, err := s.trackResolver.LoadTracks(ctx, query.LavalinkQuery())
	if err != nil {
		return nil, err
	}

	if result.Type == ports.LoadTypeEmpty || result.Type == ports.LoadTypeError ||
		len(result.Tracks) == 0 {
		return nil, ErrNoResults
	}

	// Create domain track from first result
	trackInfo := result.Tracks[0]
	track := domain.NewTrack(
		domain.TrackID(trackInfo.Identifier),
		trackInfo.Encoded,
		trackInfo.Title,
		trackInfo.Artist,
		trackInfo.Duration,
		trackInfo.URI,
		trackInfo.ArtworkURL,
		trackInfo.SourceName,
		trackInfo.IsStream,
		input.RequesterID,
		input.RequesterName,
		input.RequesterAvatarURL,
	)

	return &LoadTrackOutput{
		Track: track,
	}, nil
}

// LoadTracks loads tracks from the given query.
// For playlists, returns all tracks. For single tracks/searches, returns one track.
func (s *TrackLoaderService) LoadTracks(
	ctx context.Context,
	input LoadTracksInput,
) (*LoadTracksOutput, error) {
	query := domain.NewSearchQuery(input.Query)
	result, err := s.trackResolver.LoadTracks(ctx, query.LavalinkQuery())
	if err != nil {
		return nil, err
	}

	if result.Type == ports.LoadTypeEmpty || result.Type == ports.LoadTypeError ||
		len(result.Tracks) == 0 {
		return nil, ErrNoResults
	}

	// Determine which tracks to convert
	// For playlists, convert all tracks; otherwise just the first one
	tracksToConvert := result.Tracks
	if result.Type != ports.LoadTypePlaylist {
		tracksToConvert = result.Tracks[:1]
	}

	tracks := make([]*domain.Track, 0, len(tracksToConvert))
	for _, trackInfo := range tracksToConvert {
		track := domain.NewTrack(
			domain.TrackID(trackInfo.Identifier),
			trackInfo.Encoded,
			trackInfo.Title,
			trackInfo.Artist,
			trackInfo.Duration,
			trackInfo.URI,
			trackInfo.ArtworkURL,
			trackInfo.SourceName,
			trackInfo.IsStream,
			input.RequesterID,
			input.RequesterName,
			input.RequesterAvatarURL,
		)
		tracks = append(tracks, track)
	}

	return &LoadTracksOutput{
		Tracks:       tracks,
		IsPlaylist:   result.Type == ports.LoadTypePlaylist,
		PlaylistName: result.PlaylistID,
	}, nil
}

// ResolveQueryInput contains the input for resolving a query into track info.
type ResolveQueryInput struct {
	Query string
	Limit int // Max individual tracks to return (default 24)
}

// ResolveQueryOutput contains the result of resolving a query.
type ResolveQueryOutput struct {
	Tracks       []*ports.TrackInfo
	IsPlaylist   bool
	PlaylistName string
	TotalTracks  int
}

// ResolveQuery resolves a query into track information without creating domain tracks.
// For playlists, returns playlist metadata and a limited list of individual tracks.
// For non-playlists, returns the tracks normally.
func (s *TrackLoaderService) ResolveQuery(
	ctx context.Context,
	input ResolveQueryInput,
) (*ResolveQueryOutput, error) {
	if s.trackResolver == nil {
		return &ResolveQueryOutput{}, nil
	}

	query := domain.NewSearchQuery(input.Query)
	result, err := s.trackResolver.LoadTracks(ctx, query.LavalinkQuery())
	if err != nil {
		return nil, err
	}

	if result.Type == ports.LoadTypeEmpty || result.Type == ports.LoadTypeError ||
		len(result.Tracks) == 0 {
		return &ResolveQueryOutput{}, nil
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

	return &ResolveQueryOutput{
		IsPlaylist:   result.Type == ports.LoadTypePlaylist,
		PlaylistName: result.PlaylistID,
		TotalTracks:  len(result.Tracks),
		Tracks:       tracks,
	}, nil
}

// SearchTracksInput contains the input for the SearchTracks use case.
type SearchTracksInput struct {
	Query string
	Limit int
}

// SearchTracksOutput contains the result of the SearchTracks use case.
type SearchTracksOutput struct {
	Tracks []*ports.TrackInfo
}

// SearchTracks searches for tracks matching the query.
func (s *TrackLoaderService) SearchTracks(
	ctx context.Context,
	input SearchTracksInput,
) (*SearchTracksOutput, error) {
	if input.Query == "" {
		return &SearchTracksOutput{Tracks: nil}, nil
	}

	query := domain.NewSearchQuery(input.Query)
	result, err := s.trackResolver.LoadTracks(ctx, query.LavalinkQuery())
	if err != nil {
		return nil, err
	}

	if result.Type == ports.LoadTypeEmpty || result.Type == ports.LoadTypeError {
		return &SearchTracksOutput{Tracks: nil}, nil
	}

	limit := input.Limit
	if limit <= 0 || limit > len(result.Tracks) {
		limit = len(result.Tracks)
	}

	return &SearchTracksOutput{
		Tracks: result.Tracks[:limit],
	}, nil
}
