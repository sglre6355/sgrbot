package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestSkipTrackUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type want struct {
		skippedID string
		played    int
		stopped   int
		err       error
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
			name: "skip with next track plays next",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1", "2") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{skippedID: "1", played: 1},
		},
		{
			name: "skip last track stops playback",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{skippedID: "1", stopped: 1},
		},
		{
			name: "skip idle returns ErrNotPlaying",
			deps: deps{
				state: newIdleState,
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{err: ErrNotPlaying},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			audio := &stubAudioGateway{}
			events := &stubEventPublisher{}

			uc := NewSkipTrackUsecase[string](
				newPlayerService(),
				newStubPlayerStateRepo(state),
				audio,
				events,
				tt.deps.locator(state.ID()),
			)

			out, err := uc.Execute(context.Background(), SkipTrackInput[string]{
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
			if out.SkippedTrack.ID != tt.want.skippedID {
				t.Errorf("SkippedTrack.ID: got %q, want %q", out.SkippedTrack.ID, tt.want.skippedID)
			}
			if len(audio.playedEntries) != tt.want.played {
				t.Errorf("Play calls: got %d, want %d", len(audio.playedEntries), tt.want.played)
			}
			if len(audio.stopped) != tt.want.stopped {
				t.Errorf("Stop calls: got %d, want %d", len(audio.stopped), tt.want.stopped)
			}
		})
	}
}
