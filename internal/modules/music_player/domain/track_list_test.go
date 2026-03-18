package domain

import (
	"testing"
	"time"
)

func TestTrackListType_String(t *testing.T) {
	type args struct {
		typ TrackListType
	}
	type want struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "track", args: args{typ: TrackListTypeTrack}, want: want{str: "track"}},
		{name: "playlist", args: args{typ: TrackListTypePlaylist}, want: want{str: "playlist"}},
		{name: "search", args: args{typ: TrackListTypeSearch}, want: want{str: "search"}},
		{name: "unknown value", args: args{typ: TrackListType(99)}, want: want{str: "unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.typ.String(); got != tt.want.str {
				t.Fatalf("got %q, want %q", got, tt.want.str)
			}
		})
	}
}

func TestNewTrackList(t *testing.T) {
	tracks := []Track{
		*ConstructTrack(TrackID("1"), "A", "A", time.Minute, "", "", TrackSourceYouTube, false),
		*ConstructTrack(TrackID("2"), "B", "B", time.Minute, "", "", TrackSourceYouTube, false),
	}

	type args struct {
		trackListType TrackListType
		tracks        []Track
		opts          []TrackListOption
	}
	type want struct {
		typ           TrackListType
		trackCount    int
		hasIdentifier bool
		identifier    string
		hasName       bool
		name          string
		hasUrl        bool
		url           string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "basic creation without options",
			args: args{
				trackListType: TrackListTypeSearch,
				tracks:        tracks,
				opts:          nil,
			},
			want: want{
				typ:           TrackListTypeSearch,
				trackCount:    2,
				hasIdentifier: false,
				hasName:       false,
				hasUrl:        false,
			},
		},
		{
			name: "with playlist info",
			args: args{
				trackListType: TrackListTypePlaylist,
				tracks:        tracks,
				opts: []TrackListOption{
					WithPlaylistInfo("PL123", "My Playlist", "https://example.com/playlist"),
				},
			},
			want: want{
				typ:           TrackListTypePlaylist,
				trackCount:    2,
				hasIdentifier: true,
				identifier:    "PL123",
				hasName:       true,
				name:          "My Playlist",
				hasUrl:        true,
				url:           "https://example.com/playlist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := NewTrackList(tt.args.trackListType, tt.args.tracks, tt.args.opts...)

			if tl.Type != tt.want.typ {
				t.Errorf("Type: got %v, want %v", tl.Type, tt.want.typ)
			}
			if len(tl.Tracks) != tt.want.trackCount {
				t.Errorf("Tracks len: got %d, want %d", len(tl.Tracks), tt.want.trackCount)
			}
			if tt.want.hasIdentifier {
				if tl.Identifier == nil || *tl.Identifier != tt.want.identifier {
					t.Errorf("Identifier: got %v, want %q", tl.Identifier, tt.want.identifier)
				}
			} else if tl.Identifier != nil {
				t.Error("Identifier should be nil")
			}
			if tt.want.hasName {
				if tl.Name == nil || *tl.Name != tt.want.name {
					t.Errorf("Name: got %v, want %q", tl.Name, tt.want.name)
				}
			} else if tl.Name != nil {
				t.Error("Name should be nil")
			}
			if tt.want.hasUrl {
				if tl.Url == nil || *tl.Url != tt.want.url {
					t.Errorf("Url: got %v, want %q", tl.Url, tt.want.url)
				}
			} else if tl.Url != nil {
				t.Error("Url should be nil")
			}
		})
	}
}
