package domain

import (
	"testing"
	"time"
)

func TestParseTrackID(t *testing.T) {
	type args struct {
		id string
	}
	type want struct {
		trackID TrackID
		err     error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid ID",
			args: args{id: "dQw4w9WgXcQ"},
			want: want{trackID: TrackID("dQw4w9WgXcQ"), err: nil},
		},
		{
			name: "empty ID returns error",
			args: args{id: ""},
			want: want{trackID: "", err: ErrEmptyTrackID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTrackID(tt.args.id)
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if got != tt.want.trackID {
				t.Fatalf("id: got %q, want %q", got, tt.want.trackID)
			}
		})
	}
}

func TestTrackID_String(t *testing.T) {
	type args struct {
		id TrackID
	}
	type want struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "returns underlying string",
			args: args{id: TrackID("abc123")},
			want: want{str: "abc123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.id.String(); got != tt.want.str {
				t.Fatalf("got %q, want %q", got, tt.want.str)
			}
		})
	}
}

func TestParseTrackSource(t *testing.T) {
	type args struct {
		name string
	}
	type want struct {
		source TrackSource
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "youtube", args: args{name: "youtube"}, want: want{source: TrackSourceYouTube}},
		{name: "spotify", args: args{name: "spotify"}, want: want{source: TrackSourceSpotify}},
		{
			name: "soundcloud",
			args: args{name: "soundcloud"},
			want: want{source: TrackSourceSoundCloud},
		},
		{name: "twitch", args: args{name: "twitch"}, want: want{source: TrackSourceTwitch}},
		{name: "unknown string", args: args{name: "unknown"}, want: want{source: TrackSourceOther}},
		{name: "empty string", args: args{name: ""}, want: want{source: TrackSourceOther}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseTrackSource(tt.args.name); got != tt.want.source {
				t.Fatalf("got %q, want %q", got, tt.want.source)
			}
		})
	}
}

func TestTrackSource_String(t *testing.T) {
	type args struct {
		source TrackSource
	}
	type want struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "youtube", args: args{source: TrackSourceYouTube}, want: want{str: "youtube"}},
		{name: "spotify", args: args{source: TrackSourceSpotify}, want: want{str: "spotify"}},
		{
			name: "soundcloud",
			args: args{source: TrackSourceSoundCloud},
			want: want{str: "soundcloud"},
		},
		{name: "twitch", args: args{source: TrackSourceTwitch}, want: want{str: "twitch"}},
		{name: "other", args: args{source: TrackSourceOther}, want: want{str: "other"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.source.String(); got != tt.want.str {
				t.Fatalf("got %q, want %q", got, tt.want.str)
			}
		})
	}
}

func TestConstructTrack(t *testing.T) {
	type args struct {
		id         TrackID
		title      string
		author     string
		duration   time.Duration
		url        string
		artworkURL string
		source     TrackSource
		isStream   bool
	}
	type want struct {
		id         TrackID
		title      string
		author     string
		duration   time.Duration
		url        string
		artworkURL string
		source     TrackSource
		isStream   bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "regular track",
			args: args{
				id: TrackID("id1"), title: "Test Title", author: "Test Author",
				duration: 3 * time.Minute, url: "https://example.com",
				artworkURL: "https://example.com/art.jpg",
				source:     TrackSourceYouTube, isStream: false,
			},
			want: want{
				id: TrackID("id1"), title: "Test Title", author: "Test Author",
				duration: 3 * time.Minute, url: "https://example.com",
				artworkURL: "https://example.com/art.jpg",
				source:     TrackSourceYouTube, isStream: false,
			},
		},
		{
			name: "live stream",
			args: args{
				id: TrackID("stream1"), title: "Live Stream", author: "Streamer",
				duration: 0, url: "https://twitch.tv/streamer",
				artworkURL: "", source: TrackSourceTwitch, isStream: true,
			},
			want: want{
				id: TrackID("stream1"), title: "Live Stream", author: "Streamer",
				duration: 0, url: "https://twitch.tv/streamer",
				artworkURL: "", source: TrackSourceTwitch, isStream: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := ConstructTrack(
				tt.args.id, tt.args.title, tt.args.author, tt.args.duration,
				tt.args.url, tt.args.artworkURL, tt.args.source, tt.args.isStream,
			)

			if track.ID() != tt.want.id {
				t.Errorf("ID: got %q, want %q", track.ID(), tt.want.id)
			}
			if track.Title() != tt.want.title {
				t.Errorf("Title: got %q, want %q", track.Title(), tt.want.title)
			}
			if track.Author() != tt.want.author {
				t.Errorf("Author: got %q, want %q", track.Author(), tt.want.author)
			}
			if track.Duration() != tt.want.duration {
				t.Errorf("Duration: got %v, want %v", track.Duration(), tt.want.duration)
			}
			if track.URL() != tt.want.url {
				t.Errorf("URL: got %q, want %q", track.URL(), tt.want.url)
			}
			if track.ArtworkURL() != tt.want.artworkURL {
				t.Errorf("ArtworkURL: got %q, want %q", track.ArtworkURL(), tt.want.artworkURL)
			}
			if track.Source() != tt.want.source {
				t.Errorf("Source: got %q, want %q", track.Source(), tt.want.source)
			}
			if track.IsStream() != tt.want.isStream {
				t.Errorf("IsStream: got %v, want %v", track.IsStream(), tt.want.isStream)
			}
		})
	}
}
