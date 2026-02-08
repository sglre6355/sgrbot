package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestTrackLoaderService_LoadTrack(t *testing.T) {
	requesterID := snowflake.ID(123)

	successResult := &ports.LoadResult{
		Type: ports.LoadTypeTrack,
		Tracks: []*ports.TrackInfo{
			{
				Identifier: "dQw4w9WgXcQ",
				Encoded:    "encoded-data",
				Title:      "Test Song",
				Artist:     "Test Artist",
				Duration:   3 * time.Minute,
				URI:        "https://example.com/track",
				ArtworkURL: "https://example.com/art.jpg",
				SourceName: "youtube",
				IsStream:   false,
			},
		},
	}

	tests := []struct {
		name          string
		input         LoadTrackInput
		setupResolver func(*mockTrackResolver)
		wantErr       error
		wantTitle     string
	}{
		{
			name: "successful track loading with search query",
			input: LoadTrackInput{
				Query:       "test song",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = successResult
			},
			wantTitle: "Test Song",
		},
		{
			name: "successful track loading with URL",
			input: LoadTrackInput{
				Query:       "https://youtube.com/watch?v=123",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = successResult
			},
			wantTitle: "Test Song",
		},
		{
			name: "no results - empty type",
			input: LoadTrackInput{
				Query:       "nonexistent song",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeEmpty, Tracks: nil}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "no results - error type",
			input: LoadTrackInput{
				Query:       "error query",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeError, Tracks: nil}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "no results - empty tracks array",
			input: LoadTrackInput{
				Query:       "empty result",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{
					Type:   ports.LoadTypeSearch,
					Tracks: []*ports.TrackInfo{},
				}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "resolver error",
			input: LoadTrackInput{
				Query:       "test song",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadErr = errors.New("resolver connection failed")
			},
			wantErr: errors.New("resolver connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockTrackResolver{}

			if tt.setupResolver != nil {
				tt.setupResolver(resolver)
			}

			service := NewTrackLoaderService(resolver)
			output, err := service.LoadTrack(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.Track == nil {
				t.Error("expected Track to be set")
				return
			}

			if output.Track.Title != tt.wantTitle {
				t.Errorf("Track.Title = %q, want %q", output.Track.Title, tt.wantTitle)
			}

			if output.Track.RequesterID != requesterID {
				t.Errorf("Track.RequesterID = %d, want %d", output.Track.RequesterID, requesterID)
			}

			if output.Track.ID == "" {
				t.Error("expected Track.ID to be set")
			}
		})
	}
}

func TestTrackLoaderService_LoadTracks(t *testing.T) {
	requesterID := snowflake.ID(123)
	requesterName := "TestUser"
	requesterAvatarURL := "https://example.com/avatar.jpg"

	singleTrackResult := &ports.LoadResult{
		Type: ports.LoadTypeTrack,
		Tracks: []*ports.TrackInfo{
			{
				Identifier: "track-1",
				Encoded:    "encoded-1",
				Title:      "Single Track",
				Artist:     "Artist 1",
				Duration:   3 * time.Minute,
				URI:        "https://example.com/track1",
				SourceName: "youtube",
			},
		},
	}

	searchResult := &ports.LoadResult{
		Type: ports.LoadTypeSearch,
		Tracks: []*ports.TrackInfo{
			{Identifier: "search-1", Title: "Search Result 1"},
			{Identifier: "search-2", Title: "Search Result 2"},
			{Identifier: "search-3", Title: "Search Result 3"},
		},
	}

	playlistResult := &ports.LoadResult{
		Type:       ports.LoadTypePlaylist,
		PlaylistID: "My Awesome Playlist",
		Tracks: []*ports.TrackInfo{
			{Identifier: "playlist-1", Title: "Playlist Track 1", Artist: "Artist 1"},
			{Identifier: "playlist-2", Title: "Playlist Track 2", Artist: "Artist 2"},
			{Identifier: "playlist-3", Title: "Playlist Track 3", Artist: "Artist 3"},
		},
	}

	tests := []struct {
		name             string
		input            LoadTracksInput
		setupResolver    func(*mockTrackResolver)
		wantErr          error
		wantTrackCount   int
		wantIsPlaylist   bool
		wantPlaylistName string
		wantFirstTitle   string
	}{
		{
			name: "single track result returns one track",
			input: LoadTracksInput{
				Query:              "https://youtube.com/watch?v=123",
				RequesterID:        requesterID,
				RequesterName:      requesterName,
				RequesterAvatarURL: requesterAvatarURL,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = singleTrackResult
			},
			wantTrackCount: 1,
			wantIsPlaylist: false,
			wantFirstTitle: "Single Track",
		},
		{
			name: "search result returns only first track",
			input: LoadTracksInput{
				Query:       "search query",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = searchResult
			},
			wantTrackCount: 1,
			wantIsPlaylist: false,
			wantFirstTitle: "Search Result 1",
		},
		{
			name: "playlist result returns all tracks",
			input: LoadTracksInput{
				Query:              "https://youtube.com/playlist?list=abc",
				RequesterID:        requesterID,
				RequesterName:      requesterName,
				RequesterAvatarURL: requesterAvatarURL,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = playlistResult
			},
			wantTrackCount:   3,
			wantIsPlaylist:   true,
			wantPlaylistName: "My Awesome Playlist",
			wantFirstTitle:   "Playlist Track 1",
		},
		{
			name: "no results - empty type",
			input: LoadTracksInput{
				Query:       "nonexistent",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeEmpty}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "no results - error type",
			input: LoadTracksInput{
				Query:       "error",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeError}
			},
			wantErr: ErrNoResults,
		},
		{
			name: "resolver error",
			input: LoadTracksInput{
				Query:       "test",
				RequesterID: requesterID,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadErr = errors.New("connection failed")
			},
			wantErr: errors.New("connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockTrackResolver{}
			if tt.setupResolver != nil {
				tt.setupResolver(resolver)
			}

			service := NewTrackLoaderService(resolver)
			output, err := service.LoadTracks(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(output.Tracks) != tt.wantTrackCount {
				t.Errorf("got %d tracks, want %d", len(output.Tracks), tt.wantTrackCount)
			}

			if output.IsPlaylist != tt.wantIsPlaylist {
				t.Errorf("IsPlaylist = %v, want %v", output.IsPlaylist, tt.wantIsPlaylist)
			}

			if output.PlaylistName != tt.wantPlaylistName {
				t.Errorf("PlaylistName = %q, want %q", output.PlaylistName, tt.wantPlaylistName)
			}

			if len(output.Tracks) > 0 && output.Tracks[0].Title != tt.wantFirstTitle {
				t.Errorf(
					"first track title = %q, want %q",
					output.Tracks[0].Title,
					tt.wantFirstTitle,
				)
			}
		})
	}
}

func TestTrackLoaderService_ResolveQuery(t *testing.T) {
	singleTrackResult := &ports.LoadResult{
		Type: ports.LoadTypeTrack,
		Tracks: []*ports.TrackInfo{
			{Identifier: "track-1", Title: "Single Track", Artist: "Artist"},
		},
	}

	searchResult := &ports.LoadResult{
		Type: ports.LoadTypeSearch,
		Tracks: []*ports.TrackInfo{
			{Identifier: "search-1", Title: "Search Result 1"},
			{Identifier: "search-2", Title: "Search Result 2"},
			{Identifier: "search-3", Title: "Search Result 3"},
		},
	}

	playlistResult := &ports.LoadResult{
		Type:       ports.LoadTypePlaylist,
		PlaylistID: "My Playlist",
		Tracks: []*ports.TrackInfo{
			{Identifier: "pl-1", Title: "Playlist Track 1"},
			{Identifier: "pl-2", Title: "Playlist Track 2"},
			{Identifier: "pl-3", Title: "Playlist Track 3"},
			{Identifier: "pl-4", Title: "Playlist Track 4"},
			{Identifier: "pl-5", Title: "Playlist Track 5"},
		},
	}

	tests := []struct {
		name             string
		input            ResolveQueryInput
		setupResolver    func(*mockTrackResolver)
		wantIsPlaylist   bool
		wantPlaylistName string
		wantTotalTracks  int
		wantTracksLen    int
		wantErr          bool
	}{
		{
			name: "single track result",
			input: ResolveQueryInput{
				Query: "https://youtube.com/watch?v=123",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = singleTrackResult
			},
			wantIsPlaylist:   false,
			wantPlaylistName: "",
			wantTotalTracks:  1,
			wantTracksLen:    1,
		},
		{
			name: "search result",
			input: ResolveQueryInput{
				Query: "search query",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = searchResult
			},
			wantIsPlaylist:   false,
			wantPlaylistName: "",
			wantTotalTracks:  3,
			wantTracksLen:    3,
		},
		{
			name: "playlist result",
			input: ResolveQueryInput{
				Query: "https://youtube.com/playlist?list=abc",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = playlistResult
			},
			wantIsPlaylist:   true,
			wantPlaylistName: "My Playlist",
			wantTotalTracks:  5,
			wantTracksLen:    5,
		},
		{
			name: "playlist result with limit",
			input: ResolveQueryInput{
				Query: "https://youtube.com/playlist?list=abc",
				Limit: 2,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = playlistResult
			},
			wantIsPlaylist:   true,
			wantPlaylistName: "My Playlist",
			wantTotalTracks:  5, // Total tracks in playlist
			wantTracksLen:    2, // Limited to 2
		},
		{
			name: "empty result",
			input: ResolveQueryInput{
				Query: "nonexistent",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeEmpty}
			},
			wantIsPlaylist:  false,
			wantTotalTracks: 0,
			wantTracksLen:   0,
		},
		{
			name: "error result",
			input: ResolveQueryInput{
				Query: "error",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeError}
			},
			wantIsPlaylist:  false,
			wantTotalTracks: 0,
			wantTracksLen:   0,
		},
		{
			name: "resolver error",
			input: ResolveQueryInput{
				Query: "test",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadErr = errors.New("connection failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &mockTrackResolver{}
			if tt.setupResolver != nil {
				tt.setupResolver(resolver)
			}

			service := NewTrackLoaderService(resolver)
			output, err := service.ResolveQuery(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output.IsPlaylist != tt.wantIsPlaylist {
				t.Errorf("IsPlaylist = %v, want %v", output.IsPlaylist, tt.wantIsPlaylist)
			}

			if output.PlaylistName != tt.wantPlaylistName {
				t.Errorf("PlaylistName = %q, want %q", output.PlaylistName, tt.wantPlaylistName)
			}

			if output.TotalTracks != tt.wantTotalTracks {
				t.Errorf("TotalTracks = %d, want %d", output.TotalTracks, tt.wantTotalTracks)
			}

			if len(output.Tracks) != tt.wantTracksLen {
				t.Errorf("len(Tracks) = %d, want %d", len(output.Tracks), tt.wantTracksLen)
			}
		})
	}
}

func TestTrackLoaderService_ResolveQuery_NilResolver(t *testing.T) {
	service := NewTrackLoaderService(nil)
	output, err := service.ResolveQuery(
		context.Background(),
		ResolveQueryInput{
			Query: "test",
		},
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if output.IsPlaylist {
		t.Error("expected IsPlaylist = false for nil resolver")
	}

	if len(output.Tracks) != 0 {
		t.Errorf("expected 0 tracks for nil resolver, got %d", len(output.Tracks))
	}
}

func TestTrackLoaderService_ResolveQuery_DefaultLimit(t *testing.T) {
	// Create a playlist with 30 tracks
	tracks := make([]*ports.TrackInfo, 30)
	for i := range 30 {
		tracks[i] = &ports.TrackInfo{
			Identifier: string(rune('a' + i)),
			Title:      "Track " + string(rune('A'+i)),
		}
	}

	resolver := &mockTrackResolver{
		loadResult: &ports.LoadResult{
			Type:       ports.LoadTypePlaylist,
			PlaylistID: "Large Playlist",
			Tracks:     tracks,
		},
	}

	service := NewTrackLoaderService(resolver)
	output, err := service.ResolveQuery(
		context.Background(),
		ResolveQueryInput{
			Query: "https://example.com/playlist",
			// Limit not specified, should default to 24
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TotalTracks != 30 {
		t.Errorf("TotalTracks = %d, want 30", output.TotalTracks)
	}

	if len(output.Tracks) != 24 {
		t.Errorf("len(Tracks) = %d, want 24 (default limit)", len(output.Tracks))
	}
}

func TestTrackLoaderService_LoadTracks_RequesterInfo(t *testing.T) {
	requesterID := snowflake.ID(456)
	requesterName := "TestUser"
	requesterAvatarURL := "https://example.com/avatar.jpg"

	resolver := &mockTrackResolver{
		loadResult: &ports.LoadResult{
			Type:       ports.LoadTypePlaylist,
			PlaylistID: "Test Playlist",
			Tracks: []*ports.TrackInfo{
				{Identifier: "track-1", Title: "Track 1"},
				{Identifier: "track-2", Title: "Track 2"},
			},
		},
	}

	service := NewTrackLoaderService(resolver)
	output, err := service.LoadTracks(context.Background(), LoadTracksInput{
		Query:              "https://example.com/playlist",
		RequesterID:        requesterID,
		RequesterName:      requesterName,
		RequesterAvatarURL: requesterAvatarURL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify requester info is set on all tracks
	for i, track := range output.Tracks {
		if track.RequesterID != requesterID {
			t.Errorf("track %d: RequesterID = %d, want %d", i, track.RequesterID, requesterID)
		}
		if track.RequesterName != requesterName {
			t.Errorf("track %d: RequesterName = %q, want %q", i, track.RequesterName, requesterName)
		}
		if track.RequesterAvatarURL != requesterAvatarURL {
			t.Errorf(
				"track %d: RequesterAvatarURL = %q, want %q",
				i,
				track.RequesterAvatarURL,
				requesterAvatarURL,
			)
		}
		if track.ID == "" {
			t.Errorf("track %d: ID should be set", i)
		}
		if track.ID != domain.TrackID(resolver.loadResult.Tracks[i].Identifier) {
			t.Errorf(
				"track %d: ID = %q, want %q",
				i,
				track.ID,
				resolver.loadResult.Tracks[i].Identifier,
			)
		}
	}
}
