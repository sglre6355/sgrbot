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
func toTrackInfo(t domain.Track) TrackInfo {
	return TrackInfo{
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

type LoadTrackInput struct {
	TrackID string
}

type LoadTrackOutput struct {
	Track TrackInfo
}

// LoadTrack loads a single track by ID and returns its info.
func (s *TrackLoaderService) LoadTrack(
	ctx context.Context,
	input LoadTrackInput,
) (LoadTrackOutput, error) {
	track, err := s.trackResolver.LoadTrack(ctx, domain.TrackID(input.TrackID))
	if err != nil {
		return LoadTrackOutput{}, err
	}
	return LoadTrackOutput{Track: toTrackInfo(track)}, nil
}

type LoadTracksInput struct {
	TrackIDs []string
}

type LoadTracksOutput struct {
	Tracks []TrackInfo
}

// LoadTracks loads multiple tracks by ID and returns their info.
func (s *TrackLoaderService) LoadTracks(
	ctx context.Context,
	input LoadTracksInput,
) (LoadTracksOutput, error) {
	trackIDs := make([]domain.TrackID, len(input.TrackIDs))
	for i, id := range input.TrackIDs {
		trackIDs[i] = domain.TrackID(id)
	}

	tracks, err := s.trackResolver.LoadTracks(ctx, trackIDs...)
	if err != nil {
		return LoadTracksOutput{}, err
	}

	trackInfos := make([]TrackInfo, len(input.TrackIDs))
	for i, track := range tracks {
		trackInfos[i] = toTrackInfo(track)
	}
	return LoadTracksOutput{Tracks: trackInfos}, nil
}

// ResolveQueryInput contains the input for the ResolveQuery use case.
type ResolveQueryInput struct {
	Query string
}

// ResolveQueryOutput contains the result of the ResolveQuery use case.
type ResolveQueryOutput struct {
	Tracks       []TrackInfo
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

	infos := make([]TrackInfo, len(result.Tracks))
	for i := range result.Tracks {
		infos[i] = toTrackInfo(result.Tracks[i])
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
