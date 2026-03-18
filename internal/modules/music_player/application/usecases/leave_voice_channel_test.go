package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func TestLeaveVoiceChannelUsecase_Execute(t *testing.T) {
	type deps struct {
		state   func() domain.PlayerState
		locator func(domain.PlayerStateID) *stubPlayerStateLocator
	}
	type want struct {
		left    int
		cleared int
		deleted bool
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
			name: "leaves voice and cleans up state",
			deps: deps{
				state: func() domain.PlayerState { return newActiveState("1") },
				locator: func(id domain.PlayerStateID) *stubPlayerStateLocator {
					return newStubLocator(map[string]domain.PlayerStateID{"guild1": id})
				},
			},
			want: want{left: 1, cleared: 1, deleted: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.deps.state()
			repo := newStubPlayerStateRepo(state)
			voice := &stubVoiceConnectionGateway{}
			nowPlaying := &stubNowPlayingPublisher{}

			uc := NewLeaveVoiceChannelUsecase[string, string](
				repo,
				nowPlaying,
				tt.deps.locator(state.ID()),
				voice,
			)

			_, err := uc.Execute(context.Background(), LeaveVoiceChannelInput[string]{
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
			if len(voice.left) != tt.want.left {
				t.Errorf("Leave calls: got %d, want %d", len(voice.left), tt.want.left)
			}
			if len(nowPlaying.cleared) != tt.want.cleared {
				t.Errorf(
					"NowPlaying.Clear calls: got %d, want %d",
					len(nowPlaying.cleared),
					tt.want.cleared,
				)
			}
			if tt.want.deleted && len(repo.states) != 0 {
				t.Errorf("states remaining: got %d, want 0", len(repo.states))
			}
		})
	}
}
