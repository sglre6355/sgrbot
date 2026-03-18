package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestRestartQueueUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type want struct {
		played int
		err    error
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
			name: "restart seeks to first track and plays",
			deps: deps{
				state: func() domain.PlayerState {
					s := newActiveState("1", "2", "3")
					s.Seek(2)
					return s
				},
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{played: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			audio := &stubAudioGateway{}
			events := &stubEventPublisher{}

			uc := NewRestartQueueUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				audio,
				events,
				tt.deps.locator(state.ID()),
			)

			_, err := uc.Execute(context.Background(), RestartQueueInput[string]{
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
			if len(audio.playedEntries) != tt.want.played {
				t.Errorf("Play calls: got %d, want %d", len(audio.playedEntries), tt.want.played)
			}
			if len(events.published) == 0 {
				t.Error("expected events to be published")
			}
		})
	}
}
