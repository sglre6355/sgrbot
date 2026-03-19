package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestResolveQueryUsecase_Execute(t *testing.T) {
	type deps struct {
		resolver *stubTrackResolver
	}
	type args struct {
		query string
		limit int
	}
	type want struct {
		trackCount int
		firstID    string
		err        error
	}

	tests := []struct {
		name string
		deps deps
		args args
		want want
	}{
		{
			name: "successful resolution returns tracks",
			deps: deps{resolver: &stubTrackResolver{
				result: domain.NewTrackList(
					domain.TrackListTypeSearch,
					[]domain.Track{newTestTrack("t1")},
				),
			}},
			args: args{query: "test"},
			want: want{trackCount: 1, firstID: "t1"},
		},
		{
			name: "no results returns ErrNoResults",
			deps: deps{resolver: &stubTrackResolver{
				result: domain.NewTrackList(domain.TrackListTypeSearch, []domain.Track{}),
			}},
			args: args{query: "nothing"},
			want: want{err: ErrNoResults},
		},
		{
			name: "limit truncates search results",
			deps: deps{resolver: &stubTrackResolver{
				result: domain.NewTrackList(
					domain.TrackListTypeSearch,
					[]domain.Track{newTestTrack("t1"), newTestTrack("t2"), newTestTrack("t3")},
				),
			}},
			args: args{query: "test", limit: 1},
			want: want{trackCount: 1, firstID: "t1"},
		},
		{
			name: "zero limit returns all results",
			deps: deps{resolver: &stubTrackResolver{
				result: domain.NewTrackList(
					domain.TrackListTypeSearch,
					[]domain.Track{newTestTrack("t1"), newTestTrack("t2"), newTestTrack("t3")},
				),
			}},
			args: args{query: "test", limit: 0},
			want: want{trackCount: 3, firstID: "t1"},
		},
		{
			name: "limit does not truncate when results are within limit",
			deps: deps{resolver: &stubTrackResolver{
				result: domain.NewTrackList(
					domain.TrackListTypeSearch,
					[]domain.Track{newTestTrack("t1")},
				),
			}},
			args: args{query: "test", limit: 5},
			want: want{trackCount: 1, firstID: "t1"},
		},
		{
			name: "resolver error returns ErrInternal",
			deps: deps{resolver: &stubTrackResolver{
				err: errors.New("connection failed"),
			}},
			args: args{query: "test"},
			want: want{err: ErrInternal},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewResolveQueryUsecase(tt.deps.resolver)

			out, err := uc.Execute(
				context.Background(),
				ResolveQueryInput{Query: tt.args.query, Limit: tt.args.limit},
			)

			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err: got %v, want %v", err, tt.want.err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Tracks) != tt.want.trackCount {
				t.Errorf("Tracks: got %d, want %d", len(out.Tracks), tt.want.trackCount)
			}
			if out.Tracks[0].ID != tt.want.firstID {
				t.Errorf("Track ID: got %q, want %q", out.Tracks[0].ID, tt.want.firstID)
			}
		})
	}
}
