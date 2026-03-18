package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestShuffleQueueUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type want struct {
		eventCount int
		err        error
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
			name: "shuffle non-empty queue succeeds",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2", "3") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{eventCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			events := &stubEventPublisher{}

			uc := NewShuffleQueueUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				events,
				tt.deps.locator(state.ID()),
			)

			_, err := uc.Execute(context.Background(), ShuffleQueueInput[string]{
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
			if len(events.published) != tt.want.eventCount {
				t.Errorf("events: got %d, want %d", len(events.published), tt.want.eventCount)
			}
		})
	}
}
