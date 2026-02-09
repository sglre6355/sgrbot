package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// ResolveQueryInput contains the input for the ResolveQuery use case.
type ResolveQueryInput struct {
	Query              string
	RequesterID        snowflake.ID
	RequesterName      string
	RequesterAvatarURL string
}

// ResolveQueryOutput contains the result of the ResolveQuery use case.
type ResolveQueryOutput struct {
	Tracks       []*domain.Track
	IsPlaylist   bool
	PlaylistName string
}

// PreviewQueryInput contains the input for previewing a query into track info.
type PreviewQueryInput struct {
	Query string
	Limit int // Max individual tracks to return (default 24)
}

// PreviewQueryOutput contains the result of previewing a query.
type PreviewQueryOutput struct {
	Tracks       []*ports.TrackInfo
	IsPlaylist   bool
	PlaylistName string
	TotalTracks  int
}

// TrackLoaderService handles track loading operations and implements TrackProvider via caching.
type TrackLoaderService struct {
	trackResolver ports.TrackResolver
	mu            sync.RWMutex
	cache         map[domain.TrackID]*domain.Track
}

// Compile-time check that TrackLoaderService implements TrackProvider.
var _ ports.TrackProvider = (*TrackLoaderService)(nil)

// NewTrackLoaderService creates a new TrackLoaderService.
func NewTrackLoaderService(trackResolver ports.TrackResolver) *TrackLoaderService {
	return &TrackLoaderService{
		trackResolver: trackResolver,
		cache:         make(map[domain.TrackID]*domain.Track),
	}
}

// LoadTrack returns a Track from the cache by ID.
func (s *TrackLoaderService) LoadTrack(id domain.TrackID) (domain.Track, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	track, ok := s.cache[id]
	if !ok {
		return domain.Track{}, fmt.Errorf("track %q not found in cache", id)
	}
	return *track, nil
}

// LoadTracks returns multiple Tracks from the cache by IDs.
func (s *TrackLoaderService) LoadTracks(ids ...domain.TrackID) ([]domain.Track, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tracks := make([]domain.Track, 0, len(ids))
	for _, id := range ids {
		track, ok := s.cache[id]
		if !ok {
			return nil, fmt.Errorf("track %q not found in cache", id)
		}
		tracks = append(tracks, *track)
	}
	return tracks, nil
}

// ResolveQuery loads tracks from the given query.
// For playlists, returns all tracks. For single tracks/searches, returns one track.
// All resolved tracks are cached for later retrieval via LoadTrack/LoadTracks.
func (s *TrackLoaderService) ResolveQuery(
	ctx context.Context,
	input ResolveQueryInput,
) (*ResolveQueryOutput, error) {
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

	// Cache all resolved tracks
	s.mu.Lock()
	for _, track := range tracks {
		s.cache[track.ID] = track
	}
	s.mu.Unlock()

	return &ResolveQueryOutput{
		Tracks:       tracks,
		IsPlaylist:   result.Type == ports.LoadTypePlaylist,
		PlaylistName: result.PlaylistID,
	}, nil
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

	query := domain.NewSearchQuery(input.Query)
	result, err := s.trackResolver.LoadTracks(ctx, query.LavalinkQuery())
	if err != nil {
		return nil, err
	}

	if result.Type == ports.LoadTypeEmpty || result.Type == ports.LoadTypeError ||
		len(result.Tracks) == 0 {
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

	return &PreviewQueryOutput{
		IsPlaylist:   result.Type == ports.LoadTypePlaylist,
		PlaylistName: result.PlaylistID,
		TotalTracks:  len(result.Tracks),
		Tracks:       tracks,
	}, nil
}
