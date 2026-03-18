package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestSetLoopModeUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		mode string
	}
	type want struct {
		changed bool
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
			args: args{mode: "track"},
			want: want{err: ErrNotConnected},
		},
		{
			name: "set different mode reports changed",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{mode: "track"},
			want: want{changed: true},
		},
		{
			name: "set same mode reports not changed",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{mode: "none"},
			want: want{changed: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()

			uc := NewSetLoopModeUsecase[string](
				newStubPlayerStateRepo(state),
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), SetLoopModeInput[string]{
				ConnectionInfo: "guild1",
				Mode:           tt.args.mode,
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
			if out.Changed != tt.want.changed {
				t.Errorf("Changed: got %v, want %v", out.Changed, tt.want.changed)
			}
		})
	}
}
