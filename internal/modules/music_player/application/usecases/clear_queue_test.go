package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestClearQueueUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		keepCurrent bool
	}
	type want struct {
		clearedCount int
		stopped      int
		err          error
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
			want: want{err: ErrNotConnected},
		},
		{
			name: "empty queue returns ErrQueueEmpty",
			deps: deps{
				state: newIdleState,
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{err: ErrQueueEmpty},
		},
		{
			name: "clear all stops playback",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{keepCurrent: false},
			want: want{clearedCount: 2, stopped: 1},
		},
		{
			name: "keep current track clears others",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2", "3") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{keepCurrent: true},
			want: want{clearedCount: 2, stopped: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			audio := &stubAudioGateway{}
			events := &stubEventPublisher{}

			uc := NewClearQueueUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				audio,
				events,
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), ClearQueueInput[string]{
				ConnectionInfo:   "guild1",
				KeepCurrentTrack: tt.args.keepCurrent,
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
			if out.ClearedCount != tt.want.clearedCount {
				t.Errorf("ClearedCount: got %d, want %d", out.ClearedCount, tt.want.clearedCount)
			}
			if len(audio.stopped) != tt.want.stopped {
				t.Errorf("Stop calls: got %d, want %d", len(audio.stopped), tt.want.stopped)
			}
		})
	}
}
