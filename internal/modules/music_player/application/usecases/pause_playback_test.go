package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestPausePlaybackUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func(*testing.T) domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type want struct {
		paused int
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
				state:   func(*testing.T) domain.PlayerState { return newIdleState() },
				locator: func(_ domain.PlayerStateID) *stubPlayerStateLocator { return newStubLocatorNil() },
			},
			want: want{err: ErrNotConnected},
		},
		{
			name: "pause active playback succeeds",
			deps: deps{
				state: func(*testing.T) domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{paused: 1},
		},
		{
			name: "pause already paused returns ErrAlreadyPaused",
			deps: deps{
				state: func(t *testing.T) domain.PlayerState {
					t.Helper()
					s := newActiveState("1")
					if err := s.Pause(); err != nil {
						t.Fatalf("setup Pause: %v", err)
					}
					return s
				},
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{err: ErrAlreadyPaused},
		},
		{
			name: "pause idle returns ErrNotPlaying",
			deps: deps{
				state: func(*testing.T) domain.PlayerState { return newIdleState() },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{err: ErrNotPlaying},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state(t)
			audio := &stubAudioGateway{}
			events := &stubEventPublisher{}

			uc := NewPausePlaybackUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				audio,
				events,
				tt.deps.locator(state.ID()),
			)

			_, err := uc.Execute(context.Background(), PausePlaybackInput[string]{
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
			if len(audio.paused) != tt.want.paused {
				t.Errorf("Pause calls: got %d, want %d", len(audio.paused), tt.want.paused)
			}
			if len(events.published) == 0 {
				t.Error("expected at least one event to be published")
			}
		})
	}
}
