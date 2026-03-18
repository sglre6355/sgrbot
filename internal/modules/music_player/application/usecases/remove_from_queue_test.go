package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestRemoveFromQueueUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		index int
	}
	type want struct {
		removedID string
		err       error
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
			name: "remove current track returns ErrIsCurrentTrack",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{index: 0},
			want: want{err: ErrIsCurrentTrack},
		},
		{
			name: "remove non-current track succeeds",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{index: 1},
			want: want{removedID: "2"},
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
			events := &stubEventPublisher{}

			uc := NewRemoveFromQueueUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				events,
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), RemoveFromQueueInput[string]{
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
			if out.RemovedTrack.ID != tt.want.removedID {
				t.Errorf("RemovedTrack.ID: got %q, want %q", out.RemovedTrack.ID, tt.want.removedID)
			}
			if len(events.published) == 0 {
				t.Error("expected events to be published")
			}
		})
	}
}
