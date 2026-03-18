package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestSeekQueueUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		index int
	}
	type want struct {
		trackID string
		played  int
		err     error
	}

	tests := []struct {
		name string
		deps deps
		args args
		want want
	}{
		{
			name: "not connected returns ErrNotConnected",
			deps: deps{
				state:   newIdleState,
				locator: func(_ domain.PlayerStateID) *stubPlayerStateLocator { return newStubLocatorNil() },
			},
			args: args{index: 0},
			want: want{err: ErrNotConnected},
		},
		{
			name: "valid seek plays track at index",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2", "3") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{index: 2},
			want: want{trackID: "3", played: 1},
		},
		{
			name: "invalid index returns ErrInvalidIndex",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{index: 99},
			want: want{err: ErrInvalidIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			audio := &stubAudioGateway{}
			events := &stubEventPublisher{}

			uc := NewSeekQueueUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				audio,
				events,
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), SeekQueueInput[string]{
				ConnectionInfo: "guild1",
				Index:          tt.args.index,
			})

			if tt.want.err != nil {
				if !errors.Is(err, tt.want.err) {
					t.Fatalf("err: got %v, want %v", err, tt.want.err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Track.ID != tt.want.trackID {
				t.Errorf("Track.ID: got %q, want %q", out.Track.ID, tt.want.trackID)
			}
			if len(audio.playedEntries) != tt.want.played {
				t.Errorf("Play calls: got %d, want %d", len(audio.playedEntries), tt.want.played)
			}
		})
	}
}
