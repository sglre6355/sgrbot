package domain

import (
	"testing"
)

func TestNewTrackList(t *testing.T) {
	tracks := []Track{
		{ID: "t1", Title: "Track 1"},
		{ID: "t2", Title: "Track 2"},
		{ID: "t3", Title: "Track 3"},
	}

	tests := []struct {
		name           string
		trackListType  TrackListType
		tracks         []Track
		opts           []TrackListOption
		wantTrackCount int
		wantFirstID    TrackID
		wantName       string
	}{
		{
			name:           "search keeps all tracks",
			trackListType:  TrackListTypeSearch,
			tracks:         tracks,
			wantTrackCount: 3,
			wantFirstID:    "t1",
		},
		{
			name:           "playlist keeps all tracks",
			trackListType:  TrackListTypePlaylist,
			tracks:         tracks,
			wantTrackCount: 3,
			wantFirstID:    "t1",
		},
		{
			name:           "track type keeps its track",
			trackListType:  TrackListTypeTrack,
			tracks:         tracks[:1],
			wantTrackCount: 1,
			wantFirstID:    "t1",
		},
		{
			name:          "playlist with metadata",
			trackListType: TrackListTypePlaylist,
			tracks:        tracks,
			opts: []TrackListOption{
				WithPlaylistInfo("id-123", "My Playlist", "https://example.com/playlist"),
			},
			wantTrackCount: 3,
			wantFirstID:    "t1",
			wantName:       "My Playlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := NewTrackList(tt.trackListType, tt.tracks, tt.opts...)

			if len(tl.Tracks) != tt.wantTrackCount {
				t.Errorf("got %d tracks, want %d", len(tl.Tracks), tt.wantTrackCount)
			}

			if tt.wantTrackCount > 0 && tl.Tracks[0].ID != tt.wantFirstID {
				t.Errorf("first track ID = %q, want %q", tl.Tracks[0].ID, tt.wantFirstID)
			}

			gotName := ""
			if tl.Name != nil {
				gotName = *tl.Name
			}
			if gotName != tt.wantName {
				t.Errorf("Name = %q, want %q", gotName, tt.wantName)
			}
		})
	}
}
