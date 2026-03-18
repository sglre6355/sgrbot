package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestCycleLoopModeUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type want struct {
		newMode string
		err     error
	}

	tests := []struct {
		name string
		deps deps
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
			name: "cycles none to track",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{newMode: "track"},
		},
		{
			name: "cycles track to queue",
			deps: deps{
				state: func() domain.PlayerState {
					s := newActiveState("1")
					s.SetLoopMode(domain.LoopModeTrack)
					return s
				},
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{newMode: "queue"},
		},
		{
			name: "cycles queue to none",
			deps: deps{
				state: func() domain.PlayerState {
					s := newActiveState("1")
					s.SetLoopMode(domain.LoopModeQueue)
					return s
				},
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{newMode: "none"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()

			uc := NewCycleLoopModeUsecase[string](
				newStubPlayerStateRepo(state),
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), CycleLoopModeInput[string]{
				ConnectionInfo: "guild1",
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
			if out.NewMode != tt.want.newMode {
				t.Errorf("NewMode: got %q, want %q", out.NewMode, tt.want.newMode)
			}
		})
	}
}
