package usecases

import (
	"context"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

// TrackData is an application-layer DTO for track information.
// It exposes track data without leaking domain types to the presentation layer.
type TrackData struct {
	ID         string
	Title      string
	Artist     string
	Duration   time.Duration
	URI        string
	ArtworkURL string
	Source     string
	IsStream   bool
}

// toTrackData converts a domain.Track to a TrackData DTO.
func toTrackData(t domain.Track) TrackData {
	return TrackData{
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
	Track TrackData
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
	return LoadTrackOutput{Track: toTrackData(track)}, nil
}

type LoadTracksInput struct {
	TrackIDs []string
}

type LoadTracksOutput struct {
	Tracks []TrackData
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

	trackInfos := make([]TrackData, len(input.TrackIDs))
	for i, track := range tracks {
		trackInfos[i] = toTrackData(track)
	}
	return LoadTracksOutput{Tracks: trackInfos}, nil
}

// ResolveQueryInput contains the input for the ResolveQuery use case.
type ResolveQueryInput struct {
	Query string
}

// ResolveQueryOutput contains the result of the ResolveQuery use case.
type ResolveQueryOutput struct {
	Type       string
	Identifier *string
	Name       *string
	Url        *string
	Tracks     []TrackData
}

// ResolveQuery loads tracks from the given query.
// For playlists, returns all tracks. For single tracks/searches, returns one track.
func (s *TrackLoaderService) ResolveQuery(
	ctx context.Context,
	input ResolveQueryInput,
) (*ResolveQueryOutput, error) {
	output, err := s.trackResolver.ResolveQuery(ctx, input.Query)
	if err != nil {
		return nil, err
	}

	if len(output.Tracks) == 0 {
		return nil, ErrNoResults
	}

	tracks := make([]TrackData, len(output.Tracks))
	for i, track := range output.Tracks {
		tracks[i] = toTrackData(track)
	}

	return &ResolveQueryOutput{
		Type:       output.Type.String(),
		Identifier: output.Identifier,
		Name:       output.Name,
		Url:        output.Url,
		Tracks:     tracks,
	}, nil
}
