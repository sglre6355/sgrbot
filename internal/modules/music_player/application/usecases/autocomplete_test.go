package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/application/ports"
)

func TestAutocompleteService_LoadTracksForAutocomplete(t *testing.T) {
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
		input            LoadTracksForAutocompleteInput
		setupResolver    func(*mockTrackResolver)
		wantIsPlaylist   bool
		wantPlaylistName string
		wantPlaylistURL  string
		wantTrackCount   int
		wantTracksLen    int
		wantErr          bool
	}{
		{
			name: "single track result",
			input: LoadTracksForAutocompleteInput{
				Query: "https://youtube.com/watch?v=123",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = singleTrackResult
			},
			wantIsPlaylist:   false,
			wantPlaylistName: "",
			wantTrackCount:   1,
			wantTracksLen:    1,
		},
		{
			name: "search result",
			input: LoadTracksForAutocompleteInput{
				Query: "search query",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = searchResult
			},
			wantIsPlaylist:   false,
			wantPlaylistName: "",
			wantTrackCount:   3,
			wantTracksLen:    3,
		},
		{
			name: "playlist result",
			input: LoadTracksForAutocompleteInput{
				Query: "https://youtube.com/playlist?list=abc",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = playlistResult
			},
			wantIsPlaylist:   true,
			wantPlaylistName: "My Playlist",
			wantPlaylistURL:  "https://youtube.com/playlist?list=abc",
			wantTrackCount:   5,
			wantTracksLen:    5,
		},
		{
			name: "playlist result with limit",
			input: LoadTracksForAutocompleteInput{
				Query: "https://youtube.com/playlist?list=abc",
				Limit: 2,
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = playlistResult
			},
			wantIsPlaylist:   true,
			wantPlaylistName: "My Playlist",
			wantPlaylistURL:  "https://youtube.com/playlist?list=abc",
			wantTrackCount:   5, // Total tracks in playlist
			wantTracksLen:    2, // Limited to 2
		},
		{
			name: "empty result",
			input: LoadTracksForAutocompleteInput{
				Query: "nonexistent",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeEmpty}
			},
			wantIsPlaylist: false,
			wantTrackCount: 0,
			wantTracksLen:  0,
		},
		{
			name: "error result",
			input: LoadTracksForAutocompleteInput{
				Query: "error",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = &ports.LoadResult{Type: ports.LoadTypeError}
			},
			wantIsPlaylist: false,
			wantTrackCount: 0,
			wantTracksLen:  0,
		},
		{
			name: "resolver error",
			input: LoadTracksForAutocompleteInput{
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

			service := NewAutocompleteService(nil, resolver)
			output, err := service.LoadTracksForAutocomplete(context.Background(), tt.input)

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

			if tt.wantPlaylistURL != "" && output.PlaylistURL != tt.wantPlaylistURL {
				t.Errorf("PlaylistURL = %q, want %q", output.PlaylistURL, tt.wantPlaylistURL)
			}

			if output.TrackCount != tt.wantTrackCount {
				t.Errorf("TrackCount = %d, want %d", output.TrackCount, tt.wantTrackCount)
			}

			if len(output.Tracks) != tt.wantTracksLen {
				t.Errorf("len(Tracks) = %d, want %d", len(output.Tracks), tt.wantTracksLen)
			}
		})
	}
}

func TestAutocompleteService_LoadTracksForAutocomplete_NilResolver(t *testing.T) {
	service := NewAutocompleteService(nil, nil)
	output, err := service.LoadTracksForAutocomplete(
		context.Background(),
		LoadTracksForAutocompleteInput{
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

func TestAutocompleteService_LoadTracksForAutocomplete_DefaultLimit(t *testing.T) {
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

	service := NewAutocompleteService(nil, resolver)
	output, err := service.LoadTracksForAutocomplete(
		context.Background(),
		LoadTracksForAutocompleteInput{
			Query: "https://example.com/playlist",
			// Limit not specified, should default to 24
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.TrackCount != 30 {
		t.Errorf("TrackCount = %d, want 30", output.TrackCount)
	}

	if len(output.Tracks) != 24 {
		t.Errorf("len(Tracks) = %d, want 24 (default limit)", len(output.Tracks))
	}
}
