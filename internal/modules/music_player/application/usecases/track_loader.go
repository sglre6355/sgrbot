package usecases

import (
	"context"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackInfo is an application-layer DTO for track information.
// It exposes track data without leaking domain types to the presentation layer.
type TrackInfo struct {
	ID         string
	Title      string
	Artist     string
	Duration   time.Duration
	URI        string
	ArtworkURL string
	Source     string
	IsStream   bool
}

// toTrackInfo converts a domain.Track to a TrackInfo DTO.
func toTrackInfo(t *domain.Track) *TrackInfo {
	return &TrackInfo{
		ID:         t.ID.String(),
		Title:      t.Title,
		Artist:     t.Artist,
		Duration:   t.Duration,
		URI:        t.URI,
		ArtworkURL: t.ArtworkURL,
		Source:     string(t.Source),
		IsStream:   t.IsStream,
	}
}

// TrackLoaderService handles track loading operations and implements TrackProvider via caching.
type TrackLoaderService struct {
	trackResolver ports.TrackProvider
}

// NewTrackLoaderService creates a new TrackLoaderService.
func NewTrackLoaderService(trackResolver ports.TrackProvider) *TrackLoaderService {
	return &TrackLoaderService{
		trackResolver: trackResolver,
	}
}

// LoadTrack loads a single track by ID and returns its info.
func (s *TrackLoaderService) LoadTrack(ctx context.Context, trackID string) (*TrackInfo, error) {
	track, err := s.trackResolver.LoadTrack(ctx, domain.TrackID(trackID))
	if err != nil {
		return nil, err
	}
	return toTrackInfo(&track), nil
}

// ResolveQueryInput contains the input for the ResolveQuery use case.
type ResolveQueryInput struct {
	Query string
}

// ResolveQueryOutput contains the result of the ResolveQuery use case.
type ResolveQueryOutput struct {
	Tracks       []*TrackInfo
	IsPlaylist   bool
	PlaylistName string
}

// ResolveQuery loads tracks from the given query.
// For playlists, returns all tracks. For single tracks/searches, returns one track.
func (s *TrackLoaderService) ResolveQuery(
	ctx context.Context,
	input ResolveQueryInput,
) (*ResolveQueryOutput, error) {
	result, err := s.trackResolver.ResolveQuery(ctx, input.Query)
	if err != nil {
		return nil, err
	}

	if len(result.Tracks) == 0 {
		return nil, ErrNoResults
	}

	// For playlists, return all tracks; otherwise just the first one
	tracks := result.Tracks
	if result.Type != domain.TrackListTypePlaylist {
		tracks = tracks[:1]
	}

	infos := make([]*TrackInfo, len(tracks))
	for i := range tracks {
		infos[i] = toTrackInfo(&tracks[i])
	}

	var playlistName string
	if result.Name != nil {
		playlistName = *result.Name
	}

	return &ResolveQueryOutput{
		Tracks:       infos,
		IsPlaylist:   result.Type == domain.TrackListTypePlaylist,
		PlaylistName: playlistName,
	}, nil
}

// PreviewQueryInput contains the input for previewing a query into track info.
type PreviewQueryInput struct {
	Query string
	Limit int // Max individual tracks to return (default 24)
}

// PreviewQueryOutput contains the result of previewing a query.
type PreviewQueryOutput struct {
	Tracks       []TrackInfo
	IsPlaylist   bool
	PlaylistName string
	TotalTracks  int
}

// PreviewQuery resolves a query into track information without creating domain tracks.
// For playlists, returns playlist metadata and a limited list of individual tracks.
// For non-playlists, returns the tracks normally.
func (s *TrackLoaderService) PreviewQuery(
	ctx context.Context,
	input PreviewQueryInput,
) (*PreviewQueryOutput, error) {
	if s.trackResolver == nil {
		return &PreviewQueryOutput{}, nil
	}

	result, err := s.trackResolver.ResolveQuery(ctx, input.Query)
	if err != nil {
		return &PreviewQueryOutput{}, nil
	}

	if len(result.Tracks) == 0 {
		return &PreviewQueryOutput{}, nil
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

	infos := make([]TrackInfo, len(tracks))
	for i := range tracks {
		infos[i] = *toTrackInfo(&tracks[i])
	}

	var playlistName string
	if result.Name != nil {
		playlistName = *result.Name
	}

	return &PreviewQueryOutput{
		IsPlaylist:   result.Type == domain.TrackListTypePlaylist,
		PlaylistName: playlistName,
		TotalTracks:  len(result.Tracks),
		Tracks:       infos,
	}, nil
}
