package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func strPtr(s string) *string { return &s }

func TestTrackLoaderService_ResolveQuery(t *testing.T) {
	singleTrackResult := domain.TrackList{
		Type: domain.TrackListTypeTrack,
		Tracks: []domain.Track{
			{
				ID:       "track-1",
				Title:    "Single Track",
				Artist:   "Artist 1",
				Duration: 3 * time.Minute,
				URI:      "https://example.com/track1",
				Source:   domain.TrackSourceYouTube,
			},
		},
	}

	searchResult := domain.TrackList{
		Type: domain.TrackListTypeSearch,
		Tracks: []domain.Track{
			{ID: "search-1", Title: "Search Result 1"},
			{ID: "search-2", Title: "Search Result 2"},
			{ID: "search-3", Title: "Search Result 3"},
		},
	}

	playlistResult := domain.TrackList{
		Type: domain.TrackListTypePlaylist,
		Name: strPtr("My Awesome Playlist"),
		Tracks: []domain.Track{
			{ID: "playlist-1", Title: "Playlist Track 1", Artist: "Artist 1"},
			{ID: "playlist-2", Title: "Playlist Track 2", Artist: "Artist 2"},
			{ID: "playlist-3", Title: "Playlist Track 3", Artist: "Artist 3"},
		},
	}

	tests := []struct {
		name             string
		input            ResolveQueryInput
		setupResolver    func(*mockTrackResolver)
		wantErr          error
		wantTrackCount   int
		wantIsPlaylist   bool
		wantPlaylistName string
		wantFirstTitle   string
	}{
		{
			name: "single track result returns one track",
			input: ResolveQueryInput{
				Query: "https://youtube.com/watch?v=123",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = singleTrackResult
			},
			wantTrackCount: 1,
			wantIsPlaylist: false,
			wantFirstTitle: "Single Track",
		},
		{
			name: "search result returns all tracks",
			input: ResolveQueryInput{
				Query: "search query",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadResult = searchResult
			},
			wantTrackCount: 3,
			wantIsPlaylist: false,
			wantFirstTitle: "Search Result 1",
		},
		{
			name: "playlist result returns all tracks",
			input: ResolveQueryInput{
				Query: "https://youtube.com/playlist?list=abc",
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
			name: "no results from resolver",
			input: ResolveQueryInput{
				Query: "nonexistent",
			},
			setupResolver: func(m *mockTrackResolver) {
				m.loadErr = fmt.Errorf("no results found")
			},
			wantErr: fmt.Errorf("no results found"),
		},
		{
			name: "resolver error",
			input: ResolveQueryInput{
				Query: "test",
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
			output, err := service.ResolveQuery(context.Background(), tt.input)

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
