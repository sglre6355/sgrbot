package domain

import (
	"context"
	"errors"
	"testing"
)

func TestAutoPlayService_GetNextRecommendation(t *testing.T) {
	type args struct {
		recommender *stubRecommender
		trackIDs    []string
	}
	type want struct {
		trackID    string
		isAutoPlay bool
		requester  UserID
		err        error
		hasErr     bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "returns recommendation from seeds",
			args: args{
				recommender: &stubRecommender{
					tracks:     []Track{newTestTrack("recommended")},
					acceptSeed: true,
				},
				trackIDs: []string{"1", "2"},
			},
			want: want{trackID: "recommended", isAutoPlay: true, requester: UserID("bot")},
		},
		{
			name: "no acceptable seeds",
			args: args{
				recommender: &stubRecommender{acceptSeed: false},
				trackIDs:    []string{"1"},
			},
			want: want{err: ErrNoAutoPlaySeeds, hasErr: true},
		},
		{
			name: "recommender returns empty",
			args: args{
				recommender: &stubRecommender{tracks: []Track{}, acceptSeed: true},
				trackIDs:    []string{"1"},
			},
			want: want{err: ErrNoRecommendations, hasErr: true},
		},
		{
			name: "recommender returns error",
			args: args{
				recommender: &stubRecommender{err: errors.New("network error"), acceptSeed: true},
				trackIDs:    []string{"1"},
			},
			want: want{hasErr: true},
		},
		{
			name: "empty queue returns no seeds error",
			args: args{
				recommender: &stubRecommender{acceptSeed: true},
				trackIDs:    nil,
			},
			want: want{err: ErrNoAutoPlaySeeds, hasErr: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAutoPlayService(UserID("bot"), tt.args.recommender)
			var ps *PlayerState
			if tt.args.trackIDs == nil {
				ps = NewPlayerState()
			} else {
				ps = newActiveState(tt.args.trackIDs...)
			}

			entry, err := svc.GetNextRecommendation(context.Background(), ps)

			if tt.want.hasErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.want.err != nil && !errors.Is(err, tt.want.err) {
					t.Fatalf("err: got %v, want %v", err, tt.want.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(entry.Track().ID()) != tt.want.trackID {
				t.Errorf("track: got %q, want %q", entry.Track().ID(), tt.want.trackID)
			}
			if entry.IsAutoPlay() != tt.want.isAutoPlay {
				t.Errorf("IsAutoPlay: got %v, want %v", entry.IsAutoPlay(), tt.want.isAutoPlay)
			}
			if entry.RequesterID() != tt.want.requester {
				t.Errorf("requester: got %q, want %q", entry.RequesterID(), tt.want.requester)
			}
		})
	}
}
