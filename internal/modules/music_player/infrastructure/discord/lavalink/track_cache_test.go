package lavalink

import (
	"testing"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestTrackCache_Get(t *testing.T) {
	type args struct {
		setup func(cache *TrackCache)
		getID domain.TrackID
	}
	type want struct {
		ok    bool
		title string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "returns cached track",
			args: args{
				setup: func(cache *TrackCache) {
					track := domain.ConstructTrack(
						domain.TrackID("abc"), "Title", "Author", time.Minute,
						"https://example.com", "", domain.TrackSourceYouTube, false,
					)
					cache.Set(domain.TrackID("abc"), track)
				},
				getID: domain.TrackID("abc"),
			},
			want: want{ok: true, title: "Title"},
		},
		{
			name: "cache miss returns false",
			args: args{
				setup: func(_ *TrackCache) {},
				getID: domain.TrackID("nonexistent"),
			},
			want: want{ok: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTrackCache()
			tt.args.setup(cache)

			got, ok := cache.Get(tt.args.getID)

			if ok != tt.want.ok {
				t.Fatalf("ok: got %v, want %v", ok, tt.want.ok)
			}
			if ok && got.Title() != tt.want.title {
				t.Errorf("Title: got %q, want %q", got.Title(), tt.want.title)
			}
		})
	}
}

func TestTrackCache_SetOverwrites(t *testing.T) {
	type args struct {
		id     domain.TrackID
		titles []string
	}
	type want struct {
		title string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "second set overwrites first",
			args: args{id: domain.TrackID("abc"), titles: []string{"First", "Second"}},
			want: want{title: "Second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewTrackCache()

			for _, title := range tt.args.titles {
				track := domain.ConstructTrack(
					tt.args.id,
					title,
					"A",
					time.Minute,
					"",
					"",
					domain.TrackSourceYouTube,
					false,
				)
				cache.Set(tt.args.id, track)
			}

			got, ok := cache.Get(tt.args.id)
			if !ok {
				t.Fatal("expected ok=true")
			}
			if got.Title() != tt.want.title {
				t.Errorf("Title: got %q, want %q", got.Title(), tt.want.title)
			}
		})
	}
}
