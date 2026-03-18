package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestSetNowPlayingDestinationUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type args struct {
		destination string
	}
	type want struct {
		destination string
		err         error
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
			args: args{destination: "channel1"},
			want: want{err: ErrNotConnected},
		},
		{
			name: "sets destination successfully",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			args: args{destination: "channel1"},
			want: want{destination: "channel1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			setter := &stubNowPlayingDestinationSetter{}

			uc := NewSetNowPlayingDestinationUsecase[string, string](
				setter,
				tt.deps.locator(state.ID()),
			)

			_, err := uc.Execute(
				context.Background(),
				SetNowPlayingDestinationInput[string, string]{
					ConnectionInfo:        "guild1",
					NowPlayingDestination: tt.args.destination,
				},
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
			if dest, ok := setter.destinations[state.ID()]; !ok || dest != tt.want.destination {
				t.Errorf("destination: got %q, want %q", dest, tt.want.destination)
			}
		})
	}
}
