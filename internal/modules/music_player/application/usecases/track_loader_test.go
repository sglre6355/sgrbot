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

func TestTrackLoaderService_ResolveQuery(t *testing.T) {
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
			input: ResolveQueryInput{
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
			input: ResolveQueryInput{
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
			input: ResolveQueryInput{
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
			input: ResolveQueryInput{
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
			input: ResolveQueryInput{
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

func TestTrackLoaderService_PreviewQuery(t *testing.T) {
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
		input            PreviewQueryInput
		setupResolver    func(*mockTrackResolver)
		wantIsPlaylist   bool
		wantPlaylistName string
		wantTotalTracks  int
		wantTracksLen    int
		wantErr          bool
	}{
		{
			name: "single track result",
			input: PreviewQueryInput{
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
			input: PreviewQueryInput{
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
			input: PreviewQueryInput{
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
			input: PreviewQueryInput{
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
			input: PreviewQueryInput{
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
			input: PreviewQueryInput{
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
			input: PreviewQueryInput{
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
			output, err := service.PreviewQuery(context.Background(), tt.input)

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

func TestTrackLoaderService_PreviewQuery_NilResolver(t *testing.T) {
	service := NewTrackLoaderService(nil)
	output, err := service.PreviewQuery(
		context.Background(),
		PreviewQueryInput{
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

func TestTrackLoaderService_PreviewQuery_DefaultLimit(t *testing.T) {
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
	output, err := service.PreviewQuery(
		context.Background(),
		PreviewQueryInput{
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

func TestTrackLoaderService_ResolveQuery_RequesterInfo(t *testing.T) {
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
	output, err := service.ResolveQuery(context.Background(), ResolveQueryInput{
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

func TestTrackLoaderService_ResolveQuery_PopulatesCache(t *testing.T) {
	resolver := &mockTrackResolver{
		loadResult: &ports.LoadResult{
			Type: ports.LoadTypeTrack,
			Tracks: []*ports.TrackInfo{
				{
					Identifier: "cached-track",
					Encoded:    "encoded-data",
					Title:      "Cached Track",
					Artist:     "Artist",
					Duration:   3 * time.Minute,
					URI:        "https://example.com/track",
					SourceName: "youtube",
				},
			},
		},
	}

	service := NewTrackLoaderService(resolver)

	// Before ResolveQuery, LoadTrack should fail
	_, err := service.LoadTrack(domain.TrackID("cached-track"))
	if err == nil {
		t.Error("expected error before ResolveQuery populates cache")
	}

	// ResolveQuery should populate cache
	_, err = service.ResolveQuery(context.Background(), ResolveQueryInput{
		Query:       "test",
		RequesterID: snowflake.ID(123),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After ResolveQuery, LoadTrack should succeed
	track, err := service.LoadTrack(domain.TrackID("cached-track"))
	if err != nil {
		t.Fatalf("unexpected error after cache populated: %v", err)
	}
	if track.Title != "Cached Track" {
		t.Errorf("Title = %q, want %q", track.Title, "Cached Track")
	}
}

func TestTrackLoaderService_LoadTrack_CacheBehavior(t *testing.T) {
	service := NewTrackLoaderService(nil)

	// LoadTrack on empty cache should return error
	_, err := service.LoadTrack(domain.TrackID("nonexistent"))
	if err == nil {
		t.Error("expected error for non-cached track")
	}

	// Manually populate cache via a public method (ResolveQuery)
	// For this test, we test that a resolved track can be loaded
	resolver := &mockTrackResolver{
		loadResult: &ports.LoadResult{
			Type: ports.LoadTypeTrack,
			Tracks: []*ports.TrackInfo{
				{Identifier: "test-id", Title: "Test Track"},
			},
		},
	}

	service2 := NewTrackLoaderService(resolver)
	_, _ = service2.ResolveQuery(context.Background(), ResolveQueryInput{
		Query:       "test",
		RequesterID: snowflake.ID(1),
	})

	track, err := service2.LoadTrack(domain.TrackID("test-id"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if track.Title != "Test Track" {
		t.Errorf("Title = %q, want %q", track.Title, "Test Track")
	}
}

func TestTrackLoaderService_LoadTracks_CacheBehavior(t *testing.T) {
	resolver := &mockTrackResolver{
		loadResult: &ports.LoadResult{
			Type:       ports.LoadTypePlaylist,
			PlaylistID: "Test Playlist",
			Tracks: []*ports.TrackInfo{
				{Identifier: "id-1", Title: "Track 1"},
				{Identifier: "id-2", Title: "Track 2"},
				{Identifier: "id-3", Title: "Track 3"},
			},
		},
	}

	service := NewTrackLoaderService(resolver)

	// Resolve to populate cache
	_, err := service.ResolveQuery(context.Background(), ResolveQueryInput{
		Query:       "test",
		RequesterID: snowflake.ID(1),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// LoadTracks should return all cached tracks
	tracks, err := service.LoadTracks(
		domain.TrackID("id-1"),
		domain.TrackID("id-2"),
		domain.TrackID("id-3"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 3 {
		t.Errorf("expected 3 tracks, got %d", len(tracks))
	}

	// LoadTracks with missing ID should return error
	_, err = service.LoadTracks(
		domain.TrackID("id-1"),
		domain.TrackID("missing"),
	)
	if err == nil {
		t.Error("expected error for missing track ID")
	}
}
